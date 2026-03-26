package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
)

type StateCommitment struct {
	BlockHeight int64
	BlockHash   string
	StateRoot   []byte
	UTXORoot    []byte
	ReceiptRoot []byte
	TxCount     int64
	StateHashes map[string][]byte
	Timestamp   int64
}

type CommitmentInfo struct {
	StateRoot []byte
	UTXORoot  []byte
	TxCount   int64
	Accounts  int
	Storage   int64
}

const (
	CommitmentInterval  = 100
	DefaultCommitHeight = 1
)

func (bc *Blockchain) ComputeStateCommitment(height int64) (*StateCommitment, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if height >= int64(len(bc.blocks)) {
		return nil, fmt.Errorf("height %d beyond chain length %d", height, len(bc.blocks))
	}

	block := bc.blocks[height]

	stateRoot, err := bc.computeStateRoot()
	if err != nil {
		return nil, fmt.Errorf("compute state root: %w", err)
	}

	utxoRoot, err := bc.computeUTXORoot()
	if err != nil {
		return nil, fmt.Errorf("compute UTXO root: %w", err)
	}

	return &StateCommitment{
		BlockHeight: height,
		BlockHash:   hex.EncodeToString(block.Hash),
		StateRoot:   stateRoot,
		UTXORoot:    utxoRoot,
		TxCount:     int64(len(block.Transactions)),
		StateHashes: make(map[string][]byte),
		Timestamp:   block.TimestampUnix,
	}, nil
}

func (bc *Blockchain) computeStateRoot() ([]byte, error) {
	addrs := make([]string, 0, len(bc.state))
	for addr := range bc.state {
		addrs = append(addrs, addr)
	}
	sort.Strings(addrs)

	leaves := make([][]byte, 0, len(addrs))
	for _, addr := range addrs {
		acc := bc.state[addr]
		leaf := hashAccountState(addr, acc.Balance, acc.Nonce)
		leaves = append(leaves, leaf)
	}

	if len(leaves) == 0 {
		return sha256.New().Sum(nil), nil
	}

	return computeMerkleRoot(leaves), nil
}

func (bc *Blockchain) computeUTXORoot() ([]byte, error) {
	return bc.computeStateRoot()
}

func hashAccountState(addr string, balance uint64, nonce uint64) []byte {
	h := sha256.New()

	h.Write([]byte(addr))

	var balBytes [8]byte
	binary.LittleEndian.PutUint64(balBytes[:], balance)
	h.Write(balBytes[:])

	var nonceBytes [8]byte
	binary.LittleEndian.PutUint64(nonceBytes[:], nonce)
	h.Write(nonceBytes[:])

	return h.Sum(nil)
}

func computeMerkleRoot(leaves [][]byte) []byte {
	if len(leaves) == 0 {
		return sha256.New().Sum(nil)
	}

	if len(leaves) == 1 {
		return leaves[0]
	}

	n := 1
	for n < len(leaves) {
		n *= 2
	}

	emptyHash := sha256.Sum256([]byte("empty"))
	padded := make([][]byte, n)
	copy(padded, leaves)
	for i := len(leaves); i < n; i++ {
		padded[i] = emptyHash[:]
	}

	for len(padded) > 1 {
		var next [][]byte
		for i := 0; i < len(padded); i += 2 {
			h := sha256.New()
			h.Write(padded[i])
			h.Write(padded[i+1])
			next = append(next, h.Sum(nil))
		}
		padded = next
	}

	return padded[0]
}

func (bc *Blockchain) VerifyStateCommitment(height int64, commitment *StateCommitment) (bool, error) {
	computed, err := bc.ComputeStateCommitment(height)
	if err != nil {
		return false, err
	}

	if !bytesEqual(computed.StateRoot, commitment.StateRoot) {
		return false, nil
	}

	return true, nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (bc *Blockchain) GetCommitmentInfo(height int64) (*CommitmentInfo, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	stateRoot, _ := bc.computeStateRoot()
	utxoRoot, _ := bc.computeUTXORoot()

	return &CommitmentInfo{
		StateRoot: stateRoot,
		UTXORoot:  utxoRoot,
		TxCount:   int64(len(bc.blocks)),
		Accounts:  len(bc.state),
	}, nil
}

func (bc *Blockchain) StoreCommitment(c *StateCommitment) error {
	return bc.store.WriteCommitment(c)
}

func (bc *Blockchain) LoadCommitment(height int64) (*StateCommitment, error) {
	return bc.store.ReadCommitment(height)
}

func ShouldCommit(height uint64) bool {
	return height == 1 || height%CommitmentInterval == 0
}

func (bc *Blockchain) ComputeAndStoreCommitment(height uint64) error {
	if !ShouldCommit(height) {
		return nil
	}

	commitment, err := bc.ComputeStateCommitment(int64(height))
	if err != nil {
		return fmt.Errorf("compute commitment: %w", err)
	}

	if err := bc.store.WriteCommitment(commitment); err != nil {
		return fmt.Errorf("store commitment: %w", err)
	}

	bc.mu.Lock()
	if height < uint64(len(bc.blocks)) {
		bc.blocks[height].StateRoot = commitment.StateRoot
		bc.blocks[height].UTXORoot = commitment.UTXORoot
	}
	bc.mu.Unlock()

	return nil
}
