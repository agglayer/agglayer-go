package interop

import (
	"context"
	"testing"

	"github.com/0xPolygon/beethoven/config"
	"github.com/0xPolygon/beethoven/test"
	"github.com/0xPolygon/beethoven/tx"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
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
		// Set your desired config values here
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
}
