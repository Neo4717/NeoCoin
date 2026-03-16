package main

import (
	"testing"
)

func TestVMBasic(t *testing.T) {
	code := []byte{OP_PUSH1, 42, OP_RETURN}
	vm := NewVM(code, make(map[string][]byte), 1000)
	result := vm.Run()

	if !result.Success {
		t.Errorf("VM failed: %s", result.Error)
	}
}

func TestVMAddition(t *testing.T) {
	code := []byte{OP_PUSH1, 10, OP_PUSH1, 20, OP_ADD, OP_RETURN}
	vm := NewVM(code, make(map[string][]byte), 1000)
	result := vm.Run()

	if !result.Success {
		t.Errorf("VM failed: %s", result.Error)
	}
}

func TestVMGasTracking(t *testing.T) {
	code := []byte{OP_PUSH1, 42, OP_RETURN}
	vm := NewVM(code, make(map[string][]byte), 100)
	result := vm.Run()

	if result.GasUsed <= 0 {
		t.Errorf("Expected gas usage, got %d", result.GasUsed)
	}
}

func TestTokenContract(t *testing.T) {
	contract := NewTokenContract("TestToken", "TST", 8, 1000000)

	contract.Balances["test_addr"] = 100

	if contract.BalanceOf("test_addr") != 100 {
		t.Errorf("Expected balance 100, got %d", contract.BalanceOf("test_addr"))
	}

	err := contract.Transfer("test_addr", "other_addr", 50)
	if err != nil {
		t.Errorf("Transfer failed: %v", err)
	}

	if contract.BalanceOf("test_addr") != 50 {
		t.Errorf("Expected balance 50, got %d", contract.BalanceOf("test_addr"))
	}
}

func TestMultiSigContract(t *testing.T) {
	pubKeys := []string{"pubkey1", "pubkey2", "pubkey3"}
	contract := NewMultiSigContract(2, pubKeys)

	if contract.GetRequired() != 2 {
		t.Errorf("Expected required 2, got %d", contract.GetRequired())
	}

	sigs := [][]byte{[]byte("sig1"), []byte("sig2")}
	if !contract.ValidateSignatures(sigs) {
		t.Errorf("Expected signatures to be valid")
	}

	badSigs := [][]byte{[]byte("sig1")}
	if contract.ValidateSignatures(badSigs) {
		t.Errorf("Expected signatures to be invalid")
	}
}

func TestSPVMerkleTree(t *testing.T) {
	txHashes := []string{"tx1", "tx2", "tx3", "tx4"}
	tree := NewSPVMerkleTree(txHashes)

	root := tree.GetRoot()
	if root == "" {
		t.Error("Expected non-empty merkle root")
	}
}

func TestLightClientAddressSync(t *testing.T) {
	config := LightClientConfig{
		ServerURL:   "http://localhost:8080",
		MaxSPVDepth: 100,
	}
	client := NewLightClient(config)

	if client.serverURL != "http://localhost:8080" {
		t.Errorf("Expected server URL to be set")
	}
}

func TestAddressValidation(t *testing.T) {
	valid := "NEO006c11712656d8b683fd00ec52594cb47a0ac43e5365168104a8bec2ebdf3507898e2d974e"
	if err := ValidateAddress(valid); err != nil {
		t.Errorf("Valid address rejected: %v", err)
	}

	invalid := "NEOinvalid"
	if err := ValidateAddress(invalid); err == nil {
		t.Error("Expected invalid address to be rejected")
	}
}

func TestGenerateAddress(t *testing.T) {
	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i)
	}

	addr := GenerateAddress(pubKey)
	if len(addr) < 10 {
		t.Errorf("Address too short: %s", addr)
	}

	if addr[:3] != "NEO" {
		t.Errorf("Expected NEO prefix, got %s", addr[:3])
	}
}
