package networking

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	neocoinpb "github.com/Neo4717/NeoCoin/proto"
)

type PeerManagerConfig struct {
	MaxPeers       int
	MaxPeerScore   float64
	BanThreshold   float64
	BanDuration    time.Duration
	DialTimeout    time.Duration
	ConnRetryDelay time.Duration
	MaxRetries     int
}

func DefaultPeerManagerConfig() PeerManagerConfig {
	return PeerManagerConfig{
		MaxPeers:       200,
		MaxPeerScore:   100.0,
		BanThreshold:   10.0,
		BanDuration:    24 * time.Hour,
		DialTimeout:    5 * time.Second,
		ConnRetryDelay: 30 * time.Second,
		MaxRetries:     3,
	}
}

type PeerManager struct {
	config       PeerManagerConfig
	peers        map[string]*Peer
	pendingPeers map[string]*Peer
	bannedPeers  map[string]time.Time
	knownAddrs   []string

	mu         sync.RWMutex
	scorer     *PeerScorer
	listenAddr string
	nodeID     string
	chainID    uint64

	notifyCh chan PeerEvent
	doneCh   chan struct{}
	chain    ChainInfoProvider

	dataDir string
}

type PeerEvent struct {
	Type  PeerEventType
	Peer  *Peer
	Error error
}

type PeerEventType int

const (
	PeerEventConnected    PeerEventType = 0
	PeerEventDisconnected PeerEventType = 1
	PeerEventBanned       PeerEventType = 2
	PeerEventError        PeerEventType = 3
)

func NewPeerManager(config PeerManagerConfig, nodeID string) *PeerManager {
	if config.MaxPeers <= 0 {
		config.MaxPeers = 200
	}
	return &PeerManager{
		config:       config,
		peers:        make(map[string]*Peer),
		pendingPeers: make(map[string]*Peer),
		bannedPeers:  make(map[string]time.Time),
		scorer:       NewPeerScorer(config.MaxPeers),
		nodeID:       nodeID,
		notifyCh:     make(chan PeerEvent, 100),
		doneCh:       make(chan struct{}),
	}
}

func (pm *PeerManager) SetDataDir(dir string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.dataDir = dir
}

func (pm *PeerManager) SavePeers() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.dataDir == "" {
		return nil
	}

	addrs := make([]string, 0, len(pm.peers))
	for addr := range pm.peers {
		addrs = append(addrs, addr)
	}

	data, err := json.Marshal(addrs)
	if err != nil {
		return err
	}

	return os.WriteFile(pm.dataDir+"/peers.json", data, 0644)
}

func (pm *PeerManager) LoadPeers() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.dataDir == "" {
		return nil
	}

	data, err := os.ReadFile(pm.dataDir + "/peers.json")
	if err != nil {
		return err
	}

	var addrs []string
	if err := json.Unmarshal(data, &addrs); err != nil {
		return err
	}

	pm.knownAddrs = addrs
	return nil
}

func (pm *PeerManager) SetChain(chain ChainInfoProvider) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.chain = chain
}

func (pm *PeerManager) SetChainID(chainID uint64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.chainID = chainID
}

func ParsePeersEnv(peersEnv string) []string {
	var peers []string
	for _, raw := range strings.Split(peersEnv, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		peers = append(peers, raw)
	}
	return peers
}

func (pm *PeerManager) AddPeer(address string, id string, isOutgoing bool) *Peer {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.isBanned(address) {
		return nil
	}

	if len(pm.peers) >= pm.config.MaxPeers {
		return nil
	}

	peer := NewPeer(address, id, isOutgoing)
	pm.peers[address] = peer
	return peer
}

func (pm *PeerManager) RemovePeer(address string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, ok := pm.peers[address]; ok {
		pm.scorer.RemovePeer(address)
		delete(pm.peers, address)
	}
}

func (pm *PeerManager) GetPeer(address string) *Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if peer, ok := pm.peers[address]; ok {
		return peer
	}
	return nil
}

func (pm *PeerManager) Peers() []*Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]*Peer, 0, len(pm.peers))
	for _, peer := range pm.peers {
		result = append(result, peer)
	}
	return result
}

func (pm *PeerManager) PeerCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.peers)
}

func (pm *PeerManager) PeerIDs() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]string, 0, len(pm.peers))
	for _, peer := range pm.peers {
		result = append(result, peer.ID)
	}
	return result
}

func (pm *PeerManager) GetPeerAddresses() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]string, 0, len(pm.peers))
	for _, peer := range pm.peers {
		result = append(result, peer.Address)
	}
	return result
}

func (pm *PeerManager) GetHealthyPeers() []*Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var result []*Peer
	for _, peer := range pm.peers {
		if peer.IsHealthy() {
			result = append(result, peer)
		}
	}
	return result
}

func (pm *PeerManager) AddAddress(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, known := range pm.knownAddrs {
		if known == addr {
			return
		}
	}
	pm.knownAddrs = append(pm.knownAddrs, addr)
}

func (pm *PeerManager) AddAddresses(addrs []string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, addr := range addrs {
		found := false
		for _, known := range pm.knownAddrs {
			if known == addr {
				found = true
				break
			}
		}
		if !found {
			pm.knownAddrs = append(pm.knownAddrs, addr)
		}
	}
}

func (pm *PeerManager) GetAddresses() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return append([]string(nil), pm.knownAddrs...)
}

func (pm *PeerManager) BanPeer(address string, reason string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.peers, address)
	pm.bannedPeers[address] = time.Now().Add(pm.config.BanDuration)
	log.Printf("peer banned: %s (reason: %s)", address, reason)
}

func (pm *PeerManager) UnbanPeer(address string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.bannedPeers, address)
}

func (pm *PeerManager) isBanned(address string) bool {
	if banTime, ok := pm.bannedPeers[address]; ok {
		if time.Now().Before(banTime) {
			return true
		}
		delete(pm.bannedPeers, address)
	}
	return false
}

func (pm *PeerManager) IsBanned(address string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.isBanned(address)
}

func (pm *PeerManager) cleanupBanned() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	for address, banTime := range pm.bannedPeers {
		if now.After(banTime) {
			delete(pm.bannedPeers, address)
		}
	}
}

func (pm *PeerManager) RecordSuccess(address string, latencyMs int64) {
	pm.scorer.RecordSuccess(address, latencyMs)
}

func (pm *PeerManager) RecordFailure(address string) {
	pm.scorer.RecordFailure(address)
}

func (pm *PeerManager) GetScore(address string) float64 {
	return pm.scorer.GetScore(address)
}

func (pm *PeerManager) GetTopPeers(n int) []string {
	return pm.scorer.GetTopPeers(n)
}

func (pm *PeerManager) UpdatePeerScore(address string, score float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if peer, ok := pm.peers[address]; ok {
		peer.UpdateScore(score)
		if score < pm.config.BanThreshold {
			pm.BanPeer(address, "score too low")
		}
	}
}

func (pm *PeerManager) CanAcceptMorePeers() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.peers) < pm.config.MaxPeers
}

func (pm *PeerManager) NodeID() string {
	return pm.nodeID
}

func (pm *PeerManager) Subscribe() chan PeerEvent {
	return pm.notifyCh
}

func (pm *PeerManager) notify(event PeerEvent) {
	select {
	case pm.notifyCh <- event:
	default:
	}
}

func (pm *PeerManager) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pm.cleanupBanned()
			pm.evictLowScorePeers()
		}
	}
}

func (pm *PeerManager) evictLowScorePeers() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.peers) < pm.config.MaxPeers {
		return
	}

	var worstPeer string
	lowestScore := pm.config.MaxPeerScore + 1

	for addr, peer := range pm.peers {
		if peer.Score < lowestScore && !peer.IsOutgoing {
			lowestScore = peer.Score
			worstPeer = addr
		}
	}

	if worstPeer != "" && lowestScore < pm.config.BanThreshold {
		delete(pm.peers, worstPeer)
	}
}

func (pm *PeerManager) FetchChainInfo(ctx context.Context, peer *Peer) (*PeerChainInfo, error) {
	if peer == nil || !peer.IsConnected() {
		return nil, fmt.Errorf("peer not connected")
	}

	pm.mu.RLock()
	chainID := pm.chainID
	pm.mu.RUnlock()

	req := map[string]interface{}{
		"version":   ProtocolVersion,
		"chainId":   chainID,
		"timestamp": time.Now().Unix(),
	}

	type chainInfoResp struct {
		ChainID     uint64 `json:"chainId"`
		Height      uint64 `json:"height"`
		RulesHash   string `json:"rulesHash"`
		GenesisHash string `json:"genesisHash"`
		LatestHash  string `json:"latestHash"`
	}

	var resp chainInfoResp
	err := peer.Request(ctx, "chain_info_req", req, &resp)
	if err != nil {
		return nil, err
	}

	return &PeerChainInfo{
		ChainID:     resp.ChainID,
		Height:      resp.Height,
		RulesHash:   resp.RulesHash,
		GenesisHash: resp.GenesisHash,
		Version:     ProtocolVersion,
	}, nil
}

func (pm *PeerManager) FetchHeadersFrom(ctx context.Context, peer *Peer, from uint64, count int) ([]BlockHeader, error) {
	type simpleReq struct {
		From  uint64 `json:"from"`
		Count int    `json:"count"`
	}

	type headersResp struct {
		Headers []BlockHeader `json:"headers"`
	}

	var resp headersResp
	err := peer.Request(ctx, "headers_from_req", simpleReq{From: from, Count: count}, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Headers, nil
}

func (pm *PeerManager) FetchBlockByHash(ctx context.Context, peer *Peer, hashHex string) (*BlockData, error) {
	type blockReq struct {
		Hash string `json:"hash"`
	}
	type blockResp struct {
		Block *BlockData `json:"block"`
	}

	var resp blockResp
	err := peer.Request(ctx, "block_by_hash_req", blockReq{Hash: hashHex}, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Block, nil
}

func (pm *PeerManager) SyncBlocks(ctx context.Context) {
	pm.mu.RLock()
	chain := pm.chain
	pm.mu.RUnlock()

	if chain == nil {
		return
	}

	localHeight := uint64(0)
	if tip := chain.LatestBlock(); tip != nil {
		localHeight = tip.GetHeight()
	}

	var bestPeer *Peer
	var bestInfo *PeerChainInfo

	for _, peer := range pm.GetHealthyPeers() {
		info, err := pm.FetchChainInfo(ctx, peer)
		if err != nil {
			continue
		}

		pm.mu.RLock()
		chainID := pm.chainID
		pm.mu.RUnlock()

		if info.ChainID != chainID {
			continue
		}

		if info.Height > localHeight {
			if bestInfo == nil || info.Height > bestInfo.Height {
				bestPeer = peer
				bestInfo = info
			}
		}
	}

	if bestPeer == nil {
		return
	}

	log.Printf("parallel sync: syncing %d blocks from %s", bestInfo.Height-localHeight, bestPeer.Address)

	batchSize := uint64(100)
	var wg sync.WaitGroup

	for from := localHeight + 1; from <= bestInfo.Height; from += batchSize {
		to := from + batchSize - 1
		if to > bestInfo.Height {
			to = bestInfo.Height
		}

		wg.Add(1)
		go func(start, end uint64) {
			defer wg.Done()

			headers, err := pm.FetchHeadersFrom(ctx, bestPeer, start, int(end-start+1))
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
				go func(hdr BlockHeader) {
					defer blockWg.Done()

					hashHex := fmt.Sprintf("%x", hdr.Hash)
					block, err := pm.FetchBlockByHash(ctx, bestPeer, hashHex)
					if err != nil {
						log.Printf("parallel sync: fetch block %s failed: %v", hashHex, err)
						return
					}

					if _, err := chain.AddBlock(block); err != nil {
						log.Printf("parallel sync: add block %d failed: %v", hdr.Height, err)
					}
				}(hdr)
			}
			blockWg.Wait()
		}(from, to)
	}

	wg.Wait()
}

type PeerScorer struct {
	mu       sync.RWMutex
	peers    map[string]*PeerScore
	maxPeers int
}

type PeerScore struct {
	Peer           string
	Score          float64
	SuccessCount   int
	FailureCount   int
	TotalLatencyMs int64
	LastSeen       time.Time
	FirstSeen      time.Time
}

func NewPeerScorer(maxPeers int) *PeerScorer {
	if maxPeers <= 0 {
		maxPeers = 100
	}
	return &PeerScorer{
		peers:    make(map[string]*PeerScore),
		maxPeers: maxPeers,
	}
}

func (ps *PeerScorer) RecordSuccess(peer string, latencyMs int64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	if p, ok := ps.peers[peer]; ok {
		p.SuccessCount++
		p.TotalLatencyMs += latencyMs
		p.LastSeen = now
		p.Score = ps.calculateScore(p)
	} else {
		ps.peers[peer] = &PeerScore{
			Peer:           peer,
			Score:          50.0,
			SuccessCount:   1,
			FailureCount:   0,
			TotalLatencyMs: latencyMs,
			LastSeen:       now,
			FirstSeen:      now,
		}
		ps.evictIfNeeded()
	}
}

func (ps *PeerScorer) RecordFailure(peer string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	if p, ok := ps.peers[peer]; ok {
		p.FailureCount++
		p.LastSeen = now
		p.Score = ps.calculateScore(p)
	} else {
		ps.peers[peer] = &PeerScore{
			Peer:           peer,
			Score:          25.0,
			SuccessCount:   0,
			FailureCount:   1,
			TotalLatencyMs: 0,
			LastSeen:       now,
			FirstSeen:      now,
		}
		ps.evictIfNeeded()
	}
}

func (ps *PeerScorer) calculateScore(p *PeerScore) float64 {
	total := p.SuccessCount + p.FailureCount
	if total == 0 {
		return 50.0
	}

	successRate := float64(p.SuccessCount) / float64(total)

	var avgLatency float64 = 1000
	if p.SuccessCount > 0 {
		avgLatency = float64(p.TotalLatencyMs) / float64(p.SuccessCount)
	}

	latencyFactor := 1.0
	if avgLatency < 100 {
		latencyFactor = 1.5
	} else if avgLatency < 500 {
		latencyFactor = 1.2
	} else if avgLatency > 2000 {
		latencyFactor = 0.5
	} else if avgLatency > 5000 {
		latencyFactor = 0.2
	}

	score := successRate * 100 * latencyFactor

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

func (ps *PeerScorer) evictIfNeeded() {
	if len(ps.peers) > ps.maxPeers {
		var worst string
		lowestScore := 101.0
		for peer, p := range ps.peers {
			if p.Score < lowestScore {
				lowestScore = p.Score
				worst = peer
			}
		}
		if worst != "" {
			delete(ps.peers, worst)
		}
	}
}

func (ps *PeerScorer) GetScore(peer string) float64 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	if p, ok := ps.peers[peer]; ok {
		return p.Score
	}
	return 0
}

func (ps *PeerScorer) GetTopPeers(n int) []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	type scoredPeer struct {
		peer  string
		score float64
	}

	var scored []scoredPeer
	for peer, p := range ps.peers {
		scored = append(scored, scoredPeer{peer: peer, score: p.Score})
	}

	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	if n > len(scored) {
		n = len(scored)
	}

	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = scored[i].peer
	}
	return result
}

func (ps *PeerScorer) RemovePeer(peer string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.peers, peer)
}

func (ps *PeerScorer) Count() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.peers)
}

type P2PPeerManager struct {
	peers  []string
	client *P2PClient
}

func NewP2PPeerManager(chainID uint64, rulesHash string, nodeID string, peers []string) *P2PPeerManager {
	return &P2PPeerManager{
		peers:  peers,
		client: NewP2PClient(chainID, rulesHash, nodeID),
	}
}

func (pm *P2PPeerManager) Peers() []string { return append([]string(nil), pm.peers...) }

func (pm *P2PPeerManager) FetchChainInfo(ctx context.Context, peer string) (*ChainInfo, error) {
	var out ChainInfo
	if err := pm.client.do(ctx, peer, "chain_info_req", nil, &out, "chain_info"); err != nil {
		return nil, err
	}
	return &out, nil
}

func (pm *P2PPeerManager) FetchHeadersFrom(ctx context.Context, peer string, fromHeight uint64, count int) ([]BlockHeader, error) {
	var out []BlockHeader
	if err := pm.client.do(ctx, peer, "headers_from_req", HeadersRequest{From: fromHeight, Count: count}, &out, "headers"); err != nil {
		return nil, err
	}
	return out, nil
}

func (pm *P2PPeerManager) FetchBlockByHash(ctx context.Context, peer, hashHex string) ([]byte, error) {
	var out []byte
	err := pm.client.do(ctx, peer, "block_by_hash_req", BlockByHashRequest{HashHex: hashHex}, &out, "block")
	if err != nil {
		if errors.Is(err, errors.New("not found")) {
			return nil, errors.New("not found")
		}
		return nil, err
	}
	return out, nil
}

func (pm *P2PPeerManager) BroadcastTransaction(ctx context.Context, txHex string, _ int) {
	for _, peer := range pm.peers {
		go func(p string) {
			_, err := pm.client.BroadcastTransaction(ctx, p, txHex)
			if err != nil {
				log.Printf("p2p broadcast tx to %s failed: %v", p, err)
			}
		}(peer)
	}
}

func (pm *P2PPeerManager) BroadcastBlock(ctx context.Context, blockHex string) {
	for _, peer := range pm.peers {
		go func(p string) {
			_, err := pm.client.BroadcastBlock(ctx, p, blockHex)
			if err != nil {
				log.Printf("p2p broadcast block to %s failed: %v", p, err)
			}
		}(peer)
	}
}

func (pm *P2PPeerManager) FetchAnyBlockByHash(ctx context.Context, hashHex string) ([]byte, string, error) {
	var lastErr error
	for _, peer := range pm.peers {
		var block []byte
		err := pm.client.do(ctx, peer, "block_by_hash_req", BlockByHashRequest{HashHex: hashHex}, &block, "block")
		if err == nil {
			return block, peer, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no peers configured")
	}
	return nil, "", lastErr
}

type BlockByHashRequest struct {
	HashHex string `json:"hashHex"`
}

func EnsureAncestors(ctx context.Context, pm *P2PPeerManager, bc interface {
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
			return errors.New("ancestor fetch cycle")
		}
		visited[need] = struct{}{}

		b, _, err := pm.FetchAnyBlockByHash(ctx, need)
		if err != nil {
			return err
		}

		_, err = bc.AddBlock(b)
		if err == nil {
			return nil
		}
		if errors.Is(err, fmt.Errorf("unknown parent")) {
			continue
		}
		return err
	}
	return errors.New("max ancestor depth exceeded")
}

type P2PClient struct {
	chainID   uint64
	rulesHash string
	nodeID    string

	dialTimeout time.Duration
	ioTimeout   time.Duration
	maxMsgBytes int
	enableProto bool
	codec       *ProtobufCodec
}

func NewP2PClient(chainID uint64, rulesHash string, nodeID string) *P2PClient {
	return NewP2PClientWithProto(chainID, rulesHash, nodeID, true)
}

func NewP2PClientWithProto(chainID uint64, rulesHash string, nodeID string, enableProtobuf bool) *P2PClient {
	if nodeID == "" {
		nodeID = "unknown"
	}
	return &P2PClient{
		chainID:     chainID,
		rulesHash:   strings.TrimSpace(rulesHash),
		nodeID:      nodeID,
		dialTimeout: 5 * time.Second,
		ioTimeout:   10 * time.Second,
		maxMsgBytes: 4 << 20,
		enableProto: enableProtobuf,
		codec:       NewProtobufCodec(enableProtobuf),
	}
}

func (c *P2PClient) do(ctx context.Context, peer string, reqType string, reqPayload any, resp any, expectedRespType string) error {
	peer = strings.TrimSpace(peer)
	if peer == "" {
		return errors.New("empty peer")
	}

	d := P2PDialer{Timeout: c.dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", peer)
	if err != nil {
		return err
	}
	defer conn.Close()

	return c.doWithConn(ctx, conn, reqType, reqPayload, resp, expectedRespType)
}

func (c *P2PClient) doWithConn(ctx context.Context, conn *P2PConnection, reqType string, reqPayload any, resp any, expectedRespType string) error {
	hello := NewHello(c.chainID, c.rulesHash, c.nodeID)
	hello.TimeUnix = time.Now().Unix()

	if c.enableProto {
		return c.doWithConnProto(conn, reqType, reqPayload, resp, expectedRespType, hello)
	}

	payload := MustJSON(hello)
	if err := WriteJSON(conn, Envelope{Type: "hello", Payload: payload}); err != nil {
		return err
	}

	raw, err := ReadJSON(conn, 1<<20)
	if err != nil {
		return err
	}

	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}

	if env.Type != "hello" {
		return errors.New("bad hello response")
	}

	var helloResp Hello
	if err := json.Unmarshal(env.Payload, &helloResp); err != nil {
		return err
	}

	if helloResp.Protocol != 1 || helloResp.ChainID != c.chainID {
		return errors.New("wrong chain/protocol")
	}
	if c.rulesHash != "" && helloResp.RulesHash != c.rulesHash {
		return errors.New("rules hash mismatch")
	}

	var payload2 []byte
	if reqPayload != nil {
		payload2 = MustJSON(reqPayload)
	}
	if err := WriteJSON(conn, Envelope{Type: reqType, Payload: payload2}); err != nil {
		return err
	}

	raw, err = ReadJSON(conn, c.maxMsgBytes)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}

	if expectedRespType != "" && env.Type != expectedRespType {
		if env.Type == "not_found" {
			return errors.New("not found")
		}
		return errors.New("unexpected response type: " + env.Type)
	}

	if resp != nil {
		return json.Unmarshal(env.Payload, resp)
	}
	return nil
}

func (c *P2PClient) doWithConnProto(conn *P2PConnection, reqType string, reqPayload any, resp any, expectedRespType string, hello Hello) error {
	helloProto := &neocoinpb.Hello{
		Protocol:  hello.Protocol,
		ChainId:   hello.ChainID,
		RulesHash: hello.RulesHash,
		NodeId:    hello.NodeID,
		TimeUnix:  hello.TimeUnix,
		Port:      uint32(hello.Port),
		Services:  hello.Services,
	}

	encoded, err := c.codec.Encode("hello", helloProto)
	if err != nil {
		return fmt.Errorf("encode hello: %w", err)
	}
	if _, err := conn.Write(encoded); err != nil {
		return fmt.Errorf("write hello: %w", err)
	}

	msgType, payload, err := c.codec.Decode(conn)
	if err != nil {
		return fmt.Errorf("read hello response: %w", err)
	}

	if msgType != "" && msgType != "hello" {
		return fmt.Errorf("expected hello response, got: %s", msgType)
	}

	helloResp, err := UnmarshalProtoMessage("hello", payload)
	if err != nil {
		return fmt.Errorf("unmarshal hello: %w", err)
	}
	helloProtoResp := helloResp.(*neocoinpb.Hello)

	if helloProtoResp.Protocol != 1 || helloProtoResp.ChainId != c.chainID {
		return errors.New("wrong chain/protocol")
	}
	if c.rulesHash != "" && helloProtoResp.RulesHash != c.rulesHash {
		return errors.New("rules hash mismatch")
	}

	var encodedPayload []byte
	if reqPayload != nil {
		protoPayload, err := MarshalProtoMessage(reqPayload)
		if err != nil {
			return fmt.Errorf("marshal request payload: %w", err)
		}
		encodedPayload, err = c.codec.Encode(reqType, protoPayload)
	} else {
		encodedPayload, err = c.codec.Encode(reqType, nil)
	}
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	if _, err := conn.Write(encodedPayload); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	msgType, payload, err = c.codec.Decode(conn)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if expectedRespType != "" && msgType != "" && msgType != expectedRespType {
		if msgType == "not_found" {
			return errors.New("not found")
		}
		return fmt.Errorf("unexpected response type: %s", msgType)
	}

	if resp != nil && len(payload) > 0 {
		decoded, err := UnmarshalProtoMessage(expectedRespType, payload)
		if err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
		respBytes, err := MarshalProtoMessage(decoded)
		if err != nil {
			return fmt.Errorf("re-encode for JSON resp: %w", err)
		}
		return json.Unmarshal(respBytes, resp)
	}
	return nil
}

type p2pTxResponse struct {
	TxID string `json:"txid"`
}

type p2pBlockResponse struct {
	Hash string `json:"hash"`
}

func (c *P2PClient) RequestTransaction(ctx context.Context, peer string, txHex string) (string, error) {
	var resp p2pTxResponse
	err := c.do(ctx, peer, "tx_req", TXRequest{TxHex: txHex}, &resp, "tx_ack")
	if err != nil {
		return "", err
	}
	return resp.TxID, nil
}

func (c *P2PClient) BroadcastTransaction(ctx context.Context, peer string, txHex string) (string, error) {
	var resp p2pTxResponse
	err := c.do(ctx, peer, "tx_broadcast", TransactionBroadcast{TxHex: txHex}, &resp, "tx_broadcast_ack")
	if err != nil {
		return "", err
	}
	return resp.TxID, nil
}

func (c *P2PClient) BroadcastBlock(ctx context.Context, peer string, blockHex string) (string, error) {
	var resp p2pBlockResponse
	err := c.do(ctx, peer, "block_broadcast", BlockBroadcast{BlockHex: blockHex}, &resp, "block_broadcast_ack")
	if err != nil {
		return "", err
	}
	return resp.Hash, nil
}

func (c *P2PClient) BroadcastInv(ctx context.Context, peer string, inv InvMessage) (string, error) {
	var resp map[string]any
	err := c.do(ctx, peer, "inv", inv, &resp, "inv_ack")
	if err != nil {
		return "", err
	}
	return "", nil
}

func (c *P2PClient) RequestBlock(ctx context.Context, peer string, hashHex string) ([]byte, error) {
	var block []byte
	err := c.do(ctx, peer, "block_req", GetBlockRequest{HashHex: hashHex}, &block, "block")
	if err != nil {
		return nil, err
	}
	return block, nil
}

type TXRequest struct {
	TxHex string `json:"txHex"`
}

type GetBlockRequest struct {
	HashHex string `json:"hashHex"`
}

type P2PDialer struct {
	Timeout time.Duration
	net.Dialer
}

func (d *P2PDialer) DialContext(ctx context.Context, network, address string) (*P2PConnection, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &P2PConnection{Conn: conn}, nil
}

type P2PConnection struct {
	net.Conn
}

func ReadJSON(r io.Reader, maxBytes int) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint32(lenBuf[:]))
	if n <= 0 {
		return nil, io.ErrUnexpectedEOF
	}
	if maxBytes > 0 && n > maxBytes {
		_, _ = io.CopyN(io.Discard, r, int64(n))
		return nil, ErrMessageTooLarge
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, err
	}
	return b, nil
}

func WriteJSON(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(b)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func MustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
