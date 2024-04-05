package txmanager

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/0xPolygon/agglayer/config"
	"github.com/0xPolygon/agglayer/log"
	txmTypes "github.com/0xPolygon/agglayer/txmanager/types"
	aggLayerTypes "github.com/0xPolygon/agglayer/types"
	"github.com/0xPolygonHermez/zkevm-node/state"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
)

const failureIntervalInSeconds = 5

// Client for eth tx manager
type Client struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg      config.EthTxManagerConfig
	etherman aggLayerTypes.IEtherman
	storage  txmTypes.StorageInterface
	state    txmTypes.StateInterface
}

// New creates new eth tx manager
func New(cfg config.EthTxManagerConfig, ethMan aggLayerTypes.IEtherman, storage txmTypes.StorageInterface, state txmTypes.StateInterface) *Client {
	c := &Client{
		cfg:      cfg,
		etherman: ethMan,
		storage:  storage,
		state:    state,
	}

	return c
}

// Start will start the tx management, reading txs from storage,
// send then to the blockchain and keep monitoring them until they
// get mined
func (c *Client) Start() {
	// infinite loop to manage txs as they arrive
	c.ctx, c.cancel = context.WithCancel(context.Background())

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(c.cfg.FrequencyToMonitorTxs.Duration):
			err := c.monitorTxs(context.Background())
			if err != nil {
				c.logErrorAndWait("failed to monitor txs: %v", err)
			}
		}
	}
}

// Stop will stops the monitored tx management
func (c *Client) Stop() {
	c.cancel()
}

// Add a transaction to be sent and monitored
func (c *Client) Add(ctx context.Context, owner, id string, from common.Address, to *common.Address, value *big.Int, data []byte, gasOffset uint64, dbTx pgx.Tx) error {
	// get nonce
	nonce, err := c.getTxNonce(ctx, from)
	if err != nil {
		err := fmt.Errorf("failed to get nonce: %w", err)
		log.Errorf(err.Error())
		return err
	}

	// get gas
	gas, err := c.etherman.EstimateGas(ctx, from, to, value, data)
	if err != nil {
		err := fmt.Errorf("failed to estimate gas: %w, data: %v", err, common.Bytes2Hex(data))
		log.Error(err.Error())
		if c.cfg.ForcedGas > 0 {
			gas = c.cfg.ForcedGas
		} else {
			return err
		}
	}

	// get gas price
	gasPrice, err := c.suggestedGasPrice(ctx)
	if err != nil {
		err := fmt.Errorf("failed to get suggested gas price: %w", err)
		log.Errorf(err.Error())
		return err
	}

	// create monitored tx
	mTx := txmTypes.MonitoredTx{
		Owner:     owner,
		ID:        id,
		From:      from,
		To:        to,
		Nonce:     nonce,
		Value:     value,
		Data:      data,
		Gas:       gas,
		GasOffset: gasOffset,
		GasPrice:  gasPrice,
		Status:    txmTypes.MonitoredTxStatusCreated,
	}

	// add to storage
	err = c.storage.Add(ctx, mTx, dbTx)
	if err != nil {
		err := fmt.Errorf("failed to add tx to get monitored: %w", err)
		log.Errorf(err.Error())
		return err
	}

	mTxLog := log.WithFields("monitoredTx", mTx.ID, "createdAt", mTx.CreatedAt)
	mTxLog.Infof("created")

	return nil
}

// Result returns the current result of the transaction execution with all the details
func (c *Client) Result(ctx context.Context, owner, id string, dbTx pgx.Tx) (txmTypes.MonitoredTxResult, error) {
	mTx, err := c.storage.Get(ctx, owner, id, dbTx)
	if err != nil {
		return txmTypes.MonitoredTxResult{}, err
	}

	return c.buildResult(ctx, mTx)
}

func (c *Client) buildResult(ctx context.Context, mTx txmTypes.MonitoredTx) (txmTypes.MonitoredTxResult, error) {
	history := mTx.HistoryHashSlice()
	txs := make(map[common.Hash]txmTypes.TxResult, len(history))

	for _, txHash := range history {
		tx, _, err := c.etherman.GetTx(ctx, txHash)
		if !errors.Is(err, ethereum.NotFound) && err != nil {
			return txmTypes.MonitoredTxResult{}, err
		}

		receipt, err := c.etherman.GetTxReceipt(ctx, txHash)
		if !errors.Is(err, ethereum.NotFound) && err != nil {
			return txmTypes.MonitoredTxResult{}, err
		}

		revertMessage, err := c.etherman.GetRevertMessage(ctx, tx)
		if !errors.Is(err, ethereum.NotFound) && err != nil && err.Error() != txmTypes.ErrExecutionReverted.Error() {
			return txmTypes.MonitoredTxResult{}, err
		}

		txs[txHash] = txmTypes.TxResult{
			Tx:            tx,
			Receipt:       receipt,
			RevertMessage: revertMessage,
		}
	}

	result := txmTypes.MonitoredTxResult{
		ID:     mTx.ID,
		Status: mTx.Status,
		Txs:    txs,
	}

	return result, nil
}

// monitorTxs process all pending monitored tx
func (c *Client) monitorTxs(ctx context.Context) error {
	statusesFilter := []txmTypes.MonitoredTxStatus{txmTypes.MonitoredTxStatusCreated, txmTypes.MonitoredTxStatusSent, txmTypes.MonitoredTxStatusReorged}
	mTxs, err := c.storage.GetByStatus(ctx, nil, statusesFilter, nil)
	if err != nil {
		return fmt.Errorf("failed to get created monitored txs: %v", err)
	}

	log.Infof("found %v monitored tx to process", len(mTxs))

	wg := sync.WaitGroup{}
	wg.Add(len(mTxs))
	for _, mTx := range mTxs {
		if mTx.NumRetries == 0 {
			// this is only done for old monitored txs that were not updated before this fix
			mTx.NumRetries = uint64(len(mTx.History))
		}

		mTx := mTx // force variable shadowing to avoid pointer conflicts
		go func(c *Client, mTx txmTypes.MonitoredTx) {
			mTxLogger := createMonitoredTxLogger(mTx)
			defer func(mTx txmTypes.MonitoredTx, mTxLogger *zap.SugaredLogger) {
				if err := recover(); err != nil {
					mTxLogger.Error("monitoring recovered from this err: %v", err)
				}
				wg.Done()
			}(mTx, mTxLogger)
			c.monitorTx(ctx, mTx, mTxLogger)
		}(c, mTx)
	}
	wg.Wait()

	return nil
}

// monitorTx does all the monitoring steps to the monitored tx
func (c *Client) monitorTx(ctx context.Context, mTx txmTypes.MonitoredTx, logger *zap.SugaredLogger) {
	var err error
	logger.Info("processing")
	// check if any of the txs in the history was confirmed
	var lastReceiptChecked types.Receipt
	// monitored tx is confirmed until we find a successful receipt
	confirmed := false
	// monitored tx doesn't have a failed receipt until we find a failed receipt for any
	// tx in the monitored tx history
	hasFailedReceipts := false
	// all history txs are considered mined until we can't find a receipt for any
	// tx in the monitored tx history
	allHistoryTxsWereMined := true
	for txHash := range mTx.History {
		mined, receipt, err := c.etherman.CheckTxWasMined(ctx, txHash)
		if err != nil {
			logger.Errorf("failed to check if tx %v was mined: %v", txHash.String(), err)
			continue
		}

		// if the tx is not mined yet, check that not all the tx were mined and go to the next
		if !mined {
			allHistoryTxsWereMined = false
			continue
		}

		lastReceiptChecked = *receipt

		// if the tx was mined successfully we can set it as confirmed and break the loop
		if lastReceiptChecked.Status == types.ReceiptStatusSuccessful {
			confirmed = true
			break
		}

		// if the tx was mined but failed, we continue to consider it was not confirmed
		// and set that we have found a failed receipt. This info will be used later
		// to check if nonce needs to be reviewed
		confirmed = false
		hasFailedReceipts = true
	}

	// we need to check if we need to review the nonce carefully, to avoid sending
	// duplicated data to the roll-up and causing an unnecessary trusted state reorg.
	//
	// if we have failed receipts, this means at least one of the generated txs was mined,
	// in this case maybe the current nonce was already consumed(if this is the first iteration
	// of this cycle, next iteration might have the nonce already updated by the preivous one),
	// then we need to check if there are tx that were not mined yet, if so, we just need to wait
	// because maybe one of them will get mined successfully
	//
	// in case of the monitored tx is not confirmed yet, all tx were mined and none of them were
	// mined successfully, we need to review the nonce
	if !confirmed && hasFailedReceipts && allHistoryTxsWereMined {
		logger.Infof("nonce needs to be updated")
		err := c.reviewMonitoredTxNonce(ctx, &mTx, logger)
		if err != nil {
			logger.Errorf("failed to review monitored tx nonce: %v", err)
			return
		}
		err = c.storage.Update(ctx, mTx, nil)
		if err != nil {
			logger.Errorf("failed to update monitored tx nonce change: %v", err)
			return
		}
	}

	// if num of retires reaches the max retry limit, this means something is really wrong with
	// this Tx and we are not able to identify automatically, so we mark this as failed to let the
	// caller know something is not right and needs to be review and to avoid to monitor this
	// tx infinitely
	if mTx.NumRetries >= c.cfg.MaxRetries {
		mTx.Status = txmTypes.MonitoredTxStatusFailed
		logger.Infof("marked as failed because reached the num of retires limit: %v", err)
		// update monitored tx changes into storage
		err = c.storage.Update(ctx, mTx, nil)
		if err != nil {
			logger.Errorf("failed to update monitored tx when num of retires reached: %v", err)
		}

		return
	}

	var signedTx *types.Transaction
	if !confirmed {
		// if is a reorged, move to the next
		if mTx.Status == txmTypes.MonitoredTxStatusReorged {
			return
		}

		// review tx and increase gas and gas price if needed
		if mTx.Status == txmTypes.MonitoredTxStatusSent {
			err := c.reviewMonitoredTx(ctx, &mTx, logger)
			if err != nil {
				logger.Errorf("failed to review monitored tx: %v", err)
				mTx.NumRetries++

				// update numRetries and return
				if err := c.storage.Update(ctx, mTx, nil); err != nil {
					logger.Errorf("failed to update monitored tx review change: %v", err)
				}

				return
			}

			if err := c.storage.Update(ctx, mTx, nil); err != nil {
				logger.Errorf("failed to update monitored tx review change: %v", err)
				return
			}
		}

		// rebuild transaction
		tx := mTx.Tx()
		logger.Debugf("unsigned tx %v created", tx.Hash().String())

		// sign tx
		signedTx, err = c.etherman.SignTx(ctx, mTx.From, tx)
		if err != nil {
			logger.Errorf("failed to sign tx %v: %v", tx.Hash().String(), err)
			return
		}
		logger.Debugf("signed tx %v created", signedTx.Hash().String())

		// add tx to monitored tx history
		err = mTx.AddHistory(signedTx)
		if errors.Is(err, txmTypes.ErrAlreadyExists) {
			logger.Infof("signed tx already existed in the history")
		} else if err != nil {
			logger.Errorf("failed to add signed tx %v to monitored tx history: %v", signedTx.Hash().String(), err)
			return
		} else {
			// update monitored tx changes into storage
			err = c.storage.Update(ctx, mTx, nil)
			if err != nil {
				logger.Errorf("failed to update monitored tx: %v", err)
				return
			}
			logger.Debugf("signed tx added to the monitored tx history")
		}

		// check if the tx is already in the network, if not, send it
		_, _, err = c.etherman.GetTx(ctx, signedTx.Hash())
		// if not found, send it tx to the network
		if errors.Is(err, ethereum.NotFound) {
			logger.Debugf("signed tx not found in the network")
			err := c.etherman.SendTx(ctx, signedTx)
			if err != nil {
				logger.Errorf("failed to send tx %v to network: %v", signedTx.Hash().String(), err)
				return
			}
			logger.Infof("signed tx sent to the network: %v", signedTx.Hash().String())
			if mTx.Status == txmTypes.MonitoredTxStatusCreated {
				// update tx status to sent
				mTx.Status = txmTypes.MonitoredTxStatusSent
				logger.Debugf("status changed to %v", string(mTx.Status))
				// update monitored tx changes into storage
				err = c.storage.Update(ctx, mTx, nil)
				if err != nil {
					logger.Errorf("failed to update monitored tx changes: %v", err)
					return
				}
			}
		} else {
			logger.Infof("signed tx already found in the network")
		}

		log.Infof("waiting signedTx to be mined...")

		// wait tx to get mined
		confirmed, err = c.etherman.WaitTxToBeMined(ctx, signedTx, c.cfg.WaitTxToBeMined.Duration)
		if err != nil {
			logger.Errorf("failed to wait tx to be mined: %v", err)
			return
		}
		if !confirmed {
			log.Infof("signedTx not mined yet and timeout has been reached")
			return
		}

		// get tx receipt
		var txReceipt *types.Receipt
		txReceipt, err = c.etherman.GetTxReceipt(ctx, signedTx.Hash())
		if err != nil {
			logger.Errorf("failed to get tx receipt for tx %v: %v", signedTx.Hash().String(), err)
			return
		}
		lastReceiptChecked = *txReceipt
	}

	// if mined, check receipt and mark as Failed or Confirmed
	if lastReceiptChecked.Status == types.ReceiptStatusSuccessful {
		receiptBlockNum := lastReceiptChecked.BlockNumber.Uint64()

		// check if state is already synchronized until the block
		// where the tx was mined
		block, err := c.state.GetLastBlock(ctx, nil)
		if errors.Is(err, state.ErrStateNotSynchronized) {
			logger.Debugf("state not synchronized yet, waiting for L1 block %v to be synced", receiptBlockNum)
			return
		} else if err != nil {
			logger.Errorf("failed to check if L1 block %v is already synced: %v", receiptBlockNum, err)
			return
		} else if block.BlockNumber < receiptBlockNum {
			logger.Debugf("L1 block %v not synchronized yet, waiting for L1 block to be synced in order to confirm monitored tx", receiptBlockNum)
			return
		} else {
			mTx.Status = txmTypes.MonitoredTxStatusConfirmed
			mTx.BlockNumber = lastReceiptChecked.BlockNumber
			logger.Info("confirmed")
		}
	} else {
		// if we should continue to monitor, we move to the next one and this will
		// be reviewed in the next monitoring cycle
		if c.shouldContinueToMonitorThisTx(ctx, lastReceiptChecked) {
			return
		}
		// otherwise we understand this monitored tx has failed
		mTx.Status = txmTypes.MonitoredTxStatusFailed
		mTx.BlockNumber = lastReceiptChecked.BlockNumber
		logger.Info("failed")
	}

	// update monitored tx changes into storage
	err = c.storage.Update(ctx, mTx, nil)
	if err != nil {
		logger.Errorf("failed to update monitored tx: %v", err)
		return
	}
}

// getTxNonce get the nonce for the given account
func (c *Client) getTxNonce(ctx context.Context, from common.Address) (uint64, error) {
	// Get created transactions from the database for the given account
	createdTxs, err := c.storage.GetBySenderAndStatus(ctx, from, []txmTypes.MonitoredTxStatus{txmTypes.MonitoredTxStatusCreated}, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get created monitored txs: %w", err)
	}

	var nonce uint64
	if len(createdTxs) > 0 {
		// if there are pending txs, we adjust the nonce accordingly
		for _, createdTx := range createdTxs {
			if createdTx.Nonce > nonce {
				nonce = createdTx.Nonce
			}
		}

		nonce++
	} else {
		// if there are no pending txs, we get the pending nonce from the etherman
		if nonce, err = c.etherman.PendingNonce(ctx, from); err != nil {
			return 0, fmt.Errorf("failed to get pending nonce: %w", err)
		}
	}

	return nonce, nil
}

// shouldContinueToMonitorThisTx checks the the tx receipt and decides if it should
// continue or not to monitor the monitored tx related to the tx from this receipt
func (c *Client) shouldContinueToMonitorThisTx(ctx context.Context, receipt types.Receipt) bool {
	// if the receipt has a is successful result, stop monitoring
	if receipt.Status == types.ReceiptStatusSuccessful {
		return false
	}

	tx, _, err := c.etherman.GetTx(ctx, receipt.TxHash)
	if err != nil {
		log.Errorf("failed to get tx when monitored tx identified as failed, tx : %v", receipt.TxHash.String(), err)
		return false
	}
	_, err = c.etherman.GetRevertMessage(ctx, tx)
	if err != nil {
		// if the error when getting the revert message is not identified, continue to monitor
		if err.Error() == txmTypes.ErrExecutionReverted.Error() {
			return true
		} else {
			log.Errorf("failed to get revert message for monitored tx identified as failed, tx %v: %v", receipt.TxHash.String(), err)
		}
	}
	// if nothing weird was found, stop monitoring
	return false
}

// reviewMonitoredTx checks if some field needs to be updated
// accordingly to the current information stored and the current
// state of the blockchain
func (c *Client) reviewMonitoredTx(ctx context.Context, mTx *txmTypes.MonitoredTx, mTxLogger *zap.SugaredLogger) error {
	mTxLogger.Debug("reviewing")
	// get gas
	gas, err := c.etherman.EstimateGas(ctx, mTx.From, mTx.To, mTx.Value, mTx.Data)
	if err != nil {
		err := fmt.Errorf("failed to estimate gas: %w", err)
		mTxLogger.Errorf(err.Error())
		return err
	}

	// check gas
	if gas > mTx.Gas {
		mTxLogger.Infof("monitored tx gas updated from %v to %v", mTx.Gas, gas)
		mTx.Gas = gas
	}

	// get gas price
	gasPrice, err := c.suggestedGasPrice(ctx)
	if err != nil {
		err := fmt.Errorf("failed to get suggested gas price: %w", err)
		mTxLogger.Errorf(err.Error())
		return err
	}

	// check gas price
	if gasPrice.Cmp(mTx.GasPrice) == 1 {
		mTxLogger.Infof("monitored tx gas price updated from %v to %v", mTx.GasPrice.String(), gasPrice.String())
		mTx.GasPrice = gasPrice
	}
	return nil
}

// reviewMonitoredTxNonce checks if the nonce needs to be updated accordingly to
// the current nonce of the sender account.
//
// IMPORTANT: Nonce is reviewed apart from the other fields because it is a very
// sensible information and can make duplicated data to be sent to the blockchain,
// causing possible side effects and wasting resources.
func (c *Client) reviewMonitoredTxNonce(ctx context.Context, mTx *txmTypes.MonitoredTx, mTxLogger *zap.SugaredLogger) error {
	mTxLogger.Debug("reviewing nonce")
	nonce, err := c.getTxNonce(ctx, mTx.From)
	if err != nil {
		err := fmt.Errorf("failed to load current nonce for acc %v: %w", mTx.From.String(), err)
		mTxLogger.Errorf(err.Error())
		return err
	}

	if nonce > mTx.Nonce {
		mTxLogger.Infof("monitored tx nonce updated from %v to %v", mTx.Nonce, nonce)
		mTx.Nonce = nonce
	}

	return nil
}

func (c *Client) suggestedGasPrice(ctx context.Context) (*big.Int, error) {
	// get gas price
	gasPrice, err := c.etherman.SuggestedGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	// adjust the gas price by the margin factor
	marginFactor := big.NewFloat(0).SetFloat64(c.cfg.GasPriceMarginFactor)
	fGasPrice := big.NewFloat(0).SetInt(gasPrice)
	adjustedGasPrice, _ := big.NewFloat(0).Mul(fGasPrice, marginFactor).Int(big.NewInt(0))

	// if there is a max gas price limit configured and the current
	// adjusted gas price is over this limit, set the gas price as the limit
	if c.cfg.MaxGasPriceLimit > 0 {
		maxGasPrice := big.NewInt(0).SetUint64(c.cfg.MaxGasPriceLimit)
		if adjustedGasPrice.Cmp(maxGasPrice) == 1 {
			adjustedGasPrice.Set(maxGasPrice)
		}
	}

	return adjustedGasPrice, nil
}

// logErrorAndWait used when an error is detected before trying again
func (c *Client) logErrorAndWait(msg string, err error) {
	log.Errorf(msg, err)
	time.Sleep(failureIntervalInSeconds * time.Second)
}

// createMonitoredTxLogger creates an instance of logger with all the important
// fields already set for a monitoredTx
func createMonitoredTxLogger(mTx txmTypes.MonitoredTx) *zap.SugaredLogger {
	return log.WithFields(
		"owner", mTx.Owner,
		"monitoredTxId", mTx.ID,
		"createdAt", mTx.CreatedAt,
		"from", mTx.From,
		"to", mTx.To,
	)
}
