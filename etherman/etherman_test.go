package etherman

import (
	"context"
	"github.com/0xPolygon/beethoven/tx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"math/big"
	"testing"
)

func signer(from common.Address, tx *types.Transaction) (*types.Transaction, error) {
	return tx, nil
}

type EthermanTestSuite struct {
	suite.Suite
	ethman *Etherman
}

func (suite *EthermanTestSuite) SetupTest() {
	ethman, _ := New(
		new(ethereumClientMock),
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
	)

	suite.ethman = &ethman
}

/*func (suite *EthermanTestSuite) TestGetSequencerAddr(t *testing.T) {
	t.Parallel()

	t.Run("Returns expected error", func(t *testing.T) {
		t.Parallel()

		_, err := suite.ethman.GetSequencerAddr()

		assert.NotNil(t, err)
	})

	t.Run("Returns expected sequencer address", func(t *testing.T) {
		t.Parallel()

		address, err := suite.ethman.GetSequencerAddr()

		assert.Equal(t, err, nil)
		assert.Equal(t, address, common.HexToAddress("0x71c7656ec7ab88b098defb751b7401b5f6d8976f"))
	})
}*/

func (suite *EthermanTestSuite) TestBuildTrustedVerifyBatches(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	t.Run("Return expected ABI encoded bytes array", func(t *testing.T) {
		data, err := suite.ethman.BuildTrustedVerifyBatches(
			0,
			1,
			tx.ZKP{
				NewStateRoot:     common.HexToHash("0x001"),
				NewLocalExitRoot: common.HexToHash("0x002"),
				Proof:            common.Hex2Bytes("0x003"),
			},
		)

		assert.Equal(data, "")
		assert.Nil(err)
	})
}

/**

func TestBuildTrustedVerifyBatchesTxDataAndReturnProofConvertionError() {

}

func TestBuildTrustedVerifyBatchesTxDataAndReturnABIError() {

}

func TestBuildTrustedVerifyBatchesTxDataAndReturnExpectedPackedRLPValue() {

}


func TestCallContractAndReturnExpectedValue() {

}

func TestCallContractAndReturnExpectedError() {

}


func TestCheckTxWasMinedAndReceiptCouldntGetFound() {

}

func TestCheckTxWasMinedAndReturnsTheExpectedGenericError() {

}

func TestCheckTXWasMinedAndReturnsTheExpectedReceipt() {

}


func TestCurrentNonceAndReturnExpectedValue() {

}

func TestCurrentNonceAndReturnExpectedError() {

}


func TestGetTxAndReturnExpectedValue() {

}

func TestGetTxAndReturnExpextedError() {

}


func TestGetTxReceiptAndReturnExpectedValue() {

}

func TestGetTxReceiptAndReturnExpectedError() {

}


func TestWaitTxToBeMinedAndReturnExpectedDeadlineExceededError() {

}

func TestWaitTxToBeMinedAndReturnTheExpectedGenericError() {

}

func TestWaitTXToBeMinedAndReturnExpectedValue() {

}
*/
