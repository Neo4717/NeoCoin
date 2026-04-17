package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	MaxPeers           = 50
	MinPeers           = 3
	HandshakeTimeout   = 10 * time.Second
	HeartbeatInterval  = 30 * time.Second
	MaxMessageSize     = 1024 * 1024
	HandshakeMagic     = "NEOCHAIN_HANDSHAKE_V1"
	HandshakeExpireSec = 300
)

type P2PMessageType string

const (
	MsgHandshake P2PMessageType = "handshake"
	MsgTx        P2PMessageType = "tx"
	MsgBlock     P2PMessageType = "block"
	MsgGetBlocks P2PMessageType = "get_blocks"
	MsgGetPeers  P2PMessageType = "get_peers"
	MsgPeers     P2PMessageType = "peers"
	MsgPing      P2PMessageType = "ping"
	MsgPong      P2PMessageType = "pong"
)

type P2PMessage struct {
	Type    P2PMessageType `json:"type"`
	Payload interface{}    `json:"payload,omitempty"`
}

type Handshake struct {
	Version    int    `json:"version"`
	ChainID    uint64 `json:"chainId"`
	NodeID     string `json:"nodeId"`
	Height     uint64 `json:"height"`
	ListenPort int    `json:"listenPort"`
	Timestamp  int64  `json:"timestamp"`
	NodePubKey []byte `json:"nodePubKey,omitempty"`
	Signature  []byte `json:"signature,omitempty"`
}

func (h *Handshake) sign(nodePubKey ed25519.PublicKey, nodePrivKey ed25519.PrivateKey) error {
	msg := fmt.Sprintf("%s|%d|%d|%s|%d", HandshakeMagic, h.Version, h.ChainID, h.NodeID, h.Timestamp)
	msgHash := sha256.Sum256([]byte(msg))

	signature := ed25519.Sign(nodePrivKey, msgHash[:])
	h.NodePubKey = nodePubKey
	h.Signature = signature

	return nil
}

func (h *Handshake) verify() error {
	if h.Timestamp == 0 {
		return fmt.Errorf("handshake missing timestamp")
	}
	if time.Now().Unix()-h.Timestamp > HandshakeExpireSec {
		return fmt.Errorf("handshake expired")
	}
	if len(h.Signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length")
	}
	if len(h.NodePubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid pubkey length")
	}

	msg := fmt.Sprintf("%s|%d|%d|%s|%d", HandshakeMagic, h.Version, h.ChainID, h.NodeID, h.Timestamp)
	msgHash := sha256.Sum256([]byte(msg))

	if !ed25519.Verify(h.NodePubKey, msgHash[:], h.Signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

func P2PNewSecureHandshake(nodeID string, chainID uint64, height uint64, port int, nodeKey ed25519.PrivateKey) Handshake {
	ts := time.Now().Unix()
	return Handshake{
		Version:    1,
		ChainID:    chainID,
		NodeID:     nodeID,
		Height:     height,
		ListenPort: port,
		Timestamp:  ts,
	}
}

type Peer struct {
	ID       string
	Addr     string
	Conn     net.Conn
	height   uint64
	lastPing time.Time
	mu       sync.RWMutex
}

type P2PNetwork struct {
	nodeID    string
	chainID   uint64
	listener  net.Listener
	peers     map[string]*Peer
	discovery []string
	seeds     []string
	txChan    chan []byte
	blockChan chan []byte
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	config    P2PConfig
}

type P2PConfig struct {
	ListenAddr  string
	Seeds       []string
	MaxPeers    int
	EnableRelay bool
}

func NewP2PNetwork(nodeID string, chainID uint64, config P2PConfig) *P2PNetwork {
	if config.MaxPeers == 0 {
		config.MaxPeers = MaxPeers
	}

	p2p := &P2PNetwork{
		nodeID:    nodeID,
		chainID:   chainID,
		peers:     make(map[string]*Peer),
		discovery: config.Seeds,
		seeds:     config.Seeds,
		txChan:    make(chan []byte, 100),
		blockChan: make(chan []byte, 100),
		config:    config,
	}
	p2p.ctx, p2p.cancel = context.WithCancel(context.Background())

	return p2p
}
