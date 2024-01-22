package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestLoad(t *testing.T) {
	ctx := cli.NewContext(nil, nil, nil)
	ctx.Set(FlagCfg, "/path/to/config.yaml")

	cfg, err := Load(ctx)
	assert.NoError(t, err)
}
