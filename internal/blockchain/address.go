package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	AddressPrefix  = "NEO"
	AddressVersion = 0x00
	ChecksumLen    = 4
	HashLen        = 32
)

func GenerateAddress(pubKey []byte) string {
	hash := sha256.Sum256(pubKey)
	addressHash := hash[:HashLen]

	addressData := make([]byte, 1+len(addressHash))
	addressData[0] = AddressVersion
	copy(addressData[1:], addressHash)

	checksum := sha256.Sum256(addressData)
	addressData = append(addressData, checksum[:ChecksumLen]...)

	encoded := hex.EncodeToString(addressData)

	return fmt.Sprintf("%s%s", AddressPrefix, encoded)
}

func ValidateAddress(addr string) error {
	if len(addr) < len(AddressPrefix)+10 {
		return fmt.Errorf("address too short")
	}

	if addr[:len(AddressPrefix)] != AddressPrefix {
		return fmt.Errorf("invalid prefix, expected %s", AddressPrefix)
	}

	encoded := addr[len(AddressPrefix):]

	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("invalid hex: %w", err)
	}

	if len(decoded) < ChecksumLen+1 {
		return fmt.Errorf("invalid encoded length")
	}

	addressData := decoded[:len(decoded)-ChecksumLen]
	storedChecksum := decoded[len(decoded)-ChecksumLen:]

	checksum := sha256.Sum256(addressData)

	for i := 0; i < ChecksumLen; i++ {
		if storedChecksum[i] != checksum[i] {
			return fmt.Errorf("checksum mismatch")
		}
	}

	return nil
}
