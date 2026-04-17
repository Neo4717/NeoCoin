package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

type SecureEngine struct {
	key []byte
}

func NewSecureEngine(password string) (*SecureEngine, error) {
	sum := sha256.Sum256([]byte(password))
	key := make([]byte, 32)
	copy(key, sum[:])
	return &SecureEngine{key: key}, nil
}

func (s *SecureEngine) Encrypt(plaintext []byte) ([]byte, error) {
	if len(s.key) != 32 {
		return plaintext, errors.New("invalid key size")
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (s *SecureEngine) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(s.key) != 32 {
		return ciphertext, errors.New("invalid key size")
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed: %w", err)
	}

	return plaintext, nil
}

func (s *SecureEngine) EncryptStoreValue(key []byte, value []byte) ([]byte, error) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(len(key)))
	buf.Write(key)
	buf.Write(value)
	return s.Encrypt(buf.Bytes())
}

func (s *SecureEngine) DecryptStoreValue(data []byte) ([]byte, []byte, error) {
	plaintext, err := s.Decrypt(data)
	if err != nil {
		return nil, nil, err
	}

	if len(plaintext) < 4 {
		return nil, nil, errors.New("invalid encrypted data")
	}

	keyLen := binary.BigEndian.Uint32(plaintext[:4])
	if int(keyLen)+4 > len(plaintext) {
		return nil, nil, errors.New("invalid key length")
	}

	key := plaintext[4 : 4+keyLen]
	value := plaintext[4+keyLen:]
	return key, value, nil
}
