package types

import (
	"encoding/hex"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	// ErrNotFound when the object is not found
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists when the object already exists
	ErrAlreadyExists = errors.New("already exists")

	// ErrExecutionReverted returned when trying to get the revert message
	// but the call fails without revealing the revert reason
	ErrExecutionReverted = errors.New("execution reverted")
)

const (
	// MonitoredTxStatusCreated mean the tx was just added to the storage
	MonitoredTxStatusCreated = MonitoredTxStatus("created")

	// MonitoredTxStatusSent means that at least a eth tx was sent to the network
	MonitoredTxStatusSent = MonitoredTxStatus("sent")

	// MonitoredTxStatusFailed means the tx was already mined and failed with an
	// error that can't be recovered automatically, ex: the data in the tx is invalid
	// and the tx gets reverted
	MonitoredTxStatusFailed = MonitoredTxStatus("failed")

	// MonitoredTxStatusConfirmed means the tx was already mined and the receipt
	// status is Successful
	MonitoredTxStatusConfirmed = MonitoredTxStatus("confirmed")

	// MonitoredTxStatusReorged is used when a monitored tx was already confirmed but
	// the L1 block where this tx was confirmed has been reorged, in this situation
	// the caller needs to review this information and wait until it gets confirmed
	// again in a future block
	MonitoredTxStatusReorged = MonitoredTxStatus("reorged")

	// MonitoredTxStatusDone means the tx was set by the owner as done
	MonitoredTxStatusDone = MonitoredTxStatus("done")
)

// MonitoredTxStatus represents the status of a monitored tx
type MonitoredTxStatus string

// String returns a string representation of the status
func (s MonitoredTxStatus) String() string {
	return string(s)
}

// MonitoredTx represents a set of information used to build tx
// plus information to monitor if the transactions was sent successfully
type MonitoredTx struct {
	// Owner is the common identifier among all the monitored tx to identify who
	// created this, it's a identification provided by the caller in order to be
	// used in the future to query the monitored tx by the Owner, this allows the
	// caller to be free of implementing a persistence layer to monitor the txs
	Owner string

	// ID is the tx identifier controller by the caller
	ID string

	// sender of the tx, used to identify which private key should be used to sing the tx
	From common.Address

	// receiver of the tx
	To *common.Address

	// Nonce used to create the tx
	Nonce uint64

	// tx Value
	Value *big.Int

	// tx Data
	Data []byte

	// tx Gas
	Gas uint64

	// tx gas offset
	GasOffset uint64

	// tx gas price
	GasPrice *big.Int

	// Status of this monitoring
	Status MonitoredTxStatus

	// BlockNumber represents the block where the tx was identified
	// to be mined, it's the same as the block number found in the
	// tx receipt, this is used to control reorged monitored txs
	BlockNumber *big.Int

	// History represent all transaction hashes from
	// transactions created using this struct data and
	// sent to the network
	History map[common.Hash]bool

	// CreatedAt date time it was created
	CreatedAt time.Time

	// UpdatedAt last date time it was updated
	UpdatedAt time.Time

	// NumRetries number of times tx was sent to the network
	NumRetries uint64
}

// Tx uses the current information to build a tx
func (mTx MonitoredTx) Tx() *types.Transaction {
	tx := types.NewTx(&types.LegacyTx{
		To:       mTx.To,
		Nonce:    mTx.Nonce,
		Value:    mTx.Value,
		Data:     mTx.Data,
		Gas:      mTx.Gas + mTx.GasOffset,
		GasPrice: mTx.GasPrice,
	})

	return tx
}

// AddHistory adds a transaction to the monitoring history
func (mTx *MonitoredTx) AddHistory(tx *types.Transaction) error {
	if _, found := mTx.History[tx.Hash()]; found {
		return ErrAlreadyExists
	}

	mTx.History[tx.Hash()] = true
	mTx.NumRetries++

	return nil
}

// ToStringPtr returns the current to field as a string pointer
func (mTx *MonitoredTx) ToStringPtr() *string {
	var to *string
	if mTx.To != nil {
		s := mTx.To.String()
		to = &s
	}
	return to
}

// ValueU64Ptr returns the current value field as a uint64 pointer
func (mTx *MonitoredTx) ValueU64Ptr() *uint64 {
	var value *uint64
	if mTx.Value != nil {
		tmp := mTx.Value.Uint64()
		value = &tmp
	}
	return value
}

// DataStringPtr returns the current data field as a string pointer
func (mTx *MonitoredTx) DataStringPtr() *string {
	var data *string
	if mTx.Data != nil {
		tmp := hex.EncodeToString(mTx.Data)
		data = &tmp
	}
	return data
}

// HistoryStringSlice returns the current history field as a string slice
func (mTx *MonitoredTx) HistoryStringSlice() []string {
	history := make([]string, 0, len(mTx.History))
	for h := range mTx.History {
		history = append(history, h.String())
	}
	return history
}

// HistoryHashSlice returns the current history field as a string slice
func (mTx *MonitoredTx) HistoryHashSlice() []common.Hash {
	history := make([]common.Hash, 0, len(mTx.History))
	for h := range mTx.History {
		history = append(history, h)
	}
	return history
}

// BlockNumberU64Ptr returns the current blockNumber as a uint64 pointer
func (mTx *MonitoredTx) BlockNumberU64Ptr() *uint64 {
	var blockNumber *uint64
	if mTx.BlockNumber != nil {
		tmp := mTx.BlockNumber.Uint64()
		blockNumber = &tmp
	}
	return blockNumber
}

// MonitoredTxResult represents the result of a execution of a monitored tx
type MonitoredTxResult struct {
	ID     string
	Status MonitoredTxStatus
	Txs    map[common.Hash]TxResult
}

// TxResult represents the result of a execution of a ethereum transaction in the block chain
type TxResult struct {
	Tx            *types.Transaction
	Receipt       *types.Receipt
	RevertMessage string
}
