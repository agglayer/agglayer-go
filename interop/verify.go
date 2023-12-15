package interop

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xPolygon/beethoven/tx"

	"github.com/ethereum/go-ethereum"
)

func (i *Interop) Verify(tx tx.SignedTx) error {
	err := i.VerifyZKP(tx)
	if err != nil {
		return fmt.Errorf("failed to verify ZKP: %s", err)
	}

	return i.VerifySignature(tx)
}

func (i *Interop) VerifyZKP(stx tx.SignedTx) error {
	// Verify ZKP using eth_call
	l1TxData, err := i.etherman.BuildTrustedVerifyBatchesTxData(
		uint64(stx.Tx.LastVerifiedBatch),
		uint64(stx.Tx.NewVerifiedBatch),
		stx.Tx.ZKP,
	)
	if err != nil {
		return fmt.Errorf("failed to build verify ZKP tx: %s", err)
	}
	msg := ethereum.CallMsg{
		From: i.interopAdminAddr,
		To:   &stx.Tx.L1Contract,
		Data: l1TxData,
	}
	res, err := i.etherman.CallContract(context.Background(), msg, nil)
	if err != nil {
		return fmt.Errorf("failed to call verify ZKP response: %s, error: %s", res, err)
	}

	return nil
}

func (i *Interop) VerifySignature(stx tx.SignedTx) error {
	// Auth: check signature vs admin
	signer, err := stx.Signer()
	if err != nil {
		return errors.New("failed to get signer")
	}

	sequencer, err := i.etherman.GetSequencerAddr(stx.Tx.L1Contract)
	if err != nil {
		return errors.New("failed to get admin from L1")
	}
	if sequencer != signer {
		return errors.New("unexpected signer")
	}

	return nil
}
