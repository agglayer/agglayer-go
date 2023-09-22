package rpc

import (
	"context"
	"fmt"
	"math/big"

	"github.com/0xPolygon/cdk-validium-node/jsonrpc/client"
	"github.com/0xPolygon/cdk-validium-node/jsonrpc/types"
	"github.com/0xPolygon/silencer/tx"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// INTEROP is the namespace of the interop service
const (
	INTEROP       = "interop"
	ethTxManOwner = "interop"
)

type FullNodeRPCs map[common.Address]string

// InteropEndpoints contains implementations for the "interop" RPC endpoints
type InteropEndpoints struct {
	db               DBInterface
	etherman         EthermanInterface
	interopAdminAddr common.Address
	fullNodeRPCs     FullNodeRPCs
	ethTxManager     EthTxManager
}

// NewInteropEndpoints returns InteropEndpoints
func NewInteropEndpoints(
	interopAdminAddr common.Address,
	db DBInterface,
	etherman EthermanInterface,
	fullNodeRPCs FullNodeRPCs,
	ethTxManager EthTxManager,
) *InteropEndpoints {
	return &InteropEndpoints{
		db:               db,
		interopAdminAddr: interopAdminAddr,
		etherman:         etherman,
		fullNodeRPCs:     fullNodeRPCs,
		ethTxManager:     ethTxManager,
	}
}

func (i *InteropEndpoints) SendTx(signedTx tx.SignedTx) (interface{}, types.Error) {
	ctx := context.TODO()

	// Check if the RPC is actually registered, if not it won't be possible to assert soundness (in the future once we are stateless won't be needed)
	if _, ok := i.fullNodeRPCs[signedTx.Tx.L1Contract]; !ok {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, fmt.Sprintf("there is no RPC registered for %s", signedTx.Tx.L1Contract))
	}

	// Verify ZKP using eth_call
	l1TxData, err := i.etherman.BuildTrustedVerifyBatchesTxData(
		signedTx.Tx.L1Contract,
		uint64(signedTx.Tx.LastVerifiedBatch),
		uint64(signedTx.Tx.NewVerifiedBatch),
		signedTx.Tx.ZKP,
	)
	if err != nil {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, fmt.Sprintf("failed to build verify ZKP tx: %s", err))
	}
	msg := ethereum.CallMsg{
		From: i.interopAdminAddr,
		To:   &signedTx.Tx.L1Contract,
		Data: l1TxData,
	}
	res, err := i.etherman.CallContract(ctx, msg, nil)
	if err != nil {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, fmt.Sprintf("failed to call verify ZKP response: %s, error: %s", res, err))
	}

	// Auth: check signature vs admin
	signer, err := signedTx.Signer()
	if err != nil {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, "failed to get signer")
	}
	sequencer, err := i.etherman.GetSequencerAddr(signedTx.Tx.L1Contract)
	if err != nil {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, "failed to get admin from L1")
	}
	if sequencer != signer {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, "unexpected signer")
	}

	// Check expected root vs root from the managed full node
	// TODO: go stateless, depends on https://github.com/0xPolygonHermez/zkevm-prover/issues/581
	// when this happens we should go async from here, since processing all the batches could take a lot of time
	zkEVMClient := client.NewClient(i.fullNodeRPCs[signedTx.Tx.L1Contract])
	batch, err := zkEVMClient.BatchByNumber(
		ctx,
		big.NewInt(int64(signedTx.Tx.LastVerifiedBatch)),
	)
	if err != nil {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, fmt.Sprintf("failed to get batch from our node, error: %s", err))
	}
	if batch.StateRoot != signedTx.Tx.ZKP.NewStateRoot || batch.LocalExitRoot != signedTx.Tx.ZKP.NewLocalExitRoot {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, fmt.Sprintf(
			"Missmatch detected,  expected local exit root: %s actual: %s. expected state root: %s actual: %s",
			signedTx.Tx.ZKP.NewLocalExitRoot.Hex(),
			batch.LocalExitRoot.Hex(),
			signedTx.Tx.ZKP.NewStateRoot.Hex(),
			batch.StateRoot.Hex(),
		))
	}

	// Send L1 tx
	dbTx, err := i.db.BeginStateTransaction(ctx)
	if err != nil {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, fmt.Sprintf("failed to begin dbTx, error: %s", err))
	}
	err = i.ethTxManager.Add(ctx, ethTxManOwner, signedTx.Tx.Hash().Hex(), i.interopAdminAddr, &signedTx.Tx.L1Contract, nil, l1TxData, dbTx)
	if err != nil {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, fmt.Sprintf("failed to add tx to ethTxMan, error: %s", err))
	}
	return signedTx.Tx.Hash().Hex(), nil
}

func (i *InteropEndpoints) GetTxStatus(hash types.ArgHash) (interface{}, types.Error) {
	ctx := context.TODO()
	dbTx, err := i.db.BeginStateTransaction(ctx)
	if err != nil {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, fmt.Sprintf("failed to begin dbTx, error: %s", err))
	}
	res, err := i.ethTxManager.Result(ctx, ethTxManOwner, hash.Hash().Hex(), dbTx)
	if err != nil {
		return "0x0", types.NewRPCError(types.DefaultErrorCode, fmt.Sprintf("failed to get tx, error: %s", err))
	}
	return res.Status.String(), nil
}
