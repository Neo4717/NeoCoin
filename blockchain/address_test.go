package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestValidateAddress_Valid(t *testing.T) {
	_, pubKey, _ := ed25519.GenerateKey(rand.Reader)
	addr := GenerateAddress(pubKey)

	err := ValidateAddress(addr)
	if err != nil {
		t.Errorf("expected valid address, got error: %v", err)
	}
}

func TestIsValidNeoAddress(t *testing.T) {
	_, pubKey, _ := ed25519.GenerateKey(rand.Reader)
	addr := GenerateAddress(pubKey)

	result := IsValidNeoAddress(addr)
	if !result {
		t.Errorf("expected valid NEO address")
	}

	if IsValidNeoAddress("invalid") {
		t.Errorf("expected invalid address to return false")
	}
}

func TestGenerateAddressFromPubKey(t *testing.T) {
	_, pubKey, _ := ed25519.GenerateKey(rand.Reader)

	addr := GenerateAddress(pubKey)
	if addr == "" {
		t.Error("generated address should not be empty")
	}

	addr2 := GenerateAddress(pubKey)
	if addr != addr2 {
		t.Error("same key should produce same address")
	}
}

func TestValidateAddress_Invalid(t *testing.T) {
	invalidAddrs := []string{
		"",
		"TOO_SHORT",
	}

	for _, addr := range invalidAddrs {
		err := ValidateAddress(addr)
		if err == nil {
			t.Errorf("expected invalid address for %q, got none", addr)
		}
	}
}

func TestGetAddressFromPubKey(t *testing.T) {
	_, pubKey, _ := ed25519.GenerateKey(rand.Reader)

	addr := GetAddressFromPubKey(pubKey)
	if addr == "" {
		t.Error("address should not be empty")
	}
}

func TestDecodeAddress(t *testing.T) {
	_, pubKey, _ := ed25519.GenerateKey(rand.Reader)
	addr := GenerateAddress(pubKey)

	decoded, err := DecodeAddress(addr)
	if err != nil {
		t.Fatalf("failed to decode address: %v", err)
	}

	if len(decoded) == 0 {
		t.Error("decoded address should not be empty")
	}
}

func TestGenerateTestAddress(t *testing.T) {
	addr := GenerateTestAddress(0x42)
	if addr == "" {
		t.Error("test address should not be empty")
	}

	addr2 := GenerateTestAddress(0x42)
	if addr != addr2 {
		t.Error("same seed should produce same address")
	}
}
