package keystore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

const (
	KeyFilePerm = 0600
)

type Keystore struct {
	path string
}

func New(path string) *Keystore {
	return &Keystore{path: path}
}

func (k *Keystore) Exists() bool {
	_, err := os.Stat(k.path)
	return err == nil
}

func (k *Keystore) Encrypt(privateKeyHex string, passphrase string) error {
	key, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return fmt.Errorf("invalid key hex: %w", err)
	}

	if len(key) != 32 {
		return fmt.Errorf("key must be 32 bytes")
	}

	passHash := sha256.Sum256([]byte(passphrase))
	derivedKey := sha256.Sum256(passHash[:])

	block, err := aes.NewCipher(derivedKey[:])
	if err != nil {
		return fmt.Errorf("cipher error: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("gcm error: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)

	ciphertext := gcm.Seal(nonce, nonce, key, nil)

	encrypted := hex.EncodeToString(ciphertext)

	return os.WriteFile(k.path, []byte(encrypted), KeyFilePerm)
}

func (k *Keystore) Decrypt(passphrase string) ([]byte, error) {
	if !k.Exists() {
		return nil, fmt.Errorf("keystore not found")
	}

	data, err := os.ReadFile(k.path)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	ciphertext, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	passHash := sha256.Sum256([]byte(passphrase))
	derivedKey := sha256.Sum256(passHash[:])

	block, err := aes.NewCipher(derivedKey[:])
	if err != nil {
		return nil, fmt.Errorf("cipher error: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm error: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	key, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed (wrong passphrase?): %w", err)
	}

	return key, nil
}

func (k *Keystore) DecryptHex(passphrase string) (string, error) {
	key, err := k.Decrypt(passphrase)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}
