package mining

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type TxEntry interface {
	TxIDHex() string
}

type Miner struct {
	bc interface {
		SelectMempoolTxs(entries []TxEntry, maxTx int) ([]interface{}, [][]byte, error)
		MineTransfers(txs []interface{}) (*interface{}, error)
	}
	mp interface {
		RemoveMany(ids [][]byte)
		EntriesSortedByFeeDesc() []TxEntry
		Size() int
	}

	maxTxPerBlock    int
	forceEmptyBlocks bool

	events interface {
		Publish(interface{})
	}

	stratumServer  *StratumServer
	stratumAddr    string
	stratumEnabled bool

	templateCallbacks []TemplateCallback

	mu      sync.Mutex
	wakeCh  chan struct{}
	stopped chan struct{}
}

type TemplateCallback func(*StratumJob)

func NewMiner(bc interface {
	SelectMempoolTxs(entries []TxEntry, maxTx int) ([]interface{}, [][]byte, error)
	MineTransfers(txs []interface{}) (*interface{}, error)
}, mp interface {
	RemoveMany(ids [][]byte)
	EntriesSortedByFeeDesc() []TxEntry
	Size() int
}, maxTxPerBlock int, forceEmptyBlocks bool, stratumEnabled bool, stratumAddr string) *Miner {
	if maxTxPerBlock <= 0 {
		maxTxPerBlock = 100
	}
	m := &Miner{
		bc:               bc,
		mp:               mp,
		maxTxPerBlock:    maxTxPerBlock,
		forceEmptyBlocks: forceEmptyBlocks,
		wakeCh:           make(chan struct{}, 1),
		stopped:          make(chan struct{}),
		stratumEnabled:   stratumEnabled,
		stratumAddr:      stratumAddr,
	}

	if m.stratumEnabled {
		m.stratumServer = NewStratumServer(m.stratumAddr, m)
	}

	return m
}

func (m *Miner) SetEventSink(sink interface{ Publish(interface{}) }) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = sink
}

func (m *Miner) OnNewTemplate(cb TemplateCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.templateCallbacks = append(m.templateCallbacks, cb)
}

func (m *Miner) notifyNewTemplate(job *StratumJob) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, cb := range m.templateCallbacks {
		cb(job)
	}
}

func (m *Miner) NewTemplate() {
	if m.stratumServer != nil {
		job := &StratumJob{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			PrevHash:  "0000000000000000000000000000000000000000000000000000000000000000",
			Coinbase1: "",
			Coinbase2: "",
			Version:   "20000000",
			NBits:     "1d00ffff",
			NTime:     fmt.Sprintf("%08x", uint64(time.Now().Unix())),
			CleanJobs: true,
		}
		m.stratumServer.BroadcastJob(job)
		m.notifyNewTemplate(job)
	}
}

func (m *Miner) Wake() {
	select {
	case m.wakeCh <- struct{}{}:
	default:
	}
}

func (m *Miner) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 1 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	defer close(m.stopped)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_, _ = m.MineOnce(ctx, false)
		case <-m.wakeCh:
			_, _ = m.MineOnce(ctx, false)
		}
	}
}

func (m *Miner) MineOnce(ctx context.Context, force bool) (interface{}, error) {
	if m.mp == nil {
		return nil, nil
	}

	entries := m.mp.EntriesSortedByFeeDesc()

	selected, selectedIDs, err := m.bc.SelectMempoolTxs(entries, m.maxTxPerBlock)
	if err != nil {
		return nil, err
	}

	mineEmpty := force || m.forceEmptyBlocks
	if len(selected) == 0 && !mineEmpty {
		return nil, nil
	}

	b, err := m.bc.MineTransfers(selected)
	if err != nil {
		return nil, err
	}
	if len(selectedIDs) > 0 {
		m.mp.RemoveMany(selectedIDs)
	}
	return b, nil
}

func (m *Miner) Start(ctx context.Context) error {
	if m.stratumEnabled && m.stratumServer != nil {
		if err := m.stratumServer.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *Miner) Stop() {
	if m.stratumServer != nil {
		m.stratumServer.Stop()
	}
}

func (m *Miner) SubmitShare(jobID, extraNonce2 string, nonce, ntime uint64) (bool, error) {
	return true, nil
}
