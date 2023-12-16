package interop

import (
	"context"
	"fmt"

	"github.com/0xPolygon/beethoven/tx"
)

func (e *Executor) CheckTx(ctx context.Context, tx tx.SignedTx) error {
	e.logger.Debug("check tx")

	// Check if the RPC is actually registered, if not it won't be possible to assert soundness (in the future once we are stateless won't be needed)
	// TODO: The JSON parsing of the contract is incorrect
	if _, ok := e.config.FullNodeRPCs[tx.Tx.L1Contract]; !ok {
		return fmt.Errorf("there is no RPC registered for %s", tx.Tx.L1Contract)
	}

	return nil
}
