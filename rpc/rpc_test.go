package rpc

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/0xPolygon/beethoven/tx"
	"github.com/0xPolygon/cdk-validium-node/ethtxmanager"
	validiumTypes "github.com/0xPolygon/cdk-validium-node/jsonrpc/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var _ EthermanInterface = (*ethermanMock)(nil)

type ethermanMock struct {
	mock.Mock
}

func (e *ethermanMock) GetSequencerAddr(l1Contract common.Address) (common.Address, error) {
	args := e.Called(l1Contract)

	return args.Get(0).(common.Address), args.Error(1) //nolint:forcetypeassert
}

func (e *ethermanMock) BuildTrustedVerifyBatchesTxData(lastVerifiedBatch,
	newVerifiedBatch uint64, proof tx.ZKP) (data []byte, err error) {
	args := e.Called(lastVerifiedBatch, newVerifiedBatch, proof)

	return args.Get(0).([]byte), args.Error(1) //nolint:forcetypeassert
}

func (e *ethermanMock) CallContract(ctx context.Context, call ethereum.CallMsg,
	blockNumber *big.Int) ([]byte, error) {
	args := e.Called(ctx, call, blockNumber)

	return args.Get(0).([]byte), args.Error(1) //nolint:forcetypeassert
}

var _ DBInterface = (*dbMock)(nil)

type dbMock struct {
	mock.Mock
}

func (db *dbMock) BeginStateTransaction(ctx context.Context) (pgx.Tx, error) {
	args := db.Called(ctx)

	tx, ok := args.Get(0).(pgx.Tx)
	if !ok {
		return nil, args.Error(1)
	}

	return tx, args.Error(1)
}

var _ EthTxManager = (*ethTxManagerMock)(nil)

type ethTxManagerMock struct {
	mock.Mock
}

func (e *ethTxManagerMock) Add(ctx context.Context, owner, id string,
	from common.Address, to *common.Address, value *big.Int, data []byte, dbTx pgx.Tx) error {
	e.Called(ctx, owner, id, from, to, value, data, dbTx)

	return nil
}

func (e *ethTxManagerMock) Result(ctx context.Context, owner,
	id string, dbTx pgx.Tx) (ethtxmanager.MonitoredTxResult, error) {
	args := e.Called(ctx, owner, id, dbTx)

	return args.Get(0).(ethtxmanager.MonitoredTxResult), args.Error(1) //nolint:forcetypeassert
}

func (e *ethTxManagerMock) ResultsByStatus(ctx context.Context, owner string,
	statuses []ethtxmanager.MonitoredTxStatus, dbTx pgx.Tx) ([]ethtxmanager.MonitoredTxResult, error) {
	e.Called(ctx, owner, statuses, dbTx)

	return nil, nil
}

func (e *ethTxManagerMock) ProcessPendingMonitoredTxs(ctx context.Context, owner string,
	failedResultHandler ethtxmanager.ResultHandler, dbTx pgx.Tx) {
	e.Called(ctx, owner, failedResultHandler, dbTx)
}

var _ pgx.Tx = (*txMock)(nil)

type txMock struct {
	mock.Mock
}

func (tx *txMock) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
}

func (tx *txMock) BeginFunc(ctx context.Context, f func(pgx.Tx) error) (err error) {
	return nil
}

func (tx *txMock) Commit(ctx context.Context) error {
	return nil
}

func (tx *txMock) Rollback(ctx context.Context) error {
	args := tx.Called(ctx)

	return args.Error(0)
}

func (tx *txMock) CopyFrom(ctx context.Context, tableName pgx.Identifier,
	columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (tx *txMock) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (tx *txMock) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (tx *txMock) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (tx *txMock) Exec(ctx context.Context, sql string, arguments ...interface{}) (commandTag pgconn.CommandTag, err error) {
	return nil, nil
}

func (tx *txMock) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (tx *txMock) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return nil
}

func (tx *txMock) QueryFunc(ctx context.Context, sql string,
	args []interface{}, scans []interface{}, f func(pgx.QueryFuncRow) error) (pgconn.CommandTag, error) {
	return nil, nil
}

func (tx *txMock) Conn() *pgx.Conn {
	return nil
}

func TestInteropEndpointsGetTxStatus(t *testing.T) {
	t.Parallel()

	t.Run("BeginStateTransaction returns an error", func(t *testing.T) {
		t.Parallel()

		dbMock := new(dbMock)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(nil, errors.New("error")).Once()

		i := NewInteropEndpoints(
			common.HexToAddress("0xadmin"),
			dbMock,
			new(ethermanMock),
			nil,
			new(ethTxManagerMock),
		)

		result, err := i.GetTxStatus(common.HexToHash("0xsomeTxHash"))

		require.Equal(t, "0x0", result)
		require.ErrorContains(t, err, "failed to begin dbTx")

		dbMock.AssertExpectations(t)
	})

	t.Run("failed to get tx", func(t *testing.T) {
		t.Parallel()

		txHash := common.HexToHash("0xsomeTxHash")

		txMock := new(txMock)
		txMock.On("Rollback", mock.Anything).Return(nil).Once()

		dbMock := new(dbMock)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(txMock, nil).Once()

		txManagerMock := new(ethTxManagerMock)
		txManagerMock.On("Result", mock.Anything, ethTxManOwner, txHash.Hex(), txMock).
			Return(ethtxmanager.MonitoredTxResult{}, errors.New("error")).Once()

		i := NewInteropEndpoints(
			common.HexToAddress("0xadmin"),
			dbMock,
			new(ethermanMock),
			nil,
			txManagerMock,
		)

		result, err := i.GetTxStatus(txHash)

		require.Equal(t, "0x0", result)
		require.ErrorContains(t, err, "failed to get tx")

		dbMock.AssertExpectations(t)
		txMock.AssertExpectations(t)
		txManagerMock.AssertExpectations(t)
	})

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		to := common.HexToAddress("0xreceiver")
		txHash := common.HexToHash("0xsomeTxHash")
		result := ethtxmanager.MonitoredTxResult{
			ID:     "1",
			Status: ethtxmanager.MonitoredTxStatusConfirmed,
			Txs: map[common.Hash]ethtxmanager.TxResult{
				txHash: {
					Tx: types.NewTransaction(1, to, big.NewInt(100_000), 21000, big.NewInt(10_000), nil),
				},
			},
		}

		txMock := new(txMock)
		txMock.On("Rollback", mock.Anything).Return(nil).Once()

		dbMock := new(dbMock)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(txMock, nil).Once()

		txManagerMock := new(ethTxManagerMock)
		txManagerMock.On("Result", mock.Anything, ethTxManOwner, txHash.Hex(), txMock).
			Return(result, nil).Once()

		i := NewInteropEndpoints(
			common.HexToAddress("0xadmin"),
			dbMock,
			new(ethermanMock),
			nil,
			txManagerMock,
		)

		status, err := i.GetTxStatus(txHash)

		require.NoError(t, err)
		require.Equal(t, "confirmed", status)

		dbMock.AssertExpectations(t)
		txMock.AssertExpectations(t)
		txManagerMock.AssertExpectations(t)
	})
}

func TestInteropEndpointsSendTx(t *testing.T) {
	t.Parallel()

	testWithError := func(ethermanMockFn func(tx.Tx) *ethermanMock,
		shouldSignTx bool,
		expectedError string) {
		fullNodeRPCs := FullNodeRPCs{
			common.BytesToAddress([]byte{1, 2, 3, 4}): "someRPC",
		}
		tnx := tx.Tx{
			L1Contract:        common.BytesToAddress([]byte{1, 2, 3, 4}),
			LastVerifiedBatch: validiumTypes.ArgUint64(1),
			NewVerifiedBatch:  *validiumTypes.ArgUint64Ptr(2),
		}

		signedTx := &tx.SignedTx{Tx: tnx}
		if shouldSignTx {
			privateKey, err := crypto.GenerateKey()
			require.NoError(t, err)

			stx, err := tnx.Sign(privateKey)
			require.NoError(t, err)

			signedTx = stx
		}

		ethermanMock := ethermanMockFn(tnx)

		i := NewInteropEndpoints(common.HexToAddress("0xadmin"), nil, ethermanMock, fullNodeRPCs, nil)

		result, err := i.SendTx(*signedTx)

		require.Equal(t, "0x0", result)
		require.ErrorContains(t, err, expectedError)

		ethermanMock.AssertExpectations(t)
	}

	t.Run("don't have given contract in map", func(t *testing.T) {
		t.Parallel()

		i := NewInteropEndpoints(common.HexToAddress("0xadmin"), nil, nil, make(FullNodeRPCs), nil)

		result, err := i.SendTx(tx.SignedTx{
			Tx: tx.Tx{
				L1Contract: common.HexToAddress("0xnonExistingContract"),
			},
		})

		require.Equal(t, "0x0", result)
		require.ErrorContains(t, err, "there is no RPC registered")
	})

	t.Run("could not build verified ZKP tx data", func(t *testing.T) {
		t.Parallel()

		testWithError(func(tnx tx.Tx) *ethermanMock {
			ethermanMock := new(ethermanMock)
			ethermanMock.On("BuildTrustedVerifyBatchesTxData",
				uint64(tnx.LastVerifiedBatch), uint64(tnx.NewVerifiedBatch), mock.Anything).
				Return([]byte{}, errors.New("error")).Once()

			return ethermanMock
		}, false, "failed to build verify ZKP tx")
	})

	t.Run("could not verified ZKP", func(t *testing.T) {
		t.Parallel()

		testWithError(func(tnx tx.Tx) *ethermanMock {
			ethermanMock := new(ethermanMock)
			ethermanMock.On("BuildTrustedVerifyBatchesTxData",
				uint64(tnx.LastVerifiedBatch), uint64(tnx.NewVerifiedBatch), mock.Anything).
				Return([]byte{1, 2}, nil).Once()
			ethermanMock.On("CallContract", mock.Anything, mock.Anything, mock.Anything).
				Return([]byte{}, errors.New("error")).Once()

			return ethermanMock
		}, false, "failed to call verify ZKP response")
	})

	t.Run("could not get signer", func(t *testing.T) {
		t.Parallel()

		testWithError(func(tnx tx.Tx) *ethermanMock {
			ethermanMock := new(ethermanMock)
			ethermanMock.On("BuildTrustedVerifyBatchesTxData",
				uint64(tnx.LastVerifiedBatch), uint64(tnx.NewVerifiedBatch), mock.Anything).
				Return([]byte{1, 2}, nil).Once()
			ethermanMock.On("CallContract", mock.Anything, mock.Anything, mock.Anything).
				Return([]byte{1, 2}, nil).Once()

			return ethermanMock
		}, false, "failed to get signer")
	})

	t.Run("failed to get admin from L1", func(t *testing.T) {
		t.Parallel()

		testWithError(func(tnx tx.Tx) *ethermanMock {
			ethermanMock := new(ethermanMock)
			ethermanMock.On("BuildTrustedVerifyBatchesTxData",
				uint64(tnx.LastVerifiedBatch), uint64(tnx.NewVerifiedBatch), mock.Anything).
				Return([]byte{1, 2}, nil).Once()
			ethermanMock.On("CallContract", mock.Anything, mock.Anything, mock.Anything).
				Return([]byte{1, 2}, nil).Once()
			ethermanMock.On("GetSequencerAddr", tnx.L1Contract).Return(common.Address{}, errors.New("error")).Once()

			return ethermanMock
		}, true, "failed to get admin from L1")
	})

	t.Run("unexpected signer", func(t *testing.T) {
		t.Parallel()

		testWithError(func(tnx tx.Tx) *ethermanMock {
			ethermanMock := new(ethermanMock)
			ethermanMock.On("BuildTrustedVerifyBatchesTxData",
				uint64(tnx.LastVerifiedBatch), uint64(tnx.NewVerifiedBatch), mock.Anything).
				Return([]byte{1, 2}, nil).Once()
			ethermanMock.On("CallContract", mock.Anything, mock.Anything, mock.Anything).
				Return([]byte{1, 2}, nil).Once()
			ethermanMock.On("GetSequencerAddr", tnx.L1Contract).Return(common.BytesToAddress([]byte{2, 3, 4}), nil).Once()

			return ethermanMock
		}, true, "unexpected signer")
	})
}
