package wallet

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"

	"github.com/Neo4717/NeoCoin/internal/crypto"
)

type WalletStore struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	Address    string
}

func WalletFromPrivateKeyBytes(raw []byte) (*WalletStore, error) {
	if len(raw) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key length")
	}
	priv := make([]byte, len(raw))
	copy(priv, raw)
	pub := ed25519.PrivateKey(priv).Public().(ed25519.PublicKey)
	sum := crypto.Hash256(pub)
	return &WalletStore{
		PrivateKey: ed25519.PrivateKey(priv),
		PublicKey:  pub,
		Address:    hex.EncodeToString(sum[:]),
	}, nil
}
