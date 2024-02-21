package config

import (
	"flag"
	"testing"

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
		const configFile = "../docker/data/agglayer/agglayer.toml"

		flags := new(flag.FlagSet)
		cfgPath := ""
		flags.StringVar(&cfgPath,
			FlagCfg,
			configFile,
			"config file for the agglayer")
		ctx := cli.NewContext(nil, flags, nil)

		cfg, err := Load(ctx)
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})
}
