package mempool

import (
	"container/heap"
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Neo4717/NeoCoin/config"
	"github.com/Neo4717/NeoCoin/internal/blockchain"
)

const (
	DefaultMaxSize    = 50000
	DefaultTTL        = 24 * time.Hour
	DefaultCleanupInt = 5 * time.Minute
	MaxTXAgeForProp   = 2 * time.Hour
)

type EntryMetrics struct {
	ReceivedAt time.Time
	SizeBytes  int
	FirstSeen  time.Time
	FailCount  int
	IsFavorite bool
}

type MempoolEntry struct {
	tx       *blockchain.Transaction
	txid     string
	feePerKB int64
	weight   int
	expiry   time.Time
	metrics  EntryMetrics
	listEl   *list.Element
}

type Mempool struct {
	byHash      map[string]*MempoolEntry
	bySender    map[string][]*MempoolEntry
	byRecipient map[string][]*MempoolEntry

	feeHeap  *FeeHeap
	ttlQueue *list.List

	maxSize    int
	ttl        time.Duration
	minFeeRate int64
	maxTXAge   time.Duration

	cleanupMu   sync.Mutex
	lastCleanup time.Time
	cleanupInt  time.Duration

	mu    sync.RWMutex
	stats MempoolStats

	validateWorkers int
	pendingTxs      chan *blockchain.Transaction
	validatedTxs    chan *MempoolEntry
	stopCh          chan struct{}
}

type MempoolStats struct {
	Received    atomic.Int64
	Accepted    atomic.Int64
	Rejected    atomic.Int64
	Evicted     atomic.Int64
	Expired     atomic.Int64
	CurrentSize atomic.Int64
	TotalFees   atomic.Int64
}

type FeeHeap []*MempoolEntry

func (h FeeHeap) Len() int           { return len(h) }
func (h FeeHeap) Less(i, j int) bool { return h[i].feePerKB > h[j].feePerKB }
func (h FeeHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *FeeHeap) Push(x any)        { *h = append(*h, x.(*MempoolEntry)) }
func (h *FeeHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func NewMempool(cfg *config.Config) *Mempool {
	maxSize := cfg.MempoolMaxSize
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}

	minFeeRate := cfg.MempoolMinFeeRate
	if minFeeRate <= 0 {
		minFeeRate = 1
	}

	mp := &Mempool{
		byHash:          make(map[string]*MempoolEntry, maxSize),
		bySender:        make(map[string][]*MempoolEntry),
		byRecipient:     make(map[string][]*MempoolEntry),
		feeHeap:         &FeeHeap{},
		ttlQueue:        list.New(),
		maxSize:         maxSize,
		ttl:             DefaultTTL,
		minFeeRate:      minFeeRate,
		maxTXAge:        MaxTXAgeForProp,
		cleanupInt:      DefaultCleanupInt,
		lastCleanup:     time.Now(),
		validateWorkers: 4,
		pendingTxs:      make(chan *blockchain.Transaction, 1000),
		validatedTxs:    make(chan *MempoolEntry, 1000),
		stopCh:          make(chan struct{}),
	}

	heap.Init(mp.feeHeap)

	for i := 0; i < mp.validateWorkers; i++ {
		go mp.validationWorker(i)
	}

	go mp.cleanupLoop()

	return mp
}

func (mp *Mempool) Add(tx *blockchain.Transaction) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	txid, err := blockchain.TxIDHex(*tx)
	if err != nil {
		return fmt.Errorf("compute txid: %w", err)
	}

	if _, exists := mp.byHash[txid]; exists {
		return fmt.Errorf("tx already in mempool")
	}

	if len(mp.byHash) >= mp.maxSize {
		mp.evictToMakeSpace(1)
	}

	txBytes := tx.EstimateSize()
	feePerKB := (int64(tx.Fee) * 1000) / int64(txBytes)

	if feePerKB < mp.minFeeRate {
		return fmt.Errorf("fee too low: %d < %d sat/KB", feePerKB, mp.minFeeRate)
	}

	fromAddr, err := tx.FromAddress()
	if err != nil {
		return fmt.Errorf("get sender address: %w", err)
	}

	entry := &MempoolEntry{
		tx:       tx,
		txid:     txid,
		feePerKB: feePerKB,
		weight:   txBytes,
		expiry:   time.Now().Add(mp.ttl),
		metrics: EntryMetrics{
			ReceivedAt: time.Now(),
			SizeBytes:  txBytes,
			FirstSeen:  time.Now(),
		},
	}

	if existing := mp.getBySender(fromAddr); len(existing) > 0 {
		highestNonce := existing[len(existing)-1].tx.Nonce
		if tx.Nonce <= highestNonce {
			return fmt.Errorf("nonce too low: %d <= %d", tx.Nonce, highestNonce)
		}
		mp.removeSenderEntries(fromAddr, tx.Nonce-1)
	}

	mp.byHash[txid] = entry
	mp.bySender[fromAddr] = append(mp.bySender[fromAddr], entry)
	mp.byRecipient[tx.ToAddress] = append(mp.byRecipient[tx.ToAddress], entry)

	heap.Push(mp.feeHeap, entry)
	entry.listEl = mp.ttlQueue.PushBack(entry)

	mp.stats.Accepted.Add(1)
	mp.stats.CurrentSize.Add(1)
	mp.stats.TotalFees.Add(int64(tx.Fee))

	return nil
}

func (mp *Mempool) AddAsync(tx *blockchain.Transaction) {
	select {
	case mp.pendingTxs <- tx:
	default:
	}
}

func (mp *Mempool) validationWorker(id int) {
	for {
		select {
		case tx := <-mp.pendingTxs:
			if err := mp.Add(tx); err != nil {
				mp.stats.Rejected.Add(1)
			}
		case <-mp.stopCh:
			return
		}
	}
}

func (mp *Mempool) cleanupLoop() {
	ticker := time.NewTicker(mp.cleanupInt)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mp.Cleanup()
		case <-mp.stopCh:
			return
		}
	}
}

func (mp *Mempool) Cleanup() {
	mp.cleanupMu.Lock()
	defer mp.cleanupMu.Unlock()

	now := time.Now()
	removed := 0

	for el := mp.ttlQueue.Front(); el != nil; {
		entry := el.Value.(*MempoolEntry)
		next := el.Next()

		if now.After(entry.expiry) || entry.metrics.FailCount > 3 {
			mp.removeEntry(entry)
			mp.stats.Expired.Add(1)
			removed++
		}

		el = next
	}

	mp.lastCleanup = now
}

func (mp *Mempool) removeEntry(entry *MempoolEntry) {
	delete(mp.byHash, entry.txid)
	mp.ttlQueue.Remove(entry.listEl)

	if entries, ok := mp.bySender[entry.tx.FromAddressSafe()]; ok {
		newEntries := make([]*MempoolEntry, 0, len(entries))
		for _, e := range entries {
			if e != entry {
				newEntries = append(newEntries, e)
			}
		}
		if len(newEntries) > 0 {
			mp.bySender[entry.tx.FromAddressSafe()] = newEntries
		} else {
			delete(mp.bySender, entry.tx.FromAddressSafe())
		}
	}

	if entries, ok := mp.byRecipient[entry.tx.ToAddress]; ok {
		newEntries := make([]*MempoolEntry, 0, len(entries))
		for _, e := range entries {
			if e != entry {
				newEntries = append(newEntries, e)
			}
		}
		if len(newEntries) > 0 {
			mp.byRecipient[entry.tx.ToAddress] = newEntries
		} else {
			delete(mp.byRecipient, entry.tx.ToAddress)
		}
	}

	mp.stats.CurrentSize.Add(-1)
	mp.stats.Evicted.Add(1)
}

func (mp *Mempool) removeSenderEntries(sender string, maxNonce uint64) {
	for txid, entry := range mp.byHash {
		if entry.tx.FromAddressSafe() == sender && entry.tx.Nonce <= maxNonce {
			mp.removeEntry(entry)
			delete(mp.byHash, txid)
		}
	}
}

func (mp *Mempool) getBySender(sender string) []*MempoolEntry {
	return mp.bySender[sender]
}

func (mp *Mempool) SelectTxs(maxWeight int) []*blockchain.Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	var selected []*blockchain.Transaction
	totalWeight := 0

	h := &FeeHeap{}
	*h = append(*h, *mp.feeHeap...)
	heap.Init(h)

	for h.Len() > 0 {
		entry := heap.Pop(h).(*MempoolEntry)

		if totalWeight+entry.weight > maxWeight {
			break
		}

		if time.Now().After(entry.expiry) {
			continue
		}

		selected = append(selected, entry.tx)
		totalWeight += entry.weight
	}

	return selected
}

func (mp *Mempool) Get(txid string) (*blockchain.Transaction, bool) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if entry, ok := mp.byHash[txid]; ok {
		return entry.tx, true
	}
	return nil, false
}

func (mp *Mempool) GetByRecipient(addr string) []*blockchain.Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	var txs []*blockchain.Transaction
	for _, entry := range mp.byRecipient[addr] {
		txs = append(txs, entry.tx)
	}
	return txs
}

func (mp *Mempool) GetTxs() []*blockchain.Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	txs := make([]*blockchain.Transaction, 0, len(mp.byHash))
	for _, entry := range mp.byHash {
		txs = append(txs, entry.tx)
	}
	return txs
}

func (mp *Mempool) Size() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return len(mp.byHash)
}

func (mp *Mempool) Stats() MempoolStats {
	return MempoolStats{
		Received:    atomic.Int64{},
		Accepted:    atomic.Int64{},
		Rejected:    atomic.Int64{},
		Evicted:     atomic.Int64{},
		Expired:     atomic.Int64{},
		CurrentSize: atomic.Int64{},
		TotalFees:   atomic.Int64{},
	}
}

func (mp *Mempool) StatsValues() (received, accepted, rejected, evicted, expired, currentSize, totalFees int64) {
	return mp.stats.Received.Load(),
		mp.stats.Accepted.Load(),
		mp.stats.Rejected.Load(),
		mp.stats.Evicted.Load(),
		mp.stats.Expired.Load(),
		mp.stats.CurrentSize.Load(),
		mp.stats.TotalFees.Load()
}

func (mp *Mempool) Remove(txid string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if entry, ok := mp.byHash[txid]; ok {
		mp.removeEntry(entry)
	}
}

func (mp *Mempool) RemoveMany(txids []string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	for _, txid := range txids {
		if entry, ok := mp.byHash[txid]; ok {
			mp.removeEntry(entry)
		}
	}
}

func (mp *Mempool) evictToMakeSpace(n int) {
	target := len(mp.byHash) + n - mp.maxSize
	if target <= 0 {
		return
	}

	evicted := 0
	for evicted < target && mp.feeHeap.Len() > 0 {
		entry := heap.Pop(mp.feeHeap).(*MempoolEntry)
		mp.removeEntry(entry)
		evicted++
	}
}

func (mp *Mempool) EstimateFee(blocks int) int64 {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if mp.feeHeap.Len() == 0 {
		return mp.minFeeRate
	}

	percentile := float64(blocks) / 6.0
	if percentile > 1.0 {
		percentile = 1.0
	}

	idx := int(float64(len(*mp.feeHeap)) * percentile)
	if idx >= len(*mp.feeHeap) {
		idx = len(*mp.feeHeap) - 1
	}

	if idx < 0 {
		return mp.minFeeRate
	}

	return (*mp.feeHeap)[idx].feePerKB
}

func (mp *Mempool) Stop() {
	close(mp.stopCh)
}

func (e MempoolEntry) Tx() blockchain.Transaction {
	return *e.tx
}

func (e *MempoolEntry) TxIDHex() string {
	return e.txid
}

func (e MempoolEntry) FromAddress() (string, error) {
	return e.tx.FromAddress()
}

func (e MempoolEntry) FromAddressSafe() string {
	return e.tx.FromAddressSafe()
}

func (mp *Mempool) PendingForSender(fromAddr string) []MempoolEntry {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	var out []MempoolEntry
	for _, e := range mp.bySender[fromAddr] {
		out = append(out, *e)
	}
	return out
}

func (mp *Mempool) EntriesSortedByFeeDesc() []MempoolEntry {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	entries := make([]MempoolEntry, 0, len(mp.byHash))
	for _, e := range mp.byHash {
		entries = append(entries, *e)
	}

	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Tx().Fee > entries[i].Tx().Fee {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	return entries
}

func (mp *Mempool) AddWithTxID(tx blockchain.Transaction, txid string, p blockchain.ConsensusParams, height uint64) (string, error) {
	txPtr := &tx
	if err := mp.Add(txPtr); err != nil {
		return "", err
	}
	txidOut, _ := blockchain.TxIDHex(tx)
	return txidOut, nil
}

func (mp *Mempool) ReplaceByFeeWithTxID(tx blockchain.Transaction, txid string, p blockchain.ConsensusParams, height uint64) (string, bool, []string, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	fromAddr, err := tx.FromAddress()
	if err != nil {
		return "", false, nil, err
	}

	txPtr := &tx

	existingTxid, exists := mp.getTxidBySenderNonce(fromAddr, tx.Nonce)
	if !exists {
		return "", false, nil, fmt.Errorf("no existing tx with nonce %d", tx.Nonce)
	}

	existingEntry := mp.byHash[existingTxid]
	if tx.Fee <= existingEntry.tx.Fee {
		return "", false, nil, fmt.Errorf("replacement fee must be higher")
	}

	var evicted []string
	evicted = append(evicted, existingTxid)
	mp.removeEntry(existingEntry)

	for txid2, entry := range mp.byHash {
		if entry.tx.FromAddressSafe() == fromAddr && entry.tx.Nonce > tx.Nonce {
			evicted = append(evicted, txid2)
			mp.removeEntry(entry)
		}
	}

	if _, exists := mp.byHash[txid]; exists {
		return "", false, evicted, fmt.Errorf("tx already in mempool")
	}

	if len(mp.byHash) >= mp.maxSize {
		mp.evictToMakeSpace(1)
	}

	txBytes := tx.EstimateSize()
	feePerKB := (int64(tx.Fee) * 1000) / int64(txBytes)

	if feePerKB < mp.minFeeRate {
		return "", false, evicted, fmt.Errorf("fee too low: %d < %d sat/KB", feePerKB, mp.minFeeRate)
	}

	entry := &MempoolEntry{
		tx:       txPtr,
		txid:     txid,
		feePerKB: feePerKB,
		weight:   txBytes,
		expiry:   time.Now().Add(mp.ttl),
		metrics: EntryMetrics{
			ReceivedAt: time.Now(),
			SizeBytes:  txBytes,
			FirstSeen:  time.Now(),
		},
	}

	mp.byHash[txid] = entry
	mp.bySender[fromAddr] = append(mp.bySender[fromAddr], entry)
	mp.byRecipient[tx.ToAddress] = append(mp.byRecipient[tx.ToAddress], entry)

	heap.Push(mp.feeHeap, entry)
	entry.listEl = mp.ttlQueue.PushBack(entry)

	mp.stats.Accepted.Add(1)
	mp.stats.CurrentSize.Add(1)
	mp.stats.TotalFees.Add(int64(tx.Fee))

	return txid, true, evicted, nil
}

func (mp *Mempool) getTxidBySenderNonce(sender string, nonce uint64) (string, bool) {
	for _, entry := range mp.bySender[sender] {
		if entry.tx.Nonce == nonce {
			return entry.txid, true
		}
	}
	return "", false
}
