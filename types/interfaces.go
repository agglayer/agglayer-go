package types

import (
	"context"
	"math/big"

	"github.com/0xPolygon/beethoven/tx"
	"github.com/0xPolygonHermez/zkevm-node/ethtxmanager"
	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

//go:generate mockery --name IDB --output ../mocks --case=underscore --filename db.generated.go
type IDB interface {
	BeginStateTransaction(ctx context.Context) (pgx.Tx, error)
}

//go:generate mockery --name IEtherman --output ../mocks --case=underscore --filename etherman.generated.go
type IEtherman interface {
	GetSequencerAddr(rollupId uint32) (common.Address, error)
	BuildTrustedVerifyBatchesTxData(lastVerifiedBatch, newVerifiedBatch uint64, proof tx.ZKP, rollupId uint32) (data []byte, err error)
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

//go:generate mockery --name IEthTxManager --output ../mocks --case=underscore --filename eth_tx_manager.generated.go
type IEthTxManager interface {
	Add(ctx context.Context, owner, id string, from common.Address, to *common.Address, value *big.Int, data []byte, gasOffset uint64, dbTx pgx.Tx) error
	Result(ctx context.Context, owner, id string, dbTx pgx.Tx) (ethtxmanager.MonitoredTxResult, error)
	ResultsByStatus(ctx context.Context, owner string, statuses []ethtxmanager.MonitoredTxStatus, dbTx pgx.Tx) ([]ethtxmanager.MonitoredTxResult, error)
	ProcessPendingMonitoredTxs(ctx context.Context, owner string, failedResultHandler ethtxmanager.ResultHandler, dbTx pgx.Tx)
}

//go:generate mockery --name IZkEVMClient --output ../mocks --case=underscore --filename zk_evm_client.generated.go
type IZkEVMClient interface {
	BatchByNumber(ctx context.Context, number *big.Int) (*types.Batch, error)
}

//go:generate mockery --name IZkEVMClientClientCreator --output ../mocks --case=underscore --filename zk_evm_client_creator.generated.go
type IZkEVMClientClientCreator interface {
	NewClient(rpc string) IZkEVMClient
}
