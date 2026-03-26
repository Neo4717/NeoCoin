package blockchain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	ErrNoHeaders          = errors.New("no headers synced")
	ErrHeaderNotFound     = errors.New("header not found")
	ErrInvalidMerkleProof = errors.New("invalid merkle proof")
	ErrInvalidTxIndex     = errors.New("invalid transaction index")
	ErrSyncInProgress     = errors.New("sync already in progress")
)

type SPVConfig struct {
	ServerURL  string
	HTTPClient *http.Client
	MaxHeaders int
}

type SPVClient struct {
	cfg            SPVConfig
	headers        map[int64]*SPVHeader
	headersMu      sync.RWMutex
	bestHeight     int64
	bestHeader     *SPVHeader
	syncInProgress bool
	syncMu         sync.Mutex
	lastSyncTime   time.Time
	headerList     []int64
}

func NewSPVClient(cfg SPVConfig) *SPVClient {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	if cfg.MaxHeaders == 0 {
		cfg.MaxHeaders = 100000
	}

	return &SPVClient{
		cfg:        cfg,
		headers:    make(map[int64]*SPVHeader),
		bestHeight: 0,
	}
}

func (s *SPVClient) Connect(ctx context.Context) error {
	resp, err := s.httpGet(ctx, "/chain/info")
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *SPVClient) SyncHeaders(ctx context.Context) error {
	s.syncMu.Lock()
	if s.syncInProgress {
		s.syncMu.Unlock()
		return ErrSyncInProgress
	}
	s.syncInProgress = true
	s.syncMu.Unlock()

	defer func() {
		s.syncMu.Lock()
		s.syncInProgress = false
		s.syncMu.Unlock()
	}()

	startHeight := s.bestHeight + 1

	batchSize := int64(2000)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		endHeight := startHeight + batchSize - 1

		headers, err := s.fetchHeadersRange(ctx, startHeight, endHeight)
		if err != nil {
			return fmt.Errorf("fetch headers %d-%d: %w", startHeight, endHeight, err)
		}

		if len(headers) == 0 {
			break
		}

		for _, header := range headers {
			if err := s.validateAndAddHeader(header); err != nil {
				return fmt.Errorf("validate header at height %d: %w", header.Height, err)
			}
		}

		if int64(len(headers)) < batchSize {
			break
		}

		startHeight = endHeight + 1
	}

	s.lastSyncTime = time.Now()
	return nil
}

func (s *SPVClient) fetchHeadersRange(ctx context.Context, startHeight, endHeight int64) ([]*SPVHeader, error) {
	url := fmt.Sprintf("%s/headers/%d/%d", s.cfg.ServerURL, startHeight, endHeight)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var headers []*SPVHeader
	if err := json.NewDecoder(resp.Body).Decode(&headers); err != nil {
		return nil, fmt.Errorf("decode headers: %w", err)
	}

	return headers, nil
}

func (s *SPVClient) validateAndAddHeader(header *SPVHeader) error {
	if !header.ValidatePoW() {
		return fmt.Errorf("invalid PoW at height %d", header.Height)
	}

	if s.bestHeight > 0 {
		prevHeader := s.headers[s.bestHeight]
		if prevHeader != nil && header.PrevBlock != prevHeader.Hash {
			return fmt.Errorf("chain discontinuity at height %d", header.Height)
		}
	}

	s.headersMu.Lock()
	defer s.headersMu.Unlock()

	s.headers[header.Height] = header
	s.headerList = append(s.headerList, header.Height)
	s.bestHeight = header.Height
	s.bestHeader = header

	if len(s.headers) > s.cfg.MaxHeaders {
		s.pruneOldHeaders()
	}

	return nil
}

func (s *SPVClient) pruneOldHeaders() {
	if len(s.headerList) <= s.cfg.MaxHeaders/2 {
		return
	}

	toRemove := s.cfg.MaxHeaders / 4
	for i := 0; i < toRemove && i < len(s.headerList); i++ {
		height := s.headerList[i]
		delete(s.headers, height)
	}
	s.headerList = s.headerList[toRemove:]
}

func (s *SPVClient) GetHeader(height int64) (*SPVHeader, error) {
	s.headersMu.RLock()
	defer s.headersMu.RUnlock()

	header, ok := s.headers[height]
	if !ok {
		return nil, ErrHeaderNotFound
	}
	return header, nil
}

func (s *SPVClient) GetHeaderByHash(blockHash string) (*SPVHeader, error) {
	s.headersMu.RLock()
	defer s.headersMu.RUnlock()

	for _, h := range s.headers {
		if h.Hash == blockHash {
			return h, nil
		}
	}
	return nil, ErrHeaderNotFound
}

func (s *SPVClient) BestHeight() int64 {
	s.headersMu.RLock()
	defer s.headersMu.RUnlock()
	return s.bestHeight
}

func (s *SPVClient) BestHeader() *SPVHeader {
	s.headersMu.RLock()
	defer s.headersMu.RUnlock()
	return s.bestHeader
}

func (s *SPVClient) VerifyTxInBlock(ctx context.Context, txHash string, blockHeight int64) (*MerkleProofData, error) {
	header, err := s.GetHeader(blockHeight)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpGet(ctx, fmt.Sprintf("/merkle/%d", blockHeight))
	if err != nil {
		return nil, fmt.Errorf("fetch merkle block: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("merkle endpoint status %d", resp.StatusCode)
	}

	var merkleBlock struct {
		TxHashes []string `json:"txHashes"`
		Proof    []string `json:"proof"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&merkleBlock); err != nil {
		return nil, fmt.Errorf("decode merkle block: %w", err)
	}

	txIndex := -1
	for i, h := range merkleBlock.TxHashes {
		if h == txHash {
			txIndex = i
			break
		}
	}

	if txIndex < 0 {
		return nil, ErrInvalidTxIndex
	}

	proof := &MerkleProofData{
		BlockHash: header.Hash,
		TxIndex:   txIndex,
		TxHash:    txHash,
		Proof:     merkleBlock.Proof,
		Header:    header,
	}

	if !VerifyMerkleProof(proof, header.MerkleRoot) {
		return nil, ErrInvalidMerkleProof
	}

	return proof, nil
}

func (s *SPVClient) GetBalance(ctx context.Context, address string) (uint64, error) {
	resp, err := s.httpGet(ctx, fmt.Sprintf("/balance/%s", address))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("balance endpoint status %d", resp.StatusCode)
	}

	var result struct {
		Balance uint64 `json:"balance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode balance: %w", err)
	}

	return result.Balance, nil
}

func (s *SPVClient) GetSPVProof(ctx context.Context, address string, minDepth int64) (*SPVProof, error) {
	currentHeight := s.BestHeight()
	if currentHeight == 0 {
		return nil, ErrNoHeaders
	}

	balance, err := s.GetBalance(ctx, address)
	if err != nil {
		return nil, err
	}

	startHeight := currentHeight - minDepth
	if startHeight < 0 {
		startHeight = 0
	}

	var relevantHeaders []*SPVHeader
	s.headersMu.RLock()
	for h := startHeight; h <= currentHeight; h++ {
		if header, ok := s.headers[h]; ok {
			relevantHeaders = append(relevantHeaders, header)
		}
	}
	s.headersMu.RUnlock()

	headerHashes := make([]string, len(relevantHeaders))
	for i, h := range relevantHeaders {
		headerHashes[i] = h.Hash
	}

	headerProof := &HeaderProof{
		Headers: headerHashes,
	}

	return &SPVProof{
		Address:     address,
		Balance:     balance,
		HeaderProof: headerProof,
		BlockHeight: currentHeight,
		Timestamp:   s.bestHeader.TimestampUnix,
	}, nil
}

func (s *SPVClient) httpGet(ctx context.Context, path string) (*http.Response, error) {
	url := s.cfg.ServerURL + path
	return s.cfg.HTTPClient.Get(url)
}

func (s *SPVClient) GetHeadersCount() int {
	s.headersMu.RLock()
	defer s.headersMu.RUnlock()
	return len(s.headers)
}

func (s *SPVClient) LastSyncTime() time.Time {
	return s.lastSyncTime
}

type SPVProof struct {
	Address     string       `json:"address"`
	Balance     uint64       `json:"balance"`
	HeaderProof *HeaderProof `json:"headerProof"`
	BlockHeight int64        `json:"blockHeight"`
	Timestamp   int64        `json:"timestamp"`
}

type HeaderProof struct {
	Headers []string `json:"headers"`
}

func (s *SPVClient) IsSynced() bool {
	s.headersMu.RLock()
	defer s.headersMu.RUnlock()

	if s.bestHeader == nil {
		return false
	}

	age := time.Now().Unix() - s.bestHeader.TimestampUnix
	return age < 1200
}

func (s *SPVClient) GenerateAddressProof(ctx context.Context, address string, txHash string) (*AddressProof, error) {
	currentHeight := s.BestHeight()
	if currentHeight == 0 {
		return nil, ErrNoHeaders
	}

	header, err := s.GetHeader(currentHeight)
	if err != nil {
		return nil, err
	}

	txProof, err := s.VerifyTxInBlock(ctx, txHash, currentHeight)
	if err != nil {
		return nil, err
	}

	return &AddressProof{
		Address:     address,
		TxHash:      txHash,
		BlockHash:   header.Hash,
		BlockHeight: currentHeight,
		MerkleProof: txProof.Proof,
		TxIndex:     txProof.TxIndex,
		Timestamp:   header.TimestampUnix,
		MerkleRoot:  header.MerkleRoot,
	}, nil
}

type AddressProof struct {
	Address     string   `json:"address"`
	TxHash      string   `json:"txHash"`
	BlockHash   string   `json:"blockHash"`
	BlockHeight int64    `json:"blockHeight"`
	MerkleProof []string `json:"merkleProof"`
	TxIndex     int      `json:"txIndex"`
	Timestamp   int64    `json:"timestamp"`
	MerkleRoot  string   `json:"merkleRoot"`
}

func VerifyAddressProof(proof *AddressProof, merkleRoot string) bool {
	if proof == nil || len(proof.MerkleProof) == 0 {
		return false
	}

	hash, err := hex.DecodeString(proof.TxHash)
	if err != nil {
		return false
	}

	computedHash := Hash256(hash)
	idx := proof.TxIndex

	for _, step := range proof.MerkleProof {
		stepHash, err := hex.DecodeString(step)
		if err != nil {
			return false
		}

		if idx%2 == 0 {
			computedHash = Hash256(append(computedHash, stepHash...))
		} else {
			computedHash = Hash256(append(stepHash, computedHash...))
		}
		idx = idx / 2
	}

	computedRoot := hex.EncodeToString(computedHash)
	return computedRoot == merkleRoot
}
