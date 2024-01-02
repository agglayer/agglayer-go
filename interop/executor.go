package interop

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/0xPolygon/beethoven/config"
	"github.com/0xPolygon/beethoven/tx"

	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/client"
	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/types"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

type Executor struct {
	logger           *log.Logger
	interopAdminAddr common.Address
	config           *config.Config
	ethTxMan         EthTxManager
	etherman         EthermanInterface
}

func New(logger *log.Logger, cfg *config.Config,
	interopAdminAddr common.Address,
	etherman EthermanInterface,
	ethTxManager EthTxManager,
) *Executor {
	return &Executor{
		logger:           logger,
		interopAdminAddr: interopAdminAddr,
		config:           cfg,
		ethTxMan:         ethTxManager,
		etherman:         etherman,
	}
}

const ethTxManOwner = "interop"

func (e *Executor) CheckTx(ctx context.Context, tx tx.SignedTx) error {
	e.logger.Debug("check tx")

	// Check if the RPC is actually registered, if not it won't be possible to assert soundness (in the future once we are stateless won't be needed)
	// TODO: The JSON parsing of the contract is incorrect
	if _, ok := e.config.FullNodeRPCs[tx.Tx.L1Contract]; !ok {
		return fmt.Errorf("there is no RPC registered for %s", tx.Tx.L1Contract)
	}

	return nil
}

func (e *Executor) Verify(tx tx.SignedTx) error {
	err := e.VerifyZKP(tx)
	if err != nil {
		return fmt.Errorf("failed to verify ZKP: %s", err)
	}

	return e.VerifySignature(tx)
}

func (e *Executor) VerifyZKP(stx tx.SignedTx) error {
	// Verify ZKP using eth_call
	l1TxData, err := e.etherman.BuildTrustedVerifyBatchesTxData(
		uint64(stx.Tx.LastVerifiedBatch),
		uint64(stx.Tx.NewVerifiedBatch),
		stx.Tx.ZKP,
	)
	if err != nil {
		return fmt.Errorf("failed to build verify ZKP tx: %s", err)
	}
	msg := ethereum.CallMsg{
		From: e.interopAdminAddr,
		To:   &stx.Tx.L1Contract,
		Data: l1TxData,
	}
	res, err := e.etherman.CallContract(context.Background(), msg, nil)
	if err != nil {
		return fmt.Errorf("failed to call verify ZKP response: %s, error: %s", res, err)
	}

	return nil
}

func (e *Executor) VerifySignature(stx tx.SignedTx) error {
	// Auth: check signature vs admin
	signer, err := stx.Signer()
	if err != nil {
		return errors.New("failed to get signer")
	}

	sequencer, err := e.etherman.GetSequencerAddr(stx.Tx.L1Contract)
	if err != nil {
		return errors.New("failed to get admin from L1")
	}
	if sequencer != signer {
		return errors.New("unexpected signer")
	}

	return nil
}

func (e *Executor) Execute(signedTx tx.SignedTx) error {
	ctx := context.TODO()

	// Check expected root vs root from the managed full node
	// TODO: go stateless, depends on https://github.com/0xPolygonHermez/zkevm-prover/issues/581
	// when this happens we should go async from here, since processing all the batches could take a lot of time
	zkEVMClient := client.NewClient(e.config.FullNodeRPCs[signedTx.Tx.L1Contract])
	batch, err := zkEVMClient.BatchByNumber(
		ctx,
		big.NewInt(int64(signedTx.Tx.NewVerifiedBatch)),
	)
	if err != nil {
		return fmt.Errorf("failed to get batch from our node, error: %s", err)
	}
	if batch.StateRoot != signedTx.Tx.ZKP.NewStateRoot || batch.LocalExitRoot != signedTx.Tx.ZKP.NewLocalExitRoot {
		return fmt.Errorf(
			"Missmatch detected,  expected local exit root: %s actual: %s. expected state root: %s actual: %s",
			signedTx.Tx.ZKP.NewLocalExitRoot.Hex(),
			batch.LocalExitRoot.Hex(),
			signedTx.Tx.ZKP.NewStateRoot.Hex(),
			batch.StateRoot.Hex(),
		)
	}

	return nil
}

func (e *Executor) Settle(signedTx tx.SignedTx, dbTx pgx.Tx) (common.Hash, error) {
	// // Send L1 tx
	// Verify ZKP using eth_call
	l1TxData, err := e.etherman.BuildTrustedVerifyBatchesTxData(
		uint64(signedTx.Tx.LastVerifiedBatch),
		uint64(signedTx.Tx.NewVerifiedBatch),
		signedTx.Tx.ZKP,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to build verify ZKP tx: %s", err)
	}

	if err := e.ethTxMan.Add(
		context.Background(),
		ethTxManOwner,
		signedTx.Tx.Hash().Hex(),
		e.interopAdminAddr,
		&signedTx.Tx.L1Contract,
		nil,
		l1TxData,
		0,
		dbTx,
	); err != nil {
		return common.Hash{}, fmt.Errorf("failed to add tx to ethTxMan, error: %s", err)
	}
	log.Debugf("successfuly added tx %s to ethTxMan", signedTx.Tx.Hash().Hex())
	return signedTx.Tx.Hash(), nil
}

func (e *Executor) GetTxStatus(ctx context.Context, hash common.Hash, dbTx pgx.Tx) (result string, err types.Error) {
	res, innerErr := e.ethTxMan.Result(ctx, ethTxManOwner, hash.Hex(), dbTx)
	if innerErr != nil {
		result = "0x0"
		err = types.NewRPCError(types.DefaultErrorCode, fmt.Sprintf("failed to get tx, error: %s", innerErr))

		return
	}

	result = res.Status.String()

	return
}
