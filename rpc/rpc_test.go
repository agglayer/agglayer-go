package rpc

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xPolygon/agglayer/config"
	"github.com/0xPolygon/agglayer/interop"
	"github.com/0xPolygon/agglayer/mocks"
	"github.com/0xPolygon/agglayer/tx"
	"github.com/0xPolygon/agglayer/workflow"

	"github.com/0xPolygonHermez/zkevm-node/ethtxmanager"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInteropEndpointsGetTxStatus(t *testing.T) {
	t.Parallel()

	t.Run("BeginStateTransaction returns an error", func(t *testing.T) {
		t.Parallel()

		dbMock := mocks.NewDBMock(t)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(nil, errors.New("error")).Once()

		interopAdmin := common.HexToAddress("0xadmin")
		etherman := mocks.NewEthermanMock(t)

		e := interop.New(
			log.WithFields("module", "test"),
			&config.Config{},
			interopAdmin,
			etherman,
			mocks.NewEthTxManagerMock(t),
		)
		w := workflow.New(mocks.NewSilencerMock(t))
		i := NewInteropEndpoints(context.Background(), e, w, dbMock)

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

		dbMock := mocks.NewDBMock(t)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(txMock, nil).Once()

		txManagerMock := mocks.NewEthTxManagerMock(t)
		txManagerMock.On("Result", mock.Anything, ethTxManOwner, txHash.Hex(), txMock).
			Return(ethtxmanager.MonitoredTxResult{}, errors.New("error")).Once()

		interopAdmin := common.HexToAddress("0xadmin")
		etherman := mocks.NewEthermanMock(t)

		e := interop.New(
			log.WithFields("module", "test"),
			&config.Config{},
			interopAdmin,
			etherman,
			txManagerMock,
		)
		w := workflow.New(mocks.NewSilencerMock(t))
		i := NewInteropEndpoints(context.Background(), e, w, dbMock)

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

		dbMock := mocks.NewDBMock(t)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(txMock, nil).Once()

		txManagerMock := mocks.NewEthTxManagerMock(t)
		txManagerMock.On("Result", mock.Anything, ethTxManOwner, txHash.Hex(), txMock).
			Return(result, nil).Once()

		interopAdmin := common.HexToAddress("0xadmin")
		etherman := mocks.NewEthermanMock(t)

		e := interop.New(
			log.WithFields("module", "test"),
			&config.Config{},
			interopAdmin,
			etherman,
			txManagerMock,
		)
		w := workflow.New(mocks.NewSilencerMock(t))
		i := NewInteropEndpoints(context.Background(), e, w, dbMock)

		status, err := i.GetTxStatus(txHash)

		require.NoError(t, err)
		require.Equal(t, "confirmed", status)

		dbMock.AssertExpectations(t)
		txMock.AssertExpectations(t)
		txManagerMock.AssertExpectations(t)
	})
}

func TestInteropEndpoints_SendTx(t *testing.T) {
	interopAdmin := common.HexToAddress("0x2002")

	type testCase struct {
		dbMock           *mocks.DBMock
		ethermanMock     *mocks.EthermanMock
		silencerMock     *mocks.SilencerMock
		ethTxManagerMock *mocks.EthTxManagerMock
		tx               *tx.SignedTx
		expectedRes      interface{}
		expectedErr      error
	}

	runTest := func(t *testing.T, tc testCase) {
		ethTxManagerMock := tc.ethTxManagerMock
		if ethTxManagerMock == nil {
			ethTxManagerMock = mocks.NewEthTxManagerMock(t)
		}

		e := interop.New(
			log.WithFields("module", "test"),
			&config.Config{},
			interopAdmin,
			tc.ethermanMock,
			ethTxManagerMock,
		)
		w := workflow.New(tc.silencerMock)
		i := NewInteropEndpoints(context.Background(), e, w, tc.dbMock)

		stx := tc.tx
		if tc.tx == nil {
			stx = &tx.SignedTx{}
		}

		res, err := i.SendTx(*stx)
		require.Equal(t, tc.expectedRes, res)
		if tc.expectedErr != nil {
			require.EqualError(t, err, tc.expectedErr.Error())
		} else {
			require.NoError(t, err)
		}

		tc.dbMock.AssertExpectations(t)
		tc.ethermanMock.AssertExpectations(t)
		tc.silencerMock.AssertExpectations(t)
		ethTxManagerMock.AssertExpectations(t)
	}

	t.Run("silencer execution failed", func(t *testing.T) {
		dbMock := mocks.NewDBMock(t)
		expectedErr := errors.New("soundness check failed")

		ethermanMock := mocks.NewEthermanMock(t)
		silencerMock := mocks.NewSilencerMock(t)
		silencerMock.On("Silence", mock.Anything, mock.Anything).Return(expectedErr).Once()

		tc := testCase{
			dbMock:       dbMock,
			ethermanMock: ethermanMock,
			silencerMock: silencerMock,
			expectedRes:  "0x0",
			expectedErr:  expectedErr,
		}

		runTest(t, tc)
	})

	t.Run("begin state tx failed", func(t *testing.T) {
		dbMock := mocks.NewDBMock(t)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(nil, errors.New("failure")).Once()

		ethermanMock := mocks.NewEthermanMock(t)
		silencerMock := mocks.NewSilencerMock(t)
		silencerMock.On("Silence", mock.Anything, mock.Anything).Return(nil).Once()

		tc := testCase{
			dbMock:       dbMock,
			ethermanMock: ethermanMock,
			silencerMock: silencerMock,
			expectedRes:  "0x0",
			expectedErr:  errors.New("failed to begin dbTx, error: failure"),
		}

		runTest(t, tc)
	})

	t.Run("settle failed", func(t *testing.T) {
		expectedErr := errors.New("ABI encoding is invalid")

		txMock := new(mocks.TxMock)
		txMock.On("Rollback", mock.Anything).Return(nil).Once()

		dbMock := mocks.NewDBMock(t)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(txMock, nil).Once()

		ethermanMock := mocks.NewEthermanMock(t)
		ethermanMock.On("BuildTrustedVerifyBatchesTxData",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, expectedErr).Once()

		silencerMock := mocks.NewSilencerMock(t)
		silencerMock.On("Silence", mock.Anything, mock.Anything).Return(nil).Once()

		tc := testCase{
			dbMock:       dbMock,
			silencerMock: silencerMock,
			ethermanMock: ethermanMock,
			expectedRes:  "0x0",
			expectedErr:  fmt.Errorf("failed to add tx to ethTxMan, error: failed to build ZK proof verification tx data: %w", expectedErr),
		}

		runTest(t, tc)
	})

	t.Run("db tx commit failed", func(t *testing.T) {
		expectedErr := errors.New("commit has failed")

		txMock := new(mocks.TxMock)
		txMock.On("Commit", mock.Anything).Return(expectedErr).Once()

		dbMock := mocks.NewDBMock(t)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(txMock, nil).Once()

		ethermanMock := mocks.NewEthermanMock(t)
		ethermanMock.On("BuildTrustedVerifyBatchesTxData",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()

		silencerMock := mocks.NewSilencerMock(t)
		silencerMock.On("Silence", mock.Anything, mock.Anything).Return(nil).Once()

		ethTxManagerMock := mocks.NewEthTxManagerMock(t)
		ethTxManagerMock.On("Add",
			mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		tc := testCase{
			dbMock:           dbMock,
			silencerMock:     silencerMock,
			ethermanMock:     ethermanMock,
			ethTxManagerMock: ethTxManagerMock,
			expectedRes:      "0x0",
			expectedErr:      fmt.Errorf("failed to commit dbTx, error: %w", expectedErr),
		}

		runTest(t, tc)
	})

	t.Run("send tx successful", func(t *testing.T) {
		txMock := new(mocks.TxMock)
		txMock.On("Commit", mock.Anything).Return(nil).Once()

		dbMock := mocks.NewDBMock(t)
		dbMock.On("BeginStateTransaction", mock.Anything).Return(txMock, nil).Once()

		ethermanMock := mocks.NewEthermanMock(t)
		ethermanMock.On("BuildTrustedVerifyBatchesTxData",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()

		silencerMock := mocks.NewSilencerMock(t)
		silencerMock.On("Silence", mock.Anything, mock.Anything).Return(nil).Once()

		ethTxManagerMock := mocks.NewEthTxManagerMock(t)
		ethTxManagerMock.On("Add",
			mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		signedTx := tx.SignedTx{
			Data: tx.Tx{
				RollupID:          uint32(10),
				LastVerifiedBatch: 5,
				NewVerifiedBatch:  10,
			}}

		expectedHash := signedTx.Data.Hash()

		tc := testCase{
			dbMock:           dbMock,
			silencerMock:     silencerMock,
			ethermanMock:     ethermanMock,
			ethTxManagerMock: ethTxManagerMock,
			tx:               &signedTx,
			expectedRes:      expectedHash,
			expectedErr:      nil,
		}

		runTest(t, tc)
	})
}
