package blockchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Neo4717/NeoCoin/config"
)

func TestPrunerCreation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		PruneDepth:         100,
		CheckpointInterval: 10,
		StoreMode:          "pruned",
	}

	pruner := NewPruner(store, cfg)
	if pruner == nil {
		t.Fatal("expected non-nil pruner")
	}
	if pruner.pruneDepth != 100 {
		t.Errorf("expected pruneDepth 100, got %d", pruner.pruneDepth)
	}
	if pruner.checkpointInterval != 10 {
		t.Errorf("expected checkpointInterval 10, got %d", pruner.checkpointInterval)
	}
}

func TestPrunerCheckpointWriteAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		PruneDepth:         1000,
		CheckpointInterval: 100,
		StoreMode:          "pruned",
	}

	pruner := NewPruner(store, cfg)

	state := map[string]Account{
		"addr1": {Balance: 1000, Nonce: 1},
		"addr2": {Balance: 500, Nonce: 0},
	}

	err = pruner.WriteCheckpoint(100, state)
	if err != nil {
		t.Fatalf("failed to write checkpoint: %v", err)
	}

	readState, err := store.ReadCheckpoint(100)
	if err != nil {
		t.Fatalf("failed to read checkpoint: %v", err)
	}

	if len(readState) != len(state) {
		t.Errorf("expected %d accounts, got %d", len(state), len(readState))
	}

	if readState["addr1"].Balance != 1000 {
		t.Errorf("expected addr1 balance 1000, got %d", readState["addr1"].Balance)
	}
	if readState["addr2"].Balance != 500 {
		t.Errorf("expected addr2 balance 500, got %d", readState["addr2"].Balance)
	}
}

func TestPrunerGetCheckpointHeights(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		PruneDepth:         1000,
		CheckpointInterval: 100,
		StoreMode:          "pruned",
	}

	_ = NewPruner(store, cfg)

	for _, h := range []int64{0, 100, 200, 300} {
		state := map[string]Account{"addr": {Balance: uint64(h), Nonce: 0}}
		err := store.WriteCheckpoint(h, state)
		if err != nil {
			t.Fatalf("failed to write checkpoint at %d: %v", h, err)
		}
	}

	heights, err := store.GetCheckpointHeights()
	if err != nil {
		t.Fatalf("failed to get checkpoint heights: %v", err)
	}

	if len(heights) != 4 {
		t.Errorf("expected 4 checkpoints, got %d", len(heights))
	}

	expected := []int64{0, 100, 200, 300}
	for i, h := range heights {
		if h != expected[i] {
			t.Errorf("expected height %d at index %d, got %d", expected[i], i, h)
		}
	}
}

func TestPrunerShouldPrune(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		PruneDepth:         1000,
		CheckpointInterval: 100,
		StoreMode:          "pruned",
	}

	pruner := NewPruner(store, cfg)

	if pruner.ShouldPrune() {
		t.Error("should not prune with no blocks")
	}

	for _, h := range []int64{0, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 1100, 1200} {
		state := map[string]Account{"addr": {Balance: uint64(h), Nonce: 0}}
		err := store.WriteCheckpoint(h, state)
		if err != nil {
			t.Fatalf("failed to write checkpoint at %d: %v", h, err)
		}
	}
}

func TestPrunerPruneBelow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		PruneDepth:         100,
		CheckpointInterval: 10,
		StoreMode:          "pruned",
	}

	pruner := NewPruner(store, cfg)

	for _, h := range []int64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100} {
		state := map[string]Account{"addr": {Balance: uint64(h), Nonce: 0}}
		err := store.WriteCheckpoint(h, state)
		if err != nil {
			t.Fatalf("failed to write checkpoint at %d: %v", h, err)
		}
	}

	heights, err := store.GetCheckpointHeights()
	if err != nil {
		t.Fatalf("failed to get checkpoint heights before prune: %v", err)
	}
	if len(heights) != 11 {
		t.Errorf("expected 11 checkpoints before prune, got %d", len(heights))
	}

	err = pruner.PruneToHeight(50)
	if err != nil {
		t.Fatalf("failed to prune: %v", err)
	}

	heights, err = store.GetCheckpointHeights()
	if err != nil {
		t.Fatalf("failed to get checkpoint heights after prune: %v", err)
	}
	if len(heights) != 11 {
		t.Errorf("expected checkpoints to be preserved after prune, got %d", len(heights))
	}
}

func TestPrunerStateIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		PruneDepth:         100,
		CheckpointInterval: 10,
		StoreMode:          "pruned",
	}

	_ = NewPruner(store, cfg)

	finalState := map[string]Account{}
	for i := int64(0); i <= 50; i++ {
		state := map[string]Account{
			"addr1": {Balance: uint64(i * 100), Nonce: uint64(i)},
			"addr2": {Balance: uint64(1000 - i*10), Nonce: uint64(i)},
		}
		err := store.WriteCheckpoint(i*10, state)
		if err != nil {
			t.Fatalf("failed to write checkpoint at %d: %v", i*10, err)
		}
		finalState = state
	}

	readState, err := store.ReadCheckpoint(500)
	if err != nil {
		t.Fatalf("failed to read final checkpoint: %v", err)
	}

	if readState["addr1"].Balance != finalState["addr1"].Balance {
		t.Errorf("addr1 balance mismatch: expected %d, got %d", finalState["addr1"].Balance, readState["addr1"].Balance)
	}
	if readState["addr2"].Balance != finalState["addr2"].Balance {
		t.Errorf("addr2 balance mismatch: expected %d, got %d", finalState["addr2"].Balance, readState["addr2"].Balance)
	}
}

func TestPrunerGetStateAt(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		PruneDepth:         1000,
		CheckpointInterval: 100,
		StoreMode:          "pruned",
	}

	pruner := NewPruner(store, cfg)

	for i := int64(0); i <= 500; i += 100 {
		state := map[string]Account{
			"addr": {Balance: uint64(i), Nonce: uint64(i)},
		}
		err := store.WriteCheckpoint(i, state)
		if err != nil {
			t.Fatalf("failed to write checkpoint at %d: %v", i, err)
		}
	}

	stateAt250, err := pruner.GetStateAt(250)
	if err != nil {
		t.Fatalf("failed to get state at 250: %v", err)
	}
	if stateAt250["addr"].Balance != 200 {
		t.Errorf("expected balance 200 at height 250, got %d", stateAt250["addr"].Balance)
	}

	stateAt50, err := pruner.GetStateAt(50)
	if err != nil {
		t.Fatalf("failed to get state at 50: %v", err)
	}
	if stateAt50["addr"].Balance != 0 {
		t.Errorf("expected balance 0 at height 50, got %d", stateAt50["addr"].Balance)
	}
}

func TestStoreModeConstants(t *testing.T) {
	if StoreModeFull != 0 {
		t.Errorf("expected StoreModeFull to be 0, got %d", StoreModeFull)
	}
	if StoreModePruned != 1 {
		t.Errorf("expected StoreModePruned to be 1, got %d", StoreModePruned)
	}
	if StoreModeArchive != 2 {
		t.Errorf("expected StoreModeArchive to be 2, got %d", StoreModeArchive)
	}
}

func TestDefaultPruneDepth(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		StoreMode: "pruned",
	}

	pruner := NewPruner(store, cfg)
	if pruner.pruneDepth != 1000 {
		t.Errorf("expected default pruneDepth 1000, got %d", pruner.pruneDepth)
	}
	if pruner.checkpointInterval != 100 {
		t.Errorf("expected default checkpointInterval 100, got %d", pruner.checkpointInterval)
	}
}

func TestBoltStorePruneBelowDeletesCanonical(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	genesis := &Block{
		Height: 0,
		Hash:   []byte("genesis"),
	}
	err = store.AppendCanonical(genesis)
	if err != nil {
		t.Fatalf("failed to append genesis: %v", err)
	}
	err = store.PutBlock(genesis)
	if err != nil {
		t.Fatalf("failed to put genesis: %v", err)
	}

	for i := 1; i <= 10; i++ {
		b := &Block{
			Height:   uint64(i),
			Hash:     []byte{byte(i)},
			PrevHash: []byte("genesis"),
		}
		if i > 1 {
			b.PrevHash = []byte{byte(i - 1)}
		}
		err = store.AppendCanonical(b)
		if err != nil {
			t.Fatalf("failed to append block %d: %v", i, err)
		}
		err = store.PutBlock(b)
		if err != nil {
			t.Fatalf("failed to put block %d: %v", i, err)
		}
	}

	blocks, err := store.ReadCanonical()
	if err != nil {
		t.Fatalf("failed to read canonical: %v", err)
	}
	if len(blocks) != 11 {
		t.Errorf("expected 11 blocks before prune, got %d", len(blocks))
	}

	err = store.PruneBelow(5)
	if err != nil {
		t.Fatalf("failed to prune below 5: %v", err)
	}

	blocks, err = store.ReadCanonical()
	if err != nil {
		t.Fatalf("failed to read canonical after prune: %v", err)
	}
	if len(blocks) != 6 {
		t.Errorf("expected 6 blocks after prune, got %d", len(blocks))
	}

	if blocks[0].Height != 5 {
		t.Errorf("expected first block height 5, got %d", blocks[0].Height)
	}
	if blocks[len(blocks)-1].Height != 10 {
		t.Errorf("expected last block height 10, got %d", blocks[len(blocks)-1].Height)
	}
}

func TestPrunerPruneToHeightPreventsBelowPruneDepth(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		PruneDepth: 100,
		StoreMode:  "pruned",
	}

	pruner := NewPruner(store, cfg)

	err = pruner.PruneToHeight(50)
	if err != nil {
		t.Errorf("unexpected error when pruning: %v", err)
	}
}

func TestCheckpointRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		PruneDepth:         1000,
		CheckpointInterval: 100,
		StoreMode:          "pruned",
	}

	pruner := NewPruner(store, cfg)

	state := map[string]Account{
		"addr1": {Balance: 1000, Nonce: 1},
		"addr2": {Balance: 2000, Nonce: 2},
		"addr3": {Balance: 3000, Nonce: 3},
	}

	err = pruner.WriteCheckpoint(200, state)
	if err != nil {
		t.Fatalf("failed to write checkpoint: %v", err)
	}

	readState, err := pruner.GetStateAt(250)
	if err != nil {
		t.Fatalf("failed to get state at 250: %v", err)
	}

	if len(readState) != len(state) {
		t.Errorf("expected %d accounts, got %d", len(state), len(readState))
	}

	for addr, expectedAcct := range state {
		readAcct, ok := readState[addr]
		if !ok {
			t.Errorf("missing account %s", addr)
			continue
		}
		if readAcct.Balance != expectedAcct.Balance {
			t.Errorf("account %s: expected balance %d, got %d", addr, expectedAcct.Balance, readAcct.Balance)
		}
		if readAcct.Nonce != expectedAcct.Nonce {
			t.Errorf("account %s: expected nonce %d, got %d", addr, expectedAcct.Nonce, readAcct.Nonce)
		}
	}
}

func TestFastSyncInit(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		PruneDepth:         1000,
		CheckpointInterval: 100,
		StoreMode:          "pruned",
	}

	pruner := NewPruner(store, cfg)

	for i := int64(0); i <= 500; i += 100 {
		state := map[string]Account{
			"addr": {Balance: uint64(i * 10), Nonce: uint64(i)},
		}
		err := store.WriteCheckpoint(i, state)
		if err != nil {
			t.Fatalf("failed to write checkpoint at %d: %v", i, err)
		}
	}

	state, err := pruner.FastSyncInit(350)
	if err != nil {
		t.Fatalf("failed to fast sync init: %v", err)
	}

	if state["addr"].Balance != 3000 {
		t.Errorf("expected balance 3000, got %d", state["addr"].Balance)
	}

	_, err = pruner.FastSyncInit(1000)
	if err != nil {
		t.Fatalf("fast sync init should work for height > latest checkpoint")
	}
}

func TestPrunerWithEmptyStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := OpenBoltChainStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		PruneDepth: 1000,
		StoreMode:  "pruned",
	}

	pruner := NewPruner(store, cfg)

	if pruner.ShouldPrune() {
		t.Error("should not prune empty store")
	}

	_, err = pruner.GetStateAt(100)
	if err == nil {
		t.Error("expected error getting state from empty store")
	}
}

func cleanupTestDB(t *testing.T, path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Logf("cleanup warning: %v", err)
	}
}
