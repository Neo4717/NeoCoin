package http

import (
	"net/http"
	"time"

	"github.com/Neo4717/NeoCoin/api/websocket"
	"github.com/Neo4717/NeoCoin/internal/airdrop"
	"github.com/Neo4717/NeoCoin/internal/blockchain"
	"github.com/Neo4717/NeoCoin/internal/mempool"
)

type Server struct {
	bc          *blockchain.Blockchain
	aiAuditor   string
	requireAI   bool
	httpTimeout time.Duration

	mp       *mempool.Mempool
	txGossip bool

	wsEnable bool
	wsHub    *websocket.Hub

	adminToken string
	trustProxy bool
	limiter    *IPRateLimiter
	metrics    *Metrics

	peerManager interface {
		Peers() []string
		AddPeer(addr string)
	}

	airdroper *airdrop.Airdrop
}

func NewServer(bc *blockchain.Blockchain, aiAuditorURL string, mp *mempool.Mempool, adminToken string, limiter *IPRateLimiter, trustProxy bool, wsEnable bool, wsHub *websocket.Hub) *Server {
	return &Server{
		bc:          bc,
		aiAuditor:   aiAuditorURL,
		requireAI:   aiAuditorURL != "",
		httpTimeout: 5 * time.Second,
		mp:          mp,
		wsEnable:    wsEnable,
		wsHub:       wsHub,
		adminToken:  adminToken,
		limiter:     limiter,
		trustProxy:  trustProxy,
		metrics:     NewMetrics(),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mw := &RouteMiddleware{
		adminToken: s.adminToken,
		trustProxy: s.trustProxy,
		limiter:    s.limiter,
		metrics:    s.metrics,
	}

	mux.HandleFunc("/health", mw.Wrap("health", false, 0, s.handleHealth))
	mux.HandleFunc("/metrics", mw.Wrap("metrics", false, 0, s.metrics.ServeHTTP))
	mux.HandleFunc("/node/metrics", mw.Wrap("node_metrics", false, 0, s.handlePrometheusMetrics))
	if s.wsEnable && s.wsHub != nil {
		mux.HandleFunc("/ws", mw.Wrap("ws", false, 0, s.wsHub.ServeWS))
	}

	mux.HandleFunc("/tx", mw.Wrap("tx", false, 1<<20, s.handleSubmitTx))
	mux.HandleFunc("/tx/", mw.Wrap("tx_get", false, 0, s.handleTxByID))
	mux.HandleFunc("/tx/proof/", mw.Wrap("tx_proof", false, 0, s.handleTxProof))
	mux.HandleFunc("/wallet/create", mw.Wrap("wallet_create", false, 0, s.handleWalletCreate))
	mux.HandleFunc("/wallet/sign", mw.Wrap("wallet_sign", false, 0, s.handleWalletSign))
	mux.HandleFunc("/mempool", mw.Wrap("mempool", false, 0, s.handleMempool))
	mux.HandleFunc("/mine/once", mw.Wrap("mine_once", true, 1<<10, s.handleMineOnce))
	mux.HandleFunc("/audit/chain", mw.Wrap("audit_chain", true, 1<<16, s.handleAuditChain))
	mux.HandleFunc("/block", mw.Wrap("block_submit", true, 4<<20, s.handleAddBlock))
	mux.HandleFunc("/block/height/", mw.Wrap("block_height", false, 0, s.handleBlockByHeight))

	mux.HandleFunc("/balance/", mw.Wrap("balance", false, 0, s.handleBalance))
	mux.HandleFunc("/address/", mw.Wrap("address_txs", false, 0, s.handleAddressTxs))
	mux.HandleFunc("/chain/info", mw.Wrap("chain_info", false, 0, s.handleChainInfo))
	mux.HandleFunc("/headers/from/", mw.Wrap("headers_from", false, 0, s.handleHeadersFrom))
	mux.HandleFunc("/blocks/from/", mw.Wrap("blocks_from", false, 0, s.handleBlocksFrom))
	mux.HandleFunc("/blocks/hash/", mw.Wrap("blocks_hash", false, 0, s.handleBlockByHash))

	mux.HandleFunc("/p2p/getaddr", mw.Wrap("p2p_getaddr", false, 0, s.handleP2PGetAddr))
	mux.HandleFunc("/p2p/addr", mw.Wrap("p2p_addr", false, 1<<10, s.handleP2PAddr))

	if s.airdroper != nil {
		mux.HandleFunc("/airdrop/claim", mw.Wrap("airdrop_claim", false, 1<<10, s.handleAirdropClaim))
		mux.HandleFunc("/airdrop/stats", mw.Wrap("airdrop_stats", false, 0, s.handleAirdropStats))
		mux.HandleFunc("/airdrop/status", mw.Wrap("airdrop_status", false, 0, s.handleAirdropStatus))
	}

	mux.HandleFunc("/wallet", mw.Wrap("wallet_redirect", false, 0, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/wallet/", http.StatusFound)
	}))
	mux.HandleFunc("/wallet/", mw.Wrap("wallet", false, 0, func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/wallet/", WalletFileServer()).ServeHTTP(w, r)
	}))

	mux.HandleFunc("/", mw.Wrap("root", false, 0, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/explorer/", http.StatusFound)
	}))
	mux.HandleFunc("/explorer", mw.Wrap("explorer_redirect", false, 0, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/explorer/", http.StatusMovedPermanently)
	}))
	mux.HandleFunc("/explorer/", mw.Wrap("explorer", false, 0, func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/explorer/", ExplorerFileServer()).ServeHTTP(w, r)
	}))
	return mux
}
