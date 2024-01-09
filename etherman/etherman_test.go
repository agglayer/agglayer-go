package etherman

import (
	"context"
	"errors"
	"github.com/0xPolygon/beethoven/config"
	"math/big"
	"testing"

	"github.com/0xPolygon/beethoven/mocks"
	cdkTypes "github.com/0xPolygon/beethoven/rpc/types"
	"github.com/0xPolygon/beethoven/tx"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func signer(from common.Address, tx *types.Transaction) (*types.Transaction, error) {
	return tx, nil
}

func getEtherman(ethClientMock EthereumClient) Etherman {
	ethman, _ := New(
		ethClientMock,
		bind.TransactOpts{
			From:      common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"),
			Nonce:     big.NewInt(0),
			Signer:    signer,
			Value:     big.NewInt(0),
			GasPrice:  big.NewInt(0),
			GasTipCap: big.NewInt(0),
			GasLimit:  0,
			Context:   context.TODO(),
			NoSend:    false,
		},
		&config.Config{},
	)

	return ethman
}

func TestGetSequencerAddr(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	t.Run("Returns expected error (improperly formatted output)", func(t *testing.T) {
		t.Parallel()

		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"CallContract",
			mock.Anything,
			ethereum.CallMsg{
				From:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
				To:        &common.Address{},
				Gas:       0,
				GasPrice:  nil,
				GasFeeCap: nil,
				GasTipCap: nil,
				Value:     nil,
				Data:      []uint8{0xcf, 0xa8, 0xed, 0x47},
			},
			(*big.Int)(nil),
		).Return( // Invalid return value below to provocate error
			common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000000000"),
			nil,
		).Once()

		_, err := ethman.GetSequencerAddr(1)

		assert.ErrorContains(err, "abi: improperly formatted output:")
		ethClient.AssertExpectations(t)
	})

	t.Run("Returns expected sequencer address", func(t *testing.T) {
		t.Parallel()

		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On( // Call "RollupIdToRollupData" mapping
			"CallContract",
			mock.Anything,
			ethereum.CallMsg{},
			(*big.Int)(nil),
		).Return(
			common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000000"),
			nil,
		).Once()

		ethClient.On( // Call "TrustedSequencer" property on rollup contract
			"CallContract",
			mock.Anything,
			ethereum.CallMsg{
				From: common.HexToAddress("0x0000000000000000000000000000000000000000"),
				To:   &common.Address{},
				Data: []uint8{0xcf, 0xa8, 0xed, 0x47},
			},
			(*big.Int)(nil),
		).Return(
			common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000000"),
			nil,
		).Once()

		returnValue, err := ethman.GetSequencerAddr(1)

		assert.Equal(common.Address{}, returnValue)
		assert.NoError(err)
		ethClient.AssertExpectations(t)
	})
}

func TestBuildTrustedVerifyBatches(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	ethman := getEtherman(new(mocks.EthereumClientMock))

	// Because we cant mock the ABI dependency is this the only test case that we somehow
	// can have here in a unit test. Further test coverage can get achieved with e2e or integration tests.
	t.Run("Returns expected error on proof conversion", func(t *testing.T) {
		data, err := ethman.BuildTrustedVerifyBatchesTxData(
			0,
			1,
			tx.ZKP{
				NewStateRoot:     common.HexToHash("0x001"),
				NewLocalExitRoot: common.HexToHash("0x002"),
				Proof:            cdkTypes.ArgBytes("0x30030030030003003300300030033003000300330030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030003003003000300300300030030030"),
			},
			1,
		)

		assert.ErrorContains(err, "invalid proof length. Expected length: 1538, Actual length 1534")
		assert.Nil(data)
	})
}

func TestCallContract(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	t.Run("Returns expected value", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"CallContract",
			context.TODO(),
			ethereum.CallMsg{
				From:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
				To:        &common.Address{},
				Gas:       0,
				GasPrice:  nil,
				GasFeeCap: nil,
				GasTipCap: nil,
				Value:     nil,
				Data:      []uint8{0xcf, 0xa8, 0xed, 0x47}, // TrustedSequencer sig
			},
			(*big.Int)(nil),
		).Return( // Invalid return value below to provocate error
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"),
			nil,
		).Once()

		result, err := ethman.CallContract(
			context.TODO(),
			ethereum.CallMsg{
				From:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
				To:        &common.Address{},
				Gas:       0,
				GasPrice:  nil,
				GasFeeCap: nil,
				GasTipCap: nil,
				Value:     nil,
				Data:      []uint8{0xcf, 0xa8, 0xed, 0x47},
			},
			(*big.Int)(nil),
		)

		assert.Equal(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"), result)
		assert.Equal(nil, err)
		ethClient.AssertExpectations(t)
	})

	t.Run("Returns expected error", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"CallContract",
			context.TODO(),
			ethereum.CallMsg{
				From:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
				To:        &common.Address{},
				Gas:       0,
				GasPrice:  nil,
				GasFeeCap: nil,
				GasTipCap: nil,
				Value:     nil,
				Data:      []uint8{0xcf, 0xa8, 0xed, 0x47}, // TrustedSequencer sig
			},
			(*big.Int)(nil),
		).Return( // Invalid return value below to provocate error
			[]uint8{},
			errors.New("NOOOPE!"),
		).Once()

		result, err := ethman.CallContract(
			context.TODO(),
			ethereum.CallMsg{
				From:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
				To:        &common.Address{},
				Gas:       0,
				GasPrice:  nil,
				GasFeeCap: nil,
				GasTipCap: nil,
				Value:     nil,
				Data:      []uint8{0xcf, 0xa8, 0xed, 0x47},
			},
			(*big.Int)(nil),
		)

		assert.Equal([]uint8{}, result)
		assert.ErrorContains(err, "NOOOPE!")
		ethClient.AssertExpectations(t)
	})
}

func TestCheckTxWasMined(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	t.Run("Returns expected error on 'ethereum.NotFound'", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"TransactionReceipt",
			context.TODO(),
			common.Hash{},
		).Return(
			&types.Receipt{},
			errors.New("not found"),
		).Once()

		status, receipt, err := ethman.CheckTxWasMined(context.TODO(), common.Hash{})

		assert.False(status)
		assert.Nil(receipt)
		assert.ErrorContains(err, "not found")
		ethClient.AssertExpectations(t)
	})

	t.Run("Returns expected error", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"TransactionReceipt",
			context.TODO(),
			common.Hash{},
		).Return(
			&types.Receipt{},
			errors.New("Nooope!"),
		).Once()

		status, receipt, err := ethman.CheckTxWasMined(context.TODO(), common.Hash{})

		assert.False(status)
		assert.Nil(receipt)
		assert.ErrorContains(err, "Nooope!")
		ethClient.AssertExpectations(t)
	})

	t.Run("Returns the expected values", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"TransactionReceipt",
			context.TODO(),
			common.Hash{},
		).Return(
			&types.Receipt{},
			nil,
		).Once()

		status, receipt, err := ethman.CheckTxWasMined(context.TODO(), common.Hash{})

		assert.True(status)
		assert.Equal(&types.Receipt{}, receipt)
		assert.Nil(err)
		ethClient.AssertExpectations(t)
	})
}

func TestCurrentNonce(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	t.Run("Returns the expected nonce value", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"NonceAt",
			context.TODO(),
			common.Address{},
			(*big.Int)(nil),
		).Return(
			uint64(1),
			nil,
		).Once()

		result, err := ethman.CurrentNonce(context.TODO(), common.Address{})

		assert.Equal(uint64(1), result)
		assert.Nil(err)
		ethClient.AssertExpectations(t)
	})

	t.Run("Returns the expected error", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"NonceAt",
			context.TODO(),
			common.Address{},
			(*big.Int)(nil),
		).Return(
			uint64(0),
			errors.New("NA NA NA!"),
		).Once()

		result, err := ethman.CurrentNonce(context.TODO(), common.Address{})

		assert.Equal(uint64(0), result)
		assert.ErrorContains(err, "NA NA NA!")
		ethClient.AssertExpectations(t)
	})
}

func TestGetTx(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	t.Run("Returns the expected transaction", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"TransactionByHash",
			context.TODO(),
			common.Hash{},
		).Return(
			&types.Transaction{},
			true,
			nil,
		).Once()

		transaction, status, err := ethman.GetTx(context.TODO(), common.Hash{})

		assert.Equal(&types.Transaction{}, transaction)
		assert.True(status)
		assert.Nil(err)
		ethClient.AssertExpectations(t)
	})

	t.Run("Returns the expected error", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"TransactionByHash",
			context.TODO(),
			common.Hash{},
		).Return(
			&types.Transaction{},
			false,
			errors.New("NOPE NOPE!"),
		).Once()

		transaction, status, err := ethman.GetTx(context.TODO(), common.Hash{})

		assert.Equal(&types.Transaction{}, transaction)
		assert.False(status)
		assert.ErrorContains(err, "NOPE NOPE!")
		ethClient.AssertExpectations(t)
	})
}

func TestGetTxReceipt(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	t.Run("Returns expected receipt", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"TransactionReceipt",
			context.TODO(),
			common.Hash{},
		).Return(
			&types.Receipt{},
			nil,
		).Once()

		receipt, err := ethman.GetTxReceipt(context.TODO(), common.Hash{})

		assert.Equal(&types.Receipt{}, receipt)
		assert.Nil(err)
		ethClient.AssertExpectations(t)
	})

	t.Run("Returns expected error", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"TransactionReceipt",
			context.TODO(),
			common.Hash{},
		).Return(
			&types.Receipt{},
			errors.New("NANANA!"),
		).Once()

		receipt, err := ethman.GetTxReceipt(context.TODO(), common.Hash{})

		assert.Equal(&types.Receipt{}, receipt)
		assert.ErrorContains(err, "NANANA!")
		ethClient.AssertExpectations(t)
	})
}

func TestSendTx(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	transaction := types.NewTransaction(
		uint64(1),
		common.Address{},
		big.NewInt(1),
		uint64(1),
		big.NewInt(1),
		[]byte{},
	)

	t.Run("Returns expected value", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"SendTransaction",
			context.TODO(),
			transaction,
		).Return(
			nil,
		).Once()

		err := ethman.SendTx(context.TODO(), transaction)

		assert.Nil(err)
		ethClient.AssertExpectations(t)
	})

	t.Run("Returns expected error", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"SendTransaction",
			context.TODO(),
			transaction,
		).Return(
			errors.New("NANANA!"),
		).Once()

		err := ethman.SendTx(context.TODO(), transaction)

		assert.ErrorContains(err, "NANANA!")
		ethClient.AssertExpectations(t)
	})
}

func TestSuggestedGasPrice(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	t.Run("Returns expected value", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"SuggestGasPrice",
			context.TODO(),
		).Return(
			big.NewInt(1),
			nil,
		).Once()

		result, err := ethman.SuggestedGasPrice(context.TODO())

		assert.Equal(big.NewInt(1), result)
		assert.Nil(err)
		ethClient.AssertExpectations(t)
	})

	t.Run("Returns expected error", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"SuggestGasPrice",
			context.TODO(),
		).Return(
			(*big.Int)(nil),
			errors.New("NOPE!"),
		).Once()

		result, err := ethman.SuggestedGasPrice(context.TODO())

		assert.Equal((*big.Int)(nil), result)
		assert.ErrorContains(err, "NOPE!")
		ethClient.AssertExpectations(t)
	})
}

func TestEstimateGas(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	t.Run("Returns the expected value", func(t *testing.T) {
		ethclient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethclient)

		ethclient.On(
			"EstimateGas",
			context.TODO(),
			ethereum.CallMsg{
				From:  common.Address{},
				To:    &common.Address{},
				Value: big.NewInt(10),
				Data:  []byte{},
			},
		).Return(
			uint64(1),
			nil,
		).Once()

		result, err := ethman.EstimateGas(
			context.TODO(),
			common.Address{},
			&common.Address{},
			big.NewInt(10),
			[]byte{},
		)

		assert.Equal(uint64(1), result)
		assert.Nil(err)
		ethclient.AssertExpectations(t)
	})

	t.Run("Returns the expected error", func(t *testing.T) {
		ethclient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethclient)

		ethclient.On(
			"EstimateGas",
			context.TODO(),
			ethereum.CallMsg{
				From:  common.Address{},
				To:    &common.Address{},
				Value: big.NewInt(10),
				Data:  []byte{},
			},
		).Return(
			uint64(0),
			errors.New("NOOOPE!"),
		).Once()

		result, err := ethman.EstimateGas(
			context.TODO(),
			common.Address{},
			&common.Address{},
			big.NewInt(10),
			[]byte{},
		)

		assert.Equal(uint64(0), result)
		assert.ErrorContains(err, "NOOOPE!")
		ethclient.AssertExpectations(t)
	})
}

func TestSignTx(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	txData := types.NewTransaction(
		uint64(1),
		common.Address{},
		big.NewInt(1),
		uint64(1),
		big.NewInt(1),
		[]byte{},
	)

	t.Run("Returns the expected value", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		transaction, err := ethman.SignTx(context.TODO(), common.Address{}, txData)

		assert.Equal(txData, transaction)
		assert.Nil(err)
	})
}

func TestGetRevertMessage(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	txData := types.NewTransaction(
		uint64(1),
		common.Address{},
		big.NewInt(1),
		uint64(1),
		big.NewInt(1),
		[]byte{0xcf, 0xa8, 0xed, 0x47},
	)

	t.Run("Returns an empty string and the expected error", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"TransactionReceipt",
			context.TODO(),
			txData.Hash(),
		).Return(
			&types.Receipt{},
			errors.New("NANANA!"),
		).Once()

		result, err := ethman.GetRevertMessage(context.TODO(), txData)

		assert.Equal("", result)
		assert.ErrorContains(err, "NANANA!")
	})

	t.Run("Returns an empty string and the error set to nil", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"TransactionReceipt",
			context.TODO(),
			txData.Hash(),
		).Return(
			&types.Receipt{
				Status: types.ReceiptStatusSuccessful,
			},
			nil,
		).Once()

		result, err := ethman.GetRevertMessage(context.TODO(), txData)

		assert.Equal("", result)
		assert.Nil(err)
	})

	t.Run("Returns the expected revert reason string", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		key, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(key.PublicKey)
		signedTx, _ := types.SignTx(txData, types.NewEIP155Signer(big.NewInt(1)), key)

		ethClient.On(
			"TransactionReceipt",
			context.TODO(),
			signedTx.Hash(),
		).Return(
			&types.Receipt{
				Status:      types.ReceiptStatusFailed,
				BlockNumber: big.NewInt(1),
			},
			nil,
		).Once()

		ethClient.On(
			"CallContract",
			context.TODO(),
			ethereum.CallMsg{
				From:      addr,
				To:        &common.Address{},
				Gas:       1,
				GasPrice:  nil,
				GasFeeCap: nil,
				GasTipCap: nil,
				Value:     big.NewInt(1),
				Data:      []uint8{0xcf, 0xa8, 0xed, 0x47}, // TrustedSequencer sig
			},
			big.NewInt(1),
		).Return(
			common.Hex2Bytes("08c379a00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000548454c4c4f000000000000000000000000000000000000000000000000000000"),
			nil,
		).Once()

		result, err := ethman.GetRevertMessage(context.TODO(), signedTx)

		assert.Equal("HELLO", result)
		assert.Nil(err)
	})
}

func TestGetLastBlock(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	t.Run("Returns the expected values", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"BlockByNumber",
			context.TODO(),
			(*big.Int)(nil),
		).Return(
			types.NewBlock(
				&types.Header{},
				[]*types.Transaction{},
				[]*types.Header{},
				[]*types.Receipt{},
				types.TrieHasher(nil),
			),
			nil,
		)

		result, err := ethman.GetLastBlock(context.TODO(), new(mocks.TxMock))

		assert.Equal(uint64(0), result.BlockNumber)
		assert.Equal(common.HexToHash("0xb159a077fc2af79b9a9c748c9c0e50ff95b74c32946ed52418fcc093d0953f26"), result.BlockHash)
		assert.Equal(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"), result.ParentHash)
		assert.Nil(err)
	})

	t.Run("Returns the expected error", func(t *testing.T) {
		ethClient := new(mocks.EthereumClientMock)
		ethman := getEtherman(ethClient)

		ethClient.On(
			"BlockByNumber",
			context.TODO(),
			(*big.Int)(nil),
		).Return(
			&types.Block{},
			errors.New("NOOPE!"),
		)

		result, err := ethman.GetLastBlock(context.TODO(), new(mocks.TxMock))

		assert.ErrorContains(err, "NOOPE!")
		assert.Nil(result)
	})
}
