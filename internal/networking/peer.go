package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/Neo4717/NeoCoin/internal/blockchain"
)

type PeerState int

const (
	PeerStateDisconnected PeerState = 0
	PeerStateConnecting   PeerState = 1
	PeerStateHandshake    PeerState = 2
	PeerStateConnected    PeerState = 3
	PeerStateBanned       PeerState = 4
)

type Peer struct {
	Address     string
	ID          string
	Version     uint32
	ChainID     uint64
	RulesHash   string
	LastPing    time.Duration
	Score       float64
	IsOutgoing  bool
	State       PeerState
	ConnectedAt time.Time
	LastSeen    time.Time
	Latency     time.Duration
	Port        uint16
	Services    uint64
	inFlight    int32
	lastFailure time.Time
	banExpiry   time.Time
	Conn        net.Conn
}

func NewPeer(address string, id string, isOutgoing bool) *Peer {
	return &Peer{
		Address:     address,
		ID:          id,
		IsOutgoing:  isOutgoing,
		State:       PeerStateDisconnected,
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
		Score:       50.0,
	}
}

func (p *Peer) IsConnected() bool {
	return p.State == PeerStateConnected || p.State == PeerStateHandshake
}

func (p *Peer) IsBanned() bool {
	return p.State == PeerStateBanned
}

func (p *Peer) UpdateLastSeen() {
	p.LastSeen = time.Now()
}

func (p *Peer) UpdateLatency(latency time.Duration) {
	p.Latency = latency
	p.LastPing = latency
}

func (p *Peer) UpdateScore(score float64) {
	p.Score = score
}

func (p *Peer) RequestHeaders(startHeight, endHeight int64) ([]BlockHeader, error) {
	return nil, nil
}

func (p *Peer) RequestBlocks(hashes []string) ([]*blockchain.Block, error) {
	return nil, nil
}

func (p *Peer) SendPing() error {
	return nil
}

func (p *Peer) Close() {
}

func (p *Peer) Request(ctx context.Context, method string, payload interface{}, result interface{}) error {
	if p.Conn == nil {
		return fmt.Errorf("no connection")
	}

	req := Envelope{Type: method, Payload: MustJSON(payload)}
	if err := WriteJSON(p.Conn, req); err != nil {
		return err
	}

	raw, err := ReadJSON(p.Conn, 4<<20)
	if err != nil {
		return err
	}

	var resp Envelope
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}

	if resp.Type == "error" {
		return fmt.Errorf("peer error: %s", resp.Payload)
	}

	if resp.Type == "reject" {
		return fmt.Errorf("peer rejected: %s", resp.Payload)
	}

	if err := json.Unmarshal(resp.Payload, result); err != nil {
		return err
	}

	return nil
}

func (p *Peer) LatencyMs() int64 {
	return p.Latency.Milliseconds()
}

func (p *Peer) InFlight() int {
	return int(atomic.LoadInt32(&p.inFlight))
}

func (p *Peer) AddInFlight() {
	atomic.AddInt32(&p.inFlight, 1)
}

func (p *Peer) RemoveInFlight() {
	atomic.AddInt32(&p.inFlight, -1)
}

func (p *Peer) RecentlyFailed() bool {
	return time.Since(p.lastFailure) < 5*time.Minute
}

func (p *Peer) RecordFailure() {
	p.lastFailure = time.Now()
}

func (p *Peer) Connected() bool {
	return p.State == PeerStateConnected
}

func (p *Peer) Banned() bool {
	return p.State == PeerStateBanned || (!p.banExpiry.IsZero() && time.Now().Before(p.banExpiry))
}

func (p *Peer) IsHealthy() bool {
	return p.Connected() && !p.Banned() && p.Score > 0
}

type PeerInfo struct {
	ID         string        `json:"id"`
	Address    string        `json:"address"`
	Version    uint32        `json:"version"`
	Height     uint64        `json:"height,omitempty"`
	Latency    time.Duration `json:"latency"`
	State      string        `json:"state"`
	Score      float64       `json:"score"`
	IsOutgoing bool          `json:"isOutgoing"`
}

func (p *Peer) PeerInfo() PeerInfo {
	state := "disconnected"
	switch p.State {
	case PeerStateConnecting:
		state = "connecting"
	case PeerStateHandshake:
		state = "handshake"
	case PeerStateConnected:
		state = "connected"
	case PeerStateBanned:
		state = "banned"
	}

	return PeerInfo{
		ID:         p.ID,
		Address:    p.Address,
		Version:    p.Version,
		Latency:    p.Latency,
		State:      state,
		Score:      p.Score,
		IsOutgoing: p.IsOutgoing,
	}
}
