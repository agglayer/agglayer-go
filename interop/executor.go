package interop

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/0xPolygon/agglayer/config"
	"github.com/0xPolygon/agglayer/tx"
	"github.com/0xPolygon/agglayer/types"
	"go.uber.org/zap"

	"github.com/0xPolygon/agglayer/log"
	jRPC "github.com/0xPolygon/cdk-data-availability/rpc"
	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/client"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

var _ types.IZkEVMClientClientCreator = (*zkEVMClientCreator)(nil)

type zkEVMClientCreator struct{}

func (zc *zkEVMClientCreator) NewClient(rpc string) types.IZkEVMClient {
	return client.NewClient(rpc)
}

type Executor struct {
	logger             *zap.SugaredLogger
	interopAdminAddr   common.Address
	config             *config.Config
	ethTxMan           types.IEthTxManager
	etherman           types.IEtherman
	ZkEVMClientCreator types.IZkEVMClientClientCreator
}

func New(logger *zap.SugaredLogger, cfg *config.Config,
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
		ZkEVMClientCreator: &zkEVMClientCreator{},
	}
}

const ethTxManOwner = "interop"

func (e *Executor) CheckTx(tx tx.SignedTx) error {

	// Check if the RPC is actually registered, if not it won't be possible to assert soundness (in the future once we are stateless won't be needed)
	// TODO: The JSON parsing of the contract is incorrect
	if _, ok := e.config.FullNodeRPCs[tx.Tx.RollupID]; !ok {
		return fmt.Errorf("there is no RPC registered for %v", tx.Tx.RollupID)
	}

	return nil
}

func (e *Executor) Verify(ctx context.Context, tx tx.SignedTx) error {
	err := e.verifyZKP(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to verify ZKP: %s", err)
	}

	return e.verifySignature(tx)
}

func (e *Executor) verifyZKP(ctx context.Context, stx tx.SignedTx) error {
	// Verify ZKP using eth_call
	l1TxData, err := e.etherman.BuildTrustedVerifyBatchesTxData(
		uint64(stx.Tx.LastVerifiedBatch),
		uint64(stx.Tx.NewVerifiedBatch),
		stx.Tx.ZKP,
		stx.Tx.RollupID,
	)
	if err != nil {
		return fmt.Errorf("failed to build verify ZKP tx: %s", err)
	}

	msg := ethereum.CallMsg{
		From: e.interopAdminAddr,
		To:   &e.config.L1.RollupManagerContract,
		Data: l1TxData,
	}
	log.Debugf("verify batches trusted L1 call: %v", msg)

	res, err := e.etherman.CallContract(ctx, msg, nil)
	if err != nil {
		return fmt.Errorf("failed to call verify ZKP response: %s, error: %s", res, err)
	}

	return nil
}

func (e *Executor) verifySignature(stx tx.SignedTx) error {
	// Auth: check signature vs admin
	signer, err := stx.Signer()
	if err != nil {
		return errors.New("failed to get signer")
	}

	sequencer, err := e.etherman.GetSequencerAddr(stx.Tx.RollupID)
	if err != nil {
		return errors.New("failed to get admin from L1")
	}
	if sequencer != signer {
		return errors.New("unexpected signer")
	}

	return nil
}

func (e *Executor) Execute(ctx context.Context, signedTx tx.SignedTx) error {
	// Check expected root vs root from the managed full node
	// TODO: go stateless, depends on https://github.com/0xPolygonHermez/zkevm-prover/issues/581
	// when this happens we should go async from here, since processing all the batches could take a lot of time
	zkEVMClient := e.ZkEVMClientCreator.NewClient(e.config.FullNodeRPCs[signedTx.Tx.RollupID])
	batch, err := zkEVMClient.BatchByNumber(
		ctx,
		big.NewInt(int64(signedTx.Tx.NewVerifiedBatch)),
	)
	if err != nil {
		return fmt.Errorf("failed to get batch from our node, error: %s", err)
	}
	log.Debugf("get batch by number: %v", batch)

	if batch.StateRoot != signedTx.Tx.ZKP.NewStateRoot || batch.LocalExitRoot != signedTx.Tx.ZKP.NewLocalExitRoot {
		return fmt.Errorf(
			"Mismatch detected,  expected local exit root: %s actual: %s. expected state root: %s actual: %s",
			signedTx.Tx.ZKP.NewLocalExitRoot.Hex(),
			batch.LocalExitRoot.Hex(),
			signedTx.Tx.ZKP.NewStateRoot.Hex(),
			batch.StateRoot.Hex(),
		)
	}

	return nil
}

func (e *Executor) Settle(ctx context.Context, signedTx tx.SignedTx, dbTx pgx.Tx) (common.Hash, error) {
	// Send L1 tx
	l1TxData, err := e.etherman.BuildTrustedVerifyBatchesTxData(
		uint64(signedTx.Tx.LastVerifiedBatch),
		uint64(signedTx.Tx.NewVerifiedBatch),
		signedTx.Tx.ZKP,
		signedTx.Tx.RollupID,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to build verify ZKP tx: %s", err)
	}

	if err := e.ethTxMan.Add(
		ctx,
		ethTxManOwner,
		signedTx.Tx.Hash().Hex(),
		e.interopAdminAddr,
		&e.config.L1.RollupManagerContract,
		big.NewInt(0),
		l1TxData,
		e.config.EthTxManager.GasOffset,
		dbTx,
	); err != nil {
		return common.Hash{}, fmt.Errorf("failed to add tx to ethTxMan, error: %s", err)
	}
	log.Debugf("successfuly added tx %s to ethTxMan", signedTx.Tx.Hash().Hex())
	return signedTx.Tx.Hash(), nil
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
