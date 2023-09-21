package rpc

import (
	"context"
	"math/big"

	ethmanTypes "github.com/0xPolygonHermez/zkevm-node/etherman/types"
	"github.com/0xPolygonHermez/zkevm-node/ethtxmanager"
	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

type DBInterface interface {
	BeginStateTransaction(ctx context.Context) (pgx.Tx, error)
}

type EthermanInterface interface {
	GetSequencerAddr(l1Contract common.Address) (common.Address, error)
	BuildTrustedVerifyBatchesTxData(l1Contract common.Address, lastVerifiedBatch, newVerifiedBatch uint64, inputs *ethmanTypes.FinalProofInputs) (data []byte, err error)
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

// ethTxManager contains the methods required to send txs to ethereum.
type EthTxManager interface {
	Add(ctx context.Context, owner, id string, from common.Address, to *common.Address, value *big.Int, data []byte, dbTx pgx.Tx) error
	Result(ctx context.Context, owner, id string, dbTx pgx.Tx) (ethtxmanager.MonitoredTxResult, error)
	ResultsByStatus(ctx context.Context, owner string, statuses []ethtxmanager.MonitoredTxStatus, dbTx pgx.Tx) ([]ethtxmanager.MonitoredTxResult, error)
	ProcessPendingMonitoredTxs(ctx context.Context, owner string, failedResultHandler ethtxmanager.ResultHandler, dbTx pgx.Tx)
}

type ZkEVMClientInterface interface {
	BatchByNumber(ctx context.Context, number *big.Int) (*types.Batch, error)
}
