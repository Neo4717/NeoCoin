package blockchain

import (
	"errors"
	"sort"
	"sync"
	"time"
)

type mempoolEntry struct {
	tx       Transaction
	txIDHex  string
	received time.Time
}

func (e mempoolEntry) Tx() Transaction              { return e.tx }
func (e mempoolEntry) TxIDHex() string              { return e.txIDHex }
func (e mempoolEntry) FromAddress() (string, error) { return e.tx.FromAddress() }

type Mempool struct {
	mu sync.Mutex

	maxSize int

	entries       map[string]mempoolEntry
	bySenderNonce map[string]map[uint64]string
}

func NewMempool(maxSize int) *Mempool {
	if maxSize <= 0 {
		maxSize = 10_000
	}
	return &Mempool{
		maxSize:       maxSize,
		entries:       map[string]mempoolEntry{},
		bySenderNonce: map[string]map[uint64]string{},
	}
}

func (m *Mempool) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}

func (m *Mempool) Add(tx Transaction) (string, error) {
	txid, err := TxIDHex(tx)
	if err != nil {
		return "", err
	}
	fromAddr, err := tx.FromAddress()
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.entries[txid]; ok {
		return txid, errors.New("duplicate transaction")
	}
	if existingID, ok := m.bySenderNonce[fromAddr][tx.Nonce]; ok {
		return existingID, errors.New("nonce already in mempool")
	}
	if len(m.entries) >= m.maxSize {
		lowest := m.lowestFeeLocked()
		if lowest == "" {
			return "", errors.New("mempool full")
		}
		m.evictWithDependentsLocked(lowest)
	}

	m.entries[txid] = mempoolEntry{tx: tx, txIDHex: txid, received: time.Now()}
	m.indexLocked(fromAddr, tx.Nonce, txid)
	return txid, nil
}

func (m *Mempool) Remove(txid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeLocked(txid)
}

func (m *Mempool) RemoveMany(txids []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range txids {
		m.removeLocked(id)
	}
}

func (m *Mempool) Snapshot() []mempoolEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mempoolEntry, 0, len(m.entries))
	for _, e := range m.entries {
		out = append(out, e)
	}
	return out
}

func (m *Mempool) EntriesSortedByFeeDesc() []mempoolEntry {
	entries := m.Snapshot()
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].tx.Fee != entries[j].tx.Fee {
			return entries[i].tx.Fee > entries[j].tx.Fee
		}
		return entries[i].received.Before(entries[j].received)
	})
	return entries
}

func (m *Mempool) PendingForSender(fromAddr string) []mempoolEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []mempoolEntry
	for nonce, txid := range m.bySenderNonce[fromAddr] {
		_ = nonce
		if e, ok := m.entries[txid]; ok {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tx.Nonce < out[j].tx.Nonce })
	return out
}

func (m *Mempool) lowestFeeLocked() string {
	lowestFee := uint64(^uint64(0))
	lowestID := ""
	for id, e := range m.entries {
		if e.tx.Fee < lowestFee {
			lowestFee = e.tx.Fee
			lowestID = id
		}
	}
	return lowestID
}

func (m *Mempool) evictWithDependentsLocked(txid string) {
	victim, ok := m.entries[txid]
	if !ok {
		return
	}
	m.removeLocked(txid)

	from, err := victim.tx.FromAddress()
	if err != nil {
		return
	}
	victimNonce := victim.tx.Nonce
	for id, e := range m.entries {
		addr, err := e.tx.FromAddress()
		if err != nil {
			continue
		}
		if addr == from && e.tx.Nonce > victimNonce {
			m.removeLocked(id)
		}
	}
}

func (m *Mempool) indexLocked(fromAddr string, nonce uint64, txid string) {
	if m.bySenderNonce[fromAddr] == nil {
		m.bySenderNonce[fromAddr] = map[uint64]string{}
	}
	m.bySenderNonce[fromAddr][nonce] = txid
}

func (m *Mempool) removeLocked(txid string) {
	e, ok := m.entries[txid]
	if !ok {
		return
	}
	delete(m.entries, txid)
	from, err := e.tx.FromAddress()
	if err != nil {
		return
	}
	if m.bySenderNonce[from] != nil {
		if m.bySenderNonce[from][e.tx.Nonce] == txid {
			delete(m.bySenderNonce[from], e.tx.Nonce)
		}
		if len(m.bySenderNonce[from]) == 0 {
			delete(m.bySenderNonce, from)
		}
	}
}
