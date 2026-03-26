package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/Neo4717/NeoCoin/internal/blockchain"
	"github.com/Neo4717/NeoCoin/internal/mempool"
)

type Collector struct {
	chain    *blockchain.Blockchain
	mempool  *mempool.Mempool
	pm       interface{}
	interval time.Duration
	stopCh   chan struct{}
}

func NewCollector(chain *blockchain.Blockchain, mempool *mempool.Mempool, interval time.Duration) *Collector {
	return &Collector{
		chain:    chain,
		mempool:  mempool,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (c *Collector) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.collect()
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		}
	}
}

func (c *Collector) collect() {
	if c.chain != nil {
		if tip := c.chain.LatestBlock(); tip != nil {
			BlockHeight.Set(float64(tip.Height))
			Difficulty.Set(float64(tip.DifficultyBits))
			LastBlockTime.Set(float64(tip.TimestampUnix))
		}
		TotalSupply.Set(float64(c.chain.TotalSupply()))
	}

	if c.mempool != nil {
		MempoolSize.Set(float64(c.mempool.Size()))
	}
}

func (c *Collector) Stop() {
	close(c.stopCh)
}

var lastBlockTime int64
var muLastBlockTime sync.Mutex

func IncrementBlockMined() {
	BlocksMined.Inc()
}

func RecordBlockInterval(prevBlockTime int64) {
	muLastBlockTime.Lock()
	defer muLastBlockTime.Unlock()
	now := time.Now().Unix()
	if lastBlockTime > 0 {
		interval := now - lastBlockTime
		BlockInterval.Observe(float64(interval))
	}
	lastBlockTime = now
}

func SetLastBlockTime(t int64) {
	muLastBlockTime.Lock()
	defer muLastBlockTime.Unlock()
	lastBlockTime = t
}
