package types

import (
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

	expected := []string{
		"0x0000000000000000000000000000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000000000000000000000000000002",
		"0x0000000000000000000000000000000000000000000000000000000000000003",
	}
	result := mTx.HistoryStringSlice()

	assert.Equal(t, expected, result)
}

func TestHistoryHashSlice(t *testing.T) {
	mTx := MonitoredTx{
		History: map[common.Hash]bool{
			common.HexToHash("0x1"): true,
			common.HexToHash("0x2"): true,
			common.HexToHash("0x3"): true,
		},
	}

	expected := []common.Hash{
		common.HexToHash("0x1"),
		common.HexToHash("0x2"),
		common.HexToHash("0x3"),
	}

	result := mTx.HistoryHashSlice()

	assert.Equal(t, expected, result)
}
