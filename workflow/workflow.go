package workflow

import (
	"context"

	abciTypes "github.com/cometbft/cometbft/abci/types"

	"github.com/0xPolygon/agglayer/silencer"
	"github.com/0xPolygon/agglayer/tx"
)

var _ abciTypes.Application = (*Workflow)(nil)

type Workflow struct {
	silencer silencer.ISilencer
	// sequencer  *sequencer.Sequencer
	// aggregator *aggregator.Aggregator
}

func New(silencer silencer.ISilencer) *Workflow {
	return &Workflow{
		silencer: silencer,
	}
}

func (w *Workflow) Execute(ctx context.Context, stx tx.SignedTx) error {
	if err := w.silencer.Silence(ctx, stx); err != nil {
		return err
	}

	// TODO: Add missing parts here
	//nolint:godox
	return nil
}

// Info/Query Connection
// Return application info
func (w *Workflow) Info(_ context.Context, _ *abciTypes.RequestInfo) (*abciTypes.ResponseInfo, error) {
	panic("not implemented") // TODO: Implement
}

func (w *Workflow) Query(_ context.Context, _ *abciTypes.RequestQuery) (*abciTypes.ResponseQuery, error) {
	panic("not implemented") // TODO: Implement
}

// Mempool Connection
// Validate a tx for the mempool
func (w *Workflow) CheckTx(_ context.Context, _ *abciTypes.RequestCheckTx) (*abciTypes.ResponseCheckTx, error) {
	panic("not implemented") // TODO: It should do the soundness check
}

// Consensus Connection
// Initialize blockchain w validators/other info from CometBFT
func (w *Workflow) InitChain(_ context.Context, _ *abciTypes.RequestInitChain) (*abciTypes.ResponseInitChain, error) {
	panic("not implemented") // TODO: Implement
}

func (w *Workflow) PrepareProposal(_ context.Context, _ *abciTypes.RequestPrepareProposal) (*abciTypes.ResponsePrepareProposal, error) {
	panic("not implemented") // TODO: It should do the aggregation and ordering/sequencing
}

func (w *Workflow) ProcessProposal(_ context.Context, _ *abciTypes.RequestProcessProposal) (*abciTypes.ResponseProcessProposal, error) {
	panic("not implemented") // TODO: It should do the verification of the final proof and perform the soundness check
}

// Deliver the decided block with its txs to the Application
func (w *Workflow) FinalizeBlock(_ context.Context, _ *abciTypes.RequestFinalizeBlock) (*abciTypes.ResponseFinalizeBlock, error) {
	panic("not implemented") // TODO: Implement
}

// Create application specific vote extension
func (w *Workflow) ExtendVote(_ context.Context, _ *abciTypes.RequestExtendVote) (*abciTypes.ResponseExtendVote, error) {
	panic("not implemented") // TODO: Implement
}

// Verify application's vote extension data
func (w *Workflow) VerifyVoteExtension(_ context.Context, _ *abciTypes.RequestVerifyVoteExtension) (*abciTypes.ResponseVerifyVoteExtension, error) {
	panic("not implemented") // TODO: Implement
}

// Commit the state and return the application Merkle root hash
func (w *Workflow) Commit(_ context.Context, _ *abciTypes.RequestCommit) (*abciTypes.ResponseCommit, error) {
	panic("not implemented") // TODO: Implement
}

// State Sync Connection
// List available snapshots
func (w *Workflow) ListSnapshots(_ context.Context, _ *abciTypes.RequestListSnapshots) (*abciTypes.ResponseListSnapshots, error) {
	panic("not implemented") // TODO: Implement
}

func (w *Workflow) OfferSnapshot(_ context.Context, _ *abciTypes.RequestOfferSnapshot) (*abciTypes.ResponseOfferSnapshot, error) {
	panic("not implemented") // TODO: Implement
}

func (w *Workflow) LoadSnapshotChunk(_ context.Context, _ *abciTypes.RequestLoadSnapshotChunk) (*abciTypes.ResponseLoadSnapshotChunk, error) {
	panic("not implemented") // TODO: Implement
}

func (w *Workflow) ApplySnapshotChunk(_ context.Context, _ *abciTypes.RequestApplySnapshotChunk) (*abciTypes.ResponseApplySnapshotChunk, error) {
	panic("not implemented") // TODO: Implement
}
