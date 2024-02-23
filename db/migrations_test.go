package db

import (
	"testing"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_runMigrations(t *testing.T) {
	migrations := &migrate.EmbedFileSystemMigrationSource{FileSystem: f, Root: "migrations"}
	m, err := migrations.FindMigrations()
	require.NoError(t, err)

	assert.Greater(t, len(m), 0)
}
