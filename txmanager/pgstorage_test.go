package txmanager

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/db"
	"github.com/0xPolygonHermez/zkevm-node/test/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	aggLayerDB "github.com/0xPolygon/agglayer/db"
	txmTypes "github.com/0xPolygon/agglayer/txmanager/types"
)

func newStateDBConfig(t *testing.T) db.Config {
	t.Helper()

	const maxDBPoolConns = 50

	cfg := db.Config{
		User:      testutils.GetEnv("PGUSER", "agglayer_user"),
		Password:  testutils.GetEnv("PGPASSWORD", "agglayer_password"),
		Name:      testutils.GetEnv("PGDATABASE", "agglayer_db"),
		Host:      testutils.GetEnv("PGHOST", "localhost"),
		Port:      testutils.GetEnv("PGPORT", "5434"),
		EnableLog: false,
		MaxConns:  maxDBPoolConns,
	}

	// connect to database
	dbPool, err := db.NewSQLDB(cfg)
	require.NoError(t, err)

	defer dbPool.Close()

	c, err := pgx.ParseConfig(fmt.Sprintf("postgres://%s:%s@%s:%s/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name))
	require.NoError(t, err)

	db := stdlib.OpenDB(*c)

	require.NoError(t, aggLayerDB.RunMigrationsDown(db))
	require.NoError(t, aggLayerDB.RunMigrationsUp(db))

	return cfg
}

func TestAddGetAndUpdate(t *testing.T) {
	dbCfg := newStateDBConfig(t)
	storage, err := NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	owner := "owner"
	id := "id"
	from := common.HexToAddress("0x1")
	to := common.HexToAddress("0x2")
	nonce := uint64(1)
	value := big.NewInt(2)
	data := []byte("data")
	gas := uint64(3)
	gasPrice := big.NewInt(4)
	status := txmTypes.MonitoredTxStatusCreated
	blockNumber := big.NewInt(5)
	history := map[common.Hash]bool{common.HexToHash("0x3"): true, common.HexToHash("0x4"): true}

	mTx := txmTypes.MonitoredTx{
		Owner: owner, ID: id, From: from, To: &to, Nonce: nonce, Value: value, Data: data,
		BlockNumber: blockNumber, Gas: gas, GasPrice: gasPrice, Status: status, History: history,
	}
	err = storage.Add(context.Background(), mTx, nil)
	require.NoError(t, err)

	returnedMtx, err := storage.Get(context.Background(), owner, id, nil)
	require.NoError(t, err)

	assert.Equal(t, owner, returnedMtx.Owner)
	assert.Equal(t, id, returnedMtx.ID)
	assert.Equal(t, from.String(), returnedMtx.From.String())
	assert.Equal(t, to.String(), returnedMtx.To.String())
	assert.Equal(t, nonce, returnedMtx.Nonce)
	assert.Equal(t, value, returnedMtx.Value)
	assert.Equal(t, data, returnedMtx.Data)
	assert.Equal(t, gas, returnedMtx.Gas)
	assert.Equal(t, gasPrice, returnedMtx.GasPrice)
	assert.Equal(t, status, returnedMtx.Status)
	assert.Equal(t, 0, blockNumber.Cmp(returnedMtx.BlockNumber))
	assert.Equal(t, history, returnedMtx.History)
	assert.Greater(t, time.Now().UTC().Round(time.Microsecond), returnedMtx.CreatedAt)
	assert.Less(t, time.Time{}, returnedMtx.CreatedAt)
	assert.Greater(t, time.Now().UTC().Round(time.Microsecond), returnedMtx.UpdatedAt)
	assert.Less(t, time.Time{}, returnedMtx.UpdatedAt)

	from = common.HexToAddress("0x11")
	to = common.HexToAddress("0x22")
	nonce = uint64(11)
	value = big.NewInt(22)
	data = []byte("data data")
	gas = uint64(33)
	gasPrice = big.NewInt(44)
	status = txmTypes.MonitoredTxStatusFailed
	blockNumber = big.NewInt(55)
	history = map[common.Hash]bool{common.HexToHash("0x33"): true, common.HexToHash("0x44"): true}

	mTx = txmTypes.MonitoredTx{
		Owner: owner, ID: id, From: from, To: &to, Nonce: nonce, Value: value, Data: data,
		BlockNumber: blockNumber, Gas: gas, GasPrice: gasPrice, Status: status, History: history,
	}
	err = storage.Update(context.Background(), mTx, nil)
	require.NoError(t, err)

	returnedMtx, err = storage.Get(context.Background(), owner, id, nil)
	require.NoError(t, err)

	assert.Equal(t, owner, returnedMtx.Owner)
	assert.Equal(t, id, returnedMtx.ID)
	assert.Equal(t, from.String(), returnedMtx.From.String())
	assert.Equal(t, to.String(), returnedMtx.To.String())
	assert.Equal(t, nonce, returnedMtx.Nonce)
	assert.Equal(t, value, returnedMtx.Value)
	assert.Equal(t, data, returnedMtx.Data)
	assert.Equal(t, gas, returnedMtx.Gas)
	assert.Equal(t, gasPrice, returnedMtx.GasPrice)
	assert.Equal(t, status, returnedMtx.Status)
	assert.Equal(t, 0, blockNumber.Cmp(returnedMtx.BlockNumber))
	assert.Equal(t, history, returnedMtx.History)
	assert.Greater(t, time.Now().UTC().Round(time.Microsecond), returnedMtx.CreatedAt)
	assert.Less(t, time.Time{}, returnedMtx.CreatedAt)
	assert.Greater(t, time.Now().UTC().Round(time.Microsecond), returnedMtx.UpdatedAt)
	assert.Less(t, time.Time{}, returnedMtx.UpdatedAt)
}

func TestAddAndGetByStatus(t *testing.T) {
	dbCfg := newStateDBConfig(t)
	storage, err := NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	to := common.HexToAddress("0x2")
	baseMtx := txmTypes.MonitoredTx{
		Owner:       "owner",
		From:        common.HexToAddress("0x1"),
		To:          &to,
		Nonce:       uint64(1),
		Value:       big.NewInt(2),
		Data:        []byte("data"),
		BlockNumber: big.NewInt(1),
		Gas:         uint64(3),
		GasPrice:    big.NewInt(4),
		History:     map[common.Hash]bool{common.HexToHash("0x3"): true, common.HexToHash("0x4"): true},
	}

	type mTxReplaceInfo struct {
		id     string
		status txmTypes.MonitoredTxStatus
	}

	mTxsReplaceInfo := []mTxReplaceInfo{
		{id: "created1", status: txmTypes.MonitoredTxStatusCreated},
		{id: "sent1", status: txmTypes.MonitoredTxStatusSent},
		{id: "failed1", status: txmTypes.MonitoredTxStatusFailed},
		{id: "confirmed1", status: txmTypes.MonitoredTxStatusConfirmed},
		{id: "created2", status: txmTypes.MonitoredTxStatusCreated},
		{id: "sent2", status: txmTypes.MonitoredTxStatusSent},
		{id: "failed2", status: txmTypes.MonitoredTxStatusFailed},
		{id: "confirmed2", status: txmTypes.MonitoredTxStatusConfirmed},
	}

	for _, replaceInfo := range mTxsReplaceInfo {
		baseMtx.ID = replaceInfo.id
		baseMtx.Status = replaceInfo.status
		baseMtx.CreatedAt = baseMtx.CreatedAt.Add(time.Microsecond)
		baseMtx.UpdatedAt = baseMtx.UpdatedAt.Add(time.Microsecond)
		err = storage.Add(context.Background(), baseMtx, nil)
		require.NoError(t, err)
	}

	mTxs, err := storage.GetByStatus(context.Background(), nil, []txmTypes.MonitoredTxStatus{txmTypes.MonitoredTxStatusConfirmed}, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, len(mTxs))
	assert.Equal(t, "confirmed1", mTxs[0].ID)
	assert.Equal(t, "confirmed2", mTxs[1].ID)

	mTxs, err = storage.GetByStatus(context.Background(), nil, []txmTypes.MonitoredTxStatus{txmTypes.MonitoredTxStatusSent, txmTypes.MonitoredTxStatusCreated}, nil)
	require.NoError(t, err)
	assert.Equal(t, 4, len(mTxs))
	assert.Equal(t, "created1", mTxs[0].ID)
	assert.Equal(t, "sent1", mTxs[1].ID)
	assert.Equal(t, "created2", mTxs[2].ID)
	assert.Equal(t, "sent2", mTxs[3].ID)

	mTxs, err = storage.GetByStatus(context.Background(), nil, []txmTypes.MonitoredTxStatus{}, nil)
	require.NoError(t, err)
	assert.Equal(t, 8, len(mTxs))
	assert.Equal(t, "created1", mTxs[0].ID)
	assert.Equal(t, "sent1", mTxs[1].ID)
	assert.Equal(t, "failed1", mTxs[2].ID)
	assert.Equal(t, "confirmed1", mTxs[3].ID)
	assert.Equal(t, "created2", mTxs[4].ID)
	assert.Equal(t, "sent2", mTxs[5].ID)
	assert.Equal(t, "failed2", mTxs[6].ID)
	assert.Equal(t, "confirmed2", mTxs[7].ID)
}

func TestAddAndGetBySenderAndStatus(t *testing.T) {
	dbCfg := newStateDBConfig(t)
	storage, err := NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	from := common.HexToAddress("0x1")
	to := common.HexToAddress("0x2")
	baseMtx := txmTypes.MonitoredTx{
		Owner:       "owner",
		From:        common.HexToAddress("0x1"),
		To:          &to,
		Nonce:       uint64(1),
		Value:       big.NewInt(2),
		Data:        []byte("data"),
		BlockNumber: big.NewInt(1),
		Gas:         uint64(3),
		GasPrice:    big.NewInt(4),
		History:     map[common.Hash]bool{common.HexToHash("0x3"): true, common.HexToHash("0x4"): true},
	}

	type mTxReplaceInfo struct {
		id     string
		status txmTypes.MonitoredTxStatus
	}

	mTxsReplaceInfo := []mTxReplaceInfo{
		{id: "created1", status: txmTypes.MonitoredTxStatusCreated},
		{id: "sent1", status: txmTypes.MonitoredTxStatusSent},
		{id: "failed1", status: txmTypes.MonitoredTxStatusFailed},
		{id: "confirmed1", status: txmTypes.MonitoredTxStatusConfirmed},
		{id: "created2", status: txmTypes.MonitoredTxStatusCreated},
		{id: "sent2", status: txmTypes.MonitoredTxStatusSent},
		{id: "failed2", status: txmTypes.MonitoredTxStatusFailed},
		{id: "confirmed2", status: txmTypes.MonitoredTxStatusConfirmed},
	}

	for _, replaceInfo := range mTxsReplaceInfo {
		baseMtx.ID = replaceInfo.id
		baseMtx.Status = replaceInfo.status
		baseMtx.CreatedAt = baseMtx.CreatedAt.Add(time.Microsecond)
		baseMtx.UpdatedAt = baseMtx.UpdatedAt.Add(time.Microsecond)
		err = storage.Add(context.Background(), baseMtx, nil)
		require.NoError(t, err)
	}

	mTxs, err := storage.GetBySenderAndStatus(context.Background(), from, []txmTypes.MonitoredTxStatus{txmTypes.MonitoredTxStatusConfirmed}, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, len(mTxs))
	assert.Equal(t, "confirmed1", mTxs[0].ID)
	assert.Equal(t, "confirmed2", mTxs[1].ID)

	mTxs, err = storage.GetBySenderAndStatus(context.Background(), from, []txmTypes.MonitoredTxStatus{txmTypes.MonitoredTxStatusSent, txmTypes.MonitoredTxStatusCreated}, nil)
	require.NoError(t, err)
	assert.Equal(t, 4, len(mTxs))
	assert.Equal(t, "created1", mTxs[0].ID)
	assert.Equal(t, "sent1", mTxs[1].ID)
	assert.Equal(t, "created2", mTxs[2].ID)
	assert.Equal(t, "sent2", mTxs[3].ID)

	mTxs, err = storage.GetBySenderAndStatus(context.Background(), from, []txmTypes.MonitoredTxStatus{}, nil)
	require.NoError(t, err)
	assert.Equal(t, 8, len(mTxs))
	assert.Equal(t, "created1", mTxs[0].ID)
	assert.Equal(t, "sent1", mTxs[1].ID)
	assert.Equal(t, "failed1", mTxs[2].ID)
	assert.Equal(t, "confirmed1", mTxs[3].ID)
	assert.Equal(t, "created2", mTxs[4].ID)
	assert.Equal(t, "sent2", mTxs[5].ID)
	assert.Equal(t, "failed2", mTxs[6].ID)
	assert.Equal(t, "confirmed2", mTxs[7].ID)
}

func TestAddRepeated(t *testing.T) {
	dbCfg := newStateDBConfig(t)
	storage, err := NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	owner := "owner"
	id := "id"
	from := common.HexToAddress("0x1")
	to := common.HexToAddress("0x2")
	nonce := uint64(1)
	value := big.NewInt(2)
	data := []byte("data")
	gas := uint64(3)
	gasPrice := big.NewInt(4)
	blockNumber := big.NewInt(5)
	status := txmTypes.MonitoredTxStatusCreated
	history := map[common.Hash]bool{common.HexToHash("0x3"): true, common.HexToHash("0x4"): true}

	mTx := txmTypes.MonitoredTx{
		Owner:       owner,
		ID:          id,
		From:        from,
		To:          &to,
		Nonce:       nonce,
		Value:       value,
		Data:        data,
		BlockNumber: blockNumber,
		Gas:         gas,
		GasPrice:    gasPrice,
		Status:      status,
		History:     history,
	}

	err = storage.Add(context.Background(), mTx, nil)
	require.NoError(t, err)

	err = storage.Add(context.Background(), mTx, nil)
	require.Equal(t, txmTypes.ErrAlreadyExists, err)
}

func TestGetNotFound(t *testing.T) {
	dbCfg := newStateDBConfig(t)
	storage, err := NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	_, err = storage.Get(context.Background(), "not found owner", "not found id", nil)
	require.Equal(t, txmTypes.ErrNotFound, err)
}

func TestGetByStatusNoRows(t *testing.T) {
	dbCfg := newStateDBConfig(t)
	storage, err := NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	mTxs, err := storage.GetByStatus(context.Background(), nil, []txmTypes.MonitoredTxStatus{}, nil)
	require.NoError(t, err)
	require.Empty(t, mTxs)
}
