package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"time"

	jRPC "github.com/0xPolygon/cdk-rpc/rpc"
	dbConf "github.com/0xPolygonHermez/zkevm-node/db"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/pascaldekloe/etherkeyms"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"

	kms "cloud.google.com/go/kms/apiv1"
	agglayer "github.com/0xPolygon/agglayer"
	"github.com/0xPolygon/agglayer/config"
	"github.com/0xPolygon/agglayer/db"
	"github.com/0xPolygon/agglayer/etherman"
	"github.com/0xPolygon/agglayer/interop"
	"github.com/0xPolygon/agglayer/log"
	"github.com/0xPolygon/agglayer/network"
	"github.com/0xPolygon/agglayer/rpc"
	"github.com/0xPolygon/agglayer/txmanager"
)

const appName = "cdk-agglayer"

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
	app.Version = agglayer.Version
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

	// Prepare DB
	pg, err := dbConf.NewSQLDB(c.DB)
	if err != nil {
		return err
	}
	sqldb := stdlib.OpenDB(*pg.Config().ConnConfig)

	if err = db.RunMigrationsUp(sqldb); err != nil {
		return err
	}
	storage := db.New(pg)

	// Prepare Etherman

	// Load private key
	var (
		auth *bind.TransactOpts
		addr common.Address
	)

	if c.EthTxManager.KMSKeyName != "" {
		log.Debugf("using KMS key: %s", c.EthTxManager.KMSKeyName)
		auth, addr, err = useKMSAuth(c)
		if err != nil {
			return err
		}
	} else if len(c.EthTxManager.PrivateKeys) > 0 {
		log.Debugf("using local private key: %s", c.EthTxManager.PrivateKeys[0].Path)
		auth, addr, err = useLocalAuth(c)
		if err != nil {
			return err
		}
	} else {
		return errors.New("no private key found")
	}

	// Connect to ethereum node
	ethClient, err := ethclient.DialContext(cliCtx.Context, c.L1.NodeURL)
	if err != nil {
		return fmt.Errorf("error connecting to %s: %w", c.L1.NodeURL, err)
	}

	// Make sure the connection is okay
	if _, err = ethClient.ChainID(cliCtx.Context); err != nil {
		return fmt.Errorf("error getting chain ID from l1 with address: %w", err)
	}

	ethMan, err := etherman.New(ethClient, *auth, c)
	if err != nil {
		return err
	}

	// Prepare EthTxMan client
	ethTxManagerStorage := txmanager.NewPostgresStorage(pg)
	etm := txmanager.New(c.EthTxManager, &ethMan, ethTxManagerStorage, &ethMan)

	// Create opentelemetry metric provider
	meterProvider, err := createMeterProvider()
	if err != nil {
		return err
	}
	otel.SetMeterProvider(meterProvider)

	executor := interop.New(
		log.WithFields("module", "executor"),
		c,
		addr,
		&ethMan,
		etm,
	)

	// Register services
	server := jRPC.NewServer(
		c.RPC,
		[]jRPC.Service{
			{
				Name:    rpc.INTEROP,
				Service: rpc.NewInteropEndpoints(log.WithFields("module", "rpc"), executor, storage, c),
			},
		},
		jRPC.WithHealthHandler(http.HandlerFunc(healthHandler(storage))),
		jRPC.WithLogger(log.WithFields("module", "rpc")),
	)

	// Run RPC
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	// Run EthTxMan
	go etm.Start()

	// Run prometheus server
	closePrometheus, err := runPrometheusServer(c)
	if err != nil {
		return err
	}

	// Stop services
	waitSignal([]context.CancelFunc{
		etm.Stop,
		func() {
			if err := server.Stop(); err != nil {
				log.Error(err)
			}
		},
		ethTxManagerStorage.Close,
		closePrometheus,
		func() {
			if err := meterProvider.Shutdown(cliCtx.Context); err != nil {
				log.Error(err)
			}
		},
	})

	return nil
}

func setupLog(c log.Config) {
	if err := log.InitLogger(c); err != nil {
		panic(fmt.Errorf("could not setup logger. Err: %w", err))
	}
}

func createMeterProvider() (*metric.MeterProvider, error) {
	// The exporter embeds a default OpenTelemetry Reader and
	// implements prometheus.Collector, allowing it to be used as
	// both a Reader and Collector.
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	return metric.NewMeterProvider(metric.WithReader(exporter)), nil
}

func runPrometheusServer(c *config.Config) (func(), error) {
	if c.Telemetry.PrometheusAddr == "" {
		return nil, nil
	}

	addr, err := network.ResolveAddr(c.Telemetry.PrometheusAddr, network.AllInterfacesBinding)
	if err != nil {
		return nil, err
	}

	srv := &http.Server{
		Addr:              addr.String(),
		Handler:           promhttp.Handler(),
		ReadHeaderTimeout: 60 * time.Second,
	}

	log.Infof("prometheus server started: %s", addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Errorf("prometheus HTTP server ListenAndServe: %w", err)
			}
		}
	}()

	return func() {
		if err := srv.Close(); err != nil {
			log.Errorf("prometheus HTTP server closing failed: %w", err)
		}
	}, nil
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
				if cancel != nil {
					cancel()
				}
			}
			os.Exit(exitStatus)
		}
	}
}

func useKMSAuth(c *config.Config) (*bind.TransactOpts, common.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.EthTxManager.KMSConnectionTimeout.Duration)
	defer cancel()

	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to create kms client: %w", err)
	}

	mk, err := etherkeyms.NewManagedKey(ctx, client, c.EthTxManager.KMSKeyName)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to create managed key: %w", err)
	}
	signer := types.LatestSignerForChainID(big.NewInt(c.L1.ChainID))

	return mk.NewEthereumTransactor(context.Background(), signer), mk.EthereumAddr, nil
}

func useLocalAuth(c *config.Config) (*bind.TransactOpts, common.Address, error) {
	pk, err := config.NewKeyFromKeystore(c.EthTxManager.PrivateKeys[0])
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to create private key from keystore: %w", err)
	}
	addr := crypto.PubkeyToAddress(pk.PublicKey)

	auth, err := bind.NewKeyedTransactorWithChainID(pk, big.NewInt(c.L1.ChainID))
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to create keyed transactor: %w", err)
	}

	return auth, addr, nil
}

// healthHandler returns a handler that checks the health of the application
func healthHandler(storage *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txctx := context.Background()
		dbtx, err := storage.BeginStateTransaction(txctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err = dbtx.Rollback(txctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, _ = w.Write([]byte("Healthy"))
		w.WriteHeader(http.StatusOK)
	}
}
