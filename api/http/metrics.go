package http

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Metrics struct {
	bc *blockchainMetrics
	mp *mempoolMetrics

	mu sync.Mutex

	httpRequestsTotal map[string]map[int]int64
	httpDurations     map[string]*histogram
}

type blockchainMetrics struct {
	bc interface {
		LatestBlock() interface{ Height() uint64 }
		TotalSupply() uint64
	}
}

type mempoolMetrics struct {
	mp interface {
		Size() int
	}
}

type histogram struct {
	buckets []float64
	counts  []int64
	sum     float64
	count   int64
}

func newHistogram() *histogram {
	b := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	return &histogram{buckets: b, counts: make([]int64, len(b))}
}

func (h *histogram) observe(seconds float64) {
	h.sum += seconds
	h.count++
	for i, le := range h.buckets {
		if seconds <= le {
			h.counts[i]++
		}
	}
}

func NewMetrics() *Metrics {
	return &Metrics{
		httpRequestsTotal: map[string]map[int]int64{},
		httpDurations:     map[string]*histogram{},
	}
}

func (m *Metrics) ObserveHTTP(route string, status int, dur time.Duration) {
	if m == nil || route == "" {
		return
	}
	seconds := dur.Seconds()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.httpRequestsTotal[route] == nil {
		m.httpRequestsTotal[route] = map[int]int64{}
	}
	m.httpRequestsTotal[route][status]++
	h := m.httpDurations[route]
	if h == nil {
		h = newHistogram()
		m.httpDurations[route] = h
	}
	h.observe(seconds)
}

func (m *Metrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var lines []string
	lines = append(lines, "# HELP http_requests_total Total HTTP requests.")
	lines = append(lines, "# TYPE http_requests_total counter")
	for route, byStatus := range m.httpRequestsTotal {
		for status, count := range byStatus {
			lines = append(lines, fmt.Sprintf("http_requests_total{route=%q,status=%d} %d", route, status, count))
		}
	}

	lines = append(lines, "# HELP http_request_duration_seconds HTTP request duration.")
	lines = append(lines, "# TYPE http_request_duration_seconds histogram")
	for route, h := range m.httpDurations {
		for i, le := range h.buckets {
			lines = append(lines, fmt.Sprintf("http_request_duration_seconds_bucket{route=%q,le=%g} %d", route, le, h.counts[i]))
		}
		lines = append(lines, fmt.Sprintf("http_request_duration_seconds_bucket{route=%q,le=\"+Inf\"} %d", route, h.count))
		lines = append(lines, fmt.Sprintf("http_request_duration_seconds_sum{route=%q} %.6f", route, h.sum))
		lines = append(lines, fmt.Sprintf("http_request_duration_seconds_count{route=%q} %d", route, h.count))
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
}
