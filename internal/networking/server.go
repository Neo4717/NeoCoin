package networking

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type ServerConfig struct {
	ListenAddr string
	NodeID     string
	ChainID    uint64
	RulesHash  string
	Port       uint16
	Seeds      string
	MaxConns   int
	MaxMsgSize int
}

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ListenAddr: ":9090",
		MaxConns:   200,
		MaxMsgSize: 4 << 20,
	}
}

type Server struct {
	config  ServerConfig
	bc      BlockchainInterface
	mp      MempoolInterface
	pm      *PeerManager
	peerAPI *PeerAPI
	chainID uint64
	nodeID  string

	listener net.Listener
	sem      chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	done     chan struct{}

	pingInterval    time.Duration
	pingTimeout     time.Duration
	handshakeConfig *ProtocolHandshake
}

type BlockchainInterface interface {
	ChainID() uint64
	RulesHashHex() string
	LatestBlock() BlockInterface
	BlockByHeight(height uint64) (BlockInterface, bool)
	BlockByHash(hashHex string) (BlockInterface, bool)
	HeadersFrom(from uint64, count int) []BlockHeader
	AddBlock(block *BlockData) (interface{}, error)
}

type BlockInterface interface {
	GetHeight() uint64
	GetHash() []byte
}

type BlockData struct {
	Version        uint32
	Height         uint64
	TimestampUnix  int64
	PrevHash       []byte
	Nonce          uint64
	DifficultyBits uint32
	MinerAddress   string
	Transactions   []TransactionData
	Hash           []byte
}

type TransactionData struct {
	Type       string
	ChainID    uint64
	FromPubKey []byte
	ToAddress  string
	Amount     uint64
	Fee        uint64
	Nonce      uint64
	Data       string
	Signature  []byte
}

func (b *BlockData) GetHeight() uint64 {
	return b.Height
}

func (b *BlockData) GetHash() []byte {
	return b.Hash
}

type MempoolInterface interface {
	Add(tx interface{}) (interface{}, error)
}

func NewServer(config ServerConfig, bc BlockchainInterface, mp MempoolInterface, pm *PeerManager) *Server {
	if config.ListenAddr == "" {
		config.ListenAddr = ":9090"
	}
	if config.MaxConns <= 0 {
		config.MaxConns = 200
	}
	if config.MaxMsgSize <= 0 {
		config.MaxMsgSize = 4 << 20
	}

	s := &Server{
		config:       config,
		bc:           bc,
		mp:           mp,
		pm:           pm,
		chainID:      config.ChainID,
		nodeID:       config.NodeID,
		sem:          make(chan struct{}, config.MaxConns),
		done:         make(chan struct{}),
		pingInterval: 30 * time.Second,
		pingTimeout:  10 * time.Second,
	}

	s.handshakeConfig = NewProtocolHandshake(config.ChainID, config.RulesHash, config.NodeID, nil)

	return s
}

func (s *Server) SetPeerAPI(peerAPI *PeerAPI) {
	s.peerAPI = peerAPI
}

func (s *Server) PeerManager() *PeerManager {
	return s.pm
}

func (s *Server) ListenAddr() string {
	return s.config.ListenAddr
}

func (s *Server) Serve(ctx context.Context) error {
	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", s.config.ListenAddr)
	if err != nil {
		return err
	}
	s.listener = ln

	log.Printf("P2P listening on %s (nodeId=%s)", s.config.ListenAddr, s.nodeID)

	go s.connectToSeeds(ctx)
	go s.maintainPeers(ctx)

	for {
		c, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			case <-s.done:
				return nil
			default:
				return err
			}
		}

		select {
		case s.sem <- struct{}{}:
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				defer func() { <-s.sem }()
				_ = s.handleConn(c)
			}()
		default:
			_ = c.Close()
		}
	}
}

func (s *Server) maintainPeers(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			peerCount := s.pm.PeerCount()
			if peerCount < 3 {
				s.connectToSeeds(ctx)
			}
		}
	}
}

func (s *Server) connectToSeeds(ctx context.Context) {
	if s.config.Seeds == "" {
		return
	}

	seeds := strings.Split(s.config.Seeds, ",")
	for _, seed := range seeds {
		seed = strings.TrimSpace(seed)
		if seed == "" {
			continue
		}

		if seed == s.config.ListenAddr || seed == ":"+strings.TrimPrefix(s.config.ListenAddr, ":") {
			continue
		}

		go s.dialPeer(ctx, seed)
	}
}

func (s *Server) dialPeer(ctx context.Context, addr string) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		log.Printf("p2p dial failed %s: %v", addr, err)
		return
	}

	if err := s.performHandshake(conn, addr, true); err != nil {
		conn.Close()
		log.Printf("p2p handshake failed %s: %v", addr, err)
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer conn.Close()
		defer s.pm.RemovePeer(addr)
		s.readLoop(conn, addr, true)
	}()

	log.Printf("p2p outbound connection established to %s", addr)
}

func (s *Server) performHandshake(conn net.Conn, addr string, isOutgoing bool) error {
	hello := Hello{
		Protocol:  ProtocolVersion,
		ChainID:   s.chainID,
		RulesHash: s.config.RulesHash,
		NodeID:    s.nodeID,
		TimeUnix:  time.Now().Unix(),
		Port:      s.config.Port,
	}

	if err := WriteJSON(conn, Envelope{Type: "hello", Payload: MustJSON(hello)}); err != nil {
		return err
	}

	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))

	raw, err := ReadJSON(conn, 1<<20)
	if err != nil {
		return err
	}

	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	if env.Type != "hello" {
		return fmt.Errorf("expected hello, got %s", env.Type)
	}

	var helloResp Hello
	if err := json.Unmarshal(env.Payload, &helloResp); err != nil {
		return err
	}
	if helloResp.Protocol != 1 || helloResp.ChainID != s.chainID {
		return fmt.Errorf("wrong chain/protocol")
	}
	if s.config.RulesHash != "" && helloResp.RulesHash != s.config.RulesHash {
		return fmt.Errorf("rules hash mismatch")
	}

	peer := s.pm.AddPeer(addr, helloResp.NodeID, isOutgoing)
	if peer == nil {
		return fmt.Errorf("peer limit reached or banned")
	}

	if s.peerAPI != nil {
		s.peerAPI.addPeer(addr, conn)
	}

	return nil
}

func (s *Server) readLoop(conn net.Conn, addr string, isOutgoing bool) {
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(60 * time.Second))

	for {
		raw, err := ReadJSON(conn, s.config.MaxMsgSize)
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed") {
				log.Printf("p2p readLoop error from %s: %v", addr, err)
			}
			return
		}

		var env Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			log.Printf("p2p readLoop unmarshal error from %s: %v", addr, err)
			return
		}

		switch env.Type {
		case "chain_info_req":
			_ = s.handleChainInfo(conn)
		case "chain_info":
			_ = s.handleChainInfo(conn)
		case "headers_from_req":
			var req HeadersRequest
			if err := json.Unmarshal(env.Payload, &req); err != nil {
				return
			}
			_ = s.writeHeadersFrom(conn, req.From, req.Count)
		case "block_by_hash_req":
			var req BlockByHashReq
			if err := json.Unmarshal(env.Payload, &req); err != nil {
				return
			}
			_ = s.writeBlockByHash(conn, req.HashHex)
		case "tx_req":
			_ = s.handleTransactionReq(conn, env.Payload)
		case "tx_broadcast":
			_ = s.handleTransactionBroadcast(conn, env.Payload)
		case "block_broadcast":
			_ = s.handleBlockBroadcast(conn, env.Payload)
		case "block_req":
			_ = s.handleBlockReq(conn, env.Payload)
		case "ping":
			_ = s.handlePing(conn)
		case "getaddr":
			_ = s.handleGetAddr(conn)
		case "addr":
			_ = s.handleAddr(conn, env.Payload)
		default:
		}

		_ = conn.SetDeadline(time.Now().Add(60 * time.Second))
	}
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	close(s.done)
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) Wait() {
	s.wg.Wait()
}

func (s *Server) handleConn(c net.Conn) error {
	addr := c.RemoteAddr().String()

	_ = c.SetDeadline(time.Now().Add(15 * time.Second))

	raw, err := ReadJSON(c, 1<<20)
	if err != nil {
		c.Close()
		return err
	}
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		c.Close()
		return err
	}
	if env.Type != "hello" {
		c.Close()
		return errors.New("expected hello")
	}
	var hello Hello
	if err := json.Unmarshal(env.Payload, &hello); err != nil {
		c.Close()
		return err
	}
	if hello.Protocol != 1 || hello.ChainID != s.chainID {
		_ = WriteJSON(c, Envelope{Type: "error", Payload: MustJSON(map[string]any{"error": "wrong_chain_or_protocol"})})
		c.Close()
		return errors.New("wrong chain/protocol")
	}
	if strings.TrimSpace(hello.RulesHash) == "" || hello.RulesHash != s.config.RulesHash {
		_ = WriteJSON(c, Envelope{Type: "error", Payload: MustJSON(map[string]any{"error": "rules_hash_mismatch"})})
		c.Close()
		return errors.New("rules hash mismatch")
	}

	peer := s.pm.AddPeer(addr, hello.NodeID, false)
	if peer == nil {
		_ = WriteJSON(c, Envelope{Type: "error", Payload: MustJSON(map[string]any{"error": "peer_limit_reached"})})
		c.Close()
		return errors.New("peer limit reached")
	}

	peerAddr := addr

	_ = WriteJSON(c, Envelope{Type: "hello", Payload: MustJSON(NewHello(s.chainID, s.config.RulesHash, s.nodeID))})

	log.Printf("p2p inbound connection from %s", peerAddr)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer c.Close()
		defer s.pm.RemovePeer(peerAddr)
		s.readLoop(c, peerAddr, false)
	}()

	return nil
}

type BlockByHashReq struct {
	HashHex string `json:"hashHex"`
}

func (s *Server) handleChainInfo(conn net.Conn) error {
	return s.writeChainInfo(conn)
}

func (s *Server) writeChainInfo(w io.Writer) error {
	latest := s.bc.LatestBlock()
	genesis, _ := s.bc.BlockByHeight(0)
	peersCount := 0
	if s.pm != nil {
		peersCount = s.pm.PeerCount()
	}
	out := map[string]any{
		"chainId":              s.chainID,
		"rulesHash":            s.bc.RulesHashHex(),
		"height":               latest.GetHeight(),
		"latestHash":           fmt.Sprintf("%x", latest.GetHash()),
		"genesisHash":          fmt.Sprintf("%x", genesis.GetHash()),
		"genesisTimestampUnix": 0,
		"peersCount":           peersCount,
	}
	return WriteJSON(w, Envelope{Type: "chain_info", Payload: MustJSON(out)})
}

func (s *Server) writeHeadersFrom(w io.Writer, from uint64, count int) error {
	if count <= 0 || count > 500 {
		count = 100
	}
	headers := s.bc.HeadersFrom(from, count)
	return WriteJSON(w, Envelope{Type: "headers", Payload: MustJSON(headers)})
}

func (s *Server) writeBlockByHash(w io.Writer, hashHex string) error {
	hashHex = strings.TrimSpace(hashHex)
	if hashHex == "" {
		return WriteJSON(w, Envelope{Type: "error", Payload: MustJSON(map[string]any{"error": "missing_hash"})})
	}
	b, ok := s.bc.BlockByHash(hashHex)
	if !ok {
		return WriteJSON(w, Envelope{Type: "not_found", Payload: MustJSON(map[string]any{"hashHex": hashHex})})
	}
	blockData, ok := b.(*BlockData)
	if !ok {
		return WriteJSON(w, Envelope{Type: "error", Payload: MustJSON(map[string]any{"error": "invalid_block"})})
	}
	return WriteJSON(w, Envelope{Type: "block", Payload: MustJSON(blockData)})
}

func (s *Server) handleTransactionReq(c net.Conn, payload json.RawMessage) error {
	var req TXRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		_ = WriteJSON(c, Envelope{Type: "error", Payload: MustJSON(map[string]any{"error": "invalid_payload"})})
		return err
	}

	txid := computeHash([]byte(req.TxHex))

	if s.mp != nil {
		_, _ = s.mp.Add(req.TxHex)
	}

	return WriteJSON(c, Envelope{Type: "tx_ack", Payload: MustJSON(map[string]any{"txid": txid})})
}

func (s *Server) handleTransactionBroadcast(c net.Conn, payload json.RawMessage) error {
	var broadcast TransactionBroadcast
	if err := json.Unmarshal(payload, &broadcast); err != nil {
		return err
	}

	txid := computeHash([]byte(broadcast.TxHex))

	if s.mp != nil {
		_, _ = s.mp.Add(broadcast.TxHex)
	}

	return WriteJSON(c, Envelope{Type: "tx_broadcast_ack", Payload: MustJSON(map[string]any{"txid": txid})})
}

func (s *Server) handleBlockBroadcast(c net.Conn, payload json.RawMessage) error {
	var broadcast BlockBroadcast
	if err := json.Unmarshal(payload, &broadcast); err != nil {
		_ = WriteJSON(c, Envelope{Type: "error", Payload: MustJSON(map[string]any{"error": "invalid_block_json"})})
		return err
	}

	var block BlockData
	if err := json.Unmarshal([]byte(broadcast.BlockHex), &block); err != nil {
		_ = WriteJSON(c, Envelope{Type: "error", Payload: MustJSON(map[string]any{"error": "invalid_block_json"})})
		return err
	}

	if s.bc != nil {
		_, err := s.bc.AddBlock(&block)
		if err != nil {
			log.Printf("p2p block broadcast add result: %v", err)
		}
	}

	return WriteJSON(c, Envelope{Type: "block_broadcast_ack", Payload: MustJSON(map[string]any{"hash": fmt.Sprintf("%x", block.Hash)})})
}

func (s *Server) handleBlockReq(c net.Conn, payload json.RawMessage) error {
	var req GetBlockRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		_ = WriteJSON(c, Envelope{Type: "error", Payload: MustJSON(map[string]any{"error": "invalid_payload"})})
		return err
	}

	b, ok := s.bc.BlockByHash(req.HashHex)
	if !ok {
		_ = WriteJSON(c, Envelope{Type: "not_found", Payload: MustJSON(map[string]any{"hashHex": req.HashHex})})
		return nil
	}

	blockData, ok := b.(*BlockData)
	if !ok {
		_ = WriteJSON(c, Envelope{Type: "error", Payload: MustJSON(map[string]any{"error": "invalid_block"})})
		return nil
	}

	blockJSON, err := json.Marshal(blockData)
	if err != nil {
		_ = WriteJSON(c, Envelope{Type: "error", Payload: MustJSON(map[string]any{"error": "marshal_failed"})})
		return err
	}

	return WriteJSON(c, Envelope{Type: "block", Payload: blockJSON})
}

func (s *Server) handlePing(c net.Conn) error {
	return WriteJSON(c, Envelope{Type: "pong", Payload: nil})
}

func (s *Server) handleGetAddr(c net.Conn) error {
	if s.pm == nil {
		return nil
	}
	addrs := s.pm.GetPeerAddresses()
	type peerAddr struct {
		IP        string `json:"ip"`
		Port      int    `json:"port"`
		Timestamp int64  `json:"timestamp"`
	}
	var peerAddrs []peerAddr
	for _, addr := range addrs {
		host, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			continue
		}
		var port int
		fmt.Sscanf(portStr, "%d", &port)
		peerAddrs = append(peerAddrs, peerAddr{
			IP:        host,
			Port:      port,
			Timestamp: time.Now().Unix(),
		})
	}
	return WriteJSON(c, Envelope{Type: "addr", Payload: MustJSON(map[string]any{"addresses": peerAddrs})})
}

func (s *Server) handleAddr(c net.Conn, payload []byte) error {
	if s.pm == nil {
		return nil
	}
	type addrMsg struct {
		Addresses []struct {
			IP        string `json:"ip"`
			Port      int    `json:"port"`
			Timestamp int64  `json:"timestamp"`
		} `json:"addresses"`
	}
	var msg addrMsg
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil
	}
	for _, a := range msg.Addresses {
		addr := fmt.Sprintf("%s:%d", a.IP, a.Port)
		if addr != "" {
			s.pm.AddPeer(addr, "", true)
		}
	}
	return nil
}

func writeFramedMessage(conn net.Conn, msg *Message) error {
	var header [9]byte
	header[0] = 1
	binary.BigEndian.PutUint32(header[1:5], uint32(len(msg.Payload)))
	checksum := computeChecksum(msg.Payload)
	copy(header[5:9], checksum[:])

	if _, err := conn.Write(header[:]); err != nil {
		return err
	}
	if _, err := conn.Write(msg.Payload); err != nil {
		return err
	}
	return nil
}

func readFramedMessage(conn net.Conn) (*Message, error) {
	var header [9]byte
	if _, err := conn.Read(header[:]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(header[1:5])
	if length > 4<<20 {
		return nil, errors.New("message too large")
	}

	expectedChecksum := [4]byte{header[5], header[6], header[7], header[8]}

	payload := make([]byte, length)
	if _, err := conn.Read(payload); err != nil {
		return nil, err
	}

	checksum := computeChecksum(payload)
	if checksum[0] != expectedChecksum[0] || checksum[1] != expectedChecksum[1] ||
		checksum[2] != expectedChecksum[2] || checksum[3] != expectedChecksum[3] {
		return nil, errors.New("invalid checksum")
	}

	return &Message{
		Type:    MessageType(header[0]),
		Payload: payload,
	}, nil
}

func computeChecksum(data []byte) [4]byte {
	h := sha256HashBytes(data)
	h = sha256HashBytes(h)
	return [4]byte{h[0], h[1], h[2], h[3]}
}

func sha256HashBytes(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func sha256Hash(data string) string {
	h := sha256.Sum256([]byte(data))
	return string(h[:])
}
