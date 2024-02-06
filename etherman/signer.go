package etherman

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// NewKeyedTransactorWithChainID is a utility method to easily create a transaction signer
// from a single private key.
func NewGCPKMSTransactorWithChainID(key *ecdsa.PrivateKey, chainID *big.Int) (*bind.TransactOpts, error) {
	keyAddr := crypto.PubkeyToAddress(key.PublicKey)
	if chainID == nil {
		return nil, bind.ErrNoChainID
	}
	signer := types.LatestSignerForChainID(chainID)
	return &bind.TransactOpts{
		From: keyAddr,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != keyAddr {
				return nil, bind.ErrNotAuthorized
			}
			gcpKMS := GCPKMS{}
			return gcpKMS.Sign(cmd.Context(), tx)

			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), key)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
		Context: context.Background(),
	}, nil
}
