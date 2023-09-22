package etherman

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/0xPolygon/cdk-validium-node/encoding"
	"github.com/0xPolygon/cdk-validium-node/etherman/smartcontracts/cdkvalidium"
	"github.com/0xPolygon/cdk-validium-node/log"
	"github.com/0xPolygon/cdk-validium-node/state"
	"github.com/0xPolygon/cdk-validium-node/test/operations"
	"github.com/0xPolygon/silencer/tx"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v4"
)

type Etherman struct {
	ethClient EthereumClient
	auth      bind.TransactOpts
}

func New(url string, auth bind.TransactOpts) (Etherman, error) {
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(url)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", url, err)
		return Etherman{}, err
	}
	return Etherman{
		ethClient: ethClient,
		auth:      auth,
	}, nil
}

func (e *Etherman) GetSequencerAddr(l1Contract common.Address) (common.Address, error) {
	_, contract, err := e.contractCaller(l1Contract)
	if err != nil {
		return common.Address{}, err
	}
	return contract.TrustedSequencer(&bind.CallOpts{Pending: false})
}

func (e *Etherman) BuildTrustedVerifyBatchesTxData(l1Contract common.Address, lastVerifiedBatch, newVerifiedBatch uint64, proof tx.ZKP) (data []byte, err error) {
	opts, contract, err := e.contractCaller(l1Contract)
	if err != nil {
		return nil, err
	}
	var newLocalExitRoot [32]byte
	copy(newLocalExitRoot[:], proof.NewLocalExitRoot.Bytes())
	var newStateRoot [32]byte
	copy(newStateRoot[:], proof.NewStateRoot.Bytes())
	finalProof, err := ConvertProof(proof.Proof.Hex())
	if err != nil {
		log.Errorf("error converting proof. Error: %v, Proof: %s", err, proof.Proof)
		return nil, err
	}

	const pendStateNum = 0 // TODO hardcoded for now until we implement the pending state feature
	tx, err := contract.VerifyBatchesTrustedAggregator(
		opts,
		pendStateNum,
		lastVerifiedBatch,
		newVerifiedBatch,
		newLocalExitRoot,
		newStateRoot,
		finalProof,
	)
	if err != nil {
		if parsedErr, ok := tryParseError(err); ok {
			err = parsedErr
		}
		return nil, err
	}

	return tx.Data(), nil
}

func (e *Etherman) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return e.ethClient.CallContract(ctx, call, blockNumber)
}

func ConvertProof(p string) ([24][32]byte, error) {
	if len(p) != 24*32*2+2 {
		return [24][32]byte{}, fmt.Errorf("invalid proof length. Length: %d", len(p))
	}
	p = strings.TrimPrefix(p, "0x")
	proof := [24][32]byte{}
	for i := 0; i < 24; i++ {
		data := p[i*64 : (i+1)*64]
		p, err := encoding.DecodeBytes(&data)
		if err != nil {
			return [24][32]byte{}, fmt.Errorf("failed to decode proof, err: %w", err)
		}
		var aux [32]byte
		copy(aux[:], p)
		proof[i] = aux
	}
	return proof, nil
}

func (e *Etherman) contractCaller(to common.Address) (*bind.TransactOpts, *cdkvalidium.Cdkvalidium, error) {
	opts := bind.TransactOpts{}
	opts.NoSend = true
	// force nonce, gas limit and gas price to avoid querying it from the chain
	opts.Nonce = big.NewInt(1)
	opts.GasLimit = uint64(1)
	opts.GasPrice = big.NewInt(1)
	contract, err := cdkvalidium.NewCdkvalidium(to, e.ethClient)
	if err != nil {
		log.Errorf("error instantiating contract: %s", err)
		return nil, nil, err
	}
	return &opts, contract, nil
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
