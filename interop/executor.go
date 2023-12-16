package interop

import (
	"github.com/0xPolygon/beethoven/config"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hashicorp/go-hclog"
)

type Executor struct {
	logger           hclog.Logger
	interopAdminAddr common.Address
	config           *config.Config
	ethTxMan         EthTxManager
	etherman         EthermanInterface
}

func New(logger hclog.Logger, cfg *config.Config,
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
