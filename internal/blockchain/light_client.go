package blockchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Neo4717/NeoCoin/config"
)

type FilterType int

const (
	FilterNone FilterType = iota
	FilterBasic
	FilterFull
)

type FilterEntry struct {
	script []byte
}

type BloomFilter struct {
	data      []byte
	hashFuncs uint32
	tweak     uint32
	matched   map[string]bool
	mu        sync.RWMutex
}

func NewBloomFilter(size uint32, hashFuncs uint32, tweak uint32) *BloomFilter {
	return &BloomFilter{
		data:      make([]byte, size),
		hashFuncs: hashFuncs,
		tweak:     tweak,
		matched:   make(map[string]bool),
	}
}

func (f *BloomFilter) Add(elem []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i := uint32(0); i < f.hashFuncs; i++ {
		h := f.sipHash(elem, i)
		f.data[h%uint32(len(f.data)*8)] = 1
	}
}

func (f *BloomFilter) Contains(elem []byte) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for i := uint32(0); i < f.hashFuncs; i++ {
		h := f.sipHash(elem, i)
		if f.data[h%uint32(len(f.data)*8)] == 0 {
			return false
		}
	}
	return true
}

func (f *BloomFilter) sipHash(data []byte, i uint32) uint32 {
	h := uint64(f.tweak + i*0xfba4c795)
	for j := 0; j < len(data); j++ {
		h = h*0x100000001b3 + uint64(data[j])
	}
	return uint32(h ^ (h >> 32))
}

func (f *BloomFilter) Matched() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make([]string, 0, len(f.matched))
	for k := range f.matched {
		result = append(result, k)
	}
	return result
}

func (f *BloomFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.data = make([]byte, len(f.data))
	f.matched = make(map[string]bool)
}

type SPVHeader struct {
	Version       uint32 `json:"version"`
	Height        int64  `json:"height"`
	TimestampUnix int64  `json:"timestampUnix"`
	PrevBlock     string `json:"prevBlock"`
	Difficulty    int64  `json:"difficulty"`
	Nonce         uint64 `json:"nonce"`
	MerkleRoot    string `json:"merkleRoot"`
	Hash          string `json:"hash"`
}

func (h *SPVHeader) HashHex() string {
	return h.Hash
}

func (h *SPVHeader) ValidatePoW() bool {
	if h.Hash == "" {
		return false
	}
	hashBytes, err := hex.DecodeString(h.Hash)
	if err != nil || len(hashBytes) != 32 {
		return false
	}
	target := calcTarget(h.Difficulty)
	hashInt := bytesToUint256(hashBytes)
	return bytesCompare(hashInt, target) <= 0
}

func bytesToUint256(b []byte) []byte {
	if len(b) < 32 {
		result := make([]byte, 32)
		copy(result[32-len(b):], b)
		return result
	}
	return b[:32]
}

func bytesCompare(a, b []byte) int {
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	for i := range a {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func calcTarget(difficulty int64) []byte {
	if difficulty <= 0 {
		difficulty = 1
	}
	exp := 256 - (difficulty / 8)
	if exp < 0 {
		exp = 0
	}
	if exp > 256 {
		exp = 256
	}
	target := make([]byte, 32)
	if exp < 32 {
		target[32-exp] = 1
	} else {
		target[0] = 1 << (exp - 32)
	}
	return target
}

type LightClient struct {
	serverURL         string
	httpClient        *http.Client
	chainMu           sync.RWMutex
	headers           []*SPVHeader
	bestHeight        int64
	bestHeader        *SPVHeader
	trustedBlock      *Block
	filterMu          sync.RWMutex
	filter            *BloomFilter
	filterUpdateMu    sync.RWMutex
	needsFilterUpdate bool
	filterParams      BloomFilterParams
	syncMu            sync.Mutex
	syncing           atomic.Bool
	lastSyncTime      time.Time
	notifications     chan SPVNotification
	stopCh            chan struct{}
	cfg               *config.Config
}

type BloomFilterParams struct {
	FilterSize uint32
	HashFuncs  uint32
	Tweak      uint32
}

type SPVNotification struct {
	Type   string
	TxHash string
	Block  *SPVHeader
	Data   interface{}
}

const (
	DefaultFilterSize = 30000
	DefaultHashFuncs  = 5
	DefaultTweak      = 0
)

func NewLightClient(serverURL string, cfg *config.Config) *LightClient {
	lc := &LightClient{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		bestHeight: 0,
		filterParams: BloomFilterParams{
			FilterSize: DefaultFilterSize,
			HashFuncs:  DefaultHashFuncs,
			Tweak:      DefaultTweak,
		},
		notifications: make(chan SPVNotification, 100),
		stopCh:        make(chan struct{}),
		cfg:           cfg,
	}

	lc.filter = NewBloomFilter(
		lc.filterParams.FilterSize,
		lc.filterParams.HashFuncs,
		lc.filterParams.Tweak,
	)

	return lc
}

func (lc *LightClient) Connect(ctx context.Context) error {
	resp, err := lc.httpGet("/chain/info")
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return nil
}

func (lc *LightClient) SyncHeaders(ctx context.Context) error {
	lc.syncMu.Lock()
	defer lc.syncMu.Unlock()

	if lc.syncing.Load() {
		return fmt.Errorf("sync already in progress")
	}
	lc.syncing.Store(true)
	defer lc.syncing.Store(false)

	startHeight := int64(0)
	lc.chainMu.RLock()
	if len(lc.headers) > 0 {
		startHeight = lc.headers[len(lc.headers)-1].Height + 1
	}
	lc.chainMu.RUnlock()

	batchSize := int64(2000)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		endHeight := startHeight + batchSize - 1

		headers, err := lc.fetchHeadersRange(startHeight, endHeight)
		if err != nil {
			return fmt.Errorf("fetch headers %d-%d: %w", startHeight, endHeight, err)
		}

		if len(headers) == 0 {
			break
		}

		if err := lc.validateAndAddHeaders(headers); err != nil {
			return fmt.Errorf("validate headers: %w", err)
		}

		if int64(len(headers)) < batchSize {
			break
		}

		startHeight = endHeight + 1
	}

	lc.lastSyncTime = time.Now()
	return nil
}

func (lc *LightClient) fetchHeadersRange(startHeight, endHeight int64) ([]*SPVHeader, error) {
	resp, err := lc.httpGet(fmt.Sprintf("/headers/%d/%d", startHeight, endHeight))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var headers []*SPVHeader
	if err := decodeJSON(resp.Body, &headers); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return headers, nil
}

func (lc *LightClient) validateAndAddHeaders(headers []*SPVHeader) error {
	lc.chainMu.Lock()
	defer lc.chainMu.Unlock()

	for _, header := range headers {
		if !header.ValidatePoW() {
			return fmt.Errorf("invalid PoW at height %d", header.Height)
		}

		if len(lc.headers) > 0 {
			prev := lc.headers[len(lc.headers)-1]
			if header.PrevBlock != prev.Hash {
				return fmt.Errorf("chain discontinuity at height %d", header.Height)
			}
		} else if lc.trustedBlock != nil {
			if header.PrevBlock != HashHex(lc.trustedBlock.Hash) {
				return fmt.Errorf("doesn't connect to trusted block")
			}
		}

		if header.Height > 0 && header.Height%2016 == 0 {
			if !lc.verifyDifficulty(header, lc.headers) {
				return fmt.Errorf("difficulty adjustment failed at %d", header.Height)
			}
		}

		lc.headers = append(lc.headers, header)

		if header.Height > lc.bestHeight {
			lc.bestHeight = header.Height
			lc.bestHeader = header
		}
	}

	return nil
}

func (lc *LightClient) verifyDifficulty(header *SPVHeader, prevHeaders []*SPVHeader) bool {
	if len(prevHeaders) < 2015 {
		return true
	}

	first := prevHeaders[len(prevHeaders)-2016]
	last := prevHeaders[len(prevHeaders)-1]

	actualTime := last.TimestampUnix - first.TimestampUnix
	targetTime := int64(2016 * 600)

	if actualTime < targetTime/4 {
		actualTime = targetTime / 4
	}
	if actualTime > targetTime*4 {
		actualTime = targetTime * 4
	}

	expectedDiff := calcNewDifficulty(prevHeaders[len(prevHeaders)-1].Difficulty, actualTime, targetTime)
	return header.Difficulty == expectedDiff
}

func calcNewDifficulty(prevDiff int64, actualTime, targetTime int64) int64 {
	ratio := float64(targetTime) / float64(actualTime)
	if ratio < 0.25 {
		ratio = 0.25
	}
	if ratio > 4.0 {
		ratio = 4.0
	}

	newDiff := int64(float64(prevDiff) * ratio)
	if newDiff < 1 {
		newDiff = 1
	}

	maxDiff := prevDiff * 4
	minDiff := prevDiff / 4
	if newDiff > maxDiff {
		newDiff = maxDiff
	}
	if newDiff < minDiff {
		newDiff = minDiff
	}

	return newDiff
}

func (lc *LightClient) SetFilter(addrs []string, keys [][]byte) error {
	lc.filterMu.Lock()
	defer lc.filterMu.Unlock()

	for _, addr := range addrs {
		lc.filter.Add([]byte(addr))
	}

	for _, key := range keys {
		lc.filter.Add(key)
		h := sha256.Sum256(key)
		lc.filter.Add(h[:])
	}

	lc.needsFilterUpdate = true
	return nil
}

func (lc *LightClient) SubscribeAddress(addr string) error {
	lc.filterMu.Lock()
	defer lc.filterMu.Unlock()

	lc.filter.Add([]byte(addr))
	lc.needsFilterUpdate = true
	return nil
}

func (lc *LightClient) RescanBlockRange(ctx context.Context, startHeight, endHeight int64, foundCb func(*Transaction, *SPVHeader)) error {
	for h := startHeight; h <= endHeight; h++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		merkle, err := lc.fetchMerkleBlock(h)
		if err != nil {
			block, err := lc.fetchBlock(h)
			if err != nil {
				continue
			}
			spvHeader := blockToSPVHeader(block)
			for i := range block.Transactions {
				if lc.filterTx(&block.Transactions[i]) {
					foundCb(&block.Transactions[i], spvHeader)
				}
			}
			continue
		}

		for _, txid := range merkle.MatchedHashes {
			tx, err := lc.fetchTX(txid)
			if err != nil {
				continue
			}
			if lc.filterTx(tx) {
				foundCb(tx, merkle.Header)
			}
		}
	}

	return nil
}

func (lc *LightClient) filterTx(tx *Transaction) bool {
	lc.filterMu.RLock()
	defer lc.filterMu.RUnlock()

	fromAddr, _ := tx.FromAddress()
	if lc.filter.Contains([]byte(fromAddr)) || lc.filter.Contains([]byte(tx.ToAddress)) {
		return true
	}

	if len(tx.Signature) > 0 {
		if lc.filter.Contains(tx.Signature) {
			return true
		}
	}

	return false
}

func (lc *LightClient) fetchMerkleBlock(height int64) (*MerkleBlock, error) {
	resp, err := lc.httpGet(fmt.Sprintf("/merkle/%d", height))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var merkle MerkleBlock
	if err := decodeJSON(resp.Body, &merkle); err != nil {
		return nil, err
	}

	return &merkle, nil
}

func (lc *LightClient) fetchTX(txid string) (*Transaction, error) {
	resp, err := lc.httpGet(fmt.Sprintf("/tx/%s", txid))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var tx Transaction
	if err := decodeJSON(resp.Body, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

func (lc *LightClient) fetchBlock(height int64) (*Block, error) {
	resp, err := lc.httpGet(fmt.Sprintf("/block/height/%d", height))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var block Block
	if err := decodeJSON(resp.Body, &block); err != nil {
		return nil, err
	}

	return &block, nil
}

func (lc *LightClient) httpGet(path string) (*http.Response, error) {
	url := lc.serverURL + path
	return lc.httpClient.Get(url)
}

func (lc *LightClient) CreateMerkleProof(tx *Transaction) (*MerkleProofData, error) {
	lc.chainMu.RLock()
	defer lc.chainMu.RUnlock()

	var block *Block
	for _, h := range lc.headers {
		resp, err := lc.httpGet(fmt.Sprintf("/block/height/%d", h.Height))
		if err != nil {
			continue
		}
		var b Block
		if err := decodeJSON(resp.Body, &b); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for i := range b.Transactions {
			txID2, _ := TxIDHex(b.Transactions[i])
			txID, _ := TxIDHex(*tx)
			if txID2 == txID {
				block = &b
				break
			}
		}
		if block != nil {
			break
		}
	}

	if block == nil {
		return nil, fmt.Errorf("tx not found in headers")
	}

	txs := make([]*Transaction, len(block.Transactions))
	for i := range block.Transactions {
		txs[i] = &block.Transactions[i]
	}
	return CreateMerkleProof(txs, tx), nil
}

func (lc *LightClient) VerifyMerkleProof(proof *MerkleProofData, blockHash string) bool {
	lc.chainMu.RLock()
	defer lc.chainMu.RUnlock()

	var header *SPVHeader
	for _, h := range lc.headers {
		if h.Hash == blockHash {
			header = h
			break
		}
	}

	if header == nil {
		return false
	}

	return VerifyMerkleProof(proof, header.MerkleRoot)
}

type MerkleBlock struct {
	Header        *SPVHeader `json:"header"`
	TotalTxs      int        `json:"totalTxs"`
	MatchedHashes []string   `json:"matchedHashes"`
	Hashes        []string   `json:"hashes"`
	Flags         []byte     `json:"flags"`
}

type MerkleProofData struct {
	BlockHash string     `json:"blockHash"`
	TxIndex   int        `json:"txIndex"`
	TxHash    string     `json:"txHash"`
	Proof     []string   `json:"proof"`
	Header    *SPVHeader `json:"header"`
}

func (lc *LightClient) Notifications() <-chan SPVNotification {
	return lc.notifications
}

func (lc *LightClient) Stop() {
	close(lc.stopCh)
}

func (lc *LightClient) BestHeight() int64 {
	lc.chainMu.RLock()
	defer lc.chainMu.RUnlock()
	return lc.bestHeight
}

func (lc *LightClient) BestHeader() *SPVHeader {
	lc.chainMu.RLock()
	defer lc.chainMu.RUnlock()
	return lc.bestHeader
}

func (lc *LightClient) IsSynced() bool {
	lc.chainMu.RLock()
	defer lc.chainMu.RUnlock()

	if lc.bestHeader == nil {
		return false
	}

	age := time.Now().Unix() - lc.bestHeader.TimestampUnix
	return age < 1200
}

func (lc *LightClient) GetBalance(addr string) (int64, error) {
	resp, err := lc.httpGet(fmt.Sprintf("/balance/%s", addr))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}

	var result struct {
		Balance int64 `json:"balance"`
	}
	if err := decodeJSON(resp.Body, &result); err != nil {
		return 0, err
	}

	return result.Balance, nil
}

func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func Hash256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func blockToSPVHeader(b *Block) *SPVHeader {
	return &SPVHeader{
		Version:       b.Version,
		Height:        int64(b.Height),
		TimestampUnix: b.TimestampUnix,
		PrevBlock:     hex.EncodeToString(b.PrevHash),
		Difficulty:    int64(b.DifficultyBits),
		Nonce:         b.Nonce,
		MerkleRoot:    "",
		Hash:          hex.EncodeToString(b.Hash),
	}
}

func HashHex(hash []byte) string {
	return hex.EncodeToString(hash)
}
