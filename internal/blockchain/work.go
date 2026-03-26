package blockchain

import "math/big"

func WorkForDifficultyBits(bits uint32) *big.Int {
	if bits > maxDifficultyBits {
		bits = maxDifficultyBits
	}
	if bits == 0 {
		return big.NewInt(0)
	}
	return new(big.Int).Lsh(big.NewInt(1), uint(bits))
}
