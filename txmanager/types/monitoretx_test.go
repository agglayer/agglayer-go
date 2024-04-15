package types

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestTx(t *testing.T) {
	to := common.HexToAddress("0x2")
	nonce := uint64(1)
	value := big.NewInt(2)
	data := []byte("data")
	gas := uint64(3)
	gasOffset := uint64(4)
	gasPrice := big.NewInt(5)

	mTx := MonitoredTx{
		To:        &to,
		Nonce:     nonce,
		Value:     value,
		Data:      data,
		Gas:       gas,
		GasOffset: gasOffset,
		GasPrice:  gasPrice,
	}

	tx := mTx.Tx()

	assert.Equal(t, &to, tx.To())
	assert.Equal(t, nonce, tx.Nonce())
	assert.Equal(t, value, tx.Value())
	assert.Equal(t, data, tx.Data())
	assert.Equal(t, gas+gasOffset, tx.Gas())
	assert.Equal(t, gasPrice, tx.GasPrice())
}

func TestAddHistory(t *testing.T) {
	mTx := MonitoredTx{
		History:    make(map[common.Hash]bool),
		NumRetries: 0,
	}

	tx := types.NewTransaction(0, common.HexToAddress("0x1"), big.NewInt(0), 0, big.NewInt(0), nil)

	err := mTx.AddHistory(tx)
	assert.NoError(t, err)
	assert.True(t, mTx.History[tx.Hash()])
	assert.Equal(t, uint64(0x1), mTx.NumRetries)

	err = mTx.AddHistory(tx)
	assert.Equal(t, ErrAlreadyExists, err)
}

func TestMonitoredTx_HistoryStringSlice(t *testing.T) {
	mTx := MonitoredTx{
		History: map[common.Hash]bool{
			common.HexToHash("0x1"): true,
			common.HexToHash("0x2"): true,
			common.HexToHash("0x3"): true,
		},
	}

	result := mTx.HistoryStringSlice()
	assert.Equal(t, len(mTx.History), len(result))

	for _, hash := range result {
		assert.True(t, mTx.History[common.HexToHash(hash)])
	}
}

func TestHistoryHashSlice(t *testing.T) {
	mTx := MonitoredTx{
		History: map[common.Hash]bool{
			common.HexToHash("0x1"): true,
			common.HexToHash("0x2"): true,
			common.HexToHash("0x3"): true,
		},
	}

	result := mTx.HistoryHashSlice()
	assert.Equal(t, len(mTx.History), len(result))

	for _, hash := range result {
		assert.True(t, mTx.History[hash])
	}
}

func TestMonitoredTx_BlockNumberU64Ptr(t *testing.T) {
	// Create a monitoredTx instance with a non-nil BlockNumber
	mTx := MonitoredTx{
		BlockNumber: big.NewInt(123),
	}

	// Call the BlockNumberU64Ptr method
	result := mTx.BlockNumberU64Ptr()

	// Assert that the result is not nil
	assert.NotNil(t, result)

	// Assert that the value pointed by result is equal to the expected value
	expected := uint64(123)
	assert.Equal(t, expected, *result)

	// Create a monitoredTx instance with a nil BlockNumber
	mTx2 := MonitoredTx{
		BlockNumber: nil,
	}

	// Call the BlockNumberU64Ptr method
	result2 := mTx2.BlockNumberU64Ptr()

	// Assert that the result is nil
	assert.Nil(t, result2)
}

func TestMonitoredTx_DataStringPtr(t *testing.T) {
	mTx := MonitoredTx{
		Data: []byte("data"),
	}

	expected := hex.EncodeToString(mTx.Data)
	actual := mTx.DataStringPtr()

	assert.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}
