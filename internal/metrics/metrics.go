package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	BlockHeight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "neocoin_chain_height",
		Help: "Current blockchain height",
	})
	TotalSupply = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "neocoin_total_supply",
		Help: "Total token supply",
	})
	Difficulty = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "neocoin_difficulty",
		Help: "Current mining difficulty",
	})

	BlocksMined = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_blocks_mined_total",
		Help: "Total blocks mined",
	})
	BlockInterval = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "neocoin_block_interval_seconds",
		Help:    "Time between blocks",
		Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1200},
	})
	LastBlockTime = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "neocoin_last_block_timestamp",
		Help: "Timestamp of last mined block",
	})

	MempoolSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "neocoin_mempool_size",
		Help: "Number of transactions in mempool",
	})
	MempoolBytes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "neocoin_mempool_bytes",
		Help: "Size of mempool in bytes",
	})
	MempoolTxReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_mempool_tx_received_total",
		Help: "Total transactions received",
	})
	MempoolTxAccepted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_mempool_tx_accepted_total",
		Help: "Transactions accepted into mempool",
	})
	MempoolTxRejected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_mempool_tx_rejected_total",
		Help: "Transactions rejected from mempool",
	})
	MempoolTxExpired = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_mempool_tx_expired_total",
		Help: "Transactions expired from mempool",
	})

	PeerCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "neocoin_peer_count",
		Help: "Number of connected peers",
	})
	PeerConnectionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_peer_connection_errors_total",
		Help: "Total peer connection errors",
	})
	BytesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_p2p_bytes_sent_total",
		Help: "Total bytes sent to peers",
	})
	BytesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_p2p_bytes_received_total",
		Help: "Total bytes received from peers",
	})
	BlocksPropagated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_blocks_propagated_total",
		Help: "Total blocks propagated to peers",
	})
	TxsPropagated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_txs_propagated_total",
		Help: "Total transactions propagated to peers",
	})

	Hashrate = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "neocoin_mining_hashrate",
		Help: "Current mining hashrate (hashes/second)",
	})
	MiningAttempts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_mining_attempts_total",
		Help: "Total PoW attempts",
	})
	SharesSubmitted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_mining_shares_total",
		Help: "Total shares submitted",
	})
	SharesAccepted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_mining_shares_accepted_total",
		Help: "Shares accepted by network",
	})

	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "neocoin_http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "endpoint", "status"})
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "neocoin_http_request_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "endpoint"})
	WSConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "neocoin_websocket_connections",
		Help: "Active WebSocket connections",
	})

	CacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "neocoin_cache_hits_total",
		Help: "Cache hits by type",
	}, []string{"type"})
	CacheMisses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "neocoin_cache_misses_total",
		Help: "Cache misses by type",
	}, []string{"type"})

	DBSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "neocoin_db_size_bytes",
		Help: "Database size in bytes",
	})
	DBWrites = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_db_writes_total",
		Help: "Total database writes",
	})
	DBReads = promauto.NewCounter(prometheus.CounterOpts{
		Name: "neocoin_db_reads_total",
		Help: "Total database reads",
	})
)
