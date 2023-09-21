package db

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// DB is the database layer of the data node
type DB struct {
	pg *pgxpool.Pool
}

// New instantiates a DB
func New(pg *pgxpool.Pool) *DB {
	return &DB{
		pg: pg,
	}
}

// BeginStateTransaction begins a DB transaction. The caller is responsible for committing or rolling back the transaction
func (db *DB) BeginStateTransaction(ctx context.Context) (pgx.Tx, error) {
	return db.pg.Begin(ctx)
}
