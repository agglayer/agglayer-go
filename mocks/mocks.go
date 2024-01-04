package mocks

import (
	"context"
	"math/big"

	"github.com/0xPolygon/beethoven/tx"
	"github.com/0xPolygon/beethoven/types"

	"github.com/0xPolygonHermez/zkevm-node/ethtxmanager"
	validiumTypes "github.com/0xPolygonHermez/zkevm-node/jsonrpc/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/mock"
)

var _ types.EthermanInterface = (*EthermanMock)(nil)

type EthermanMock struct {
	mock.Mock
}

func (e *EthermanMock) GetSequencerAddr(l1Contract common.Address) (common.Address, error) {
	args := e.Called(l1Contract)

	return args.Get(0).(common.Address), args.Error(1) //nolint:forcetypeassert
}

func (e *EthermanMock) BuildTrustedVerifyBatchesTxData(lastVerifiedBatch,
	newVerifiedBatch uint64, proof tx.ZKP) (data []byte, err error) {
	args := e.Called(lastVerifiedBatch, newVerifiedBatch, proof)

	return args.Get(0).([]byte), args.Error(1) //nolint:forcetypeassert
}

func (e *EthermanMock) CallContract(ctx context.Context, call ethereum.CallMsg,
	blockNumber *big.Int) ([]byte, error) {
	args := e.Called(ctx, call, blockNumber)

	return args.Get(0).([]byte), args.Error(1) //nolint:forcetypeassert
}

var _ types.DBInterface = (*DbMock)(nil)

type DbMock struct {
	mock.Mock
}

func (db *DbMock) BeginStateTransaction(ctx context.Context) (pgx.Tx, error) {
	args := db.Called(ctx)

	tx, ok := args.Get(0).(pgx.Tx)
	if !ok {
		return nil, args.Error(1)
	}

	return tx, args.Error(1)
}

var _ types.EthTxManager = (*EthTxManagerMock)(nil)

type EthTxManagerMock struct {
	mock.Mock
}

func (e *EthTxManagerMock) Add(ctx context.Context, owner, id string,
	from common.Address, to *common.Address, value *big.Int, data []byte, gasOffset uint64, dbTx pgx.Tx) error {
	args := e.Called(ctx, owner, id, from, to, value, data, gasOffset, dbTx)

	return args.Error(0)
}

func (e *EthTxManagerMock) Result(ctx context.Context, owner,
	id string, dbTx pgx.Tx) (ethtxmanager.MonitoredTxResult, error) {
	args := e.Called(ctx, owner, id, dbTx)

	return args.Get(0).(ethtxmanager.MonitoredTxResult), args.Error(1) //nolint:forcetypeassert
}

func (e *EthTxManagerMock) ResultsByStatus(ctx context.Context, owner string,
	statuses []ethtxmanager.MonitoredTxStatus, dbTx pgx.Tx) ([]ethtxmanager.MonitoredTxResult, error) {
	e.Called(ctx, owner, statuses, dbTx)

	return nil, nil
}

func (e *EthTxManagerMock) ProcessPendingMonitoredTxs(ctx context.Context, owner string,
	failedResultHandler ethtxmanager.ResultHandler, dbTx pgx.Tx) {
	e.Called(ctx, owner, failedResultHandler, dbTx)
}

var _ types.ZkEVMClientInterface = (*ZkEVMClientMock)(nil)

type ZkEVMClientMock struct {
	mock.Mock
}

func (zkc *ZkEVMClientMock) BatchByNumber(ctx context.Context, number *big.Int) (*validiumTypes.Batch, error) {
	args := zkc.Called(ctx, number)

	batch, ok := args.Get(0).(*validiumTypes.Batch)
	if !ok {
		return nil, args.Error(1)
	}

	return batch, args.Error(1)
}

var _ types.ZkEVMClientClientCreator = (*ZkEVMClientCreatorMock)(nil)

type ZkEVMClientCreatorMock struct {
	mock.Mock
}

func (zc *ZkEVMClientCreatorMock) NewClient(rpc string) types.ZkEVMClientInterface {
	args := zc.Called(rpc)

	return args.Get(0).(types.ZkEVMClientInterface) //nolint:forcetypeassert
}
