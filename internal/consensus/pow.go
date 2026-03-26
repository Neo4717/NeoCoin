package consensus

import (
	"crypto/sha256"
	"math/big"
)

const (
	defaultDifficultyBits = uint32(18)
)

type PoW struct {
	target *big.Int
	header []byte
	nonce  uint64
}

func NewPoW(header []byte, difficultyBits uint32, nonce uint64) *PoW {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-difficultyBits))
	return &PoW{
		target: target,
		header: header,
		nonce:  nonce,
	}
}

func (pow *PoW) Validate() (bool, error) {
	sum := sha256.Sum256(pow.header)
	var hashInt big.Int
	hashInt.SetBytes(sum[:])
	return hashInt.Cmp(pow.target) == -1, nil
}

func ValidatePoW(header []byte, expectedHash []byte, difficultyBits uint32) (bool, error) {
	sum := sha256.Sum256(header)
	if len(expectedHash) != 0 && string(expectedHash) != string(sum[:]) {
		return false, nil
	}
	target := big.NewInt(1)
	target.Lsh(target, uint(256-difficultyBits))
	var hashInt big.Int
	hashInt.SetBytes(sum[:])
	return hashInt.Cmp(target) == -1, nil
}
