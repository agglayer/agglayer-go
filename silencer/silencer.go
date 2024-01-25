package silencer

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/beethoven/config"
	"github.com/0xPolygon/beethoven/tx"
	"github.com/0xPolygon/beethoven/types"
)

type ISilencer interface {
	Silence(ctx context.Context, signedTx tx.SignedTx) error
}

type Silencer struct {
	cfg               *config.Config
	interopAdmin      common.Address
	etherman          types.IEtherman
	zkEVMClientsCache types.IZkEVMClientCache
}

// New returns new instance of Silencer
func New(cfg *config.Config,
	interopAdmin common.Address,
	etherman types.IEtherman,
	zkEVMClientsCache types.IZkEVMClientCache) *Silencer {
	return &Silencer{
		cfg:               cfg,
		interopAdmin:      interopAdmin,
		etherman:          etherman,
		zkEVMClientsCache: zkEVMClientsCache,
	}
}

// Silence runs soundness check
func (s *Silencer) Silence(ctx context.Context, signedTx tx.SignedTx) error {
	if err := s.verify(ctx, signedTx); err != nil {
		return err
	}

	// Check expected root vs root from the managed full node
	// TODO: go stateless, depends on https://github.com/0xPolygonHermez/zkevm-prover/issues/581
	// when this happens we should go async from here, since processing all the batches could take a lot of time
	txData := signedTx.Data
	zkEVMClient := s.zkEVMClientsCache.GetClient(s.cfg.FullNodeRPCs[txData.RollupID])

	batchNumber := new(big.Int).SetUint64(uint64(txData.NewVerifiedBatch))
	batch, err := zkEVMClient.BatchByNumber(ctx, batchNumber)
	if err != nil {
		return fmt.Errorf("failed to get batch from our node: %w", err)
	}

	if batch.StateRoot != txData.ZKP.NewStateRoot {
		return fmt.Errorf("mismatch in state roots detected (expected: '%s', actual: '%s')",
			batch.StateRoot.Hex(),
			txData.ZKP.NewStateRoot.Hex(),
		)
	}

	if batch.LocalExitRoot != txData.ZKP.NewLocalExitRoot {
		return fmt.Errorf("mismatch in local exit roots detected (expected: '%s', actual: '%s')",
			batch.StateRoot.Hex(),
			txData.ZKP.NewLocalExitRoot.Hex(),
		)
	}

	return nil
}

// verify performs set of validations against signedTx:
// 1. ZK proof verification
// 2. signature verification
func (s *Silencer) verify(ctx context.Context, stx tx.SignedTx) error {
	if _, ok := s.cfg.FullNodeRPCs[stx.Data.RollupID]; !ok {
		return fmt.Errorf("there is no RPC registered for rollup %d", stx.Data.RollupID)
	}

	if err := s.verifyZKProof(ctx, stx); err != nil {
		return err
	}

	return s.verifySignature(stx)
}

// verifyZKProof invokes SC that is accountable for verification of the provided ZK proof
func (s *Silencer) verifyZKProof(ctx context.Context, stx tx.SignedTx) error {
	// Verify ZKP using eth_call
	l1TxData, err := s.etherman.BuildTrustedVerifyBatchesTxData(
		uint64(stx.Data.LastVerifiedBatch),
		uint64(stx.Data.NewVerifiedBatch),
		stx.Data.ZKP,
		stx.Data.RollupID,
	)
	if err != nil {
		return fmt.Errorf("failed to build ZK proof verification tx data: %w", err)
	}

	msg := ethereum.CallMsg{
		From: s.interopAdmin,
		To:   &s.cfg.L1.RollupManagerContract,
		Data: l1TxData,
	}
	res, err := s.etherman.CallContract(ctx, msg, nil)
	if err != nil {
		if len(res) > 0 {
			return fmt.Errorf("failed to call ZK proof verification (response: %s): %w", res, err)
		}

		return fmt.Errorf("failed to call ZK proof verification: %w", err)
	}

	return nil
}

// verifySignature resolves tx signer and compares it against trusted sequencer address
func (s *Silencer) verifySignature(stx tx.SignedTx) error {
	signer, err := stx.Signer()
	if err != nil {
		return fmt.Errorf("failed to resolve signer: %w", err)
	}

	trustedSequencer, err := s.etherman.GetSequencerAddr(stx.Data.RollupID)
	if err != nil {
		return fmt.Errorf("failed to get trusted sequencer address: %w", err)
	}

	if trustedSequencer != signer {
		return errors.New("unexpected signer")
	}

	return nil
}
