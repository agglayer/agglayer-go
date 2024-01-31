package rpc

import (
	"context"
	"fmt"

	jRPC "github.com/0xPolygon/cdk-data-availability/rpc"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/agglayer/interop"
	"github.com/0xPolygon/agglayer/tx"
	"github.com/0xPolygon/agglayer/types"
	"github.com/0xPolygon/agglayer/workflow"
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
	workflow *workflow.Workflow
	db       types.IDB
}

// NewInteropEndpoints returns InteropEndpoints
func NewInteropEndpoints(
	ctx context.Context,
	executor *interop.Executor,
	workflow *workflow.Workflow,
	db types.IDB,
) *InteropEndpoints {
	return &InteropEndpoints{
		ctx:      ctx,
		executor: executor,
		workflow: workflow,
		db:       db,
	}
}

func (i *InteropEndpoints) SendTx(signedTx tx.SignedTx) (interface{}, jRPC.Error) {
	if err := i.workflow.Execute(i.ctx, signedTx); err != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, err.Error())
	}

	// Send L1 tx
	dbTx, err := i.db.BeginStateTransaction(i.ctx)
	if err != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to begin dbTx, error: %s", err))
	}

	_, err = i.executor.Settle(i.ctx, signedTx, dbTx)
	if err != nil {
		if errRollback := dbTx.Rollback(i.ctx); errRollback != nil {
			log.Error("rollback err: ", errRollback)
		}
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to add tx to ethTxMan, error: %s", err))
	}
	if err := dbTx.Commit(i.ctx); err != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to commit dbTx, error: %s", err))
	}
	log.Debugf("successfuly added tx %s to ethTxMan", signedTx.Data.Hash().Hex())

	return signedTx.Data.Hash(), nil
}

func (i *InteropEndpoints) GetTxStatus(hash common.Hash) (result interface{}, err jRPC.Error) {
	dbTx, innerErr := i.db.BeginStateTransaction(i.ctx)
	if innerErr != nil {
		result = "0x0"
		err = jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to begin dbTx, error: %s", innerErr))

		return
	}

	defer func() {
		if innerErr := dbTx.Rollback(i.ctx); innerErr != nil {
			result = "0x0"
			err = jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to rollback dbTx, error: %s", innerErr))
		}
	}()

	result, innerErr = i.executor.GetTxStatus(i.ctx, hash, dbTx)
	if innerErr != nil {
		result = "0x0"
		err = jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to get tx, error: %s", innerErr))

		return
	}

	return
}
