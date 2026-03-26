package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type Propagator struct {
	pm     *PeerManager
	bc     BlockchainInterface
	mp     MempoolInterface
	client *P2PClient

	invCache map[string]time.Time
	invMu    sync.RWMutex
	invTTL   time.Duration

	txFilter    map[string]struct{}
	blockFilter map[string]struct{}
	filterMu    sync.RWMutex

	broadcastTxHook    func(txHex string)
	broadcastBlockHook func(blockHex string)
}

func NewPropagator(pm *PeerManager, bc BlockchainInterface, mp MempoolInterface, client *P2PClient) *Propagator {
	return &Propagator{
		pm:          pm,
		bc:          bc,
		mp:          mp,
		client:      client,
		invCache:    make(map[string]time.Time),
		invTTL:      10 * time.Minute,
		txFilter:    make(map[string]struct{}),
		blockFilter: make(map[string]struct{}),
	}
}

func (p *Propagator) SetBroadcastTxHook(hook func(txHex string)) {
	p.broadcastTxHook = hook
}

func (p *Propagator) SetBroadcastBlockHook(hook func(blockHex string)) {
	p.broadcastBlockHook = hook
}

func (p *Propagator) PropagateTransaction(ctx context.Context, txHex string) error {
	txHash := computeHash([]byte(txHex))

	p.filterMu.Lock()
	if _, exists := p.txFilter[txHash]; exists {
		p.filterMu.Unlock()
		return nil
	}
	p.txFilter[txHash] = struct{}{}
	p.filterMu.Unlock()

	invMsg := InvMessage{
		Items: []InvItem{
			{Type: InvTypeMSG_TX, Hash: []byte(txHash)},
		},
	}

	p.broadcastInvToAll(ctx, invMsg)

	if p.broadcastTxHook != nil {
		p.broadcastTxHook(txHex)
	}

	return nil
}

func (p *Propagator) PropagateBlock(ctx context.Context, blockHex string) error {
	blockHash := computeHash([]byte(blockHex))

	p.filterMu.Lock()
	if _, exists := p.blockFilter[blockHash]; exists {
		p.filterMu.Unlock()
		return nil
	}
	p.blockFilter[blockHash] = struct{}{}
	p.filterMu.Unlock()

	invMsg := InvMessage{
		Items: []InvItem{
			{Type: InvTypeMSG_BLOCK, Hash: []byte(blockHash)},
		},
	}

	p.broadcastInvToAll(ctx, invMsg)

	if p.broadcastBlockHook != nil {
		p.broadcastBlockHook(blockHex)
	}

	return nil
}

func (p *Propagator) broadcastInvToAll(ctx context.Context, inv InvMessage) {
	peers := p.pm.Peers()
	for _, peer := range peers {
		go func(addr string) {
			_, err := p.client.BroadcastInv(ctx, addr, inv)
			if err != nil {
				log.Printf("broadcast inv to %s failed: %v", addr, err)
			}
		}(peer.Address)
	}
}

func (p *Propagator) HandleInv(ctx context.Context, inv InvMessage, fromPeer string) []InvItem {
	var requested []InvItem

	p.filterMu.RLock()
	for _, item := range inv.Items {
		var key string
		switch item.Type {
		case InvTypeMSG_TX:
			key = string(item.Hash)
			if _, exists := p.txFilter[key]; exists {
				continue
			}
		case InvTypeMSG_BLOCK:
			key = string(item.Hash)
			if _, exists := p.blockFilter[key]; exists {
				continue
			}
		default:
			continue
		}
		requested = append(requested, item)
	}
	p.filterMu.RUnlock()

	p.invMu.Lock()
	now := time.Now()
	for hash, expiry := range p.invCache {
		if now.After(expiry) {
			delete(p.invCache, hash)
		}
	}
	for _, item := range requested {
		p.invCache[string(item.Hash)] = now.Add(p.invTTL)
	}
	p.invMu.Unlock()

	return requested
}

func (p *Propagator) HandleGetData(ctx context.Context, req GetDataRequest, fromPeer string) error {
	hashHex := string(req.Hash)

	switch req.InvType {
	case InvTypeMSG_TX:
		txHex, err := p.getTransaction(hashHex)
		if err != nil {
			return err
		}
		return p.sendTransaction(ctx, fromPeer, txHex)
	case InvTypeMSG_BLOCK:
		blockHex, err := p.getBlock(hashHex)
		if err != nil {
			return err
		}
		return p.sendBlock(ctx, fromPeer, blockHex)
	}

	return nil
}

func (p *Propagator) getTransaction(hashHex string) (string, error) {
	return "", fmt.Errorf("transaction not found: %s", hashHex)
}

func (p *Propagator) getBlock(hashHex string) (string, error) {
	b, ok := p.bc.BlockByHash(hashHex)
	if !ok {
		return "", fmt.Errorf("block not found: %s", hashHex)
	}

	blockData, ok := b.(*BlockData)
	if !ok {
		return "", fmt.Errorf("invalid block data")
	}

	blockJSON, err := json.Marshal(blockData)
	if err != nil {
		return "", err
	}
	return string(blockJSON), nil
}

func (p *Propagator) sendTransaction(ctx context.Context, peer string, txHex string) error {
	_, err := p.client.BroadcastTransaction(ctx, peer, txHex)
	return err
}

func (p *Propagator) sendBlock(ctx context.Context, peer string, blockHex string) error {
	_, err := p.client.BroadcastBlock(ctx, peer, blockHex)
	return err
}

func (p *Propagator) BroadcastTransactionToPeers(ctx context.Context, txHex string, excludePeers []string) {
	peers := p.pm.Peers()

	excludeSet := make(map[string]struct{})
	for _, p := range excludePeers {
		excludeSet[p] = struct{}{}
	}

	for _, peer := range peers {
		if _, excluded := excludeSet[peer.Address]; excluded {
			continue
		}
		go func(addr string) {
			_, err := p.client.BroadcastTransaction(ctx, addr, txHex)
			if err != nil {
				log.Printf("propagate tx to %s failed: %v", addr, err)
			}
		}(peer.Address)
	}
}

func (p *Propagator) BroadcastBlockToPeers(ctx context.Context, blockHex string, excludePeers []string) {
	peers := p.pm.Peers()

	excludeSet := make(map[string]struct{})
	for _, p := range excludePeers {
		excludeSet[p] = struct{}{}
	}

	for _, peer := range peers {
		if _, excluded := excludeSet[peer.Address]; excluded {
			continue
		}
		go func(addr string) {
			_, err := p.client.BroadcastBlock(ctx, addr, blockHex)
			if err != nil {
				log.Printf("propagate block to %s failed: %v", addr, err)
			}
		}(peer.Address)
	}
}

func (p *Propagator) RelayTransaction(ctx context.Context, txHex string, hops int) {
	if hops <= 0 {
		return
	}

	p.PropagateTransaction(ctx, txHex)
}

func (p *Propagator) RelayBlock(ctx context.Context, blockHex string, hops int) {
	if hops <= 0 {
		return
	}

	p.PropagateBlock(ctx, blockHex)
}

func (p *Propagator) IsKnownTransaction(txHash string) bool {
	p.filterMu.RLock()
	defer p.filterMu.RUnlock()
	_, exists := p.txFilter[txHash]
	return exists
}

func (p *Propagator) IsKnownBlock(blockHash string) bool {
	p.filterMu.RLock()
	defer p.filterMu.RUnlock()
	_, exists := p.blockFilter[blockHash]
	return exists
}

func (p *Propagator) MarkTransactionKnown(txHash string) {
	p.filterMu.Lock()
	defer p.filterMu.Unlock()
	p.txFilter[txHash] = struct{}{}
}

func (p *Propagator) MarkBlockKnown(blockHash string) {
	p.filterMu.Lock()
	defer p.filterMu.Unlock()
	p.blockFilter[blockHash] = struct{}{}
}

func (p *Propagator) CleanupStaleInventory() {
	p.filterMu.Lock()
	defer p.filterMu.Unlock()

	for hash := range p.txFilter {
		p.invMu.RLock()
		if _, exists := p.invCache[hash]; !exists {
			delete(p.txFilter, hash)
		}
		p.invMu.RUnlock()
	}

	for hash := range p.blockFilter {
		p.invMu.RLock()
		if _, exists := p.invCache[hash]; !exists {
			delete(p.blockFilter, hash)
		}
		p.invMu.RUnlock()
	}
}

func (p *Propagator) Stats() map[string]int {
	p.filterMu.RLock()
	defer p.filterMu.RUnlock()
	return map[string]int{
		"known_transactions": len(p.txFilter),
		"known_blocks":       len(p.blockFilter),
		"cached_inventory": func() int {
			p.invMu.RLock()
			defer p.invMu.RUnlock()
			return len(p.invCache)
		}(),
	}
}

type PropagatorP2PClient struct {
	chainID     uint64
	rulesHash   string
	nodeID      string
	dialTimeout time.Duration
	ioTimeout   time.Duration
	maxMsgBytes int
}

func NewPropagatorP2PClient(chainID uint64, rulesHash string, nodeID string) *PropagatorP2PClient {
	return &PropagatorP2PClient{
		chainID:     chainID,
		rulesHash:   rulesHash,
		nodeID:      nodeID,
		dialTimeout: 5 * time.Second,
		ioTimeout:   10 * time.Second,
		maxMsgBytes: 4 << 20,
	}
}

func (c *PropagatorP2PClient) BroadcastInv(ctx context.Context, peer string, inv InvMessage) (string, error) {
	return "", nil
}

func (c *PropagatorP2PClient) BroadcastTransaction(ctx context.Context, peer string, txHex string) (string, error) {
	return "", nil
}

func (c *PropagatorP2PClient) BroadcastBlock(ctx context.Context, peer string, blockHex string) (string, error) {
	return "", nil
}
