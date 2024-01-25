package tx

import (
	"crypto/ecdsa"
	"github.com/0xPolygon/agglayer/rpc/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// type L1Consensus string

// const (
// 	Rollup   L1Consensus = "rollup"
// 	Validium L1Consensus = "validium"
// )

type ZKP struct {
	NewStateRoot     common.Hash    `json:"newStateRoot"`
	NewLocalExitRoot common.Hash    `json:"newLocalExitRoot"`
	Proof            types.ArgBytes `json:"proof"`
}

type Tx struct {
	// L1Consensus      L1Consensus
	// Batches          []types.Batch
	RollupID          uint32
	LastVerifiedBatch types.ArgUint64 `json:"lastVerifiedBatch"`
	NewVerifiedBatch  types.ArgUint64 `json:"newVerifiedBatch"`
	ZKP               ZKP             `json:"ZKP"`
}

// Hash returns a hash that uniquely identifies the tx
func (t *Tx) Hash() common.Hash {
	return common.BytesToHash(crypto.Keccak256(
		[]byte(t.LastVerifiedBatch.Hex()),
		[]byte(t.NewVerifiedBatch.Hex()),
		t.ZKP.NewStateRoot[:],
		t.ZKP.NewLocalExitRoot[:],
		[]byte(t.ZKP.Proof.Hex()),
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
