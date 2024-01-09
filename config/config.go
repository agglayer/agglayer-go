package config

import (
	"crypto/ecdsa"
	"os"
	"path/filepath"
	"strings"

	jRPC "github.com/0xPolygon/cdk-data-availability/rpc"
	"github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/0xPolygonHermez/zkevm-node/db"
	"github.com/0xPolygonHermez/zkevm-node/ethtxmanager"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"

	beethovenTypes "github.com/0xPolygon/beethoven/rpc/types"
)

const (
	// FlagCfg flag used for config aka cfg
	FlagCfg = "cfg"
)

type FullNodeRPCs map[beethovenTypes.ArgUint64]string

// Config represents the full configuration of the data node
type Config struct {
	FullNodeRPCs FullNodeRPCs        `mapstructure:"FullNodeRPCs"`
	RPC          jRPC.Config         `mapstructure:"RPC"`
	Log          log.Config          `mapstructure:"Log"`
	DB           db.Config           `mapstructure:"DB"`
	EthTxManager ethtxmanager.Config `mapstructure:"EthTxManager"`
	L1           L1Config            `mapstructure:"L1"`
	Telemetry    Telemetry           `mapstructure:"Telemetry"`
}

type L1Config struct {
	ChainID               int64
	NodeURL               string
	RollupManagerContract common.Address
}

type Telemetry struct {
	PrometheusAddr string
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
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetEnvPrefix("DATA_NODE")
	err = viper.ReadInConfig()
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		if ok {
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
