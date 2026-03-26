package networking

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"
)

type ClientConfig struct {
	ChainID       uint64
	RulesHash     string
	NodeID        string
	DialTimeout   time.Duration
	IOTimeout     time.Duration
	MaxMsgBytes   int
	RetryInterval time.Duration
	MaxRetries    int
}

func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		DialTimeout:   5 * time.Second,
		IOTimeout:     10 * time.Second,
		MaxMsgBytes:   4 << 20,
		RetryInterval: 30 * time.Second,
		MaxRetries:    3,
	}
}

type Client struct {
	config ClientConfig

	dialer *net.Dialer
	conn   net.Conn
	mu     sync.Mutex

	peerInfo *ClientPeerInfo
	closed   bool
}

type ClientPeerInfo struct {
	NodeID   string
	Address  string
	Version  uint32
	ChainID  uint64
	Latency  time.Duration
	Score    float64
	LastSeen time.Time
}

func NewClient(config ClientConfig) *Client {
	if config.DialTimeout <= 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.IOTimeout <= 0 {
		config.IOTimeout = 10 * time.Second
	}
	if config.MaxMsgBytes <= 0 {
		config.MaxMsgBytes = 4 << 20
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}

	return &Client{
		config: config,
		dialer: &net.Dialer{Timeout: config.DialTimeout},
	}
}

func (c *Client) Connect(ctx context.Context, addr string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.New("client closed")
	}

	if c.conn != nil {
		c.conn.Close()
	}

	conn, err := c.dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	c.conn = conn
	return c.handshake(addr)
}

func (c *Client) handshake(addr string) error {
	_ = c.conn.SetDeadline(time.Now().Add(15 * time.Second))

	hello := NewHello(c.config.ChainID, c.config.RulesHash, c.config.NodeID)
	hello.TimeUnix = time.Now().Unix()

	if err := WriteJSON(c.conn, Envelope{Type: "hello", Payload: MustJSON(hello)}); err != nil {
		return err
	}

	raw, err := ReadJSON(c.conn, 1<<20)
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
	if helloResp.Protocol != 1 || helloResp.ChainID != c.config.ChainID {
		return errors.New("wrong chain/protocol")
	}
	if c.config.RulesHash != "" && helloResp.RulesHash != c.config.RulesHash {
		return errors.New("rules hash mismatch")
	}

	c.peerInfo = &ClientPeerInfo{
		NodeID:   helloResp.NodeID,
		Address:  addr,
		Version:  helloResp.Protocol,
		ChainID:  helloResp.ChainID,
		Score:    1.0,
		LastSeen: time.Now(),
	}

	_ = c.conn.SetDeadline(time.Now().Add(c.config.IOTimeout))
	return nil
}

func (c *Client) do(ctx context.Context, reqType string, reqPayload any, resp any, expectedRespType string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return errors.New("not connected")
	}

	var payload []byte
	if reqPayload != nil {
		payload = MustJSON(reqPayload)
	}
	if err := WriteJSON(c.conn, Envelope{Type: reqType, Payload: payload}); err != nil {
		return err
	}

	raw, err := ReadJSON(c.conn, c.config.MaxMsgBytes)
	if err != nil {
		return err
	}

	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}

	if expectedRespType != "" && env.Type != expectedRespType {
		if env.Type == "not_found" {
			return errors.New("not found")
		}
		if env.Type == "error" {
			var errResp map[string]any
			json.Unmarshal(env.Payload, &errResp)
			if errMsg, ok := errResp["error"]; ok {
				return errors.New(errMsg.(string))
			}
			return errors.New("unknown error")
		}
		return errors.New("unexpected response type: " + env.Type)
	}

	if resp != nil {
		return json.Unmarshal(env.Payload, resp)
	}
	return nil
}

func (c *Client) RequestChainInfo(ctx context.Context) (*ChainInfo, error) {
	var out ChainInfo
	if err := c.do(ctx, "chain_info_req", nil, &out, "chain_info"); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RequestHeaders(ctx context.Context, from uint64, count int) ([]BlockHeader, error) {
	var out []BlockHeader
	if err := c.do(ctx, "headers_from_req", HeadersRequest{From: from, Count: count}, &out, "headers"); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) RequestBlockByHash(ctx context.Context, hashHex string) ([]byte, error) {
	var out []byte
	err := c.do(ctx, "block_by_hash_req", BlockByHashReq{HashHex: hashHex}, &out, "block")
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) RequestTransaction(ctx context.Context, txHex string) (string, error) {
	var resp p2pTxResponse
	err := c.do(ctx, "tx_req", TXRequest{TxHex: txHex}, &resp, "tx_ack")
	if err != nil {
		return "", err
	}
	return resp.TxID, nil
}

func (c *Client) BroadcastTransaction(ctx context.Context, txHex string) (string, error) {
	var resp p2pTxResponse
	err := c.do(ctx, "tx_broadcast", TransactionBroadcast{TxHex: txHex}, &resp, "tx_broadcast_ack")
	if err != nil {
		return "", err
	}
	return resp.TxID, nil
}

func (c *Client) BroadcastBlock(ctx context.Context, blockHex string) (string, error) {
	var resp p2pBlockResponse
	err := c.do(ctx, "block_broadcast", BlockBroadcast{BlockHex: blockHex}, &resp, "block_broadcast_ack")
	if err != nil {
		return "", err
	}
	return resp.Hash, nil
}

func (c *Client) Ping(ctx context.Context) error {
	start := time.Now()
	err := c.do(ctx, "ping", nil, nil, "pong")
	if err != nil {
		return err
	}
	if c.peerInfo != nil {
		c.peerInfo.Latency = time.Since(start)
	}
	return nil
}

func (c *Client) PeerInfo() *ClientPeerInfo {
	return c.peerInfo
}

func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

type SecureClient struct {
	config ClientConfig
	client *Client

	nodeID  string
	pubKey  []byte
	privKey []byte
}

func NewSecureClient(config ClientConfig, nodeID string, pubKey, privKey []byte) *SecureClient {
	return &SecureClient{
		config:  config,
		client:  NewClient(config),
		nodeID:  nodeID,
		pubKey:  pubKey,
		privKey: privKey,
	}
}

func (sc *SecureClient) Connect(ctx context.Context, addr string) error {
	return sc.client.Connect(ctx, addr)
}

func (sc *SecureClient) handshake(addr string) error {
	_ = sc.client.conn.SetDeadline(time.Now().Add(15 * time.Second))

	hello := NewHello(sc.config.ChainID, sc.config.RulesHash, sc.config.NodeID)
	hello.TimeUnix = time.Now().Unix()

	if err := WriteJSON(sc.client.conn, Envelope{Type: "hello", Payload: MustJSON(hello)}); err != nil {
		return err
	}

	raw, err := ReadJSON(sc.client.conn, 1<<20)
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

	sc.client.peerInfo = &ClientPeerInfo{
		NodeID:  helloResp.NodeID,
		Address: addr,
		Version: helloResp.Protocol,
		ChainID: helloResp.ChainID,
		Score:   1.0,
	}

	_ = sc.client.conn.SetDeadline(time.Now().Add(sc.config.IOTimeout))
	return nil
}

func (sc *SecureClient) RequestChainInfo(ctx context.Context) (*ChainInfo, error) {
	return sc.client.RequestChainInfo(ctx)
}

func (sc *SecureClient) RequestHeaders(ctx context.Context, from uint64, count int) ([]BlockHeader, error) {
	return sc.client.RequestHeaders(ctx, from, count)
}

func (sc *SecureClient) RequestBlockByHash(ctx context.Context, hashHex string) ([]byte, error) {
	return sc.client.RequestBlockByHash(ctx, hashHex)
}

func (sc *SecureClient) BroadcastTransaction(ctx context.Context, txHex string) (string, error) {
	return sc.client.BroadcastTransaction(ctx, txHex)
}

func (sc *SecureClient) BroadcastBlock(ctx context.Context, blockHex string) (string, error) {
	return sc.client.BroadcastBlock(ctx, blockHex)
}

func (sc *SecureClient) Ping(ctx context.Context) error {
	return sc.client.Ping(ctx)
}

func (sc *SecureClient) PeerInfo() *ClientPeerInfo {
	return sc.client.PeerInfo()
}

func (sc *SecureClient) IsConnected() bool {
	return sc.client.IsConnected()
}

func (sc *SecureClient) Close() error {
	return sc.client.Close()
}

type ConnectionPool struct {
	clients  map[string]*Client
	mu       sync.RWMutex
	config   ClientConfig
	maxConns int
}

func NewConnectionPool(config ClientConfig, maxConns int) *ConnectionPool {
	if maxConns <= 0 {
		maxConns = 50
	}
	return &ConnectionPool{
		clients:  make(map[string]*Client),
		config:   config,
		maxConns: maxConns,
	}
}

func (cp *ConnectionPool) GetClient(addr string) (*Client, bool) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	client, ok := cp.clients[addr]
	return client, ok
}

func (cp *ConnectionPool) AddClient(addr string, client *Client) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if len(cp.clients) >= cp.maxConns {
		return
	}
	cp.clients[addr] = client
}

func (cp *ConnectionPool) RemoveClient(addr string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if client, ok := cp.clients[addr]; ok {
		client.Close()
		delete(cp.clients, addr)
	}
}

func (cp *ConnectionPool) CloseAll() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	for _, client := range cp.clients {
		client.Close()
	}
	cp.clients = make(map[string]*Client)
}

func (cp *ConnectionPool) Count() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return len(cp.clients)
}

func RetryWithBackoff(ctx context.Context, fn func() error, maxRetries int, interval time.Duration) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		if err = fn(); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval * time.Duration(i+1)):
		}
	}
	return err
}

func (c *Client) RequestBlock(ctx context.Context, hashHex string) ([]byte, error) {
	var block []byte
	err := c.do(ctx, "block_req", GetBlockRequest{HashHex: hashHex}, &block, "block")
	if err != nil {
		return nil, err
	}
	return block, nil
}
