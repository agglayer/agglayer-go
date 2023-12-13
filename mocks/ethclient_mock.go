package mocks

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"math/big"
)

type EthereumClientMock struct {
	mock.Mock
}

func (e *EthereumClientMock) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) TransactionCount(ctx context.Context, blockHash common.Hash) (uint, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	args := e.Called(ctx, account, blockNumber)

	return args.Get(0).([]byte), args.Error(1)
}

func (e *EthereumClientMock) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	//TODO implement me
	panic("implement me")
}

func (e *EthereumClientMock) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	args := e.Called(ctx, call, blockNumber)

	return args.Get(0).([]byte), args.Error(1)
}

func (e *EthereumClientMock) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	args := e.Called(ctx, txHash)

	return args.Get(0).(*types.Receipt), args.Error(1)
}

func (e *EthereumClientMock) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	args := e.Called(ctx, account, blockNumber)

	return args.Get(0).(uint64), args.Error(1)
}

func (e *EthereumClientMock) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	args := e.Called(ctx, txHash)

	return args.Get(0).(*types.Transaction), args.Bool(1), args.Error(2)
}

func (e *EthereumClientMock) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	args := e.Called(ctx, tx)

	return args.Error(0)
}

func (e *EthereumClientMock) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	args := e.Called(ctx)

	return args.Get(0).(*big.Int), args.Error(1)
}

func (e *EthereumClientMock) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	args := e.Called(ctx, call)

	return args.Get(0).(uint64), args.Error(1)
}

func (e *EthereumClientMock) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	args := e.Called(ctx, number)

	return args.Get(0).(*types.Block), args.Error(1)
}
