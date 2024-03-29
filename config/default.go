package config

import (
	"bytes"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// DefaultValues is the default configuration
const DefaultValues = `
[FullNodeRPCs]
	1 = "http://zkevm-node:8123"

[RPC]
	Host = "0.0.0.0"
	Port = 4444
	ReadTimeout = "60s"
	WriteTimeout = "60s"
	MaxRequestsPerIPAndSecond = 5000

# Address should be adjusted
[ProofSigners]
#	1 = "0x0000000000000000000000000000000000000000"

[Log]
	Environment = "development" # "production" or "development"
	Level = "debug"
	Outputs = ["stderr"]

[DB]
	User = "agglayer_user"
	Password = "agglayer_password"
	Name = "agglayer_db"
	Host = "agglayer-db"
	Port = "5432"
	EnableLog = false
	MaxConns = 200

[EthTxManager]
	FrequencyToMonitorTxs = "1s"
	WaitTxToBeMined = "2m"
	ForcedGas = 0
	GasPriceMarginFactor = 1
	MaxGasPriceLimit = 0
	PrivateKeys = [
		{Path = "/pk/agglayer.keystore", Password = "testonly"},
	]
	KMSKeyName = "gcp/resource/id"
	KMSConnectionTimeout = "30s"
	GasOffset = 100000

[L1]
	ChainID = 1337
	NodeURL = "http://zkevm-mock-l1-network:8545"
	RollupManagerContract = "0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e" # v0.0.4

[Telemetry]
	PrometheusAddr = "0.0.0.0:2223"
`

// Default parses the default configuration values.
func Default() (*Config, error) {
	var cfg Config
	viper.SetConfigType("toml")

	err := viper.ReadConfig(bytes.NewBuffer([]byte(DefaultValues)))
	if err != nil {
		return nil, err
	}
	err = viper.Unmarshal(&cfg, viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc()))
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
