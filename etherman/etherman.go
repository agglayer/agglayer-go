package etherman

import (
	"context"
	"errors"
	"fmt"
	"github.com/0xPolygon/beethoven/config"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonrollupmanager"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/state"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v4"

	"github.com/0xPolygon/beethoven/tx"
)

const (
	HashLength  = 32
	ProofLength = 24
)

type Etherman struct {
	ethClient EthereumClient
	auth      bind.TransactOpts
	config    *config.Config
}

func New(ethClient EthereumClient, auth bind.TransactOpts, cfg *config.Config) (Etherman, error) {
	return Etherman{
		ethClient: ethClient,
		auth:      auth,
		config:    cfg,
	}, nil
}

func (e *Etherman) GetSequencerAddr(rollupId uint32) (common.Address, error) {
	address, err := e.getTrustedSequencerAddress(rollupId)
	if err != nil {
		log.Errorf("error requesting the 'TrustedSequencer' address: %s", err)
		return common.Address{}, err
	}

	return address, nil
}

func (e *Etherman) BuildTrustedVerifyBatchesTxData(
	lastVerifiedBatch,
	newVerifiedBatch uint64,
	proof tx.ZKP,
	rollupId uint32,
) (data []byte, err error) {
	var newLocalExitRoot [HashLength]byte
	copy(newLocalExitRoot[:], proof.NewLocalExitRoot.Bytes())
	var newStateRoot [HashLength]byte
	copy(newStateRoot[:], proof.NewStateRoot.Bytes())
	finalProof, err := ConvertProof(proof.Proof.Hex())
	if err != nil {
		log.Errorf("error converting proof. Error: %v, Proof: %s", err, proof.Proof)
		return nil, err
	}

	const pendStateNum uint64 = 0 // TODO hardcoded for now until we implement the pending state feature
	abi, err := polygonzkevm.PolygonzkevmMetaData.GetAbi()
	if err != nil {
		log.Errorf("error geting ABI: %v, Proof: %s", err)
		return nil, err
	}

	return abi.Pack(
		"verifyBatchesTrustedAggregator",
		rollupId,
		pendStateNum,
		lastVerifiedBatch,
		newVerifiedBatch,
		newLocalExitRoot,
		newStateRoot,
		finalProof,
	)
}

func (e *Etherman) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return e.ethClient.CallContract(ctx, call, blockNumber)
}

func (e *Etherman) getRollupContractAddress(rollupId uint32) (common.Address, error) {
	contract, err := polygonrollupmanager.NewPolygonrollupmanager(e.config.L1.RollupManagerContract, e.ethClient)
	if err != nil {
		return common.Address{}, fmt.Errorf("error instantiating 'PolygonRollupManager' contract: %s", err)
	}

	rollupData, err := contract.RollupIDToRollupData(&bind.CallOpts{Pending: false}, rollupId)
	if err != nil {
		return common.Address{}, fmt.Errorf("error receiving the 'RollupData' struct: %s", err)
	}

	return rollupData.RollupContract, nil
}

func (e *Etherman) getTrustedSequencerAddress(rollupId uint32) (common.Address, error) {
	rollupContractAddress, err := e.getRollupContractAddress(rollupId)
	if err != nil {
		return common.Address{}, fmt.Errorf("error requesting the 'PolygonZkEvm' contract address from 'PolygonRollupManager': %s", err)
	}

	contract, err := polygonzkevm.NewPolygonzkevm(rollupContractAddress, e.ethClient)
	if err != nil {
		return common.Address{}, fmt.Errorf("error instantiating 'PolygonZkEvm' contract: %s", err)
	}

	return contract.TrustedSequencer(&bind.CallOpts{Pending: false})
}

// CheckTxWasMined check if a tx was already mined
func (e *Etherman) CheckTxWasMined(ctx context.Context, txHash common.Hash) (bool, *types.Receipt, error) {
	receipt, err := e.ethClient.TransactionReceipt(ctx, txHash)
	if errors.Is(err, ethereum.NotFound) {
		return false, nil, nil
	} else if err != nil {
		return false, nil, err
	}

	return true, receipt, nil
}

// CurrentNonce returns the current nonce for the provided account
func (e *Etherman) CurrentNonce(ctx context.Context, account common.Address) (uint64, error) {
	return e.ethClient.NonceAt(ctx, account, nil)
}

// GetTx function get ethereum tx
func (e *Etherman) GetTx(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	return e.ethClient.TransactionByHash(ctx, txHash)
}

// GetTxReceipt function gets ethereum tx receipt
func (e *Etherman) GetTxReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return e.ethClient.TransactionReceipt(ctx, txHash)
}

// WaitTxToBeMined waits for an L1 tx to be mined. It will return error if the tx is reverted or timeout is exceeded
func (e *Etherman) WaitTxToBeMined(ctx context.Context, tx *types.Transaction, timeout time.Duration) (bool, error) {
	err := operations.WaitTxToBeMined(ctx, e.ethClient, tx, timeout)
	if errors.Is(err, context.DeadlineExceeded) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// SendTx sends a tx to L1
func (e *Etherman) SendTx(ctx context.Context, tx *types.Transaction) error {
	return e.ethClient.SendTransaction(ctx, tx)
}

// SuggestedGasPrice returns the suggest nonce for the network at the moment
func (e *Etherman) SuggestedGasPrice(ctx context.Context) (*big.Int, error) {
	return e.ethClient.SuggestGasPrice(ctx)
}

// EstimateGas returns the estimated gas for the tx
func (e *Etherman) EstimateGas(ctx context.Context, from common.Address, to *common.Address, value *big.Int, data []byte) (uint64, error) {
	return e.ethClient.EstimateGas(ctx, ethereum.CallMsg{
		From:  from,
		To:    to,
		Value: value,
		Data:  data,
	})
}

// SignTx tries to sign a transaction accordingly to the provided sender
func (e *Etherman) SignTx(ctx context.Context, sender common.Address, tx *types.Transaction) (*types.Transaction, error) {
	return e.auth.Signer(e.auth.From, tx)
}

// GetRevertMessage tries to get a revert message of a transaction
func (e *Etherman) GetRevertMessage(ctx context.Context, tx *types.Transaction) (string, error) {
	if tx == nil {
		return "", nil
	}

	receipt, err := e.GetTxReceipt(ctx, tx.Hash())
	if err != nil {
		return "", err
	}

	if receipt.Status == types.ReceiptStatusFailed {
		revertMessage, err := operations.RevertReason(ctx, e.ethClient, tx, receipt.BlockNumber)

		if err != nil {
			return "", err
		}
		return revertMessage, nil
	}
	return "", nil
}

func (e *Etherman) GetLastBlock(ctx context.Context, dbTx pgx.Tx) (*state.Block, error) {
	block, err := e.ethClient.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &state.Block{
		BlockNumber: block.NumberU64(),
		BlockHash:   block.Hash(),
		ParentHash:  block.ParentHash(),
		ReceivedAt:  time.Unix(int64(block.Time()), 0),
	}, nil
}
