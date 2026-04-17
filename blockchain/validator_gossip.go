package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type ValidatorGossip struct {
	mu         sync.RWMutex
	nodeID     string
	addr       string
	validators map[string]*ValidatorConn
	listener   net.Listener
	tm         *Tendermint
	bc         interface{ GetHeight() uint64 }

	msgCh  chan GossipMessage
	stopCh chan struct{}

	peerAddrs []string
}

type ValidatorConn struct {
	Addr      string
	Conn      net.Conn
	LastSeen  time.Time
	LatencyMs int64
	Online    bool
}

type GossipMessage struct {
	Type      string `json:"type"`
	Src       string `json:"src"`
	Height    uint64 `json:"height"`
	Round     uint64 `json:"round"`
	BlockHash []byte `json:"blockHash,omitempty"`
	Signature []byte `json:"signature,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

const (
	VMSGProposal  = "proposal"
	VMSGPrevote   = "prevote"
	VMSGPrecommit = "precommit"
	VMSGNewHeight = "new_height"
	VMSGPing      = "vping"
	VMSGPong      = "vpong"
)

func NewValidatorGossip(nodeID, addr string, tm *Tendermint, bc interface{ GetHeight() uint64 }) *ValidatorGossip {
	return &ValidatorGossip{
		nodeID:     nodeID,
		addr:       addr,
		tm:         tm,
		bc:         bc,
		validators: make(map[string]*ValidatorConn),
		msgCh:      make(chan GossipMessage, 1000),
		stopCh:     make(chan struct{}),
	}
}

func (vg *ValidatorGossip) Start() error {
	ln, err := net.Listen("tcp", vg.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	vg.listener = ln

	go vg.acceptLoop()
	go vg.processLoop()
	go vg.pingLoop()

	log.Printf("[Gossip] Validator network started on %s", vg.addr)
	return nil
}

func (vg *ValidatorGossip) Stop() {
	close(vg.stopCh)
	if vg.listener != nil {
		vg.listener.Close()
	}
	for _, v := range vg.validators {
		if v.Conn != nil {
			v.Conn.Close()
		}
	}
}

func (vg *ValidatorGossip) acceptLoop() {
	for {
		conn, err := vg.listener.Accept()
		if err != nil {
			select {
			case <-vg.stopCh:
				return
			default:
				log.Printf("[Gossip] Accept error: %v", err)
				continue
			}
		}
		go vg.handleConn(conn)
	}
}

func (vg *ValidatorGossip) handleConn(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 65536)
	for {
		select {
		case <-vg.stopCh:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		var msg GossipMessage
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}

		vg.mu.Lock()
		vg.validators[msg.Src] = &ValidatorConn{
			Addr:     conn.RemoteAddr().String(),
			Conn:     conn,
			LastSeen: time.Now(),
			Online:   true,
		}
		vg.mu.Unlock()

		go vg.handleMessage(msg)
	}
}

func (vg *ValidatorGossip) handleMessage(msg GossipMessage) {
	switch msg.Type {
	case VMSGProposal:
		if len(msg.BlockHash) > 0 {
			vg.tm.ReceiveProposal(msg.BlockHash)
		}

	case VMSGPrevote:
		if vg.tm.IsValidator(msg.Src) {
			vg.tm.Vote(msg.Src, msg.BlockHash)
		}

	case VMSGPrecommit:
		if vg.tm.IsValidator(msg.Src) {
			vg.tm.Vote(msg.Src, msg.BlockHash)
		}

	case VMSGPing:
		vg.sendTo(msg.Src, GossipMessage{
			Type: VMSGPong,
			Src:  vg.nodeID,
		})

	case VMSGPong:
		if vc, ok := vg.validators[msg.Src]; ok {
			vc.LatencyMs = time.Now().UnixMilli() - msg.Timestamp
		}
	}

	vg.broadcastToOthers(msg)
}

func (vg *ValidatorGossip) processLoop() {
	for {
		select {
		case <-vg.stopCh:
			return
		case msg := <-vg.msgCh:
			vg.handleMessage(msg)
		}
	}
}

func (vg *ValidatorGossip) pingLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-vg.stopCh:
			return
		case <-ticker.C:
			vg.mu.RLock()
			for src := range vg.validators {
				vg.sendTo(src, GossipMessage{
					Type:      VMSGPing,
					Src:       vg.nodeID,
					Timestamp: time.Now().UnixMilli(),
				})
			}
			vg.mu.RUnlock()
		}
	}
}

func (vg *ValidatorGossip) ConnectToValidator(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}

	vg.mu.Lock()
	vg.validators[vg.nodeID] = &ValidatorConn{
		Addr:     addr,
		Conn:     conn,
		LastSeen: time.Now(),
		Online:   true,
	}
	vg.mu.Unlock()

	go vg.readFrom(conn)

	log.Printf("[Gossip] Connected to validator: %s", addr)
	return nil
}

func (vg *ValidatorGossip) readFrom(conn net.Conn) {
	buf := make([]byte, 65536)
	for {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		var msg GossipMessage
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}

		select {
		case vg.msgCh <- msg:
		default:
		}
	}
}

func (vg *ValidatorGossip) sendTo(dst string, msg GossipMessage) {
	vg.mu.RLock()
	vc, ok := vg.validators[dst]
	vg.mu.RUnlock()

	if !ok || vc.Conn == nil {
		return
	}

	data, _ := json.Marshal(msg)
	vc.Conn.Write(data)
}

func (vg *ValidatorGossip) BroadcastProposal(blockHash []byte) {
	height := vg.bc.GetHeight()
	msg := GossipMessage{
		Type:      VMSGProposal,
		Src:       vg.nodeID,
		Height:    height,
		BlockHash: blockHash,
		Timestamp: time.Now().UnixMilli(),
	}

	vg.mu.RLock()
	defer vg.mu.RUnlock()

	for src, vc := range vg.validators {
		if src == vg.nodeID {
			continue
		}
		if vc.Conn != nil {
			data, _ := json.Marshal(msg)
			vc.Conn.Write(data)
		}
	}
}

func (vg *ValidatorGossip) BroadcastPrevote(blockHash []byte) {
	height := vg.bc.GetHeight()
	msg := GossipMessage{
		Type:      VMSGPrevote,
		Src:       vg.nodeID,
		Height:    height,
		BlockHash: blockHash,
		Timestamp: time.Now().UnixMilli(),
	}

	vg.mu.RLock()
	defer vg.mu.RUnlock()

	for src, vc := range vg.validators {
		if src == vg.nodeID || vc.Conn == nil {
			continue
		}
		data, _ := json.Marshal(msg)
		vc.Conn.Write(data)
	}
}

func (vg *ValidatorGossip) BroadcastPrecommit(blockHash []byte) {
	height := vg.bc.GetHeight()
	msg := GossipMessage{
		Type:      VMSGPrecommit,
		Src:       vg.nodeID,
		Height:    height,
		BlockHash: blockHash,
		Timestamp: time.Now().UnixMilli(),
	}

	vg.mu.RLock()
	defer vg.mu.RUnlock()

	for src, vc := range vg.validators {
		if src == vg.nodeID || vc.Conn == nil {
			continue
		}
		data, _ := json.Marshal(msg)
		vc.Conn.Write(data)
	}
}

func (vg *ValidatorGossip) broadcastToOthers(msg GossipMessage) {
	vg.mu.RLock()
	defer vg.mu.RUnlock()

	for src, vc := range vg.validators {
		if src == msg.Src || vc.Conn == nil {
			continue
		}
		data, _ := json.Marshal(msg)
		vc.Conn.Write(data)
	}
}

func (vg *ValidatorGossip) GetConnectedValidators() []ValidatorConnInfo {
	vg.mu.RLock()
	defer vg.mu.RUnlock()

	var result []ValidatorConnInfo
	for _, vc := range vg.validators {
		result = append(result, ValidatorConnInfo{
			Addr:     vc.Addr,
			Online:   vc.Online,
			LastSeen: vc.LastSeen.Unix(),
			Latency:  vc.LatencyMs,
		})
	}
	return result
}

type ValidatorConnInfo struct {
	Addr     string
	Online   bool
	LastSeen int64
	Latency  int64
}

func ValidatorGossipFromEnv(tm *Tendermint, bc interface{ GetHeight() uint64 }) *ValidatorGossip {
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		h := sha256.Sum256([]byte(time.Now().String()))
		nodeID = fmt.Sprintf("%x", h[:8])
	}

	addr := os.Getenv("VALIDATOR_LISTEN")
	if addr == "" {
		addr = ":9846"
	}

	vg := NewValidatorGossip(nodeID, addr, tm, bc)

	validators := os.Getenv("VALIDATOR_PEERS")
	if validators != "" {
		for _, v := range strings.Split(validators, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				vg.peerAddrs = append(vg.peerAddrs, v)
			}
		}
	}

	return vg
}

type BlockProposer struct {
	mu     sync.RWMutex
	tm     *Tendermint
	gossip *ValidatorGossip
	bc     interface {
		GetHeight() uint64
		GetBlockHash(uint64) []byte
		ProposeBlock() []byte
	}
	miner     *Miner
	proposing bool
}

func NewBlockProposer(tm *Tendermint, gossip *ValidatorGossip, bc interface {
	GetHeight() uint64
	GetBlockHash(uint64) []byte
	ProposeBlock() []byte
}, miner *Miner) *BlockProposer {
	return &BlockProposer{
		tm:     tm,
		gossip: gossip,
		bc:     bc,
		miner:  miner,
	}
}

func (bp *BlockProposer) Start(ctx context.Context) {
	go bp.run(ctx)
	log.Printf("[Proposer] Block proposer started")
}

func (bp *BlockProposer) run(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bp.checkProposal()
		}
	}
}

func (bp *BlockProposer) checkProposal() {
	height := bp.bc.GetHeight()
	proposer := bp.tm.GetProposer(height, 0)

	if proposer == bp.gossip.nodeID {
		if !bp.proposing {
			bp.proposing = true
			bp.doProposal()
		}
	} else {
		bp.proposing = false
	}
}

func (bp *BlockProposer) doProposal() {
	block := bp.bc.ProposeBlock()
	if block == nil {
		return
	}

	blockHash := sha256.Sum256(block)

	bp.gossip.BroadcastProposal(blockHash[:])
	bp.tm.ReceiveProposal(blockHash[:])

	log.Printf("[Proposer] Proposed block at height %d", bp.bc.GetHeight())
}

func (bp *BlockProposer) OnVote(src string, blockHash []byte, voteType string) {
	if voteType == "prevote" {
		bp.gossip.BroadcastPrevote(blockHash)
	} else if voteType == "precommit" {
		bp.gossip.BroadcastPrecommit(blockHash)
	}
}
