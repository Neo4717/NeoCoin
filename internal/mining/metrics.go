package mining

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Neo4717/NeoCoin/internal/consensus"
)

type Metrics struct {
	mu sync.RWMutex

	totalShares    uint64
	acceptedShares uint64
	rejectedShares uint64

	currentHashrate float64

	totalRevenue  uint64
	revenueCount  uint64
	lastRevenueAt time.Time

	currentDifficulty uint32

	startTime time.Time
}

func NewMiningMetrics() *Metrics {
	return &Metrics{
		startTime:         time.Now(),
		currentDifficulty: 1,
	}
}

func (m *Metrics) RecordShare(accepted bool) {
	atomic.AddUint64(&m.totalShares, 1)
	if accepted {
		atomic.AddUint64(&m.acceptedShares, 1)
	} else {
		atomic.AddUint64(&m.rejectedShares, 1)
	}
}

func (m *Metrics) UpdateHashrate(hashrate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentHashrate = hashrate
}

func (m *Metrics) RecordRevenue(amount uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalRevenue += amount
	m.revenueCount++
	m.lastRevenueAt = time.Now()
}

func (m *Metrics) UpdateDifficulty(bits uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentDifficulty = bits
}

func (m *Metrics) GetHashrate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentHashrate
}

func (m *Metrics) GetTotalShares() uint64 {
	return atomic.LoadUint64(&m.totalShares)
}

func (m *Metrics) GetAcceptedShares() uint64 {
	return atomic.LoadUint64(&m.acceptedShares)
}

func (m *Metrics) GetRejectedShares() uint64 {
	return atomic.LoadUint64(&m.rejectedShares)
}

func (m *Metrics) GetTotalRevenue() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalRevenue
}

func (m *Metrics) GetDifficulty() uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentDifficulty
}

func (m *Metrics) GetShareAcceptanceRate() float64 {
	total := atomic.LoadUint64(&m.totalShares)
	if total == 0 {
		return 0
	}
	accepted := atomic.LoadUint64(&m.acceptedShares)
	return float64(accepted) / float64(total)
}

func (m *Metrics) GetUptime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.startTime)
}

func (m *Metrics) GetAverageRevenue() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.revenueCount == 0 {
		return 0
	}
	return m.totalRevenue / m.revenueCount
}

func (m *Metrics) GetEstimatedDailyRevenue() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentHashrate == 0 || m.currentDifficulty == 0 {
		return 0
	}

	blockTime := 600.0
	expectedBlocksPerDay := 86400 / blockTime
	blockReward := consensus.BlockReward(0)

	expectedDaily := uint64(float64(expectedBlocksPerDay) * float64(blockReward))

	return expectedDaily
}

func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"hashrate":              m.currentHashrate,
		"totalShares":           atomic.LoadUint64(&m.totalShares),
		"acceptedShares":        atomic.LoadUint64(&m.acceptedShares),
		"rejectedShares":        atomic.LoadUint64(&m.rejectedShares),
		"shareAcceptanceRate":   m.GetShareAcceptanceRate(),
		"totalRevenue":          m.totalRevenue,
		"averageRevenue":        m.GetAverageRevenue(),
		"estimatedDailyRevenue": m.GetEstimatedDailyRevenue(),
		"difficulty":            m.currentDifficulty,
		"uptime":                time.Since(m.startTime).String(),
	}
}
