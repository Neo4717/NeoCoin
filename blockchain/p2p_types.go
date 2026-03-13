package main

type p2pChainInfoReq struct{}

type p2pHeadersFromReq struct {
	From  uint64 `json:"from"`
	Count int    `json:"count"`
}

type p2pBlockByHashReq struct {
	HashHex string `json:"hashHex"`
}
