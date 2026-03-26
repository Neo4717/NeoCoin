package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type MetricsCollector struct {
	mu            sync.RWMutex
	startTime     time.Time
	blockCount    uint64
	txCount       uint64
	totalGasUsed  uint64
	peerCount     int
	avgBlockTime  time.Duration
	lastBlockTime time.Time
	mempoolSize   int
	chainHeight   uint64
	difficulty    uint32
	errors        uint64
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime:     time.Now(),
		avgBlockTime:  time.Minute,
		lastBlockTime: time.Now(),
	}
}

func (m *MetricsCollector) RecordBlock(height uint64, txCount int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blockCount++
	m.txCount += uint64(txCount)
	m.chainHeight = height
	m.avgBlockTime = (m.avgBlockTime + duration) / 2
	m.lastBlockTime = time.Now()
}

func (m *MetricsCollector) RecordTransaction(gasUsed uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.txCount++
	m.totalGasUsed += gasUsed
}

func (m *MetricsCollector) RecordPeerChange(delta int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.peerCount += delta
	if m.peerCount < 0 {
		m.peerCount = 0
	}
}

func (m *MetricsCollector) RecordMempoolSize(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mempoolSize = size
}

func (m *MetricsCollector) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors++
}

func (m *MetricsCollector) GetMetrics() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.startTime)

	return map[string]any{
		"uptime_seconds":     uptime.Seconds(),
		"block_height":       m.chainHeight,
		"total_blocks":       m.blockCount,
		"total_transactions": m.txCount,
		"total_gas_used":     m.totalGasUsed,
		"peers_connected":    m.peerCount,
		"mempool_size":       m.mempoolSize,
		"avg_block_time_ms":  m.avgBlockTime.Milliseconds(),
		"difficulty_bits":    m.difficulty,
		"tps":                float64(m.txCount) / uptime.Seconds(),
		"errors_count":       m.errors,
		"timestamp":          time.Now().Unix(),
	}
}

func (m *MetricsCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path == "/metrics" {
		json.NewEncoder(w).Encode(m.GetMetrics())
		return
	}

	if r.URL.Path == "/health" {
		m.mu.RLock()
		healthy := m.errors < 100
		m.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "healthy",
			"healthy":   healthy,
			"timestamp": time.Now().Unix(),
		})
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func (m *MetricsCollector) PrometheusMetrics() string {
	metrics := m.GetMetrics()

	output := `# HELP neocoin_uptime_seconds Node uptime in seconds
# TYPE neocoin_uptime_seconds gauge
neocoin_uptime_seconds %.2f

# HELP neocoin_block_height Current block height
# TYPE neocoin_block_height gauge
neocoin_block_height %d

# HELP neocoin_total_transactions Total transactions processed
# TYPE neocoin_total_transactions counter
neocoin_total_transactions %d

# HELP neocoin_peers_connected Number of connected peers
# TYPE neocoin_peers_connected gauge
neocoin_peers_connected %d

# HELP neocoin_mempool_size Current mempool size
# TYPE neocoin_mempool_size gauge
neocoin_mempool_size %d

# HELP neocoin_tps Transactions per second
# TYPE neocoin_tps gauge
neocoin_tps %.2f

# HELP neocoin_errors_total Total error count
# TYPE neocoin_errors_total counter
neocoin_errors_total %d
`
	return fmt.Sprintf(output,
		metrics["uptime_seconds"],
		metrics["block_height"],
		metrics["total_transactions"],
		metrics["peers_connected"],
		metrics["mempool_size"],
		metrics["tps"],
		metrics["errors_count"],
	)
}
