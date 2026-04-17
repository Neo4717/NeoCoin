package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
)

type HardwareWallet interface {
	GetPublicKey() ([]byte, error)
	Sign(message []byte) ([]byte, error)
	IsConnected() bool
	GetAddress() (string, error)
}

type WalletBridge struct {
	wallet   HardwareWallet
	mu       sync.RWMutex
	cachedPK []byte
}

func NewWalletBridge() *WalletBridge {
	return &WalletBridge{}
}

func (wb *WalletBridge) SetWallet(w HardwareWallet) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	wb.wallet = w
}

func (wb *WalletBridge) GetPublicKey() ([]byte, error) {
	wb.mu.RLock()
	wallet := wb.wallet
	wb.mu.RUnlock()

	if wallet == nil {
		return nil, fmt.Errorf("no wallet connected")
	}

	if len(wb.cachedPK) > 0 {
		return wb.cachedPK, nil
	}

	pk, err := wallet.GetPublicKey()
	if err != nil {
		return nil, err
	}

	wb.mu.Lock()
	wb.cachedPK = pk
	wb.mu.Unlock()

	return pk, nil
}

func (wb *WalletBridge) Sign(message []byte) ([]byte, error) {
	wb.mu.RLock()
	wallet := wb.wallet
	wb.mu.RUnlock()

	if wallet == nil {
		return nil, fmt.Errorf("no wallet connected")
	}

	return wallet.Sign(message)
}

func (wb *WalletBridge) GetAddress() (string, error) {
	pk, err := wb.GetPublicKey()
	if err != nil {
		return "", err
	}

	// Generate address from public key
	hash := sha256.Sum256(pk)
	addrData := make([]byte, 1+32+4)
	addrData[0] = 0x00
	copy(addrData[1:], hash[:32])
	checksum := sha256.Sum256(addrData)
	copy(addrData[33:], checksum[:4])

	return fmt.Sprintf("NEO%s", hex.EncodeToString(addrData)), nil
}

func (wb *WalletBridge) IsConnected() bool {
	wb.mu.RLock()
	wallet := wb.wallet
	wb.mu.RUnlock()

	if wallet == nil {
		return false
	}

	return wallet.IsConnected()
}

type LedgerWallet struct {
	devicePath string
}

func NewLedgerWallet() *LedgerWallet {
	// In production, would use hid library to detect device
	return &LedgerWallet{}
}

func (l *LedgerWallet) GetPublicKey() ([]byte, error) {
	// In production: communicate with Ledger via HID
	// This is a placeholder that checks for LEDGER_DEVICE env var
	deviceID := os.Getenv("LEDGER_DEVICE")
	if deviceID == "" {
		return nil, fmt.Errorf("no ledger device found")
	}

	// Simulated public key (in production, read from device)
	pubKey := make([]byte, ed25519.PublicKeySize)
	return pubKey, nil
}

func (l *LedgerWallet) Sign(message []byte) ([]byte, error) {
	deviceID := os.Getenv("LEDGER_DEVICE")
	if deviceID == "" {
		return nil, fmt.Errorf("no ledger device connected")
	}

	// In production: send to Ledger for signing
	// Ledger displays tx details, user confirms on device
	return nil, fmt.Errorf("signing requires hardware device")
}

func (l *LedgerWallet) IsConnected() bool {
	return os.Getenv("LEDGER_DEVICE") != ""
}

func (l *LedgerWallet) GetAddress() (string, error) {
	pk, err := l.GetPublicKey()
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(pk)
	addrData := make([]byte, 1+32+4)
	addrData[0] = 0x00
	copy(addrData[1:], hash[:32])
	checksum := sha256.Sum256(addrData)
	copy(addrData[33:], checksum[:4])

	return fmt.Sprintf("NEO%s", hex.EncodeToString(addrData)), nil
}

type TrezorWallet struct {
	devicePath string
}

func NewTrezorWallet() *TrezorWallet {
	return &TrezorWallet{}
}

func (t *TrezorWallet) GetPublicKey() ([]byte, error) {
	deviceID := os.Getenv("TREZOR_DEVICE")
	if deviceID == "" {
		return nil, fmt.Errorf("no trezor device found")
	}

	// Placeholder - in production use trezord
	return make([]byte, ed25519.PublicKeySize), nil
}

func (t *TrezorWallet) Sign(message []byte) ([]byte, error) {
	deviceID := os.Getenv("TREZOR_DEVICE")
	if deviceID == "" {
		return nil, fmt.Errorf("no trezor device connected")
	}

	return nil, fmt.Errorf("signing requires hardware device")
}

func (t *TrezorWallet) IsConnected() bool {
	return os.Getenv("TREZOR_DEVICE") != ""
}

func (t *TrezorWallet) GetAddress() (string, error) {
	pk, err := t.GetPublicKey()
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(pk)
	addrData := make([]byte, 1+32+4)
	addrData[0] = 0x00
	copy(addrData[1:], hash[:32])
	checksum := sha256.Sum256(addrData)
	copy(addrData[33:], checksum[:4])

	return fmt.Sprintf("NEO%s", hex.EncodeToString(addrData)), nil
}

func DetectAndConnectWallet() HardwareWallet {
	// Try Ledger first
	ledger := NewLedgerWallet()
	if ledger.IsConnected() {
		fmt.Println("Connected to Ledger wallet")
		return ledger
	}

	// Try Trezor
	trezor := NewTrezorWallet()
	if trezor.IsConnected() {
		fmt.Println("Connected to Trezor wallet")
		return trezor
	}

	return nil
}
