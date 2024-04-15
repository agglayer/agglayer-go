package txmanager

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"time"

	txmTypes "github.com/0xPolygon/agglayer/txmanager/types"
	"github.com/0xPolygonHermez/zkevm-node/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// PostgresStorage hold txs to be managed
type PostgresStorage struct {
	*pgxpool.Pool
}

// NewPostgresStorage creates a new instance of storage that use
// postgres to store data
func NewPostgresStorage(db *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{db}
}

// NewPostgresStorageWithCfg creates a new instance of storage that use based on provided config
func NewPostgresStorageWithCfg(dbCfg db.Config) (*PostgresStorage, error) {
	db, err := db.NewSQLDB(dbCfg)
	if err != nil {
		return nil, err
	}

	return &PostgresStorage{
		db,
	}, nil
}

// Add persist a monitored tx
func (s *PostgresStorage) Add(ctx context.Context, mTx txmTypes.MonitoredTx, dbTx pgx.Tx) error {
	conn := s.dbConn(dbTx)
	cmd := `
        INSERT INTO state.monitored_txs (owner, id, from_addr, to_addr, nonce, value, data, gas, gas_offset, gas_price, status, block_num, history, created_at, updated_at, num_retries)
                                 VALUES (   $1, $2,        $3,      $4,    $5,    $6,   $7,  $8,         $9,       $10,    $11,       $12,     $13,        $14,        $15,         $16)`

	_, err := conn.Exec(ctx, cmd, mTx.Owner,
		mTx.ID, mTx.From.String(), mTx.ToStringPtr(),
		mTx.Nonce, mTx.ValueU64Ptr(), mTx.DataStringPtr(),
		mTx.Gas, mTx.GasOffset, mTx.GasPrice.Uint64(), string(mTx.Status), mTx.BlockNumberU64Ptr(),
		mTx.HistoryStringSlice(), time.Now().UTC().Round(time.Microsecond),
		time.Now().UTC().Round(time.Microsecond), mTx.NumRetries)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.ConstraintName == "monitored_txs_pkey" {
			return txmTypes.ErrAlreadyExists
		} else {
			return err
		}
	}

	return nil
}

// Get loads a persisted monitored tx
func (s *PostgresStorage) Get(ctx context.Context, owner, id string, dbTx pgx.Tx) (txmTypes.MonitoredTx, error) {
	conn := s.dbConn(dbTx)
	cmd := `
        SELECT owner, id, from_addr, to_addr, nonce, value, data, gas, gas_offset, gas_price, status, block_num, history, created_at, updated_at, num_retries
          FROM state.monitored_txs
         WHERE owner = $1 
           AND id = $2`

	mTx := txmTypes.MonitoredTx{}

	row := conn.QueryRow(ctx, cmd, owner, id)
	err := s.scanMtx(row, &mTx)
	if errors.Is(err, pgx.ErrNoRows) {
		return mTx, txmTypes.ErrNotFound
	} else if err != nil {
		return mTx, err
	}

	return mTx, nil
}

// GetByStatus loads all monitored tx that match the provided status
func (s *PostgresStorage) GetByStatus(ctx context.Context, owner *string, statuses []txmTypes.MonitoredTxStatus, dbTx pgx.Tx) ([]txmTypes.MonitoredTx, error) {
	hasStatusToFilter := len(statuses) > 0

	conn := s.dbConn(dbTx)
	cmd := `
        SELECT owner, id, from_addr, to_addr, nonce, value, data, gas, gas_offset, gas_price, status, block_num, history, created_at, updated_at, num_retries
          FROM state.monitored_txs
         WHERE (owner = $1 OR $1 IS NULL)`
	if hasStatusToFilter {
		cmd += `
           AND status = ANY($2)`
	}
	cmd += `
         ORDER BY created_at`

	mTxs := []txmTypes.MonitoredTx{}

	var rows pgx.Rows
	var err error
	if hasStatusToFilter {
		rows, err = conn.Query(ctx, cmd, owner, statuses)
	} else {
		rows, err = conn.Query(ctx, cmd, owner)
	}

	defer rows.Close()

	if errors.Is(err, pgx.ErrNoRows) {
		return []txmTypes.MonitoredTx{}, nil
	} else if err != nil {
		return nil, err
	}

	for rows.Next() {
		mTx := txmTypes.MonitoredTx{}
		err := s.scanMtx(rows, &mTx)
		if err != nil {
			return nil, err
		}
		mTxs = append(mTxs, mTx)
	}

	return mTxs, nil
}

// GetBySenderAndStatus loads all monitored txs of the given sender that match the provided status
func (s *PostgresStorage) GetBySenderAndStatus(
	ctx context.Context, sender common.Address,
	statuses []txmTypes.MonitoredTxStatus, dbTx pgx.Tx) ([]txmTypes.MonitoredTx, error) {
	hasStatusToFilter := len(statuses) > 0

	conn := s.dbConn(dbTx)
	cmd := `
        SELECT owner, id, from_addr, to_addr, nonce, value, data, gas, gas_offset, gas_price, status, block_num, history, created_at, updated_at, num_retries
          FROM state.monitored_txs
         WHERE from_addr = $1`
	if hasStatusToFilter {
		cmd += `
           AND status = ANY($2)`
	}
	cmd += `
         ORDER BY created_at`

	mTxs := []txmTypes.MonitoredTx{}

	var rows pgx.Rows
	var err error
	if hasStatusToFilter {
		rows, err = conn.Query(ctx, cmd, sender.String(), statuses)
	} else {
		rows, err = conn.Query(ctx, cmd, sender.String())
	}

	defer rows.Close()

	if errors.Is(err, pgx.ErrNoRows) {
		return []txmTypes.MonitoredTx{}, nil
	} else if err != nil {
		return nil, err
	}

	for rows.Next() {
		mTx := txmTypes.MonitoredTx{}
		err := s.scanMtx(rows, &mTx)
		if err != nil {
			return nil, err
		}
		mTxs = append(mTxs, mTx)
	}

	return mTxs, nil
}

// Update a persisted monitored tx
func (s *PostgresStorage) Update(ctx context.Context, mTx txmTypes.MonitoredTx, dbTx pgx.Tx) error {
	conn := s.dbConn(dbTx)
	cmd := `
        UPDATE state.monitored_txs
           SET from_addr = $3
             , to_addr = $4
             , nonce = $5
             , value = $6
             , data = $7
             , gas = $8
             , gas_offset = $9
             , gas_price = $10
             , status = $11
             , block_num = $12
             , history = $13
             , updated_at = $14
			 , num_retries = $15
         WHERE owner = $1
           AND id = $2`

	var bn *uint64
	if mTx.BlockNumber != nil {
		tmp := mTx.BlockNumber.Uint64()
		bn = &tmp
	}

	_, err := conn.Exec(ctx, cmd, mTx.Owner,
		mTx.ID, mTx.From.String(), mTx.ToStringPtr(),
		mTx.Nonce, mTx.ValueU64Ptr(), mTx.DataStringPtr(),
		mTx.Gas, mTx.GasOffset, mTx.GasPrice.Uint64(), string(mTx.Status), bn,
		mTx.HistoryStringSlice(), time.Now().UTC().Round(time.Microsecond), mTx.NumRetries)

	if err != nil {
		return err
	}

	return nil
}

// scanMtx scans a row and fill the provided instance of monitoredTx with
// the row data
func (s *PostgresStorage) scanMtx(row pgx.Row, mTx *txmTypes.MonitoredTx) error {
	// id, from, to, nonce, value, data, gas, gas_offset, gas_price, status, history, created_at, updated_at, num_retries
	var from, status string
	var to, data *string
	var history []string
	var value, blockNumber *uint64
	var gasPrice uint64

	err := row.Scan(&mTx.Owner, &mTx.ID, &from, &to, &mTx.Nonce, &value,
		&data, &mTx.Gas, &mTx.GasOffset, &gasPrice, &status, &blockNumber, &history,
		&mTx.CreatedAt, &mTx.UpdatedAt, &mTx.NumRetries)
	if err != nil {
		return err
	}

	mTx.From = common.HexToAddress(from)
	mTx.GasPrice = big.NewInt(0).SetUint64(gasPrice)
	mTx.Status = txmTypes.MonitoredTxStatus(status)

	if to != nil {
		tmp := common.HexToAddress(*to)
		mTx.To = &tmp
	}
	if value != nil {
		tmp := *value
		mTx.Value = big.NewInt(0).SetUint64(tmp)
	}
	if data != nil {
		tmp := *data
		bytes, err := hex.DecodeString(tmp)
		if err != nil {
			return err
		}
		mTx.Data = bytes
	}
	if blockNumber != nil {
		tmp := *blockNumber
		mTx.BlockNumber = big.NewInt(0).SetUint64(tmp)
	}

	h := make(map[common.Hash]bool, len(history))
	for _, txHash := range history {
		h[common.HexToHash(txHash)] = true
	}
	mTx.History = h

	return nil
}

// dbConn represents an instance of an object that can
// connect to a postgres db to execute sql commands and query data
type dbConn interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// dbConn determines which db connection to use, dbTx or the main pgxpool
func (p *PostgresStorage) dbConn(dbTx pgx.Tx) dbConn {
	if dbTx != nil {
		return dbTx
	}
	return p
}
