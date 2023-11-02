package config

import (
	"bytes"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// DefaultValues is the default configuration
const DefaultValues = `
[FullNodeRPCs]
0x0DCd1Bf9A1b36cE34237eEaFef220932846BCD82 = "http://zkevm-node:8123"

[RPC]
Host = "0.0.0.0"
Port = 4444
ReadTimeout = "60s"
WriteTimeout = "60s"
MaxRequestsPerIPAndSecond = 5000

[Log]
Environment = "development" # "production" or "development"
Level = "debug"
Outputs = ["stderr"]

[DB]
User = "beethoven_user"
Password = "beethoven_password"
Name = "beethoven_db"
Host = "beethoven-db"
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
	{Path = "/pk/interop.keystore", Password = "testonly"},
]

[L1]
ChainID = 1337
NodeURL = "http://l1:8545"
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
