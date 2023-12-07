package etherman

import (
	"context"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"math/big"
)

type ethereumClientMock struct {
	mock.Mock
}

func (e *ethereumClientMock) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) TransactionCount(ctx context.Context, blockHash common.Hash) (uint, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ethereumClientMock) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	args := e.Called(ctx, call, blockNumber)

	return args.Get(0).([]byte), args.Error(1)
}

func (e *ethereumClientMock) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	args := e.Called(ctx, txHash)

	return args.Get(0).(*types.Receipt), args.Error(1)
}

func (e *ethereumClientMock) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	args := e.Called(ctx, account, blockNumber)

	return args.Get(0).(uint64), args.Error(1)
}

func (e *ethereumClientMock) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	args := e.Called(ctx, txHash)

	return args.Get(0).(*types.Transaction), args.Bool(1), args.Error(2)
}

func (e *ethereumClientMock) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	args := e.Called(ctx, tx)

	return args.Error(0)
}

func (e *ethereumClientMock) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	args := e.Called(ctx)

	return args.Get(0).(*big.Int), args.Error(1)
}

func (e *ethereumClientMock) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	args := e.Called(ctx, call)

	return args.Get(0).(uint64), args.Error(1)
}

func (e *ethereumClientMock) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	args := e.Called(ctx, number)

	return args.Get(0).(*types.Block), args.Error(1)
}
