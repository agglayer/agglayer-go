package silencer

import (
	"context"
	"errors"
	"fmt"
	"testing"

	rpctypes "github.com/0xPolygonHermez/zkevm-node/jsonrpc/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/beethoven/config"
	"github.com/0xPolygon/beethoven/mocks"
	"github.com/0xPolygon/beethoven/tx"
	"github.com/0xPolygon/beethoven/types"
)

func TestSilencer_New(t *testing.T) {
	cfg := &config.Config{FullNodeRPCs: map[uint32]string{3: "http://localhost:8545"}}
	interopAdmin := common.HexToAddress("0x1234567890abcdef")
	etherman := mocks.NewEthermanMock(t)
	clientCreator := mocks.NewZkEVMClientClientCreatorMock(t)

	executor := New(cfg, interopAdmin, etherman, clientCreator)

	require.NotNil(t, executor)
	require.Equal(t, interopAdmin, executor.interopAdmin)
	require.Equal(t, cfg, executor.cfg)
	require.Equal(t, etherman, executor.etherman)
	require.Equal(t, clientCreator, executor.zkEVMClientCreator)

}

func TestSilencer_Silence(t *testing.T) {
	const rollupID = uint32(10)
	cfg := &config.Config{FullNodeRPCs: map[uint32]string{
		rollupID: "http://localhost:10000",
	}}
	interopAdminAddr := common.HexToAddress("0x1234567890abcdef")
	tx := tx.Tx{
		LastVerifiedBatch: 1,
		NewVerifiedBatch:  2,
		RollupID:          rollupID,
		ZKP: tx.ZKP{
			NewStateRoot: common.BytesToHash([]byte("sampleNewStateRoot")),
			Proof:        []byte("sampleProof"),
		},
	}

	etherman := mocks.NewEthermanMock(t)
	etherman.On("BuildTrustedVerifyBatchesTxData",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return([]byte{}, nil).Once()

	etherman.On("CallContract",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return([]byte{}, nil).Once()

	// Mock the ZkEVMClientCreator.NewClient method
	mockZkEVMClientCreator := mocks.NewZkEVMClientClientCreatorMock(t)
	mockZkEVMClient := mocks.NewZkEVMClientMock(t)

	mockZkEVMClientCreator.On("NewClient", mock.Anything).
		Return(mockZkEVMClient).Once()
	mockZkEVMClient.On("BatchByNumber", mock.Anything, mock.Anything).
		Return(&rpctypes.Batch{
			StateRoot:     tx.ZKP.NewStateRoot,
			LocalExitRoot: tx.ZKP.NewLocalExitRoot,
		}, nil).Once()

	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	signedTx, err := tx.Sign(pk)
	require.NoError(t, err)

	etherman.On(
		"GetSequencerAddr",
		mock.Anything,
	).Return(
		crypto.PubkeyToAddress(pk.PublicKey),
		nil,
	).Once()

	silencer := New(cfg, interopAdminAddr, etherman, mockZkEVMClientCreator)

	err = silencer.Silence(context.Background(), *signedTx)
	require.NoError(t, err)

	mockZkEVMClientCreator.AssertExpectations(t)
	mockZkEVMClient.AssertExpectations(t)
}

func TestSilencer_verify(t *testing.T) {
	createSilencer := func(cfg *config.Config, etherman types.IEtherman) *Silencer {
		interopAdmin := common.HexToAddress("0x100")
		return New(cfg, interopAdmin, etherman, mocks.NewZkEVMClientClientCreatorMock(t))
	}

	defaultRollupID := uint32(2)
	defaultCfg := &config.Config{FullNodeRPCs: map[uint32]string{
		defaultRollupID: "http://localhost:8545",
	}}

	t.Run("no full node RPC registered for the given rollup", func(t *testing.T) {
		stx := tx.SignedTx{Data: tx.Tx{RollupID: 20}}

		s := createSilencer(defaultCfg, nil)
		err := s.verify(context.Background(), stx)
		require.ErrorContains(t, err, fmt.Sprintf("there is no RPC registered for rollup %d", stx.Data.RollupID))
	})

	t.Run("ZK proof verification failure", func(t *testing.T) {
		etherman := mocks.NewEthermanMock(t)
		etherman.On("BuildTrustedVerifyBatchesTxData",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return([]byte{}, errors.New("error")).Once()

		s := createSilencer(defaultCfg, etherman)
		stx := tx.SignedTx{Data: tx.Tx{RollupID: defaultRollupID}}
		err := s.verify(context.Background(), stx)
		require.ErrorContains(t, err, "failed to build ZK proof verification tx data: error")

		etherman.AssertExpectations(t)
	})

	t.Run("ZK proof verification failure", func(t *testing.T) {
		etherman := mocks.NewEthermanMock(t)
		etherman.On("BuildTrustedVerifyBatchesTxData",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return([]byte{}, nil).Once()

		etherman.On("CallContract",
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return([]byte("Hello world!"), errors.New("failure")).Once()

		s := createSilencer(defaultCfg, etherman)
		stx := tx.SignedTx{Data: tx.Tx{RollupID: defaultRollupID}}
		err := s.verify(context.Background(), stx)
		require.ErrorContains(t, err, "failed to call ZK proof verification (response: Hello world!): failure")

		etherman.AssertExpectations(t)
	})

	t.Run("signature verification failure", func(t *testing.T) {
		etherman := mocks.NewEthermanMock(t)
		etherman.On("BuildTrustedVerifyBatchesTxData",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return([]byte{}, nil).Once()

		etherman.On("CallContract",
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return([]byte{}, nil).Once()

		s := createSilencer(defaultCfg, etherman)
		stx := tx.SignedTx{Data: tx.Tx{RollupID: defaultRollupID}}
		err := s.verify(context.Background(), stx)
		require.ErrorContains(t, err, "failed to resolve signer: invalid signature length")
	})
}
