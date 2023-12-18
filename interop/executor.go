package interop

import (
	"github.com/0xPolygon/beethoven/config"

	"github.com/0xPolygon/cdk-validium-node/log"
	"github.com/ethereum/go-ethereum/common"
)

type Executor struct {
	logger           *log.Logger
	interopAdminAddr common.Address
	config           *config.Config
	ethTxMan         EthTxManager
	etherman         EthermanInterface
}

func New(logger *log.Logger, cfg *config.Config,
	interopAdminAddr common.Address,
	db DBInterface,
	etherman EthermanInterface,
	ethTxManager EthTxManager,
) *Executor {
	return &Executor{
		logger:           logger,
		interopAdminAddr: interopAdminAddr,
		config:           cfg,
		ethTxMan:         ethTxManager,
		etherman:         etherman,
	}
}
