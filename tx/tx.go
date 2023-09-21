package tx

import (
	"crypto/ecdsa"

	ethmanTypes "github.com/0xPolygonHermez/zkevm-node/etherman/types"
	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// type L1Concensus string

// const (
// 	Rollup   L1Concensus = "rollup"
// 	Validium L1Concensus = "validium"
// )

type Tx struct {
	L1Contract common.Address
	// L1Concensus      L1Con census
	// Batches          []types.Batch
	LastVerifiedBatch types.ArgUint64
	NewVerifiedBatch  types.ArgUint64
	ZKP               ethmanTypes.FinalProofInputs
	NewStateRoot      types.ArgHash
	NewLocalExitRoot  types.ArgHash
}

// Hash returns a hash that uniquely identifies the tx
func (t *Tx) Hash() common.Hash {
	return common.BytesToHash(crypto.Keccak256(
		[]byte(t.LastVerifiedBatch.Hex()),
		[]byte(t.NewVerifiedBatch.Hex()),
		[]byte(t.ZKP.FinalProof.Proof),
		t.NewStateRoot[:],
		t.NewLocalExitRoot[:],
	))
}

// Sign returns a signed batch by the private key
func (t *Tx) Sign(privateKey *ecdsa.PrivateKey) (*SignedTx, error) {
	hashToSign := t.Hash()
	sig, err := crypto.Sign(hashToSign.Bytes(), privateKey)
	if err != nil {
		return nil, err
	}
	return &SignedTx{
		Tx:        *t,
		Signature: sig,
	}, nil
}

type SignedTx struct {
	Tx        Tx             `json:"tx"`
	Signature types.ArgBytes `json:"signature"`
}

// Signer returns the address of the signer
func (s *SignedTx) Signer() (common.Address, error) {
	pubKey, err := crypto.SigToPub(s.Tx.Hash().Bytes(), s.Signature)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pubKey), nil
}
