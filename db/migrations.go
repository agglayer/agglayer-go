package db

import (
	"embed"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations
var f embed.FS

// RunMigrationsUp runs migrate-up for the given config.
func RunMigrationsUp(pg *pgxpool.Pool) error {
	log.Info("running migrations up")
	return runMigrations(pg, migrate.Up)
}

// RunMigrationsDown runs migrate-down for the given config.
func RunMigrationsDown(pg *pgxpool.Pool) error {
	log.Info("running migrations down")
	return runMigrations(pg, migrate.Down)
}

// runMigrations will execute pending migrations if needed to keep
// the database updated with the latest changes in either direction,
// up or down.
func runMigrations(pg *pgxpool.Pool, direction migrate.MigrationDirection) error {
	db := stdlib.OpenDB(*pg.Config().ConnConfig)

	var migrations = &migrate.EmbedFileSystemMigrationSource{FileSystem: f}
	nMigrations, err := migrate.Exec(db, "postgres", migrations, direction)
	if err != nil {
		return err
	}

	log.Info("successfully ran ", nMigrations, " migrations")
	return nil
}
