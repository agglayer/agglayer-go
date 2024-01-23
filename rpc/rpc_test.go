package rpc

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/0xPolygon/beethoven/config"
	"github.com/0xPolygon/beethoven/interop"
	"github.com/0xPolygon/beethoven/mocks"
	"github.com/0xPolygon/beethoven/workflow"

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
		w := workflow.New(&config.Config{}, interopAdmin, etherman)
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
		w := workflow.New(&config.Config{}, interopAdmin, etherman)
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
		w := workflow.New(&config.Config{}, interopAdmin, etherman)
		i := NewInteropEndpoints(context.Background(), e, w, dbMock)

		status, err := i.GetTxStatus(txHash)

		require.NoError(t, err)
		require.Equal(t, "confirmed", status)

		dbMock.AssertExpectations(t)
		txMock.AssertExpectations(t)
		txManagerMock.AssertExpectations(t)
	})
}
