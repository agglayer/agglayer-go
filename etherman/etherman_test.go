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
	newVerifiedBatch := uint64(0x3)
	proof := tx.ZKP{
		NewStateRoot:     common.HexToHash("0x57ad8ae2377851bc1f77f16261684fb838660d66036f4b1fd1d63199f0621711"),
		NewLocalExitRoot: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Proof:            common.Hex2Bytes("0x20227cbcef731b6cbdc0edd5850c63dc7fbc27fb58d12cd4d08298799cf66a0512c230867d3375a1f4669e7267dad2c31ebcddbaccea6abd67798ceae35ae7611c665b6069339e6812d015e239594aa71c4e217288e374448c358f6459e057c91ad2ef514570b5dea21508e214430daadabdd23433820000fe98b1c6fa81d5c512b86fbf87bd7102775f8ef1da7e8014dc7aab225503237c7927c032e589e9a01a0eab9fda82ffe834c2a4977f36cc9bcb1f2327bdac5fb48ffbeb9656efcdf70d2656c328903e9fb96e4e3f470c447b3053cc68d68cf0ad317fe10aa7f254222e47ea07f3c1c3aacb74e5926a67262f261c1ed3120576ab877b49a81fb8aac51431858662af6b1a8138a44e9d0812d032340369459ccc98b109347cc874c7202dceecc3dbb09d7f9e5658f1ca3a92d22be1fa28f9945205d853e2c866d9b649301ac9857b07b92e4865283d3d5e2b711ea5f85cb2da71965382ece050508d3d008bbe4df5458f70bd3e1bfcc50b34222b43cd28cbe39a3bab6e464664a742161df99c607638e415ced49d0cd719518539ed5f561f81d07fe40d3ce85508e0332465313e60ad9ae271d580022ffca4fbe4d72d38d18e7a6e20d020a1d1e5a8f411291ab95521386fa538ddfe6a391d4a3669cc64c40f07895f031550b32f7d73205a69c214a8ef3cdf996c495e3fd24c00873f30ea6b2bfabfd38de1c3da357d1fefe203573fdad22f675cb5cfabbec0a041b1b31274f70193da8e90cfc4d6dc054c7cd26d09c1dadd064ec52b6ddcfa0cb144d65d9e131c0c88f8004f90d363034d839aa7760167b5302c36d2c2f6714b41782070b10c51c178bd923182d28502f36e19b079b190008c46d19c399331fd60b6b6bde898bd1dd0a71ee7ec7ff7124cc3d374846614389e7b5975b77c4059bc42b810673dbb6f8b951e5b636bdf24afd2a3cbe96ce8600e8a79731b4a56c697596e0bff7b73f413bdbc75069b002b00d713fae8d6450428246f1b794d56717050fdb77bbe094ac2ee6af54a153e2fb8ce1d31a86c4fdd523783b910bedf7db58a46ba6ce48ac3ca194f3cf2275e"),
	}
	rollupId := uint32(1)

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
