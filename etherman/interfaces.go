package etherman

import (
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

//go:generate mockery --name IEthereumClient --output ../mocks --case=underscore --filename etherman_client.generated.go
type IEthereumClient interface {
	ethereum.ChainReader
	ethereum.ChainStateReader
	ethereum.ContractCaller
	ethereum.GasEstimator
	ethereum.GasPricer
	ethereum.LogFilterer
	ethereum.TransactionReader
	ethereum.TransactionSender

	bind.ContractTransactor
	bind.ContractFilterer
	bind.DeployBackend
}
