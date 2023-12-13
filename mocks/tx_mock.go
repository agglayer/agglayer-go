package mocks

import (
	"context"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/mock"
)

var _ pgx.Tx = (*TxMock)(nil)

type TxMock struct {
	mock.Mock
}

func (tx *TxMock) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
}

func (tx *TxMock) BeginFunc(ctx context.Context, f func(pgx.Tx) error) (err error) {
	return nil
}

func (tx *TxMock) Commit(ctx context.Context) error {
	args := tx.Called(ctx)

	return args.Error(0)
}

func (tx *TxMock) Rollback(ctx context.Context) error {
	args := tx.Called(ctx)

	return args.Error(0)
}

func (tx *TxMock) CopyFrom(ctx context.Context, tableName pgx.Identifier,
	columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (tx *TxMock) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (tx *TxMock) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (tx *TxMock) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (tx *TxMock) Exec(ctx context.Context, sql string, arguments ...interface{}) (commandTag pgconn.CommandTag, err error) {
	return nil, nil
}

func (tx *TxMock) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (tx *TxMock) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return nil
}

func (tx *TxMock) QueryFunc(ctx context.Context, sql string,
	args []interface{}, scans []interface{}, f func(pgx.QueryFuncRow) error) (pgconn.CommandTag, error) {
	return nil, nil
}

func (tx *TxMock) Conn() *pgx.Conn {
	return nil
}
