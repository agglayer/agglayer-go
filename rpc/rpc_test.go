package rpc

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygon/beethoven/config"
	"github.com/0xPolygon/beethoven/interop"
	"github.com/0xPolygon/beethoven/mocks"

	"github.com/0xPolygon/cdk-validium-node/ethtxmanager"
	validiumTypes "github.com/0xPolygon/cdk-validium-node/jsonrpc/types"
	"github.com/0xPolygon/cdk-validium-node/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/beethoven/tx"
)

const rpcRequestTimeout = 10 * time.Second

var _ interop.EthermanInterface = (*ethermanMock)(nil)

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

var _ interop.DBInterface = (*dbMock)(nil)

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

var _ interop.EthTxManager = (*ethTxManagerMock)(nil)

type ethTxManagerMock struct {
	mock.Mock
}

func (e *ethTxManagerMock) Add(ctx context.Context, owner, id string,
	from common.Address, to *common.Address, value *big.Int, data []byte, dbTx pgx.Tx) error {
	args := e.Called(ctx, owner, id, from, to, value, data, dbTx)

	return args.Error(0)
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

var _ interop.ZkEVMClientInterface = (*zkEVMClientMock)(nil)

type zkEVMClientMock struct {
	mock.Mock
}

func (zkc *zkEVMClientMock) BatchByNumber(ctx context.Context, number *big.Int) (*validiumTypes.Batch, error) {
	args := zkc.Called(ctx, number)

	batch, ok := args.Get(0).(*validiumTypes.Batch)
	if !ok {
		return nil, args.Error(1)
	}

	return batch, args.Error(1)
}

var _ interop.ZkEVMClientClientCreator = (*zkEVMClientCreatorMock)(nil)

type zkEVMClientCreatorMock struct {
	mock.Mock
}

func (zc *zkEVMClientCreatorMock) NewClient(rpc string) interop.ZkEVMClientInterface {
	args := zc.Called(rpc)

	return args.Get(0).(interop.ZkEVMClientInterface) //nolint:forcetypeassert
}

func TestInteropEndpointsGetTxStatus(t *testing.T) {
	t.Parallel()

	t.Run("BeginStateTransaction returns an error", func(t *testing.T) {
		t.Parallel()

		dbMock := new(dbMock)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(nil, errors.New("error")).Once()

		e := interop.New(
			log.WithFields("module", "test"),
			&config.Config{},
			common.HexToAddress("0xadmin"),
			new(ethermanMock),
			new(ethTxManagerMock),
		)
		i := NewInteropEndpoints(context.Background(), e, dbMock)

		result, err := i.GetTxStatus(common.HexToHash("0xsomeTxHash"))

		require.Equal(t, "0x0", result)
		require.ErrorContains(t, err, "failed to begin dbTx")

		dbMock.AssertExpectations(t)
	})

	t.Run("failed to get tx", func(t *testing.T) {
		t.Parallel()

		txHash := common.HexToHash("0xsomeTxHash")

		txMock := new(mocks.TxMock)
		txMock.On("Rollback", mock.Anything).Return(nil).Once()

		dbMock := new(dbMock)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(txMock, nil).Once()

		txManagerMock := new(ethTxManagerMock)
		txManagerMock.On("Result", mock.Anything, ethTxManOwner, txHash.Hex(), txMock).
			Return(ethtxmanager.MonitoredTxResult{}, errors.New("error")).Once()

		e := interop.New(
			log.WithFields("module", "test"),
			&config.Config{},
			common.HexToAddress("0xadmin"),
			new(ethermanMock),
			new(ethTxManagerMock),
		)
		i := NewInteropEndpoints(context.Background(), e, dbMock)

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

		txMock := new(mocks.TxMock)
		txMock.On("Rollback", mock.Anything).Return(nil).Once()

		dbMock := new(dbMock)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(txMock, nil).Once()

		txManagerMock := new(ethTxManagerMock)
		txManagerMock.On("Result", mock.Anything, ethTxManOwner, txHash.Hex(), txMock).
			Return(result, nil).Once()

		e := interop.New(
			log.WithFields("module", "test"),
			&config.Config{},
			common.HexToAddress("0xadmin"),
			new(ethermanMock),
			new(ethTxManagerMock),
		)
		i := NewInteropEndpoints(context.Background(), e, dbMock)

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

	type testConfig struct {
		isL1ContractInMap   bool
		canBuildZKProof     bool
		isZKProofValid      bool
		isTxSigned          bool
		isAdminRetrieved    bool
		isSignerValid       bool
		canGetBatch         bool
		isBatchValid        bool
		isDbTxOpen          bool
		isTxAddedToEthTxMan bool
		isTxCommitted       bool

		expectedError string
	}

	testFn := func(cfg testConfig) {
		fullNodeRPCs := config.FullNodeRPCs{
			common.BytesToAddress([]byte{1, 2, 3, 4}): "someRPC",
		}
		tnx := tx.Tx{
			L1Contract:        common.BytesToAddress([]byte{1, 2, 3, 4}),
			LastVerifiedBatch: validiumTypes.ArgUint64(1),
			NewVerifiedBatch:  *validiumTypes.ArgUint64Ptr(2),
			ZKP: tx.ZKP{
				NewStateRoot:     common.BigToHash(big.NewInt(11)),
				NewLocalExitRoot: common.BigToHash(big.NewInt(11)),
			},
		}
		signedTx := &tx.SignedTx{Tx: tnx}
		ethermanMock := new(ethermanMock)
		zkEVMClientCreatorMock := new(zkEVMClientCreatorMock)
		zkEVMClientMock := new(zkEVMClientMock)
		dbMock := new(dbMock)
		txMock := new(mocks.TxMock)
		ethTxManagerMock := new(ethTxManagerMock)

		executeTestFn := func() {
			e := interop.New(
				log.WithFields("module", "test"),
				&config.Config{
					FullNodeRPCs: fullNodeRPCs,
				},
				common.HexToAddress("0xadmin"),
				ethermanMock,
				ethTxManagerMock,
			)
			i := NewInteropEndpoints(context.Background(), e, dbMock)
			// i.zkEVMClientCreator = zkEVMClientCreatorMock

			result, err := i.SendTx(*signedTx)

			if cfg.expectedError != "" {
				require.Equal(t, "0x0", result)
				require.ErrorContains(t, err, cfg.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, signedTx.Tx.Hash(), result)
			}

			ethermanMock.AssertExpectations(t)
			zkEVMClientCreatorMock.AssertExpectations(t)
			zkEVMClientMock.AssertExpectations(t)
			dbMock.AssertExpectations(t)
			txMock.AssertExpectations(t)
			ethTxManagerMock.AssertExpectations(t)
		}

		if !cfg.isL1ContractInMap {
			fullNodeRPCs = config.FullNodeRPCs{}
			executeTestFn()

			return
		}

		if !cfg.canBuildZKProof {
			ethermanMock.On("BuildTrustedVerifyBatchesTxData",
				uint64(tnx.LastVerifiedBatch), uint64(tnx.NewVerifiedBatch), mock.Anything).
				Return([]byte{}, errors.New("error")).Once()
			executeTestFn()

			return
		}

		ethermanMock.On("BuildTrustedVerifyBatchesTxData",
			uint64(tnx.LastVerifiedBatch), uint64(tnx.NewVerifiedBatch), mock.Anything).
			Return([]byte{1, 2}, nil).Once()

		if !cfg.isZKProofValid {
			ethermanMock.On("CallContract", mock.Anything, mock.Anything, mock.Anything).
				Return([]byte{}, errors.New("error")).Once()
			executeTestFn()

			return
		}

		ethermanMock.On("CallContract", mock.Anything, mock.Anything, mock.Anything).
			Return([]byte{1, 2}, nil).Once()

		if !cfg.isTxSigned {
			executeTestFn()

			return
		}

		privateKey, err := crypto.GenerateKey()
		require.NoError(t, err)

		stx, err := tnx.Sign(privateKey)
		require.NoError(t, err)

		signedTx = stx

		if !cfg.isAdminRetrieved {
			ethermanMock.On("GetSequencerAddr", tnx.L1Contract).Return(common.Address{}, errors.New("error")).Once()
			executeTestFn()

			return
		}

		if !cfg.isSignerValid {
			ethermanMock.On("GetSequencerAddr", tnx.L1Contract).Return(common.BytesToAddress([]byte{1, 2, 3, 4}), nil).Once()
			executeTestFn()

			return
		}

		ethermanMock.On("GetSequencerAddr", tnx.L1Contract).Return(crypto.PubkeyToAddress(privateKey.PublicKey), nil).Once()
		zkEVMClientCreatorMock.On("NewClient", mock.Anything).Return(zkEVMClientMock)

		if !cfg.canGetBatch {
			zkEVMClientMock.On("BatchByNumber", mock.Anything, big.NewInt(int64(signedTx.Tx.NewVerifiedBatch))).Return(
				nil, errors.New("error"),
			).Once()
			executeTestFn()

			return
		}

		if !cfg.isBatchValid {
			zkEVMClientMock.On("BatchByNumber", mock.Anything, big.NewInt(int64(signedTx.Tx.NewVerifiedBatch))).Return(
				&validiumTypes.Batch{
					StateRoot: common.BigToHash(big.NewInt(12)),
				}, nil,
			).Once()
			executeTestFn()

			return
		}

		zkEVMClientMock.On("BatchByNumber", mock.Anything, big.NewInt(int64(signedTx.Tx.NewVerifiedBatch))).Return(
			&validiumTypes.Batch{
				StateRoot:     common.BigToHash(big.NewInt(11)),
				LocalExitRoot: common.BigToHash(big.NewInt(11)),
			}, nil,
		).Once()

		if !cfg.isDbTxOpen {
			dbMock.On("BeginStateTransaction", mock.Anything).Return(nil, errors.New("error")).Once()
			executeTestFn()

			return
		}

		dbMock.On("BeginStateTransaction", mock.Anything).Return(txMock, nil).Once()

		if !cfg.isTxAddedToEthTxMan {
			ethTxManagerMock.On("Add", mock.Anything, ethTxManOwner, signedTx.Tx.Hash().Hex(), mock.Anything,
				mock.Anything, mock.Anything, mock.Anything, txMock).Return(errors.New("error")).Once()
			txMock.On("Rollback", mock.Anything).Return(nil).Once()
			executeTestFn()

			return
		}

		ethTxManagerMock.On("Add", mock.Anything, ethTxManOwner, signedTx.Tx.Hash().Hex(), mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, txMock).Return(nil).Once()

		if !cfg.isTxCommitted {
			txMock.On("Commit", mock.Anything).Return(errors.New("error")).Once()
			executeTestFn()

			return
		}

		txMock.On("Commit", mock.Anything).Return(nil).Once()
		executeTestFn()
	}

	t.Run("don't have given contract in map", func(t *testing.T) {
		t.Parallel()

		testFn(testConfig{
			isL1ContractInMap: false,
			expectedError:     "there is no RPC registered",
		})
	})

	t.Run("could not build verified ZKP tx data", func(t *testing.T) {
		t.Parallel()

		testFn(testConfig{
			isL1ContractInMap: true,
			canBuildZKProof:   false,
			expectedError:     "failed to build verify ZKP tx",
		})
	})

	t.Run("could not verified ZKP", func(t *testing.T) {
		t.Parallel()

		testFn(testConfig{
			isL1ContractInMap: true,
			canBuildZKProof:   true,
			isZKProofValid:    false,
			expectedError:     "failed to call verify ZKP response",
		})
	})

	t.Run("could not get signer", func(t *testing.T) {
		t.Parallel()

		testFn(testConfig{
			isL1ContractInMap: true,
			canBuildZKProof:   true,
			isZKProofValid:    true,
			isTxSigned:        false,
			expectedError:     "failed to get signer",
		})
	})

	t.Run("failed to get admin from L1", func(t *testing.T) {
		t.Parallel()

		testFn(testConfig{
			isL1ContractInMap: true,
			canBuildZKProof:   true,
			isZKProofValid:    true,
			isTxSigned:        true,
			isAdminRetrieved:  false,
			expectedError:     "failed to get admin from L1",
		})
	})

	t.Run("unexpected signer", func(t *testing.T) {
		t.Parallel()

		testFn(testConfig{
			isL1ContractInMap: true,
			canBuildZKProof:   true,
			isZKProofValid:    true,
			isTxSigned:        true,
			isAdminRetrieved:  true,
			isSignerValid:     false,
			expectedError:     "unexpected signer",
		})
	})

	t.Run("error on batch retrieval", func(t *testing.T) {
		testFn(testConfig{
			isL1ContractInMap: true,
			canBuildZKProof:   true,
			isZKProofValid:    true,
			isTxSigned:        true,
			isAdminRetrieved:  true,
			isSignerValid:     true,
			canGetBatch:       false,
			expectedError:     "failed to get batch from our node",
		})
	})

	t.Run("unexpected batch", func(t *testing.T) {
		testFn(testConfig{
			isL1ContractInMap: true,
			canBuildZKProof:   true,
			isZKProofValid:    true,
			isTxSigned:        true,
			isAdminRetrieved:  true,
			isSignerValid:     true,
			canGetBatch:       true,
			isBatchValid:      false,
			expectedError:     "Mismatch detected",
		})
	})

	t.Run("failed to begin dbTx", func(t *testing.T) {
		testFn(testConfig{
			isL1ContractInMap: true,
			canBuildZKProof:   true,
			isZKProofValid:    true,
			isTxSigned:        true,
			isAdminRetrieved:  true,
			isSignerValid:     true,
			canGetBatch:       true,
			isBatchValid:      true,
			isDbTxOpen:        false,
			expectedError:     "failed to begin dbTx",
		})
	})

	t.Run("failed to add tx to ethTxMan", func(t *testing.T) {
		testFn(testConfig{
			isL1ContractInMap:   true,
			canBuildZKProof:     true,
			isZKProofValid:      true,
			isTxSigned:          true,
			isAdminRetrieved:    true,
			isSignerValid:       true,
			canGetBatch:         true,
			isBatchValid:        true,
			isDbTxOpen:          true,
			isTxAddedToEthTxMan: false,
			expectedError:       "failed to add tx to ethTxMan",
		})
	})

	t.Run("failed to commit tx", func(t *testing.T) {
		testFn(testConfig{
			isL1ContractInMap:   true,
			canBuildZKProof:     true,
			isZKProofValid:      true,
			isTxSigned:          true,
			isAdminRetrieved:    true,
			isSignerValid:       true,
			canGetBatch:         true,
			isBatchValid:        true,
			isDbTxOpen:          true,
			isTxAddedToEthTxMan: true,
			isTxCommitted:       false,
			expectedError:       "failed to commit dbTx",
		})
	})

	t.Run("happy path", func(t *testing.T) {
		testFn(testConfig{
			isL1ContractInMap:   true,
			canBuildZKProof:     true,
			isZKProofValid:      true,
			isTxSigned:          true,
			isAdminRetrieved:    true,
			isSignerValid:       true,
			canGetBatch:         true,
			isBatchValid:        true,
			isDbTxOpen:          true,
			isTxAddedToEthTxMan: true,
			isTxCommitted:       true,
		})
	})
}
