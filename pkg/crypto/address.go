package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	AddressPrefix  = "NEO"
	AddressVersion = 0x00
	ChecksumLen    = 4
	HashLen        = 32
)

var (
	ErrInvalidPrefix   = fmt.Errorf("invalid prefix, expected %s", AddressPrefix)
	ErrAddressTooShort = errors.New("address too short")
	ErrInvalidHex      = errors.New("invalid hex encoding")
	ErrChecksum        = errors.New("checksum mismatch")
	ErrHashLen         = errors.New("invalid hash length")

	prefixLen = len(AddressPrefix)
	minLen    = prefixLen + 10
)

func GenerateAddress(pubKey []byte) string {
	if len(pubKey) != 32 {
		return ""
	}

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
	if len(addr) < minLen {
		return ErrAddressTooShort
	}

	if !strings.HasPrefix(addr, AddressPrefix) {
		return ErrInvalidPrefix
	}

	encoded := addr[prefixLen:]

	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidHex, err)
	}

	if len(decoded) < ChecksumLen+1 {
		return errors.New("encoded address too short")
	}

	addressData := decoded[:len(decoded)-ChecksumLen]
	storedChecksum := decoded[len(decoded)-ChecksumLen:]

	checksum := sha256.Sum256(addressData)

	for i := 0; i < ChecksumLen; i++ {
		if storedChecksum[i] != checksum[i] {
			return ErrChecksum
		}
	}

	return nil
}

func DecodeAddress(addr string) ([]byte, error) {
	if err := ValidateAddress(addr); err != nil {
		return nil, err
	}

	encoded := addr[prefixLen:]
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	hashLen := len(decoded) - ChecksumLen
	if hashLen != HashLen+1 {
		return nil, fmt.Errorf("invalid hash length: %d", hashLen)
	}

	return decoded[1 : HashLen+1], nil
}

func RawHexToAddress(rawHex string) (string, error) {
	b, err := hex.DecodeString(rawHex)
	if err != nil {
		return "", err
	}

	if len(b) != 32 {
		return "", fmt.Errorf("raw hex must be 32 bytes, got %d", len(b))
	}

	return GenerateAddress(b), nil
}

func IsValidAddress(addr string) bool {
	return ValidateAddress(addr) == nil
}

func NormalizeAddress(addr string) string {
	if strings.HasPrefix(addr, AddressPrefix) {
		return addr
	}

	raw, err := hex.DecodeString(addr)
	if err != nil || len(raw) != 32 {
		return addr
	}

	return GenerateAddress(raw)
}
