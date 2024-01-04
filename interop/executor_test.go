package interop

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xPolygon/beethoven/config"
	"github.com/0xPolygon/beethoven/test"
	"github.com/0xPolygon/beethoven/tx"

	rpctypes "github.com/0xPolygonHermez/zkevm-node/jsonrpc/types"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewExecutor(t *testing.T) {
	cfg := &config.Config{
		// Set your desired config values here
	}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	etherman := &test.EthermanMock{}
	ethTxManager := &test.EthTxManagerMock{}

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
		FullNodeRPCs: map[common.Address]string{
			common.HexToAddress("0x1234567890abcdef"): "http://localhost:8545",
		},
	}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	etherman := &test.EthermanMock{}
	ethTxManager := &test.EthTxManagerMock{}

	executor := New(log.WithFields("test", "test"), cfg, interopAdminAddr, etherman, ethTxManager)

	// Create a sample signed transaction for testing
	signedTx := tx.SignedTx{
		Tx: tx.Tx{
			LastVerifiedBatch: 0,
			NewVerifiedBatch:  1,
			ZKP: tx.ZKP{
				Proof: []byte("sampleProof"),
			},
			L1Contract: common.HexToAddress("0x1234567890abcdef"),
		},
	}

	err := executor.CheckTx(context.Background(), signedTx)
	assert.NoError(t, err)

	signedTx = tx.SignedTx{
		Tx: tx.Tx{
			LastVerifiedBatch: 0,
			NewVerifiedBatch:  1,
			ZKP: tx.ZKP{
				Proof: []byte("sampleProof"),
			},
			L1Contract: common.HexToAddress("0xdeadbeef"),
		},
	}

	err = executor.CheckTx(context.Background(), signedTx)
	assert.Error(t, err)
}

func TestExecutor_VerifyZKP(t *testing.T) {
	cfg := &config.Config{
		// Set your desired config values here
	}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	etherman := &test.EthermanMock{}
	ethTxManager := &test.EthTxManagerMock{}
	tnx := tx.Tx{
		LastVerifiedBatch: 0,
		NewVerifiedBatch:  1,
		ZKP: tx.ZKP{
			Proof: []byte("sampleProof"),
		},
		L1Contract: common.HexToAddress("0x1234567890abcdef"),
	}

	etherman.On("BuildTrustedVerifyBatchesTxData",
		uint64(tnx.LastVerifiedBatch), uint64(tnx.NewVerifiedBatch), mock.Anything).
		Return([]byte{}, nil).Once()

	etherman.On("CallContract", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte{}, nil).Once()

	executor := New(nil, cfg, interopAdminAddr, etherman, ethTxManager)

	// Create a sample signed transaction for testing
	signedTx := tx.SignedTx{
		Tx: tnx,
	}

	err := executor.VerifyZKP(signedTx)
	assert.NoError(t, err)
	etherman.AssertExpectations(t)
}

func TestExecutor_VerifySignature(t *testing.T) {
	cfg := &config.Config{
		// Set your desired config values here
	}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	etherman := &test.EthermanMock{}
	ethTxManager := &test.EthTxManagerMock{}

	executor := New(nil, cfg, interopAdminAddr, etherman, ethTxManager)

	txn := tx.Tx{
		LastVerifiedBatch: 0,
		NewVerifiedBatch:  1,
		ZKP: tx.ZKP{
			Proof: []byte("sampleProof"),
		},
		L1Contract: common.HexToAddress("0x1234567890abcdef"),
	}

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	signedTx, err := txn.Sign(pk)
	require.NoError(t, err)

	etherman.On("GetSequencerAddr", mock.Anything).
		Return(crypto.PubkeyToAddress(pk.PublicKey), nil).Once()

	err = executor.VerifySignature(*signedTx)
	require.NoError(t, err)
	etherman.AssertExpectations(t)
}

func TestExecutor_Execute(t *testing.T) {
	cfg := &config.Config{}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	etherman := &test.EthermanMock{}
	ethTxManager := &test.EthTxManagerMock{}

	executor := New(log.WithFields("test", "test"), cfg, interopAdminAddr, etherman, ethTxManager)

	// Create a sample signed transaction for testing
	signedTx := tx.SignedTx{
		Tx: tx.Tx{
			LastVerifiedBatch: 0,
			NewVerifiedBatch:  1,
			ZKP: tx.ZKP{
				NewStateRoot: common.BytesToHash([]byte("sampleNewStateRoot")),
				Proof:        []byte("sampleProof"),
			},
			L1Contract: common.HexToAddress("0x1234567890abcdef"),
		},
	}

	// Mock the ZkEVMClientCreator.NewClient method
	mockZkEVMClientCreator := &test.ZkEVMClientCreatorMock{}
	mockZkEVMClient := &test.ZkEVMClientMock{}

	mockZkEVMClientCreator.On("NewClient", mock.Anything).Return(mockZkEVMClient).Once()
	mockZkEVMClient.On("BatchByNumber", mock.Anything, big.NewInt(int64(signedTx.Tx.NewVerifiedBatch))).
		Return(&rpctypes.Batch{
			StateRoot:     signedTx.Tx.ZKP.NewStateRoot,
			LocalExitRoot: signedTx.Tx.ZKP.NewLocalExitRoot,
			// Add other necessary fields here
		}, nil).Once()

	// Set the ZkEVMClientCreator to return the mock ZkEVMClient
	executor.ZkEVMClientCreator = mockZkEVMClientCreator

	err := executor.Execute(signedTx)
	require.NoError(t, err)
	mockZkEVMClientCreator.AssertExpectations(t)
	mockZkEVMClient.AssertExpectations(t)
}
