package workflow

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-node/pool"
	"github.com/0xPolygonHermez/zkevm-node/sequencer"
	"github.com/0xPolygonHermez/zkevm-node/state"
	"github.com/cometbft/cometbft/abci/types"

	"github.com/0xPolygon/beethoven/aggregator"
	"github.com/0xPolygon/beethoven/silencer"
)

var _ types.Application = (*Workflow)(nil)

type Workflow struct {
	silencer   *silencer.Silencer
	sequencer  *sequencer.Sequencer
	aggregator *aggregator.Aggregator
}

func New() (*Workflow, error) {
	seq, err := sequencer.New(sequencer.Config{}, state.BatchConfig{}, pool.Config{}, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	return &Workflow{
		silencer:   silencer.New(),
		aggregator: aggregator.New(),
		sequencer:  seq,
	}, nil
}

func (w *Workflow) Execute() error {
	// TODO: Implement
	//nolint:godox
	return nil
}

// Info/Query Connection
// Return application info
func (w *Workflow) Info(_ context.Context, _ *types.RequestInfo) (*types.ResponseInfo, error) {
	panic("not implemented") // TODO: Implement
}

func (w *Workflow) Query(_ context.Context, _ *types.RequestQuery) (*types.ResponseQuery, error) {
	panic("not implemented") // TODO: Implement
}

// Mempool Connection
// Validate a tx for the mempool
func (w *Workflow) CheckTx(_ context.Context, _ *types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	panic("not implemented") // TODO: It should do the soundness check
}

// Consensus Connection
// Initialize blockchain w validators/other info from CometBFT
func (w *Workflow) InitChain(_ context.Context, _ *types.RequestInitChain) (*types.ResponseInitChain, error) {
	panic("not implemented") // TODO: Implement
}

func (w *Workflow) PrepareProposal(_ context.Context, _ *types.RequestPrepareProposal) (*types.ResponsePrepareProposal, error) {
	panic("not implemented") // TODO: It should do the aggregation and ordering/sequencing
}

func (w *Workflow) ProcessProposal(_ context.Context, _ *types.RequestProcessProposal) (*types.ResponseProcessProposal, error) {
	panic("not implemented") // TODO: It should do the verification of the final proof and perform the soundness check
}

// Deliver the decided block with its txs to the Application
func (w *Workflow) FinalizeBlock(_ context.Context, _ *types.RequestFinalizeBlock) (*types.ResponseFinalizeBlock, error) {
	panic("not implemented") // TODO: Implement
}

// Create application specific vote extension
func (w *Workflow) ExtendVote(_ context.Context, _ *types.RequestExtendVote) (*types.ResponseExtendVote, error) {
	panic("not implemented") // TODO: Implement
}

// Verify application's vote extension data
func (w *Workflow) VerifyVoteExtension(_ context.Context, _ *types.RequestVerifyVoteExtension) (*types.ResponseVerifyVoteExtension, error) {
	panic("not implemented") // TODO: Implement
}

// Commit the state and return the application Merkle root hash
func (w *Workflow) Commit(_ context.Context, _ *types.RequestCommit) (*types.ResponseCommit, error) {
	panic("not implemented") // TODO: Implement
}

// State Sync Connection
// List available snapshots
func (w *Workflow) ListSnapshots(_ context.Context, _ *types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	panic("not implemented") // TODO: Implement
}

func (w *Workflow) OfferSnapshot(_ context.Context, _ *types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	panic("not implemented") // TODO: Implement
}

func (w *Workflow) LoadSnapshotChunk(_ context.Context, _ *types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	panic("not implemented") // TODO: Implement
}

func (w *Workflow) ApplySnapshotChunk(_ context.Context, _ *types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	panic("not implemented") // TODO: Implement
}
