package http

type RouteConfig struct {
	Path         string
	Handler      string
	Admin        bool
	MaxBodyBytes int64
}

var Routes = []RouteConfig{
	{Path: "/health", Handler: "health", Admin: false, MaxBodyBytes: 0},
	{Path: "/metrics", Handler: "metrics", Admin: false, MaxBodyBytes: 0},
	{Path: "/ws", Handler: "ws", Admin: false, MaxBodyBytes: 0},
	{Path: "/tx", Handler: "tx", Admin: false, MaxBodyBytes: 1 << 20},
	{Path: "/tx/{id}", Handler: "tx_get", Admin: false, MaxBodyBytes: 0},
	{Path: "/tx/proof/{id}", Handler: "tx_proof", Admin: false, MaxBodyBytes: 0},
	{Path: "/wallet/create", Handler: "wallet_create", Admin: false, MaxBodyBytes: 0},
	{Path: "/wallet/sign", Handler: "wallet_sign", Admin: false, MaxBodyBytes: 0},
	{Path: "/mempool", Handler: "mempool", Admin: false, MaxBodyBytes: 0},
	{Path: "/mine/once", Handler: "mine_once", Admin: true, MaxBodyBytes: 1 << 10},
	{Path: "/audit/chain", Handler: "audit_chain", Admin: true, MaxBodyBytes: 1 << 16},
	{Path: "/block", Handler: "block_submit", Admin: true, MaxBodyBytes: 4 << 20},
	{Path: "/block/height/{height}", Handler: "block_height", Admin: false, MaxBodyBytes: 0},
	{Path: "/balance/{address}", Handler: "balance", Admin: false, MaxBodyBytes: 0},
	{Path: "/address/{address}/txs", Handler: "address_txs", Admin: false, MaxBodyBytes: 0},
	{Path: "/chain/info", Handler: "chain_info", Admin: false, MaxBodyBytes: 0},
	{Path: "/headers/from/{height}", Handler: "headers_from", Admin: false, MaxBodyBytes: 0},
	{Path: "/blocks/from/{height}", Handler: "blocks_from", Admin: false, MaxBodyBytes: 0},
	{Path: "/blocks/hash/{hash}", Handler: "blocks_hash", Admin: false, MaxBodyBytes: 0},
	{Path: "/wallet", Handler: "wallet_redirect", Admin: false, MaxBodyBytes: 0},
	{Path: "/wallet/", Handler: "wallet", Admin: false, MaxBodyBytes: 0},
	{Path: "/explorer", Handler: "explorer_redirect", Admin: false, MaxBodyBytes: 0},
	{Path: "/explorer/", Handler: "explorer", Admin: false, MaxBodyBytes: 0},
}
