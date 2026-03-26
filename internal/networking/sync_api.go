package networking

import (
	"context"
)

type ChainInfoProvider interface {
	ChainID() uint64
	RulesHashHex() string
	LatestBlock() BlockInterface
	BlockByHeight(height uint64) (BlockInterface, bool)
	BlockByHash(hashHex string) (BlockInterface, bool)
	AddBlock(block *BlockData) (interface{}, error)
	HeadersFrom(from uint64, count int) []BlockHeader
}

type PeerInfoProvider interface {
	Peers() []*Peer
	FetchChainInfo(ctx context.Context, peer *Peer) (*PeerChainInfo, error)
	FetchHeadersFrom(ctx context.Context, peer *Peer, from uint64, count int) ([]BlockHeader, error)
	FetchBlockByHash(ctx context.Context, peer *Peer, hashHex string) (*BlockData, error)
	GetHealthyPeers() []*Peer
}

type PeerChainInfo struct {
	ChainID     uint64
	Height      uint64
	RulesHash   string
	GenesisHash string
	Version     uint64
}

type SyncAPI struct {
	chain ChainInfoProvider
	peers PeerInfoProvider
}

func NewSyncAPI(chain ChainInfoProvider, peers PeerInfoProvider) *SyncAPI {
	return &SyncAPI{chain: chain, peers: peers}
}

func (s *SyncAPI) Peers() []*Peer {
	return s.peers.Peers()
}

func (s *SyncAPI) FetchChainInfo(ctx context.Context, peer *Peer) (*PeerChainInfo, error) {
	return s.peers.FetchChainInfo(ctx, peer)
}

func (s *SyncAPI) FetchHeadersFrom(ctx context.Context, peer *Peer, from uint64, count int) ([]BlockHeader, error) {
	return s.peers.FetchHeadersFrom(ctx, peer, from, count)
}

func (s *SyncAPI) FetchBlockByHash(ctx context.Context, peer *Peer, hashHex string) (*BlockData, error) {
	return s.peers.FetchBlockByHash(ctx, peer, hashHex)
}
