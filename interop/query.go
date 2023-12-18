package interop

import (
	"context"
	"fmt"

	"github.com/0xPolygon/cdk-validium-node/jsonrpc/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

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
