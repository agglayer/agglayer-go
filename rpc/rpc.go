package rpc

import (
	"context"
	"fmt"

	"github.com/0xPolygon/beethoven/interop"
	"github.com/0xPolygon/beethoven/types"

	rpctypes "github.com/0xPolygonHermez/zkevm-node/jsonrpc/types"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/beethoven/tx"
)

// INTEROP is the namespace of the interop service
const (
	INTEROP       = "interop"
	ethTxManOwner = "interop"
)

// InteropEndpoints contains implementations for the "interop" RPC endpoints
type InteropEndpoints struct {
	ctx      context.Context
	executor *interop.Executor
	db       types.DBInterface
}

// NewInteropEndpoints returns InteropEndpoints
func NewInteropEndpoints(
	ctx context.Context,
	executor *interop.Executor,
	db types.DBInterface,
) *InteropEndpoints {
	return &InteropEndpoints{
		ctx:      ctx,
		executor: executor,
		db:       db,
	}
}

func (i *InteropEndpoints) SendTx(signedTx tx.SignedTx) (interface{}, rpctypes.Error) {
	// Check if the RPC is actually registered, if not it won't be possible to assert soundness (in the future once we are stateless won't be needed)
	if err := i.executor.CheckTx(i.ctx, signedTx); err != nil {
		return "0x0", rpctypes.NewRPCError(rpctypes.DefaultErrorCode, fmt.Sprintf("there is no RPC registered for %s", signedTx.Tx.L1Contract))
	}

	// Verify ZKP using eth_call
	if err := i.executor.Verify(signedTx); err != nil {
		return "0x0", rpctypes.NewRPCError(rpctypes.DefaultErrorCode, fmt.Sprintf("failed to verify tx: %s", err))
	}

	if err := i.executor.Execute(signedTx); err != nil {
		return "0x0", rpctypes.NewRPCError(rpctypes.DefaultErrorCode, fmt.Sprintf("failed to execute tx: %s", err))
	}

	// Send L1 tx
	dbTx, err := i.db.BeginStateTransaction(i.ctx)
	if err != nil {
		return "0x0", rpctypes.NewRPCError(rpctypes.DefaultErrorCode, fmt.Sprintf("failed to begin dbTx, error: %s", err))
	}

	_, err = i.executor.Settle(i.ctx, signedTx, dbTx)
	if err != nil {
		if errRollback := dbTx.Rollback(i.ctx); errRollback != nil {
			log.Error("rollback err: ", errRollback)
		}
		return "0x0", rpctypes.NewRPCError(rpctypes.DefaultErrorCode, fmt.Sprintf("failed to add tx to ethTxMan, error: %s", err))
	}
	if err := dbTx.Commit(i.ctx); err != nil {
		return "0x0", rpctypes.NewRPCError(rpctypes.DefaultErrorCode, fmt.Sprintf("failed to commit dbTx, error: %s", err))
	}
	log.Debugf("successfuly added tx %s to ethTxMan", signedTx.Tx.Hash().Hex())

	return signedTx.Tx.Hash(), nil
}

func (i *InteropEndpoints) GetTxStatus(hash common.Hash) (result interface{}, err rpctypes.Error) {
	dbTx, innerErr := i.db.BeginStateTransaction(i.ctx)
	if innerErr != nil {
		result = "0x0"
		err = rpctypes.NewRPCError(rpctypes.DefaultErrorCode, fmt.Sprintf("failed to begin dbTx, error: %s", innerErr))

		return
	}

	defer func() {
		if innerErr := dbTx.Rollback(i.ctx); innerErr != nil {
			result = "0x0"
			err = rpctypes.NewRPCError(rpctypes.DefaultErrorCode, fmt.Sprintf("failed to rollback dbTx, error: %s", innerErr))
		}
	}()

	result, innerErr = i.executor.GetTxStatus(i.ctx, hash, dbTx)
	if innerErr != nil {
		result = "0x0"
		err = rpctypes.NewRPCError(rpctypes.DefaultErrorCode, fmt.Sprintf("failed to get tx, error: %s", innerErr))

		return
	}

	return
}
