package interop

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/0xPolygon/beethoven/config"
	"github.com/0xPolygon/beethoven/tx"
	"github.com/0xPolygon/beethoven/types"

	jRPC "github.com/0xPolygon/cdk-data-availability/rpc"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

type Executor struct {
	logger             *log.Logger
	interopAdminAddr   common.Address
	config             *config.Config
	ethTxMan           types.IEthTxManager
	etherman           types.IEtherman
	ZkEVMClientCreator types.IZkEVMClientClientCreator
}

func New(logger *log.Logger, cfg *config.Config,
	interopAdminAddr common.Address,
	etherman types.IEtherman,
	ethTxManager types.IEthTxManager,
) *Executor {
	return &Executor{
		logger:             logger,
		interopAdminAddr:   interopAdminAddr,
		config:             cfg,
		ethTxMan:           ethTxManager,
		etherman:           etherman,
		ZkEVMClientCreator: &types.ZkEVMClientCreator{},
	}
}

const ethTxManOwner = "interop"

// @Stefan-Ethernal: Moved to Silencer.validate
func (e *Executor) CheckTx(tx tx.SignedTx) error {
	// Check if the RPC is actually registered, if not it won't be possible to assert soundness (in the future once we are stateless won't be needed)
	// TODO: The JSON parsing of the contract is incorrect
	if _, ok := e.config.FullNodeRPCs[tx.Data.RollupID]; !ok {
		return fmt.Errorf("there is no RPC registered for %v", tx.Data.RollupID)
	}

	return nil
}

// @Stefan-Ethernal: Moved to Silencer.verify
func (e *Executor) Verify(ctx context.Context, tx tx.SignedTx) error {
	err := e.verifyZKP(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to verify ZKP: %s", err)
	}

	return e.verifySignature(tx)
}

// @Stefan-Ethernal: Moved to Silencer.verifyZKProof
func (e *Executor) verifyZKP(ctx context.Context, stx tx.SignedTx) error {
	// Verify ZKP using eth_call
	l1TxData, err := e.etherman.BuildTrustedVerifyBatchesTxData(
		uint64(stx.Data.LastVerifiedBatch),
		uint64(stx.Data.NewVerifiedBatch),
		stx.Data.ZKP,
		stx.Data.RollupID,
	)
	if err != nil {
		return fmt.Errorf("failed to build verify ZKP tx: %s", err)
	}
	msg := ethereum.CallMsg{
		From: e.interopAdminAddr,
		To:   &e.config.L1.RollupManagerContract,
		Data: l1TxData,
	}
	res, err := e.etherman.CallContract(ctx, msg, nil)
	if err != nil {
		return fmt.Errorf("failed to call verify ZKP response: %s, error: %w", res, err)
	}

	return nil
}

// @Stefan-Ethernal: Moved to Silencer.verifyZKProof
func (e *Executor) verifySignature(stx tx.SignedTx) error {
	// Auth: check signature vs admin
	signer, err := stx.Signer()
	if err != nil {
		return errors.New("failed to get signer")
	}

	sequencer, err := e.etherman.GetSequencerAddr(stx.Data.RollupID)
	if err != nil {
		return errors.New("failed to get admin from L1")
	}
	if sequencer != signer {
		return errors.New("unexpected signer")
	}

	return nil
}

// @Stefan-Ethernal: Remove, moved into Silencer.Silence
func (e *Executor) Execute(ctx context.Context, signedTx tx.SignedTx) error {
	// Check expected root vs root from the managed full node
	// TODO: go stateless, depends on https://github.com/0xPolygonHermez/zkevm-prover/issues/581
	// when this happens we should go async from here, since processing all the batches could take a lot of time
	zkEVMClient := e.ZkEVMClientCreator.NewClient(e.config.FullNodeRPCs[signedTx.Data.RollupID])
	batch, err := zkEVMClient.BatchByNumber(
		ctx,
		new(big.Int).SetUint64(uint64(signedTx.Data.NewVerifiedBatch)),
	)
	if err != nil {
		return fmt.Errorf("failed to get batch from our node: %w", err)
	}

	if batch.StateRoot != signedTx.Data.ZKP.NewStateRoot {
		return fmt.Errorf("mismatch in state roots detected (expected: '%s', actual: '%s')",
			signedTx.Data.ZKP.NewStateRoot.Hex(),
			batch.StateRoot.Hex(),
		)
	}

	if batch.LocalExitRoot != signedTx.Data.ZKP.NewLocalExitRoot {
		return fmt.Errorf("mismatch in local exit roots detected (expected: '%s', actual: '%s')",
			signedTx.Data.ZKP.NewLocalExitRoot.Hex(),
			batch.StateRoot.Hex())
	}

	return nil
}

func (e *Executor) Settle(ctx context.Context, signedTx tx.SignedTx, dbTx pgx.Tx) (common.Hash, error) {
	// Send L1 tx
	l1TxData, err := e.etherman.BuildTrustedVerifyBatchesTxData(
		uint64(signedTx.Data.LastVerifiedBatch),
		uint64(signedTx.Data.NewVerifiedBatch),
		signedTx.Data.ZKP,
		signedTx.Data.RollupID,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to build verify ZKP tx: %s", err)
	}

	if err := e.ethTxMan.Add(
		ctx,
		ethTxManOwner,
		signedTx.Data.Hash().Hex(),
		e.interopAdminAddr,
		&e.config.L1.RollupManagerContract,
		big.NewInt(0),
		l1TxData,
		0,
		dbTx,
	); err != nil {
		return common.Hash{}, fmt.Errorf("failed to add tx to ethTxMan, error: %s", err)
	}
	log.Debugf("successfuly added tx %s to ethTxMan", signedTx.Data.Hash().Hex())
	return signedTx.Data.Hash(), nil
}

func (e *Executor) GetTxStatus(ctx context.Context, hash common.Hash, dbTx pgx.Tx) (result string, err jRPC.Error) {
	res, innerErr := e.ethTxMan.Result(ctx, ethTxManOwner, hash.Hex(), dbTx)
	if innerErr != nil {
		result = "0x0"
		err = jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to get tx, error: %s", innerErr))

		return
	}

	result = res.Status.String()

	return
}
