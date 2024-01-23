package interop

import (
	"context"
	"fmt"
	"math/big"

	"github.com/0xPolygon/beethoven/config"
	"github.com/0xPolygon/beethoven/tx"
	"github.com/0xPolygon/beethoven/types"

	jRPC "github.com/0xPolygon/cdk-data-availability/rpc"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

type Executor struct {
	logger           *log.Logger
	interopAdminAddr common.Address
	config           *config.Config
	ethTxMan         types.IEthTxManager
	etherman         types.IEtherman
}

func New(logger *log.Logger, cfg *config.Config,
	interopAdminAddr common.Address,
	etherman types.IEtherman,
	ethTxManager types.IEthTxManager,
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
