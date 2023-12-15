package interop

import (
	"crypto/ecdsa"
	"log"
	"math/big"

	"github.com/0xPolygon/beethoven/config"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/go-hclog"
)

const (
	AppVersion uint64 = 1
)

type Interop struct {
	logger           hclog.Logger
	addr             common.Address
	config           *config.Config
	ethTxMan         *ethtxmanager.Client
	etherman         etherman.EthermanInterface
	interopAdminAddr common.Address
}

func NewSilencer(logger hclog.Logger, cfg *config.Config, privateKey *ecdsa.PrivateKey, state *State) *Interop {
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Prepare Etherman
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(cfg.L1.ChainID))
	if err != nil {
		log.Fatal(err)
	}
	ethMan, err := etherman.New(cfg.L1.NodeURL, *auth)
	if err != nil {
		log.Fatal(err)
	}

	// Prepare EthTxMan client
	ethTxManagerStorage, err := ethtxmanager.NewBoltDBStorage(cfg.BaseConfig.RootDir + "/ethtxmanager.db")
	if err != nil {
		log.Fatal(err)
	}
	etm := ethtxmanager.New(cfg.EthTxManager, &ethMan, ethTxManagerStorage, &ethMan)

	e, err := etherman.New(cfg.L1.NodeURL, bind.TransactOpts{})
	if err != nil {
		log.Fatal(err)
	}

	return &Interop{
		ID:       cfg.Moniker,
		logger:   logger,
		addr:     addr,
		state:    state,
		config:   cfg,
		ethTxMan: etm,
		etherman: &e,
	}
}
