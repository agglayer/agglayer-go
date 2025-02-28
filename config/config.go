package config

import (
	"crypto/ecdsa"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xPolygon/agglayer/log"
	cdkrpc "github.com/0xPolygon/cdk-rpc/rpc"
	"github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/0xPolygonHermez/zkevm-node/db"
	"github.com/0xPolygonHermez/zkevm-node/ethtxmanager"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
)

const (
	// FlagCfg flag used for config aka cfg
	FlagCfg = "cfg"
)

type FullNodeRPCs map[uint32]string

// ProofSigners holds the address for authorized signers of proofs for a given rollup ip
type ProofSigners map[uint32]common.Address

// Config represents the full configuration of the data node
type Config struct {
	FullNodeRPCs FullNodeRPCs       `mapstructure:"FullNodeRPCs"`
	RPC          cdkrpc.Config      `mapstructure:"RPC"`
	ProofSigners ProofSigners       `mapstructure:"ProofSigners"`
	Log          log.Config         `mapstructure:"Log"`
	DB           db.Config          `mapstructure:"DB"`
	EthTxManager EthTxManagerConfig `mapstructure:"EthTxManager"`
	L1           L1Config           `mapstructure:"L1"`
	Telemetry    Telemetry          `mapstructure:"Telemetry"`
}

type L1Config struct {
	ChainID               int64
	NodeURL               string
	RollupManagerContract common.Address
}

type Telemetry struct {
	PrometheusAddr string
}

type EthTxManagerConfig struct {
	ethtxmanager.Config  `mapstructure:",squash"`
	GasOffset            uint64         `mapstructure:"GasOffset"`
	KMSKeyName           string         `mapstructure:"KMSKeyName"`
	KMSConnectionTimeout types.Duration `mapstructure:"KMSConnectionTimeout"`
	MaxRetries           uint64         `mapstructure:"MaxRetries"`
}

// Load loads the configuration baseed on the cli context
func Load(ctx *cli.Context) (*Config, error) {
	cfg, err := Default()
	if err != nil {
		return nil, err
	}

	configFilePath := ctx.String(FlagCfg)
	if configFilePath != "" {
		dirName, fileName := filepath.Split(configFilePath)

		fileExtension := strings.TrimPrefix(filepath.Ext(fileName), ".")
		fileNameWithoutExtension := strings.TrimSuffix(fileName, "."+fileExtension)

		viper.AddConfigPath(dirName)
		viper.SetConfigName(fileNameWithoutExtension)
		viper.SetConfigType(fileExtension)
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("DATA_NODE")

	if err = viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Infof("config file not found")
		} else {
			log.Infof("error reading config file: ", err)
			return nil, err
		}
	}

	decodeHooks := []viper.DecoderConfigOption{
		// this allows arrays to be decoded from env var separated by ",", example: MY_VAR="value1,value2,value3"
		viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(mapstructure.TextUnmarshallerHookFunc(), mapstructure.StringToSliceHookFunc(","))),
	}
	err = viper.Unmarshal(&cfg, decodeHooks...)
	return cfg, err
}

// NewKeyFromKeystore creates a private key from a keystore file
func NewKeyFromKeystore(cfg types.KeystoreFileConfig) (*ecdsa.PrivateKey, error) {
	if cfg.Path == "" && cfg.Password == "" {
		return nil, nil
	}
	keystoreEncrypted, err := os.ReadFile(filepath.Clean(cfg.Path))
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(keystoreEncrypted, cfg.Password)
	if err != nil {
		return nil, err
	}
	return key.PrivateKey, nil
}
