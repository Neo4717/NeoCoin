package main

type p2pChainInfoReq struct{}

type p2pHeadersFromReq struct {
	From  uint64 `json:"from"`
	Count int    `json:"count"`
}

type p2pBlockByHashReq struct {
	HashHex string `json:"hashHex"`
}

type p2pTransactionReq struct {
	TxHex string `json:"txHex"`
}

type p2pTransactionBroadcast struct {
	TxHex string `json:"txHex"`
}
