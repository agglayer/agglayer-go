package interop

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xPolygon/agglayer/log"
	txmTypes "github.com/0xPolygon/agglayer/txmanager/types"
	jRPC "github.com/0xPolygon/cdk-rpc/rpc"
	rpctypes "github.com/0xPolygonHermez/zkevm-node/jsonrpc/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/agglayer/config"
	"github.com/0xPolygon/agglayer/mocks"
	"github.com/0xPolygon/agglayer/tx"
)

func TestNewExecutor(t *testing.T) {
	cfg := &config.Config{}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	etherman := mocks.NewEthermanMock(t)
	ethTxManager := mocks.NewEthTxManagerMock(t)

	executor := New(nil, cfg, interopAdminAddr, etherman, ethTxManager)

	assert.NotNil(t, executor)
	assert.Equal(t, interopAdminAddr, executor.interopAdminAddr)
	assert.Equal(t, cfg, executor.config)
	assert.Equal(t, ethTxManager, executor.ethTxMan)
	assert.Equal(t, etherman, executor.etherman)
	assert.NotNil(t, executor.ZkEVMClientCreator)
}

func TestExecutor_CheckTx(t *testing.T) {
	cfg := &config.Config{
		FullNodeRPCs: map[uint32]string{
			1: "http://localhost:8545",
		},
	}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	etherman := mocks.NewEthermanMock(t)
	ethTxManager := mocks.NewEthTxManagerMock(t)

	executor := New(log.WithFields("test", "test"), cfg, interopAdminAddr, etherman, ethTxManager)

	// Create a sample signed transaction for testing
	signedTx := tx.SignedTx{
		Tx: tx.Tx{
			LastVerifiedBatch: 0,
			NewVerifiedBatch:  1,
			ZKP: tx.ZKP{
				Proof: []byte("sampleProof"),
			},
			RollupID: 1,
		},
	}

	err := executor.CheckTx(signedTx)
	assert.NoError(t, err)

	signedTx = tx.SignedTx{
		Tx: tx.Tx{
			LastVerifiedBatch: 0,
			NewVerifiedBatch:  1,
			ZKP: tx.ZKP{
				Proof: []byte("sampleProof"),
			},
			RollupID: 0,
		},
	}

	err = executor.CheckTx(signedTx)
	assert.Error(t, err)
}

func TestExecutor_VerifyZKP(t *testing.T) {
	cfg := &config.Config{}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	etherman := mocks.NewEthermanMock(t)
	ethTxManager := mocks.NewEthTxManagerMock(t)
	tnx := tx.Tx{
		LastVerifiedBatch: 0,
		NewVerifiedBatch:  1,
		ZKP: tx.ZKP{
			Proof: []byte("sampleProof"),
		},
		RollupID: 1,
	}

	etherman.On(
		"BuildTrustedVerifyBatchesTxData",
		uint64(tnx.LastVerifiedBatch),
		uint64(tnx.NewVerifiedBatch),
		mock.Anything,
		uint32(1),
	).Return(
		[]byte{},
		nil,
	).Once()

	etherman.On(
		"CallContract",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(
		[]byte{},
		nil,
	).Once()

	executor := New(nil, cfg, interopAdminAddr, etherman, ethTxManager)

	// Create a sample signed transaction for testing
	signedTx := tx.SignedTx{
		Tx: tnx,
	}

	err := executor.verifyZKP(context.Background(), signedTx)
	assert.NoError(t, err)
	etherman.AssertExpectations(t)
}

func TestExecutor_VerifySignature(t *testing.T) {
	cfg := &config.Config{}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	etherman := mocks.NewEthermanMock(t)
	ethTxManager := mocks.NewEthTxManagerMock(t)

	executor := New(nil, cfg, interopAdminAddr, etherman, ethTxManager)

	txn := tx.Tx{
		LastVerifiedBatch: 0,
		NewVerifiedBatch:  1,
		ZKP: tx.ZKP{
			Proof: []byte("sampleProof"),
		},
		RollupID: 1,
	}

	sequencerKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	t.Run("use sequencer key, correct signature", func(t *testing.T) {
		etherman.On(
			"GetSequencerAddr",
			uint32(1),
		).Return(
			crypto.PubkeyToAddress(sequencerKey.PublicKey),
			nil,
		).Once()

		signedTx, err := txn.Sign(sequencerKey)
		require.NoError(t, err)

		err = executor.verifySignature(*signedTx)
		require.NoError(t, err)
		etherman.AssertExpectations(t)
	})

	t.Run("use sequencer key, wrong signature", func(t *testing.T) {
		etherman.On(
			"GetSequencerAddr",
			uint32(1),
		).Return(
			common.Address{0x1},
			nil,
		).Once()

		signedTx, err := txn.Sign(sequencerKey)
		require.NoError(t, err)

		err = executor.verifySignature(*signedTx)
		require.Error(t, err)
		etherman.AssertExpectations(t)
	})

	t.Run("configured proof signers, correct signature", func(t *testing.T) {
		anotherKey, err := crypto.GenerateKey()
		require.NoError(t, err)

		cfg.ProofSigners = config.ProofSigners{1: crypto.PubkeyToAddress(anotherKey.PublicKey)}

		signedTx, err := txn.Sign(anotherKey)
		require.NoError(t, err)

		executor = New(nil, cfg, interopAdminAddr, etherman, ethTxManager)

		err = executor.verifySignature(*signedTx)
		require.NoError(t, err)
	})

	t.Run("configured proof signers, wrong signature", func(t *testing.T) {
		anotherKey, err := crypto.GenerateKey()
		require.NoError(t, err)

		cfg.ProofSigners = config.ProofSigners{1: common.Address{0x1}}

		signedTx, err := txn.Sign(anotherKey)
		require.NoError(t, err)

		executor = New(nil, cfg, interopAdminAddr, etherman, ethTxManager)

		err = executor.verifySignature(*signedTx)
		require.Error(t, err)
	})
}

func TestExecutor_Execute(t *testing.T) {
	t.Parallel()

	// Create a sample signed transaction for testing
	signedTx := tx.SignedTx{
		Tx: tx.Tx{
			LastVerifiedBatch: 0,
			NewVerifiedBatch:  1,
			ZKP: tx.ZKP{
				NewStateRoot:     common.BytesToHash([]byte("sampleNewStateRoot")),
				NewLocalExitRoot: common.BytesToHash([]byte("sampleNewLocalExitRoot")),
				Proof:            []byte("sampleProof"),
			},
		},
	}

	t.Run("Batch is not nil and roots match", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{}
		interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
		etherman := mocks.NewEthermanMock(t)
		ethTxManager := mocks.NewEthTxManagerMock(t)

		executor := New(log.WithFields("test", "test"), cfg, interopAdminAddr, etherman, ethTxManager)

		// Mock the ZkEVMClientCreator.NewClient method
		mockZkEVMClientCreator := mocks.NewZkEVMClientClientCreatorMock(t)
		mockZkEVMClient := mocks.NewZkEVMClientMock(t)

		mockZkEVMClientCreator.On("NewClient", mock.Anything).Return(mockZkEVMClient).Once()
		mockZkEVMClient.On("BatchByNumber", mock.Anything, big.NewInt(int64(signedTx.Tx.NewVerifiedBatch))).
			Return(&rpctypes.Batch{
				StateRoot:     signedTx.Tx.ZKP.NewStateRoot,
				LocalExitRoot: signedTx.Tx.ZKP.NewLocalExitRoot,
				// Add other necessary fields here
			}, nil).Once()

		// Set the ZkEVMClientCreator to return the mock ZkEVMClient
		executor.ZkEVMClientCreator = mockZkEVMClientCreator

		err := executor.Execute(context.Background(), signedTx)
		require.NoError(t, err)
		mockZkEVMClientCreator.AssertExpectations(t)
		mockZkEVMClient.AssertExpectations(t)
	})

	t.Run("Returns expected error when Batch is nil", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{}
		interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
		etherman := mocks.NewEthermanMock(t)
		ethTxManager := mocks.NewEthTxManagerMock(t)

		executor := New(log.WithFields("test", "test"), cfg, interopAdminAddr, etherman, ethTxManager)

		// Mock the ZkEVMClientCreator.NewClient method
		mockZkEVMClientCreator := mocks.NewZkEVMClientClientCreatorMock(t)
		mockZkEVMClient := mocks.NewZkEVMClientMock(t)

		mockZkEVMClientCreator.On("NewClient", mock.Anything).Return(mockZkEVMClient).Once()
		mockZkEVMClient.On("BatchByNumber", mock.Anything, big.NewInt(int64(signedTx.Tx.NewVerifiedBatch))).
			Return(nil, nil).Once()

		// Set the ZkEVMClientCreator to return the mock ZkEVMClient
		executor.ZkEVMClientCreator = mockZkEVMClientCreator

		err := executor.Execute(context.Background(), signedTx)
		require.Error(t, err)
		expectedError := fmt.Sprintf(
			"unable to perform soundness check because batch with number %d is undefined",
			signedTx.Tx.NewVerifiedBatch,
		)
		assert.Contains(t, err.Error(), expectedError)
		mockZkEVMClientCreator.AssertExpectations(t)
		mockZkEVMClient.AssertExpectations(t)
	})
}

func TestExecutor_Settle(t *testing.T) {
	cfg := &config.Config{}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	etherman := mocks.NewEthermanMock(t)
	ethTxManager := mocks.NewEthTxManagerMock(t)
	dbTx := &mocks.TxMock{}

	executor := New(nil, cfg, interopAdminAddr, etherman, ethTxManager)

	signedTx := tx.SignedTx{
		Tx: tx.Tx{
			LastVerifiedBatch: 0,
			NewVerifiedBatch:  1,
			ZKP: tx.ZKP{
				Proof: []byte("sampleProof"),
			},
			RollupID: 1,
		},
	}

	l1TxData := []byte("sampleL1TxData")
	etherman.On(
		"BuildTrustedVerifyBatchesTxData",
		uint64(signedTx.Tx.LastVerifiedBatch),
		uint64(signedTx.Tx.NewVerifiedBatch),
		signedTx.Tx.ZKP,
		uint32(1),
	).Return(
		l1TxData,
		nil,
	).Once()

	ctx := context.Background()
	txHash := signedTx.Tx.Hash().Hex()
	ethTxManager.On(
		"Add",
		ctx, ethTxManOwner,
		txHash,
		interopAdminAddr,
		&cfg.L1.RollupManagerContract,
		big.NewInt(0),
		l1TxData,
		uint64(0),
		dbTx,
	).Return(
		nil,
	).Once()

	hash, err := executor.Settle(ctx, signedTx, dbTx)
	require.NoError(t, err)
	assert.Equal(t, signedTx.Tx.Hash(), hash)

	etherman.AssertExpectations(t)
	ethTxManager.AssertExpectations(t)
}

func TestExecutor_GetTxStatus(t *testing.T) {
	cfg := &config.Config{}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	etherman := mocks.NewEthermanMock(t)
	ethTxManager := mocks.NewEthTxManagerMock(t)
	dbTx := &mocks.TxMock{}

	executor := New(nil, cfg, interopAdminAddr, etherman, ethTxManager)

	hash := common.HexToHash("0x1234567890abcdef")
	expectedResult := "0x1"
	expectedError := jRPC.NewRPCError(rpctypes.DefaultErrorCode, "failed to get tx, error: sampleError")

	ethTxManager.On("Result", mock.Anything, ethTxManOwner, hash.Hex(), dbTx).
		Return(txmTypes.MonitoredTxResult{
			ID:     "0x1",
			Status: txmTypes.MonitoredTxStatus("0x1"),
		}, nil).Once()

	result, err := executor.GetTxStatus(context.Background(), hash, dbTx)

	assert.Equal(t, expectedResult, result)
	assert.NoError(t, err)

	ethTxManager.On("Result", mock.Anything, ethTxManOwner, hash.Hex(), dbTx).
		Return(txmTypes.MonitoredTxResult{
			ID:     "0x0",
			Status: txmTypes.MonitoredTxStatus("0x1"),
		}, errors.New("sampleError")).Once()

	result, err = executor.GetTxStatus(context.Background(), hash, dbTx)

	assert.Equal(t, "0x0", result)
	assert.Equal(t, expectedError, err)

	etherman.AssertExpectations(t)
	ethTxManager.AssertExpectations(t)
}
