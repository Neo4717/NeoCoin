package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type PeerAPI struct {
	server    *Server
	peers     map[string]*peerConn
	mu        sync.RWMutex
	chainID   uint64
	nodeID    string
	rulesHash string
}

type peerConn struct {
	addr   string
	conn   net.Conn
	height uint64
	mu     sync.RWMutex
}

func NewPeerAPI(server *Server, chainID uint64, nodeID, rulesHash string) *PeerAPI {
	return &PeerAPI{
		server:    server,
		peers:     make(map[string]*peerConn),
		chainID:   chainID,
		nodeID:    nodeID,
		rulesHash: rulesHash,
	}
}

func (p *PeerAPI) addPeer(addr string, conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.peers[addr]; !exists {
		p.peers[addr] = &peerConn{addr: addr, conn: conn}
		log.Printf("PeerAPI: added peer %s", addr)
	}
}

func (p *PeerAPI) removePeer(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.peers, addr)
	log.Printf("PeerAPI: removed peer %s", addr)
}

func (p *PeerAPI) getPeerConn(addr string) (net.Conn, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if pc, ok := p.peers[addr]; ok {
		return pc.conn, true
	}
	return nil, false
}

func (p *PeerAPI) Peers() []*SyncPeerInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*SyncPeerInfo, 0, len(p.peers))
	for addr, pc := range p.peers {
		pc.mu.RLock()
		result = append(result, &SyncPeerInfo{
			Addr:   addr,
			Height: pc.height,
		})
		pc.mu.RUnlock()
	}
	return result
}

type SyncPeerInfo struct {
	Addr   string
	Height uint64
}

func (p *PeerAPI) FetchChainInfo(ctx context.Context, peer *SyncPeerInfo) (*SyncChainInfo, error) {
	conn, ok := p.getPeerConn(peer.Addr)
	if !ok {
		return nil, fmt.Errorf("peer not connected: %s", peer.Addr)
	}

	req := Envelope{Type: "chain_info_req", Payload: nil}
	if err := WriteJSON(conn, req); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	raw, err := ReadJSON(conn, 4<<20)
	if err != nil {
		return nil, err
	}
	conn.SetReadDeadline(time.Time{})

	var resp Envelope
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}

	if resp.Type != "chain_info" {
		return nil, fmt.Errorf("expected chain_info, got %s", resp.Type)
	}

	var info SyncChainInfo
	if err := json.Unmarshal(resp.Payload, &info); err != nil {
		return nil, err
	}

	p.mu.Lock()
	if pc, ok := p.peers[peer.Addr]; ok {
		pc.mu.Lock()
		pc.height = info.Height
		pc.mu.Unlock()
	}
	p.mu.Unlock()

	return &info, nil
}

type SyncChainInfo struct {
	ChainID     uint64 `json:"chainId"`
	GenesisHash string `json:"genesisHash"`
	RulesHash   string `json:"rulesHash"`
	Height      uint64 `json:"height"`
	LatestHash  string `json:"latestHash"`
}

func (p *PeerAPI) FetchHeadersFrom(ctx context.Context, peer *SyncPeerInfo, from uint64, count int) ([]SyncHeaderInfo, error) {
	conn, ok := p.getPeerConn(peer.Addr)
	if !ok {
		return nil, fmt.Errorf("peer not connected: %s", peer.Addr)
	}

	req := HeadersRequest{From: from, Count: count}
	env := Envelope{Type: "headers_from_req", Payload: MustJSON(req)}
	if err := WriteJSON(conn, env); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	raw, err := ReadJSON(conn, 4<<20)
	if err != nil {
		return nil, err
	}
	conn.SetReadDeadline(time.Time{})

	var resp Envelope
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}

	if resp.Type != "headers" {
		return nil, fmt.Errorf("expected headers, got %s", resp.Type)
	}

	var headers []SyncHeaderInfo
	if err := json.Unmarshal(resp.Payload, &headers); err != nil {
		return nil, err
	}

	return headers, nil
}

type SyncHeaderInfo struct {
	Version        uint32 `json:"version"`
	Height         uint64 `json:"height"`
	TimestampUnix  int64  `json:"timestampUnix"`
	PrevHash       []byte `json:"prevHash"`
	Hash           []byte `json:"hash"`
	DifficultyBits uint32 `json:"difficultyBits"`
	MerkleRoot     []byte `json:"merkleRoot,omitempty"`
}

func (h *SyncHeaderInfo) HashHex() string {
	return fmt.Sprintf("%x", h.Hash)
}

func (h *SyncHeaderInfo) PrevHashHex() string {
	return fmt.Sprintf("%x", h.PrevHash)
}

func (p *PeerAPI) FetchBlockByHash(ctx context.Context, peer *SyncPeerInfo, hashHex string) (*SyncBlockResponse, error) {
	conn, ok := p.getPeerConn(peer.Addr)
	if !ok {
		return nil, fmt.Errorf("peer not connected: %s", peer.Addr)
	}

	req := BlockByHashReq{HashHex: hashHex}
	env := Envelope{Type: "block_by_hash_req", Payload: MustJSON(req)}
	if err := WriteJSON(conn, env); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	raw, err := ReadJSON(conn, 4<<20)
	if err != nil {
		return nil, err
	}
	conn.SetReadDeadline(time.Time{})

	var resp Envelope
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}

	if resp.Type == "not_found" {
		return nil, fmt.Errorf("block not found: %s", hashHex)
	}
	if resp.Type != "block" {
		return nil, fmt.Errorf("expected block, got %s", resp.Type)
	}

	var block SyncBlockResponse
	if err := json.Unmarshal(resp.Payload, &block); err != nil {
		return nil, err
	}

	return &block, nil
}

type SyncBlockResponse struct {
	Version        uint32                `json:"version"`
	Height         uint64                `json:"height"`
	TimestampUnix  int64                 `json:"timestampUnix"`
	PrevHash       []byte                `json:"prevHash"`
	Nonce          uint64                `json:"nonce"`
	DifficultyBits uint32                `json:"difficultyBits"`
	MinerAddress   string                `json:"minerAddress"`
	Transactions   []SyncTransactionData `json:"transactions"`
	Hash           []byte                `json:"hash"`
}

type SyncTransactionData struct {
	Type       string `json:"type"`
	ChainID    uint64 `json:"chainId"`
	FromPubKey []byte `json:"fromPubKey"`
	ToAddress  string `json:"toAddress"`
	Amount     uint64 `json:"amount"`
	Fee        uint64 `json:"fee"`
	Nonce      uint64 `json:"nonce"`
	Data       string `json:"data"`
	Signature  []byte `json:"signature"`
}

func (b *SyncBlockResponse) HashHex() string {
	return fmt.Sprintf("%x", b.Hash)
}

func (b *SyncBlockResponse) PrevHashHex() string {
	return fmt.Sprintf("%x", b.PrevHash)
}

func (p *PeerAPI) EnsureAncestors(ctx context.Context, bc interface {
	BlockByHash(hash string) (interface{}, bool)
	AddBlock(block interface{}) (interface{}, error)
}, missingHashHex string) error {
	need := missingHashHex
	visited := map[string]struct{}{}
	for depth := 0; depth < 256; depth++ {
		if _, ok := bc.BlockByHash(need); ok {
			return nil
		}
		if _, ok := visited[need]; ok {
			return fmt.Errorf("ancestor fetch cycle")
		}
		visited[need] = struct{}{}

		b, err := p.FetchAnyBlockByHash(ctx, need)
		if err != nil {
			return err
		}

		_, err = bc.AddBlock(b)
		if err == nil {
			return nil
		}
		if err.Error() == "unknown parent" {
			continue
		}
		return err
	}
	return fmt.Errorf("max ancestor depth exceeded")
}

func (p *PeerAPI) FetchAnyBlockByHash(ctx context.Context, hashHex string) ([]byte, error) {
	p.mu.RLock()
	peers := make([]*peerConn, 0, len(p.peers))
	for _, pc := range p.peers {
		peers = append(peers, pc)
	}
	p.mu.RUnlock()

	var lastErr error
	for _, pc := range peers {
		req := BlockByHashReq{HashHex: hashHex}
		env := Envelope{Type: "block_by_hash_req", Payload: MustJSON(req)}
		if err := WriteJSON(pc.conn, env); err != nil {
			lastErr = err
			continue
		}

		pc.conn.SetReadDeadline(time.Now().Add(15 * time.Second))
		raw, err := ReadJSON(pc.conn, 4<<20)
		pc.conn.SetReadDeadline(time.Time{})
		if err != nil {
			lastErr = err
			continue
		}

		var resp Envelope
		if err := json.Unmarshal(raw, &resp); err != nil {
			lastErr = err
			continue
		}

		if resp.Type == "block" {
			return resp.Payload, nil
		}
		lastErr = fmt.Errorf("unexpected response: %s", resp.Type)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no peers available")
	}
	return nil, lastErr
}

func (p *PeerAPI) DialAndAddPeer(ctx context.Context, addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}

	hello := NewHello(p.chainID, p.rulesHash, p.nodeID)
	hello.TimeUnix = time.Now().Unix()

	if err := WriteJSON(conn, Envelope{Type: "hello", Payload: MustJSON(hello)}); err != nil {
		conn.Close()
		return fmt.Errorf("send hello: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	raw, err := ReadJSON(conn, 1<<20)
	if err != nil {
		conn.Close()
		return fmt.Errorf("read hello response: %w", err)
	}
	conn.SetReadDeadline(time.Time{})

	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		conn.Close()
		return fmt.Errorf("unmarshal hello response: %w", err)
	}
	if env.Type != "hello" {
		conn.Close()
		return fmt.Errorf("expected hello, got %s", env.Type)
	}

	var helloResp Hello
	if err := json.Unmarshal(env.Payload, &helloResp); err != nil {
		conn.Close()
		return fmt.Errorf("unmarshal hello: %w", err)
	}
	if helloResp.Protocol != 1 || helloResp.ChainID != p.chainID {
		conn.Close()
		return fmt.Errorf("wrong chain/protocol")
	}
	if p.rulesHash != "" && helloResp.RulesHash != p.rulesHash {
		conn.Close()
		return fmt.Errorf("rules hash mismatch")
	}

	p.addPeer(addr, conn)
	log.Printf("PeerAPI: connected to outbound peer %s", addr)

	go p.readPeerLoop(conn, addr)

	return nil
}

func (p *PeerAPI) readPeerLoop(conn net.Conn, addr string) {
	defer conn.Close()
	defer p.removePeer(addr)

	for {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		raw, err := ReadJSON(conn, 4<<20)
		if err != nil {
			return
		}

		var env Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return
		}

		switch env.Type {
		case "chain_info":
			var info SyncChainInfo
			if err := json.Unmarshal(env.Payload, &info); err == nil {
				p.mu.Lock()
				if pc, ok := p.peers[addr]; ok {
					pc.mu.Lock()
					pc.height = info.Height
					pc.mu.Unlock()
				}
				p.mu.Unlock()
			}
		case "ping":
			WriteJSON(conn, Envelope{Type: "pong", Payload: nil})
		case "tx_broadcast", "block_broadcast":
		}
	}
}

func (p *PeerAPI) SyncBlocksParallel(ctx context.Context, bc BlockchainInterface) {
	if bc == nil {
		return
	}

	localHeight := uint64(0)
	if tip := bc.LatestBlock(); tip != nil {
		localHeight = tip.GetHeight()
	}

	peers := p.Peers()
	if len(peers) == 0 {
		return
	}

	var bestPeer *SyncPeerInfo
	var bestHeight uint64

	for _, peer := range peers {
		info, err := p.FetchChainInfo(ctx, peer)
		if err != nil {
			continue
		}
		if info.Height > localHeight {
			if bestPeer == nil || info.Height > bestHeight {
				bestPeer = peer
				bestHeight = info.Height
			}
		}
	}

	if bestPeer == nil {
		return
	}

	log.Printf("parallel sync: syncing %d blocks from %s", bestHeight-localHeight, bestPeer.Addr)

	batchSize := uint64(100)
	var wg sync.WaitGroup

	for from := localHeight + 1; from <= bestHeight; from += batchSize {
		to := from + batchSize - 1
		if to > bestHeight {
			to = bestHeight
		}

		wg.Add(1)
		go func(start, end uint64) {
			defer wg.Done()

			headers, err := p.FetchHeadersFrom(ctx, bestPeer, start, int(end-start+1))
			if err != nil {
				log.Printf("parallel sync: fetch headers %d-%d failed: %v", start, end, err)
				return
			}

			var blockWg sync.WaitGroup
			for _, hdr := range headers {
				if hdr.Height <= localHeight {
					continue
				}

				blockWg.Add(1)
				go func(hashHex string, height uint64) {
					defer blockWg.Done()

					block, err := p.FetchBlockByHash(ctx, bestPeer, hashHex)
					if err != nil {
						log.Printf("parallel sync: fetch block %s failed: %v", hashHex, err)
						return
					}

					txs := make([]TransactionData, len(block.Transactions))
					for i, tx := range block.Transactions {
						txs[i] = TransactionData{
							Type:       tx.Type,
							ChainID:    tx.ChainID,
							FromPubKey: tx.FromPubKey,
							ToAddress:  tx.ToAddress,
							Amount:     tx.Amount,
							Fee:        tx.Fee,
							Nonce:      tx.Nonce,
							Data:       tx.Data,
							Signature:  tx.Signature,
						}
					}

					b := &BlockData{
						Version:        block.Version,
						Height:         block.Height,
						TimestampUnix:  block.TimestampUnix,
						PrevHash:       block.PrevHash,
						Nonce:          block.Nonce,
						DifficultyBits: block.DifficultyBits,
						MinerAddress:   block.MinerAddress,
						Transactions:   txs,
						Hash:           block.Hash,
					}

					if _, err := bc.AddBlock(b); err != nil {
						log.Printf("parallel sync: add block %d failed: %v", height, err)
					}
				}(hdr.HashHex(), hdr.Height)
			}
			blockWg.Wait()
		}(from, to)
	}

	wg.Wait()
}
