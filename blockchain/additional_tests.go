package main

import (
	"testing"
)

func TestNewAISpamFilter(t *testing.T) {
	filter := NewAISpamFilter("", false)
	if filter == nil {
		t.Error("NewAISpamFilter should not return nil")
	}
}

func TestIsValidNeoAddress_EdgeCases(t *testing.T) {
	if IsValidNeoAddress("NEOaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") {
		t.Log("valid NEO address detected")
	}

	if IsValidNeoAddress("") {
		t.Error("empty string should not be valid")
	}

	if IsValidNeoAddress("BTC123") {
		t.Error("BTC prefix should not be valid")
	}
}

func TestWorkForDifficultyBits(t *testing.T) {
	tests := []uint32{1, 8, 16, 32}

	for _, bits := range tests {
		t.Run("", func(t *testing.T) {
			got := WorkForDifficultyBits(bits)
			if got == nil {
				t.Error("WorkForDifficultyBits should not return nil")
			}
		})
	}
}

func TestWallet_New(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Skipf("NewWallet: %v", err)
	}

	if wallet == nil {
		t.Error("NewWallet should not return nil")
	}

	if wallet.Address == "" {
		t.Error("wallet address should not be empty")
	}
}

func TestWalletFromPrivateKeyBase64(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Skipf("NewWallet: %v", err)
	}

	keyB64 := wallet.PrivateKeyBase64()
	wallet2, err := WalletFromPrivateKeyBase64(keyB64)
	if err != nil {
		t.Errorf("WalletFromPrivateKeyBase64: %v", err)
		return
	}

	if wallet2.Address != wallet.Address {
		t.Error("should recover same address")
	}
}

func TestEnvParsing_Bool(t *testing.T) {
	t.Setenv("TEST_BOOL", "true")
	if !envBool("TEST_BOOL", false) {
		t.Error("TEST_BOOL should be true")
	}
}

func TestEnvParsing_Int(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	if envInt("TEST_INT", 0) != 42 {
		t.Error("TEST_INT should be 42")
	}
}

func TestEnvParsing_Uint64(t *testing.T) {
	t.Setenv("TEST_UINT64", "100")
	if envUint64("TEST_UINT64", 0) != 100 {
		t.Error("TEST_UINT64 should be 100")
	}
}

func TestEnvParsing_Int64(t *testing.T) {
	t.Setenv("TEST_INT64", "-50")
	if envInt64("TEST_INT64", 0) != -50 {
		t.Error("TEST_INT64 should be -50")
	}
}

func TestEnvParsing_Duration(t *testing.T) {
	t.Setenv("TEST_DURATION", "1000")
	d := envDurationMS("TEST_DURATION", 0)
	if d == 0 {
		t.Error("TEST_DURATION should not be 0")
	}
}

func TestEnvParsing_Uint32(t *testing.T) {
	t.Setenv("TEST_UINT32", "200")
	if envUint32("TEST_UINT32", 0) != 200 {
		t.Error("TEST_UINT32 should be 200")
	}
}

func TestGenesisConfig(t *testing.T) {
	cfg := &GenesisConfig{}
	_ = cfg
}

func TestChainStore(t *testing.T) {
	store, err := OpenChainStoreFromEnv()
	if err != nil {
		t.Logf("OpenChainStoreFromEnv error (expected if no db): %v", err)
		return
	}
	if store != nil {
		t.Log("Chain store opened")
	}
}

func TestBlockchain_RulesHashHex(t *testing.T) {
	store, _ := OpenChainStoreFromEnv()
	if store == nil {
		t.Skip("store nil")
	}

	bc, _ := LoadBlockchain(1, "", store, 0)
	hash := bc.RulesHashHex()
	_ = hash
}

func TestMiner_New(t *testing.T) {
	store, _ := OpenChainStoreFromEnv()
	if store == nil {
		t.Skip("store nil")
	}

	bc, _ := LoadBlockchain(1, "", store, 0)
	mp := NewMempool(100)

	miner := NewMiner(bc, mp, 10, false)
	if miner == nil {
		t.Error("NewMiner should not return nil")
	}
}

func TestMempool_New(t *testing.T) {
	mp := NewMempool(100)
	if mp == nil {
		t.Error("NewMempool should not return nil")
	}
}

func TestMempoolWithAI_New(t *testing.T) {
	filter := NewAISpamFilter("", false)
	mp := NewMempoolWithAI(100, filter)
	if mp == nil {
		t.Error("NewMempoolWithAI should not return nil")
	}
}

func TestMempool_Size_Additional(t *testing.T) {
	mp := NewMempool(100)
	size := mp.Size()
	if size < 0 {
		t.Error("size should be non-negative")
	}
}

func TestNewMempoolStats(t *testing.T) {
	mp := NewMempool(100)
	_, _, _ = mp.GetSpamStats()
}
