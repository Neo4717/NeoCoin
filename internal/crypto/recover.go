package crypto

import (
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
)

// RecoverPrivateKey recovers the private key 'd' given two signatures (r, s1) and (r, s2)
// for message hashes z1 and z2, sharing the same nonce 'k'.
func RecoverPrivateKey(r, s1, z1, s2, z2 *big.Int) *big.Int {
	n := btcec.S256().N

	// k = (z1 - z2) * inv(s1 - s2) mod n
	sDiff := new(big.Int).Sub(s1, s2)
	sDiff.Mod(sDiff, n)
	if sDiff.Sign() == 0 {
		return nil
	}
	sDiffInv := new(big.Int).ModInverse(sDiff, n)

	zDiff := new(big.Int).Sub(z1, z2)
	zDiff.Mod(zDiff, n)

	k := new(big.Int).Mul(zDiff, sDiffInv)
	k.Mod(k, n)

	// d = (s1 * k - z1) * inv(r) mod n
	rInv := new(big.Int).ModInverse(r, n)
	if rInv == nil {
		return nil
	}

	sk := new(big.Int).Mul(s1, k)
	sk.Sub(sk, z1)
	sk.Mod(sk, n)

	d := new(big.Int).Mul(sk, rInv)
	d.Mod(d, n)

	return d
}
