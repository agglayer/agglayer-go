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
	tests := []struct {
		name               string
		txStateRoot        common.Hash
		txLocalExitRoot    common.Hash
		batchStateRoot     common.Hash
		batchLocalExitRoot common.Hash
		clientErr          error
		expectedErrMsg     string
	}{
		{
			name:            "happy path",
			txStateRoot:     common.BytesToHash([]byte("sampleNewStateRoot")),
			txLocalExitRoot: common.BytesToHash([]byte("sampleExitRoot")),
			clientErr:       nil,
			expectedErrMsg:  "",
		},
		{
			name:            "failed to retrieve batch",
			txStateRoot:     common.BytesToHash([]byte("sampleNewStateRoot")),
			txLocalExitRoot: common.BytesToHash([]byte("sampleExitRoot")),
			clientErr:       errors.New("timeout"),
			expectedErrMsg:  "failed to get batch from our node: timeout",
		},
		{
			name:               "state roots mismatch",
			txStateRoot:        common.BytesToHash([]byte("txStateRoot")),
			txLocalExitRoot:    common.BytesToHash([]byte("txExitRoot")),
			batchStateRoot:     common.BytesToHash([]byte("batchStateRoot")),
			batchLocalExitRoot: common.BytesToHash([]byte("batchExitRoot")),
			clientErr:          nil,
			expectedErrMsg:     "mismatch in state roots detected",
		},
		{
			name:               "local exit roots mismatch",
			txStateRoot:        common.BytesToHash([]byte("stateRoot")),
			txLocalExitRoot:    common.BytesToHash([]byte("txExitRoot")),
			batchStateRoot:     common.BytesToHash([]byte("stateRoot")),
			batchLocalExitRoot: common.BytesToHash([]byte("batchExitRoot")),
			clientErr:          nil,
			expectedErrMsg:     "mismatch in local exit roots detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const rollupID = uint32(10)

			cfg := &config.Config{FullNodeRPCs: map[uint32]string{rollupID: "http://localhost:10000"}}
			interopAdminAddr := common.HexToAddress("0x1001")

			createMockEtherman := func(sequencerAddr common.Address) *mocks.EthermanMock {
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

				etherman.On(
					"GetSequencerAddr",
					mock.Anything,
				).Return(
					sequencerAddr,
					nil,
				).Once()

				return etherman
			}

			setupMockZkEVMClient := func(batchStateRoot, batchLocalExitRoot common.Hash, err error) (
				*mocks.ZkEVMClientClientCreatorMock,
				*mocks.ZkEVMClientMock) {
				clientMock := mocks.NewZkEVMClientMock(t)
				if err == nil {
					batch := &rpctypes.Batch{
						StateRoot:     batchStateRoot,
						LocalExitRoot: batchLocalExitRoot,
					}
					clientMock.On("BatchByNumber", mock.Anything, mock.Anything).
						Return(batch, nil).Once()
				} else {
					clientMock.On("BatchByNumber", mock.Anything, mock.Anything).
						Return(nil, err).Once()
				}

				clientCreatorMock := mocks.NewZkEVMClientClientCreatorMock(t)
				clientCreatorMock.On("NewClient", mock.Anything).
					Return(clientMock).Once()

				return clientCreatorMock, clientMock
			}

			tx := tx.Tx{
				LastVerifiedBatch: 1,
				NewVerifiedBatch:  2,
				RollupID:          rollupID,
				ZKP: tx.ZKP{
					NewStateRoot:     tt.txStateRoot,
					NewLocalExitRoot: tt.txLocalExitRoot,
				},
			}

			sequencerKey, err := crypto.GenerateKey()
			require.NoError(t, err)

			signedTx, err := tx.Sign(sequencerKey)
			require.NoError(t, err)

			etherman := createMockEtherman(crypto.PubkeyToAddress(sequencerKey.PublicKey))
			batchStateRoot := tx.ZKP.NewStateRoot
			batchLocalExitRoot := tx.ZKP.NewLocalExitRoot
			if (tt.batchStateRoot != common.Hash{}) {
				batchStateRoot = tt.batchStateRoot
			}
			if (tt.batchLocalExitRoot != common.Hash{}) {
				batchLocalExitRoot = tt.batchLocalExitRoot
			}
			clientCreatorMock, clientMock := setupMockZkEVMClient(batchStateRoot, batchLocalExitRoot, tt.clientErr)

			silencer := New(cfg, interopAdminAddr, etherman, clientCreatorMock)
			err = silencer.Silence(context.Background(), *signedTx)

			if tt.expectedErrMsg != "" {
				require.ErrorContains(t, err, tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}

			clientCreatorMock.AssertExpectations(t)
			clientMock.AssertExpectations(t)
		})
	}
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

	t.Run("signature verification failure (no signature)", func(t *testing.T) {
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

	t.Run("signature verification failure (failed to retrieve sequencer addr)", func(t *testing.T) {
		getSequencerAddrErr := errors.New("execution failed")

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

		etherman.On(
			"GetSequencerAddr",
			mock.Anything,
		).Return(
			common.Address{},
			getSequencerAddrErr,
		).Once()

		s := createSilencer(defaultCfg, etherman)
		txData := tx.Tx{RollupID: defaultRollupID}

		pk, err := crypto.GenerateKey()
		require.NoError(t, err)

		signedTx, err := txData.Sign(pk)
		require.NoError(t, err)

		err = s.verify(context.Background(), *signedTx)
		require.ErrorContains(t, err, fmt.Sprintf("failed to get trusted sequencer address: %s", getSequencerAddrErr.Error()))
	})

	t.Run("signature verification failure (sequencer is not the tx signer)", func(t *testing.T) {
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

		etherman.On(
			"GetSequencerAddr",
			mock.Anything,
		).Return(
			common.HexToAddress("0x12345"),
			nil,
		).Once()

		s := createSilencer(defaultCfg, etherman)
		txData := tx.Tx{RollupID: defaultRollupID}

		pk, err := crypto.GenerateKey()
		require.NoError(t, err)

		signedTx, err := txData.Sign(pk)
		require.NoError(t, err)

		err = s.verify(context.Background(), *signedTx)
		require.ErrorContains(t, err, "unexpected signer")
	})
}
