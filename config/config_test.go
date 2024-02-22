package config

import (
	"flag"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestLoad(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		ctx := cli.NewContext(nil, nil, nil)
		_ = ctx.Set(FlagCfg, "/path/to/config.yaml")

		cfg, err := Load(ctx)
		require.NoError(t, err)

		defaultCfg, err := Default()
		require.NoError(t, err)
		require.Equal(t, defaultCfg, cfg)
	})

	t.Run("the agglayer.toml file config", func(t *testing.T) {
		const (
			cfgFile            = "../docker/data/agglayer/agglayer.toml"
			ethTxManagerCfgKey = "EthTxManager"
		)

		// simulate command with the cfg flag
		flags := new(flag.FlagSet)
		cfgPath := ""
		flags.StringVar(&cfgPath,
			FlagCfg,
			cfgFile,
			"config file for the agglayer")
		ctx := cli.NewContext(nil, flags, nil)

		// read the config independently from the Load function
		viper.SetConfigFile(cfgFile)
		err := viper.ReadInConfig()
		require.NoError(t, err)

		decodeOpts := []viper.DecoderConfigOption{
			viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(mapstructure.TextUnmarshallerHookFunc())),
		}

		var ethTxManagerCfg EthTxManagerConfig
		err = viper.UnmarshalKey(ethTxManagerCfgKey, &ethTxManagerCfg, decodeOpts...)
		require.NoError(t, err)

		cfg, err := Load(ctx)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, ethTxManagerCfg, cfg.EthTxManager)
	})
}
