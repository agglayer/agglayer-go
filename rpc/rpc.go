package rpc

import (
	"context"
	"fmt"

	jRPC "github.com/0xPolygon/cdk-data-availability/rpc"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/agglayer/config"
	"github.com/0xPolygon/agglayer/interop"
	"github.com/0xPolygon/agglayer/tx"
	"github.com/0xPolygon/agglayer/types"
)

// INTEROP is the namespace of the interop service
const (
	INTEROP       = "interop"
	ethTxManOwner = "interop"
)

// InteropEndpoints contains implementations for the "interop" RPC endpoints
type InteropEndpoints struct {
	executor *interop.Executor
	db       types.IDB
	config   *config.Config
}

// NewInteropEndpoints returns InteropEndpoints
func NewInteropEndpoints(
	executor *interop.Executor,
	db types.IDB,
	conf *config.Config,
) *InteropEndpoints {
	return &InteropEndpoints{
		executor: executor,
		db:       db,
		config:   conf,
	}
}

func (i *InteropEndpoints) SendTx(signedTx tx.SignedTx) (interface{}, jRPC.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), i.config.RPC.WriteTimeout.Duration)
	defer cancel()

	// Check if the RPC is actually registered, if not it won't be possible to assert soundness (in the future once we are stateless won't be needed)
	if err := i.executor.CheckTx(signedTx); err != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("there is no RPC registered for %d", signedTx.Tx.RollupID))
	}

	// Verify ZKP using eth_call
	if err := i.executor.Verify(ctx, signedTx); err != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to verify tx: %s", err))
	}

	if err := i.executor.Execute(ctx, signedTx); err != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to execute tx: %s", err))
	}

	// Send L1 tx
	dbTx, err := i.db.BeginStateTransaction(ctx)
	if err != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to begin dbTx, error: %s", err))
	}

	_, err = i.executor.Settle(ctx, signedTx, dbTx)
	if err != nil {
		if errRollback := dbTx.Rollback(ctx); errRollback != nil {
			log.Error("rollback err: ", errRollback)
		}
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to add tx to ethTxMan, error: %s", err))
	}
	if err := dbTx.Commit(ctx); err != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to commit dbTx, error: %s", err))
	}
	log.Debugf("successfuly added tx %s to ethTxMan", signedTx.Tx.Hash().Hex())

	return signedTx.Tx.Hash(), nil
}

func (i *InteropEndpoints) GetTxStatus(hash common.Hash) (result interface{}, err jRPC.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), i.config.RPC.ReadTimeout.Duration)
	defer cancel()

	dbTx, innerErr := i.db.BeginStateTransaction(ctx)
	if innerErr != nil {
		result = "0x0"
		err = jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to begin dbTx, error: %s", innerErr))

		return
	}

	defer func() {
		if innerErr := dbTx.Rollback(ctx); innerErr != nil {
			result = "0x0"
			err = jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to rollback dbTx, error: %s", innerErr))
		}
	}()

	result, innerErr = i.executor.GetTxStatus(ctx, hash, dbTx)
	if innerErr != nil {
		result = "0x0"
		err = jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to get tx, error: %s", innerErr))

		return
	}

	return
}
