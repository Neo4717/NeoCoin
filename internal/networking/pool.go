package networking

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Neo4717/NeoCoin/config"
)

type PooledConnection struct {
	ID       uint64
	Conn     *Peer
	InUse    atomic.Bool
	LastUsed time.Time
	latency  atomic.Int64
	failures atomic.Int32
}

type PoolManager struct {
	maxConns     int
	maxConnsPeer int
	idleTimeout  time.Duration
	maxLatency   int64

	mu       sync.RWMutex
	pools    map[string][]*PooledConnection
	allConns map[uint64]*PooledConnection

	nextID atomic.Uint64
	stats  PoolStats

	ctx    context.Context
	cancel context.CancelFunc
}

type PoolStats struct {
	Acquired   atomic.Int64
	Released   atomic.Int64
	Created    atomic.Int64
	Closed     atomic.Int64
	LatencyAvg atomic.Int64
	Reused     atomic.Int64
}

const DefaultMaxPoolConns = 100
const DefaultMaxConnsPerPeer = 3
const DefaultIdleTimeout = 5 * time.Minute

func NewPoolManager(cfg *config.Config) *PoolManager {
	maxConns := DefaultMaxPoolConns
	if cfg != nil && cfg.MaxPoolConns > 0 {
		maxConns = cfg.MaxPoolConns
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool := &PoolManager{
		maxConns:     maxConns,
		maxConnsPeer: DefaultMaxConnsPerPeer,
		idleTimeout:  DefaultIdleTimeout,
		pools:        make(map[string][]*PooledConnection),
		allConns:     make(map[uint64]*PooledConnection),
		ctx:          ctx,
		cancel:       cancel,
	}

	go pool.cleanupLoop()

	return pool
}

func (p *PoolManager) Acquire(addr string, createConn func() (*Peer, error)) (*PooledConnection, error) {
	p.mu.Lock()

	if conns, ok := p.pools[addr]; ok {
		for i := len(conns) - 1; i >= 0; i-- {
			conn := conns[i]
			if !conn.InUse.Load() && conn.failures.Load() < 3 {
				if conn.latency.Load() < p.maxLatency || p.maxLatency == 0 {
					conn.InUse.Store(true)
					conn.LastUsed = time.Now()
					p.mu.Unlock()
					p.stats.Reused.Add(1)
					p.stats.Acquired.Add(1)
					return conn, nil
				}
			}
		}
	}

	if len(p.allConns) >= p.maxConns {
		p.closeIdleConnections(1)
	}

	p.mu.Unlock()

	peer, err := createConn()
	if err != nil {
		return nil, fmt.Errorf("create conn: %w", err)
	}

	pooled := &PooledConnection{
		ID:    p.nextID.Add(1),
		Conn:  peer,
		InUse: atomic.Bool{},
	}
	pooled.InUse.Store(true)
	pooled.LastUsed = time.Now()

	p.mu.Lock()
	p.pools[addr] = append(p.pools[addr], pooled)
	p.allConns[pooled.ID] = pooled
	p.mu.Unlock()

	p.stats.Created.Add(1)
	p.stats.Acquired.Add(1)

	return pooled, nil
}

func (p *PoolManager) Release(conn *PooledConnection) {
	conn.InUse.Store(false)
	conn.LastUsed = time.Now()
	p.stats.Released.Add(1)
}

func (p *PoolManager) Close() {
	p.cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.allConns {
		if conn.Conn != nil {
			conn.Conn.Close()
		}
	}
	p.allConns = make(map[uint64]*PooledConnection)
	p.pools = make(map[string][]*PooledConnection)
}

func (p *PoolManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			p.closeIdleConnections(p.maxConns / 10)
			p.mu.Unlock()
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *PoolManager) closeIdleConnections(n int) {
	count := 0
	for addr, conns := range p.pools {
		for i := len(conns) - 1; i >= 0; i-- {
			conn := conns[i]
			if !conn.InUse.Load() && time.Since(conn.LastUsed) > p.idleTimeout {
				if conn.Conn != nil {
					conn.Conn.Close()
				}
				delete(p.allConns, conn.ID)
				conns = append(conns[:i], conns[i+1:]...)
				count++
				p.stats.Closed.Add(1)
			}
		}
		if len(conns) == 0 {
			delete(p.pools, addr)
		} else {
			p.pools[addr] = conns
		}
		if count >= n {
			break
		}
	}
}

func (p *PoolManager) Stats() PoolStats {
	return PoolStats{
		Acquired: p.stats.Acquired,
		Released: p.stats.Released,
		Created:  p.stats.Created,
		Closed:   p.stats.Closed,
		Reused:   p.stats.Reused,
	}
}

func (c *PooledConnection) MeasureLatency() int64 {
	if c.Conn == nil {
		return 0
	}

	start := time.Now()
	err := c.Conn.SendPing()
	if err != nil {
		c.failures.Add(1)
		return 0
	}

	latency := time.Since(start).Milliseconds()
	c.latency.Store(latency)
	return latency
}
