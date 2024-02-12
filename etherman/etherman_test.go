package etherman

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/0xPolygon/agglayer/config"
	cdkTypes "github.com/0xPolygon/agglayer/rpc/types"
	"github.com/0xPolygon/agglayer/tx"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonrollupmanager"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xPolygon/agglayer/mocks"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func signer(from common.Address, tx *types.Transaction) (*types.Transaction, error) {
	return tx, nil
}

func getEtherman(ethClientMock IEthereumClient) Etherman {
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

func Test_BuildTrustedVerifyBatchesTxData(t *testing.T) {
	lastVerifiedBatch := uint64(0x0)
	newVerifiedBatch := uint64(0x1)
	proof := tx.ZKP{
		NewStateRoot:     common.HexToHash("0x2fb6911a0d3be8f856bc3dd596fb9275b04166b8abc2a9244ce132d12a896582"),
		NewLocalExitRoot: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Proof:            common.Hex2Bytes("13ee4e96cfdcbea64a28f819f318d764eb798e6a3d645de48b0208d867965db412938b34d3952783ca435e9cd2b8e64c62f50dfc7a8c90addbf1c18d2c6affb623f0196f7191088e3cea80dc74ad0f0ac1f4caaa6a3e6436a719e072eba225281213de98dc4e48e1458fff8542a0afb46b36a1c734be0141dfb81b81ebeef76f023eb19dde7e733ca5693e7ad075d96eef05589a79a187155b25bd17117ffd470bf184c1c801fa4be542e36af35176eb88a35322f9b9c23abc6bd41d24c3c6e22cca41305eafd24cab154b365ee1f7a30d44eea9b2701fb3d8b0e75d1f3c510a0ede7078c54d64a1cc24d80cc8cd4adf79ff1d1e368dacae78b8ba3ed5c96be90cd544749cdb5641d71ae1fc9d1ed546181e0aa0416533b84790704c3a42f7350401763dcec4b1afcb497db51e634d98e854b735c8a6ef73f0ae62732286c2460894d8ba7dfb8cd7c1d08ec0f81ffa68b426b1f46101ac51b724d5b352cd097f257c6b02d829673198eb108a903eb12009c66c9dd2fea161b248f7c758e603e80119f677ff788008c55c80e87bf51c77fac43b0a6f1b34e785317c2f4e29789210b53b7cb3ade16e3105d0adef1d97036318977c1cc7d834e03efcdf4159b4e502dc8437042505440c4a665a2e067fdd3cdcf34da65e648c8831ea3cff7b76f82bd90260f353e9382774edc5ccd764dacebe249ed520c60c40bd04cfcc635a1f24c528720a33a21cfcf17cfce1f95212940113a92b9587f62630c2fc2749076a1c9617d863e05792434c6033cbc9ffb77c91538581d3b0b62eb3ee45b4bc25f902d70e1fea2ca22d1018cc7a72353af87dca9cae58bbd5825c8eb62c60dffd840255a86f7d394e9f25c82ea100f6ccf326836cd69a53bb699c93efd22b272c631998a20450a2eceebc8a3a5d198094e1ba2f0559a2984382a13764a99da2034b2ef61d3c5bce7cbc205be9553fea89dfe58af3ae94a49c49b67358ded70ae236064c9c21a2453900bf296370c5e50d71d43e82ffe0ef07829ff03642a61b1db02153840d1a856c32944251bab9b134022ef0ca879c096835b6883906bb9e9efd"),
	}
	rollupId := uint32(4)

	var newLocalExitRoot [HashLength]byte
	copy(newLocalExitRoot[:], proof.NewLocalExitRoot.Bytes())
	var newStateRoot [HashLength]byte
	copy(newStateRoot[:], proof.NewStateRoot.Bytes())
	finalProof, err := ConvertProof(proof.Proof.Hex())
	if err != nil {
		t.Logf("error converting proof. Error: %v, Proof: %s", err, proof.Proof)
	}

	const pendStateNum uint64 = 0 // TODO hardcoded for now until we implement the pending state feature
	abi, err := polygonrollupmanager.PolygonrollupmanagerMetaData.GetAbi()
	if err != nil {
		t.Logf("error geting ABI: %v, Proof: %s", err, abi)
	}

	sa := common.HexToAddress("0x95Af2Ec2577d292E47d370dA0f424481Ebd051b5")

	out, _ := abi.Pack(
		"verifyBatchesTrustedAggregator",
		rollupId,
		pendStateNum,
		lastVerifiedBatch,
		newVerifiedBatch,
		newLocalExitRoot,
		newStateRoot,
		sa,
		finalProof,
	)

	t.Log(common.Bytes2Hex(out))
}

func TestGetSequencerAddr(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	t.Run("Returns expected error on 'TrustedSequencer' call (improperly formatted output)", func(t *testing.T) {
		t.Parallel()

		ethClient := mocks.NewEthereumClientMock(t)
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
				Data:      common.Hex2Bytes("f9c4c2ae0000000000000000000000000000000000000000000000000000000000000001"),
			},
			(*big.Int)(nil),
		).Return( // Invalid return value below to provocate error
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001"),
			nil,
		).Once()

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

	t.Run("Returns expected error on 'RollupIDToRollupData' call (improperly formatted output)", func(t *testing.T) {
		t.Parallel()

		ethClient := mocks.NewEthereumClientMock(t)
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
				Data:      common.Hex2Bytes("f9c4c2ae0000000000000000000000000000000000000000000000000000000000000001"),
			},
			(*big.Int)(nil),
		).Return( // Invalid return value below to provocate error
			common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001"),
			nil,
		).Once()

		_, err := ethman.GetSequencerAddr(1)

		assert.ErrorContains(err, "abi: improperly formatted output:")
		ethClient.AssertExpectations(t)
	})

	t.Run("Returns expected sequencer address", func(t *testing.T) {
		t.Parallel()

		ethClient := mocks.NewEthereumClientMock(t)
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
				Data:      common.Hex2Bytes("f9c4c2ae0000000000000000000000000000000000000000000000000000000000000001"),
			},
			(*big.Int)(nil),
		).Return( // Invalid return value below to provocate error
			common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001"),
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
	ethman := getEtherman(mocks.NewEthereumClientMock(t))

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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethclient := mocks.NewEthereumClientMock(t)
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
		ethclient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
		ethClient := mocks.NewEthereumClientMock(t)
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
