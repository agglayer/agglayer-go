package db

import (
	"database/sql"
	"embed"

	"github.com/0xPolygonHermez/zkevm-node/log"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations
var f embed.FS

// RunMigrationsUp runs migrate-up for the given config.
func RunMigrationsUp(db *sql.DB) error {
	log.Info("running migrations up")
	return runMigrations(db, migrate.Up)
}

// RunMigrationsDown runs migrate-down for the given config.
func RunMigrationsDown(db *sql.DB) error {
	log.Info("running migrations down")
	return runMigrations(db, migrate.Down)
}

// runMigrations will execute pending migrations if needed to keep
// the database updated with the latest changes in either direction,
// up or down.
func runMigrations(db *sql.DB, direction migrate.MigrationDirection) error {
	migrations := &migrate.EmbedFileSystemMigrationSource{FileSystem: f, Root: "migrations"}
	nMigrations, err := migrate.Exec(db, "postgres", migrations, direction)
	if err != nil {
		return err
	}

	log.Info("successfully ran ", nMigrations, " migrations")
	return nil
}
