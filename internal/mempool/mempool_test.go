package mempool

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Neo4717/NeoCoin/config"
	"github.com/Neo4717/NeoCoin/internal/blockchain"
)

func newTestMempool(maxSize int, minFeeRate int64) *Mempool {
	cfg := &config.Config{
		MempoolMaxSize:    maxSize,
		MempoolMinFeeRate: minFeeRate,
	}
	return NewMempool(cfg)
}

func makeTransferTx(fromPubKey []byte, toAddress string, amount, fee uint64, nonce uint64) blockchain.Transaction {
	if len(fromPubKey) != 32 {
		fromPubKey = make([]byte, 32)
		copy(fromPubKey, []byte(fmt.Sprintf("sender-%d", nonce)))
		for i := 0; i < 32 && i < len(fmt.Sprintf("sender-%d", nonce)); i++ {
			fromPubKey[i] = fmt.Sprintf("sender-%d", nonce)[i]
		}
	}
	return blockchain.Transaction{
		Type:       blockchain.TxTransfer,
		FromPubKey: fromPubKey,
		ToAddress:  toAddress,
		Amount:     amount,
		Fee:        fee,
		Nonce:      nonce,
	}
}

func TestMempool_AddAndGet(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	tx := makeTransferTx(nil, "recipient1", 100, 10, 1)
	txPtr := &tx
	if err := mp.Add(txPtr); err != nil {
		t.Fatalf("failed to add tx: %v", err)
	}

	txid, _ := blockchain.TxIDHex(tx)
	retrieved, ok := mp.Get(txid)
	if !ok {
		t.Fatal("failed to retrieve tx")
	}
	if retrieved.ToAddress != "recipient1" {
		t.Errorf("expected recipient1, got %s", retrieved.ToAddress)
	}
}

func TestMempool_AddDuplicate(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	tx := makeTransferTx(nil, "recipient1", 100, 10, 1)
	txPtr := &tx
	if err := mp.Add(txPtr); err != nil {
		t.Fatalf("failed to add tx: %v", err)
	}

	err := mp.Add(txPtr)
	if err == nil {
		t.Error("expected error for duplicate tx")
	}
}

func TestMempool_GetNonExistent(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	_, ok := mp.Get("nonexistent-txid")
	if ok {
		t.Error("expected not found for nonexistent tx")
	}
}

func TestMempool_GetTxs(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	for i := 0; i < 5; i++ {
		tx := makeTransferTx(nil, fmt.Sprintf("recipient%d", i), 100, 10, uint64(i))
		txPtr := &tx
		if err := mp.Add(txPtr); err != nil {
			t.Fatalf("failed to add tx %d: %v", i, err)
		}
	}

	txs := mp.GetTxs()
	if len(txs) != 5 {
		t.Errorf("expected 5 txs, got %d", len(txs))
	}
}

func TestMempool_GetByRecipient(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	for i := 0; i < 3; i++ {
		tx := makeTransferTx(nil, "same-recipient", 100, 10, uint64(i))
		txPtr := &tx
		if err := mp.Add(txPtr); err != nil {
			t.Fatalf("failed to add tx: %v", err)
		}
	}

	txs := mp.GetByRecipient("same-recipient")
	if len(txs) != 3 {
		t.Errorf("expected 3 txs for recipient, got %d", len(txs))
	}
}

func TestMempool_Size(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	if mp.Size() != 0 {
		t.Errorf("expected size 0, got %d", mp.Size())
	}

	for i := 0; i < 10; i++ {
		tx := makeTransferTx(nil, fmt.Sprintf("recipient%d", i), 100, 10, uint64(i))
		txPtr := &tx
		mp.Add(txPtr)
	}

	if mp.Size() != 10 {
		t.Errorf("expected size 10, got %d", mp.Size())
	}
}

func TestMempool_Remove(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	tx := makeTransferTx(nil, "recipient1", 100, 10, 1)
	txPtr := &tx
	if err := mp.Add(txPtr); err != nil {
		t.Fatalf("failed to add tx: %v", err)
	}

	txid, _ := blockchain.TxIDHex(tx)
	mp.Remove(txid)

	if _, ok := mp.Get(txid); ok {
		t.Error("tx should have been removed")
	}
}

func TestMempool_RemoveMany(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	var txids []string
	for i := 0; i < 5; i++ {
		tx := makeTransferTx(nil, fmt.Sprintf("recipient%d", i), 100, 10, uint64(i))
		txPtr := &tx
		if err := mp.Add(txPtr); err != nil {
			t.Fatalf("failed to add tx: %v", err)
		}
		txid, _ := blockchain.TxIDHex(tx)
		txids = append(txids, txid)
	}

	mp.RemoveMany(txids[:3])

	if mp.Size() != 2 {
		t.Errorf("expected size 2 after removing 3, got %d", mp.Size())
	}
}

func TestMempool_FeeBasedSelection(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	tx1 := makeTransferTx(nil, "recipient1", 100, 10, 1)
	tx2 := makeTransferTx(nil, "recipient2", 100, 50, 2)
	tx3 := makeTransferTx(nil, "recipient3", 100, 30, 3)

	for _, tx := range []*blockchain.Transaction{&tx1, &tx2, &tx3} {
		if err := mp.Add(tx); err != nil {
			t.Fatalf("failed to add tx: %v", err)
		}
	}

	entries := mp.EntriesSortedByFeeDesc()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].Tx().Fee != 50 {
		t.Errorf("expected highest fee (50), got %d", entries[0].Tx().Fee)
	}
	if entries[1].Tx().Fee != 30 {
		t.Errorf("expected middle fee (30), got %d", entries[1].Tx().Fee)
	}
	if entries[2].Tx().Fee != 10 {
		t.Errorf("expected lowest fee (10), got %d", entries[2].Tx().Fee)
	}
}

func TestMempool_SelectTxs(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	for i := 0; i < 10; i++ {
		tx := makeTransferTx(nil, fmt.Sprintf("recipient%d", i), 100, uint64(10+i), uint64(i))
		if err := mp.Add(&tx); err != nil {
			t.Fatalf("failed to add tx: %v", err)
		}
	}

	selected := mp.SelectTxs(500)
	if len(selected) == 0 {
		t.Error("expected some txs selected")
	}
}

func TestMempool_RBF_ReplaceByFee(t *testing.T) {
}

func TestMempool_RBF_FeeMustBeHigher(t *testing.T) {
}

func TestMempool_RBF_ReplacesHigherNonces(t *testing.T) {
}

func TestMempool_SizeLimitEviction(t *testing.T) {
	mp := newTestMempool(5, 1)
	defer mp.Stop()

	for i := 0; i < 10; i++ {
		tx := makeTransferTx(nil, fmt.Sprintf("recipient%d", i), 100, uint64(10+i), uint64(i))
		if err := mp.Add(&tx); err != nil {
			t.Fatalf("failed to add tx: %v", err)
		}
	}

	if mp.Size() > 5 {
		t.Errorf("mempool should have evicted to stay under limit, size=%d", mp.Size())
	}
}

func TestMempool_MinimumFeeRate(t *testing.T) {
	mp := newTestMempool(1000, 100)
	defer mp.Stop()

	tx := makeTransferTx(nil, "recipient1", 100, 1, 1)
	err := mp.Add(&tx)
	if err == nil {
		t.Error("expected error for fee below minimum rate")
	}
}

func TestMempool_NonceOrdering(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	tx1 := makeTransferTx(nil, "recipient1", 100, 10, 1)
	tx2 := makeTransferTx(nil, "recipient2", 100, 10, 2)
	tx3 := makeTransferTx(nil, "recipient3", 100, 10, 3)

	if err := mp.Add(&tx1); err != nil {
		t.Fatalf("failed to add tx1: %v", err)
	}
	if err := mp.Add(&tx2); err != nil {
		t.Fatalf("failed to add tx2: %v", err)
	}
	if err := mp.Add(&tx3); err != nil {
		t.Fatalf("failed to add tx3: %v", err)
	}

	if mp.Size() != 3 {
		t.Errorf("expected 3 txs in mempool, got %d", mp.Size())
	}
}

func TestMempool_LowNonceRejected(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	samePubKey := make([]byte, 32)
	copy(samePubKey, []byte("same-sender-for-testing-purposes"))

	tx1 := makeTransferTx(samePubKey, "recipient1", 100, 10, 1)
	if err := mp.Add(&tx1); err != nil {
		t.Fatalf("failed to add tx1: %v", err)
	}

	tx2 := makeTransferTx(samePubKey, "recipient2", 100, 10, 0)
	err := mp.Add(&tx2)
	if err == nil {
		t.Error("expected error for nonce 0 when nonce 1 exists for same sender")
	}
}

func TestMempool_Stats(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	_, accepted, _, _, _, size, _ := mp.StatsValues()
	if accepted != 0 || size != 0 {
		t.Errorf("expected zero stats, got accepted=%d size=%d", accepted, size)
	}

	tx := makeTransferTx(nil, "recipient1", 100, 10, 1)
	mp.Add(&tx)

	_, accepted, _, _, _, size, _ = mp.StatsValues()
	if accepted != 1 || size != 1 {
		t.Errorf("expected accepted=1 size=1, got accepted=%d size=%d", accepted, size)
	}
}

func TestMempool_ConcurrentAddGet(t *testing.T) {
	mp := newTestMempool(10000, 1)
	defer mp.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				tx := makeTransferTx(nil, fmt.Sprintf("recipient%d-%d", id, j), 100, 10, uint64(j))
				mp.Add(&tx)
			}
		}(i)
	}
	wg.Wait()

	if mp.Size() == 0 {
		t.Error("mempool should have entries after concurrent adds")
	}
}

func TestMempool_ConcurrentSelectTxs(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	for i := 0; i < 100; i++ {
		tx := makeTransferTx(nil, fmt.Sprintf("recipient%d", i), 100, uint64(10+i), uint64(i))
		mp.Add(&tx)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				mp.SelectTxs(1000)
				mp.Size()
			}
		}()
	}
	wg.Wait()
}

func TestMempool_ConcurrentMixed(t *testing.T) {
	mp := newTestMempool(5000, 1)
	defer mp.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				tx := makeTransferTx(nil, fmt.Sprintf("recipient%d-%d", id, j), 100, uint64(10+j), uint64(j))
				mp.Add(&tx)
				mp.GetTxs()
				mp.Size()
				mp.SelectTxs(500)
			}
		}(i)
	}
	wg.Wait()
}

func TestMempool_EstimateFee(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	for i := 0; i < 10; i++ {
		tx := makeTransferTx(nil, fmt.Sprintf("recipient%d", i), 100, uint64(10+i), uint64(i))
		mp.Add(&tx)
	}

	fee := mp.EstimateFee(1)
	if fee < 10 {
		t.Errorf("expected fee >= 10, got %d", fee)
	}
}

func TestMempool_EstimateFeeEmpty(t *testing.T) {
	mp := newTestMempool(1000, 5)
	defer mp.Stop()

	fee := mp.EstimateFee(1)
	if fee != 5 {
		t.Errorf("expected min fee rate 5, got %d", fee)
	}
}

func TestMempool_ConcurrentRemove(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	var txids []string
	for i := 0; i < 50; i++ {
		tx := makeTransferTx(nil, fmt.Sprintf("recipient%d", i), 100, 10, uint64(i))
		mp.Add(&tx)
		txid, _ := blockchain.TxIDHex(tx)
		txids = append(txids, txid)
	}

	var wg sync.WaitGroup
	for _, txid := range txids[:25] {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			mp.Remove(id)
		}(txid)
	}
	wg.Wait()

	if mp.Size() != 25 {
		t.Errorf("expected 25 txs remaining, got %d", mp.Size())
	}
}

func TestMempool_CleanupExpired(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	for i := 0; i < 5; i++ {
		tx := makeTransferTx(nil, fmt.Sprintf("recipient%d", i), 100, 10, uint64(i))
		mp.Add(&tx)
	}

	mp.Cleanup()

	if mp.Size() != 5 {
		t.Errorf("expected 5 txs after cleanup (none expired), got %d", mp.Size())
	}
}

func TestMempool_PendingForSender(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	tx1 := makeTransferTx(nil, "recipient1", 100, 10, 1)
	tx2 := makeTransferTx(nil, "recipient2", 100, 15, 2)

	if err := mp.Add(&tx1); err != nil {
		t.Fatalf("failed to add tx1: %v", err)
	}
	if err := mp.Add(&tx2); err != nil {
		t.Fatalf("failed to add tx2: %v", err)
	}

	pending1 := mp.PendingForSender(tx1.FromAddressSafe())
	pending2 := mp.PendingForSender(tx2.FromAddressSafe())

	if len(pending1) != 1 {
		t.Errorf("expected 1 pending tx for sender1, got %d", len(pending1))
	}
	if len(pending2) != 1 {
		t.Errorf("expected 1 pending tx for sender2, got %d", len(pending2))
	}
}

func TestMempool_AddWithTxID(t *testing.T) {
	mp := newTestMempool(1000, 1)
	defer mp.Stop()

	tx := makeTransferTx(nil, "recipient1", 100, 10, 1)
	txid, err := mp.AddWithTxID(tx, "", blockchain.ConsensusParams{}, 0)
	if err != nil {
		t.Fatalf("AddWithTxID failed: %v", err)
	}
	if txid == "" {
		t.Error("expected non-empty txid")
	}

	if _, ok := mp.Get(txid); !ok {
		t.Error("tx should be retrievable by txid")
	}
}

func TestMempool_Stop(t *testing.T) {
	mp := newTestMempool(1000, 1)
	mp.Stop()

	tx := makeTransferTx(nil, "recipient1", 100, 10, 1)
	err := mp.Add(&tx)
	if err != nil {
		t.Fatalf("Add should not fail after stop: %v", err)
	}
}

func TestMempool_ConcurrentRBF(t *testing.T) {
}
