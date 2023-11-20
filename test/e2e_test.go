package test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygon/cdk-validium-node/log"
	"github.com/0xPolygon/cdk-validium-node/test/operations"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestEthTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	cluster, err := newTestCluster("")
	require.NoError(t, err)

	ctx := context.Background()
	defer func() {
		msg, err := cluster.stop()
		require.NoError(t, err, string(msg))
	}()

	log.Info("restarting docker containers for the test")
	msg, err := cluster.stop()
	require.NoError(t, err, string(msg))
	msg, err = cluster.start()
	require.NoError(t, err, string(msg))
	time.Sleep(5 * time.Second)

	// Load account with balance on local genesis
	log.Info("generating txs")
	auth, err := operations.GetAuth(operations.DefaultSequencerPrivateKey, operations.DefaultL2ChainID)
	require.NoError(t, err)
	// Load eth client
	client, err := ethclient.Dial(operations.DefaultL2NetworkURL)
	require.NoError(t, err)
	// Send txs
	nTxs := 10
	amount := big.NewInt(10000)
	toAddress := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{From: auth.From, To: &toAddress, Value: amount})
	require.NoError(t, err)
	gasPrice, err := client.SuggestGasPrice(ctx)
	require.NoError(t, err)
	nonce, err := client.PendingNonceAt(ctx, auth.From)
	require.NoError(t, err)
	txs := make([]*types.Transaction, 0, nTxs)
	for i := 0; i < nTxs; i++ {
		tx := types.NewTransaction(nonce+uint64(i), toAddress, amount, gasLimit, gasPrice, nil)
		txs = append(txs, tx)
	}

	log.Info("sending txs")
	_, err = operations.ApplyL2Txs(ctx, txs, auth, client, operations.VerifiedConfirmationLevel)
	require.NoError(t, err)
}
