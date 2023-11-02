package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"

	silencer "github.com/0xPolygon/beethoven"
	"github.com/0xPolygon/beethoven/config"
	"github.com/0xPolygon/beethoven/db"
	"github.com/0xPolygon/beethoven/etherman"
	"github.com/0xPolygon/beethoven/rpc"
	"github.com/0xPolygon/cdk-data-availability/dummyinterfaces"
	dbConf "github.com/0xPolygon/cdk-validium-node/db"
	"github.com/0xPolygon/cdk-validium-node/ethtxmanager"
	"github.com/0xPolygon/cdk-validium-node/jsonrpc"
	"github.com/0xPolygon/cdk-validium-node/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
)

const appName = "cdk-silencer"

var (
	configFileFlag = cli.StringFlag{
		Name:     config.FlagCfg,
		Aliases:  []string{"c"},
		Usage:    "Configuration `FILE`",
		Required: false,
	}
)

func main() {
	app := cli.NewApp()
	app.Name = appName
	app.Version = silencer.Version
	app.Commands = []*cli.Command{
		{
			Name:    "run",
			Aliases: []string{},
			Usage:   fmt.Sprintf("Run the %v", appName),
			Action:  start,
			Flags:   []cli.Flag{&configFileFlag},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func start(cliCtx *cli.Context) error {
	// Load config
	c, err := config.Load(cliCtx)
	if err != nil {
		panic(err)
	}
	setupLog(c.Log)

	// Load private key
	pk, err := config.NewKeyFromKeystore(c.EthTxManager.PrivateKeys[0])
	if err != nil {
		log.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(pk.PublicKey)

	// Prepare DB
	pg, err := dbConf.NewSQLDB(c.DB)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.RunMigrationsUp(pg); err != nil {
		log.Fatal(err)
	}
	storage := db.New(pg)

	// Prepare Etherman
	auth, err := bind.NewKeyedTransactorWithChainID(pk, big.NewInt(c.L1.ChainID))
	if err != nil {
		log.Fatal(err)
	}
	ethMan, err := etherman.New(c.L1.NodeURL, *auth)
	if err != nil {
		log.Fatal(err)
	}

	// Prepare EthTxMan client
	ethTxManagerStorage, err := ethtxmanager.NewPostgresStorage(c.DB)
	if err != nil {
		log.Fatal(err)
	}
	etm := ethtxmanager.New(c.EthTxManager, &ethMan, ethTxManagerStorage, &ethMan)

	// Register services
	server := jsonrpc.NewServer(
		c.RPC,
		0,
		&dummyinterfaces.DummyPool{},
		&dummyinterfaces.DummyState{},
		&dummyinterfaces.DummyStorage{},
		[]jsonrpc.Service{
			{
				Name:    rpc.INTEROP,
				Service: rpc.NewInteropEndpoints(addr, storage, &ethMan, c.FullNodeRPCs, etm),
			},
		},
	)

	// Run RPC
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal(err)
		}
	}()
	// Run EthTxMan
	go etm.Start()

	waitSignal(nil)
	return nil
}

func setupLog(c log.Config) {
	log.Init(c)
}

func waitSignal(cancelFuncs []context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	for sig := range signals {
		switch sig {
		case os.Interrupt, os.Kill:
			log.Info("terminating application gracefully...")

			exitStatus := 0
			for _, cancel := range cancelFuncs {
				cancel()
			}
			os.Exit(exitStatus)
		}
	}
}
