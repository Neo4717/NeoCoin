package http

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	apws "github.com/Neo4717/NeoCoin/api/websocket"
	"github.com/Neo4717/NeoCoin/internal/blockchain"
	"github.com/Neo4717/NeoCoin/internal/mempool"
	"github.com/Neo4717/NeoCoin/internal/wallet"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const relayHopsHeader = "X-Relay-Hops"

var version = "dev"
var buildTime = "unknown"

var minFee = uint64(1)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

func (s *Server) handleChainInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	latest := s.bc.LatestBlock()
	genesis, _ := s.bc.BlockByHeight(0)
	chainWork := s.bc.CanonicalWork().String()
	policy := s.bc.Consensus().MonetaryPolicy
	nextHeight := latest.Height + 1
	currentReward := policy.BlockReward(nextHeight)
	nextHalving := uint64(0)
	if policy.HalvingInterval > 0 {
		nextHalving = (latest.Height/policy.HalvingInterval + 1) * policy.HalvingInterval
	}
	totalSupply := s.bc.TotalSupply()
	out := map[string]any{
		"version":                        version,
		"buildTime":                      buildTime,
		"chainId":                        s.bc.ChainID,
		"rulesHash":                      s.bc.RulesHashHex(),
		"height":                         latest.Height,
		"latestHash":                     fmt.Sprintf("%x", latest.Hash),
		"genesisHash":                    fmt.Sprintf("%x", genesis.Hash),
		"genesisTimestampUnix":           genesis.TimestampUnix,
		"genesisMinerAddress":            genesis.MinerAddress,
		"minerAddress":                   s.bc.MinerAddress,
		"peersCount":                     0,
		"chainWork":                      chainWork,
		"totalSupply":                    totalSupply,
		"currentReward":                  currentReward,
		"nextHalvingHeight":              nextHalving,
		"difficultyBits":                 latest.DifficultyBits,
		"nextDifficultyBits":             s.bc.NextDifficultyBits(),
		"difficultyEnable":               s.bc.Consensus().DifficultyEnable,
		"difficultyTargetMs":             int64(s.bc.Consensus().TargetBlockTime / time.Millisecond),
		"difficultyWindow":               s.bc.Consensus().DifficultyWindow,
		"difficultyMinBits":              s.bc.Consensus().MinDifficultyBits,
		"difficultyMaxBits":              s.bc.Consensus().MaxDifficultyBits,
		"difficultyMaxStepBits":          s.bc.Consensus().DifficultyMaxStep,
		"maxBlockSize":                   s.bc.Consensus().MaxBlockSize,
		"maxTimeDrift":                   s.bc.Consensus().MaxTimeDrift,
		"merkleEnable":                   s.bc.Consensus().MerkleEnable,
		"merkleActivationHeight":         s.bc.Consensus().MerkleActivationHeight,
		"binaryEncodingEnable":           s.bc.Consensus().BinaryEncodingEnable,
		"binaryEncodingActivationHeight": s.bc.Consensus().BinaryEncodingActivationHeight,
		"monetaryPolicy": map[string]any{
			"initialBlockReward": policy.InitialBlockReward,
			"halvingInterval":    policy.HalvingInterval,
			"minerFeeShare":      policy.MinerFeeShare,
			"tailEmission":       policy.TailEmission,
		},
		"consensusParams": map[string]any{
			"difficultyEnable":               s.bc.Consensus().DifficultyEnable,
			"difficultyTargetMs":             int64(s.bc.Consensus().TargetBlockTime / time.Millisecond),
			"difficultyWindow":               s.bc.Consensus().DifficultyWindow,
			"difficultyMinBits":              s.bc.Consensus().MinDifficultyBits,
			"difficultyMaxBits":              s.bc.Consensus().MaxDifficultyBits,
			"difficultyMaxStepBits":          s.bc.Consensus().DifficultyMaxStep,
			"medianTimePastWindow":           s.bc.Consensus().MedianTimePastWindow,
			"maxTimeDrift":                   s.bc.Consensus().MaxTimeDrift,
			"maxBlockSize":                   s.bc.Consensus().MaxBlockSize,
			"merkleEnable":                   s.bc.Consensus().MerkleEnable,
			"merkleActivationHeight":         s.bc.Consensus().MerkleActivationHeight,
			"binaryEncodingEnable":           s.bc.Consensus().BinaryEncodingEnable,
			"binaryEncodingActivationHeight": s.bc.Consensus().BinaryEncodingActivationHeight,
		},
	}
	_ = writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleHeadersFrom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hStr := strings.TrimPrefix(r.URL.Path, "/headers/from/")
	h, err := strconv.ParseUint(hStr, 10, 64)
	if err != nil {
		http.Error(w, "bad height", http.StatusBadRequest)
		return
	}
	count := 100
	if q := r.URL.Query().Get("count"); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			count = n
		}
	}
	_ = writeJSON(w, http.StatusOK, s.bc.HeadersFrom(h, count))
}

func (s *Server) handleBlocksFrom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hStr := strings.TrimPrefix(r.URL.Path, "/blocks/from/")
	h, err := strconv.ParseUint(hStr, 10, 64)
	if err != nil {
		http.Error(w, "bad height", http.StatusBadRequest)
		return
	}
	count := 20
	if q := r.URL.Query().Get("count"); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			count = n
		}
	}
	_ = writeJSON(w, http.StatusOK, s.bc.BlocksFrom(h, count))
}

func (s *Server) handleBlockByHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hashHex := strings.TrimPrefix(r.URL.Path, "/blocks/hash/")
	if hashHex == "" {
		http.Error(w, "missing hash", http.StatusBadRequest)
		return
	}
	b, ok := s.bc.BlockByHash(hashHex)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	_ = writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleBlockByHeight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hStr := strings.TrimPrefix(r.URL.Path, "/block/height/")
	h, err := strconv.ParseUint(hStr, 10, 64)
	if err != nil {
		http.Error(w, "bad height", http.StatusBadRequest)
		return
	}
	b, ok := s.bc.BlockByHeight(h)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	_ = writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleTxByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	txid := strings.TrimPrefix(r.URL.Path, "/tx/")
	if txid == "" {
		http.Error(w, "missing txid", http.StatusBadRequest)
		return
	}
	tx, loc, ok := s.bc.TxByID(txid)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{
		"txId":        txid,
		"transaction": tx,
		"location":    loc,
	})
}

func (s *Server) handleTxProof(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	txid := strings.TrimPrefix(r.URL.Path, "/tx/proof/")
	if txid == "" {
		http.Error(w, "missing txid", http.StatusBadRequest)
		return
	}

	s.bc.Mu().RLock()
	var (
		foundBlock *blockchain.Block
		foundIndex int
		blockHash  string
	)
	for _, b := range s.bc.Blocks() {
		for i, tx := range b.Transactions {
			id, err := blockchain.TxIDHexForConsensus(tx, s.bc.Consensus(), b.Height)
			if err != nil {
				continue
			}
			if id == txid {
				foundBlock = b
				foundIndex = i
				blockHash = fmt.Sprintf("%x", b.Hash)
				break
			}
		}
		if foundBlock != nil {
			break
		}
	}
	s.bc.Mu().RUnlock()

	if foundBlock == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if foundBlock.Version != 2 {
		http.Error(w, "merkle proofs are only available for v2 blocks", http.StatusConflict)
		return
	}

	leaves := make([][]byte, 0, len(foundBlock.Transactions))
	for _, tx := range foundBlock.Transactions {
		h, err := blockchain.TxSigningHashForConsensus(tx, s.bc.Consensus(), foundBlock.Height)
		if err != nil {
			http.Error(w, "tx hash failed", http.StatusInternalServerError)
			return
		}
		leaves = append(leaves, h)
	}

	branch, siblingLeft, root, err := blockchain.MerkleProofFromLeaves(leaves, foundIndex)
	if err != nil {
		http.Error(w, "proof failed", http.StatusInternalServerError)
		return
	}
	branchHex := make([]string, 0, len(branch))
	for _, h := range branch {
		branchHex = append(branchHex, fmt.Sprintf("%x", h))
	}

	_ = writeJSON(w, http.StatusOK, map[string]any{
		"txId":        txid,
		"blockHeight": foundBlock.Height,
		"blockHash":   blockHash,
		"txIndex":     foundIndex,
		"merkleRoot":  fmt.Sprintf("%x", root),
		"branch":      branchHex,
		"siblingLeft": siblingLeft,
	})
}

func (s *Server) handleAddBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var b blockchain.Block
	if err := json.Unmarshal(body, &b); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	reorged, err := s.bc.AddBlock(&b)
	if err != nil {
		_ = writeJSON(w, http.StatusBadRequest, map[string]any{"accepted": false, "message": err.Error()})
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"accepted": true, "reorged": reorged})
}

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	addr := strings.TrimPrefix(r.URL.Path, "/balance/")
	if addr == "" {
		http.Error(w, "missing address", http.StatusBadRequest)
		return
	}
	acct, ok := s.bc.Balance(addr)
	if !ok {
		acct = blockchain.Account{}
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"address": addr, "balance": acct.Balance, "nonce": acct.Nonce})
}

func (s *Server) handleAddressTxs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/address/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "txs" {
		http.Error(w, "expected /address/{addr}/txs", http.StatusBadRequest)
		return
	}
	addr := parts[0]
	if err := blockchain.ValidateAddress(addr); err != nil {
		http.Error(w, "invalid address", http.StatusBadRequest)
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}
	cursor := 0
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			cursor = n
		}
	}

	txs, nextCursor, more := s.bc.AddressTxs(addr, limit, cursor)
	_ = writeJSON(w, http.StatusOK, map[string]any{
		"address":    addr,
		"txs":        txs,
		"nextCursor": nextCursor,
		"more":       more,
	})
}

type submitTxResponse struct {
	Accepted  bool   `json:"accepted"`
	Message   string `json:"message"`
	TxID      string `json:"txId,omitempty"`
	BlockHash string `json:"blockHash,omitempty"`
	Height    uint64 `json:"height,omitempty"`
}

func (s *Server) handleSubmitTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var tx blockchain.Transaction
	if err := json.Unmarshal(body, &tx); err != nil {
		_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: "invalid json"})
		return
	}

	if tx.ChainID == 0 {
		tx.ChainID = s.bc.ChainID
	}
	if tx.ChainID != s.bc.ChainID {
		_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: "wrong chainId"})
		return
	}

	s.bc.Mu().RLock()
	nextHeight := s.bc.LatestBlock().Height + 1
	s.bc.Mu().RUnlock()

	if err := tx.VerifyForConsensus(s.bc.Consensus(), nextHeight); err != nil {
		_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: err.Error()})
		return
	}
	if tx.Fee < minFee {
		_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: "fee too low"})
		return
	}
	if s.mp == nil {
		_ = writeJSON(w, http.StatusInternalServerError, submitTxResponse{Accepted: false, Message: "mempool not configured"})
		return
	}

	fromAddr, err := tx.FromAddress()
	if err != nil {
		_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: err.Error()})
		return
	}
	acct, _ := s.bc.Balance(fromAddr)
	pending := s.mp.PendingForSender(fromAddr)
	expectedNonce := acct.Nonce + 1
	var pendingDebitBefore uint64
	pendingByNonce := map[uint64]mempool.MempoolEntry{}
	for _, p := range pending {
		pendingByNonce[p.Tx().Nonce] = p
	}

	for {
		p, ok := pendingByNonce[expectedNonce]
		if !ok {
			break
		}
		pendingDebitBefore += p.Tx().Amount + p.Tx().Fee
		expectedNonce++
	}

	isReplacement := false
	if tx.Nonce == expectedNonce {
		totalDebit := tx.Amount + tx.Fee
		if acct.Balance < pendingDebitBefore+totalDebit {
			_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: "insufficient funds"})
			return
		}
	} else {
		existing, ok := pendingByNonce[tx.Nonce]
		if !ok {
			_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: fmt.Sprintf("bad nonce: expected %d, got %d", expectedNonce, tx.Nonce)})
			return
		}
		debitBefore := uint64(0)
		for n := acct.Nonce + 1; n < tx.Nonce; n++ {
			p, ok := pendingByNonce[n]
			if !ok {
				_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: "nonce gap in mempool"})
				return
			}
			debitBefore += p.Tx().Amount + p.Tx().Fee
		}
		totalDebit := tx.Amount + tx.Fee
		if acct.Balance < debitBefore+totalDebit {
			_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: "insufficient funds"})
			return
		}
		if tx.Fee <= existing.Tx().Fee {
			_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: "replacement fee must be higher"})
			return
		}
		isReplacement = true
	}

	aiApproved := true
	if s.requireAI {
		ok, err := s.callAIAuditor(r.Context(), tx)
		if err != nil {
			_ = writeJSON(w, http.StatusBadGateway, submitTxResponse{Accepted: false, Message: "ai auditor error"})
			return
		}
		aiApproved = ok
	}

	if s.requireAI && !aiApproved {
		_ = writeJSON(w, http.StatusOK, submitTxResponse{Accepted: false, Message: "rejected by AI auditor"})
		return
	}

	var txid string
	var evicted []string
	if isReplacement {
		var replaced bool
		txid, replaced, evicted, err = s.mp.ReplaceByFeeWithTxID(tx, "", s.bc.Consensus(), nextHeight)
		if err != nil {
			_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: err.Error()})
			return
		}
		if !replaced {
			_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: "replacement rejected"})
			return
		}
	} else {
		txid, err = s.mp.AddWithTxID(tx, "", s.bc.Consensus(), nextHeight)
		if err != nil {
			if err.Error() == "duplicate transaction" {
				_ = writeJSON(w, http.StatusOK, submitTxResponse{Accepted: true, Message: "duplicate", TxID: txid})
				return
			}
			_ = writeJSON(w, http.StatusBadRequest, submitTxResponse{Accepted: false, Message: err.Error()})
			return
		}
	}

	if s.wsHub != nil {
		if len(evicted) > 0 {
			s.wsHub.Publish(apws.WSEvent{Type: "mempool_removed", Data: map[string]any{"txIds": evicted, "reason": "rbf"}})
		}
		from, _ := tx.FromAddress()
		s.wsHub.Publish(apws.WSEvent{
			Type: "mempool_added",
			Data: map[string]any{
				"txId":      txid,
				"fromAddr":  from,
				"toAddress": tx.ToAddress,
				"amount":    tx.Amount,
				"fee":       tx.Fee,
				"nonce":     tx.Nonce,
			},
		})
	}

	_ = writeJSON(w, http.StatusOK, submitTxResponse{
		Accepted: true,
		Message:  "queued",
		TxID:     txid,
	})
}

func (s *Server) handleAuditChain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.bc.AuditChain(); err != nil {
		_ = writeJSON(w, http.StatusOK, map[string]any{"status": "FAILED", "message": err.Error()})
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"status": "SUCCESS", "message": "ok"})
}

func (s *Server) handleMempool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.mp == nil {
		_ = writeJSON(w, http.StatusOK, map[string]any{"size": 0, "txs": []any{}})
		return
	}
	entries := s.mp.EntriesSortedByFeeDesc()
	type view struct {
		TxID     string `json:"txId"`
		Fee      uint64 `json:"fee"`
		Amount   uint64 `json:"amount"`
		Nonce    uint64 `json:"nonce"`
		FromAddr string `json:"fromAddr"`
		To       string `json:"toAddress"`
	}
	out := make([]view, 0, len(entries))
	for _, e := range entries {
		from, _ := e.Tx().FromAddress()
		out = append(out, view{
			TxID:     e.TxIDHex(),
			Fee:      e.Tx().Fee,
			Amount:   e.Tx().Amount,
			Nonce:    e.Tx().Nonce,
			FromAddr: from,
			To:       e.Tx().ToAddress,
		})
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"size": len(out), "txs": out})
}

func (s *Server) handleMineOnce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_ = writeJSON(w, http.StatusBadRequest, map[string]any{"mined": false, "message": "mining not available via HTTP API in refactored version"})
}

func (s *Server) callAIAuditor(ctx context.Context, tx blockchain.Transaction) (bool, error) {
	if s.aiAuditor == "" {
		return true, nil
	}
	if tx.Type != blockchain.TxTransfer {
		return false, errors.New("ai auditor only supports transfer tx")
	}
	fromAddr, err := tx.FromAddress()
	if err != nil {
		return false, err
	}
	payload := map[string]any{
		"transaction": map[string]any{
			"sender":    fromAddr,
			"recipient": tx.ToAddress,
			"amount":    tx.Amount,
			"data":      tx.Data,
		},
	}
	b, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.aiAuditor, bytes.NewReader(b))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: s.httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("ai auditor status: %s", resp.Status)
	}
	var out struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return false, err
	}
	return out.Valid, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (s *Server) handleWalletCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")

	wlt, err := wallet.NewWallet()
	if err != nil {
		_ = writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{
		"address":    wlt.Address,
		"publicKey":  wlt.PublicKeyBase64(),
		"privateKey": wlt.PrivateKeyBase64(),
	})
}

type signRequest struct {
	PrivateKey string `json:"privateKey"`
	ToAddress  string `json:"toAddress"`
	Amount     uint64 `json:"amount"`
	Fee        uint64 `json:"fee"`
	Nonce      uint64 `json:"nonce"`
	Data       string `json:"data"`
}

func (s *Server) handleWalletSign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		_ = writeJSON(w, http.StatusBadRequest, map[string]any{"error": "read body failed"})
		return
	}
	defer r.Body.Close()

	var req signRequest
	if err := json.Unmarshal(body, &req); err != nil {
		_ = writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	if req.PrivateKey == "" {
		_ = writeJSON(w, http.StatusBadRequest, map[string]any{"error": "privateKey required"})
		return
	}
	if req.ToAddress == "" {
		_ = writeJSON(w, http.StatusBadRequest, map[string]any{"error": "toAddress required"})
		return
	}
	if req.Amount == 0 {
		_ = writeJSON(w, http.StatusBadRequest, map[string]any{"error": "amount required"})
		return
	}
	if req.Fee == 0 {
		req.Fee = minFee
	}

	wlt, err := wallet.WalletFromPrivateKeyBase64(req.PrivateKey)
	if err != nil {
		_ = writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid private key: " + err.Error()})
		return
	}

	if req.Nonce == 0 {
		acct, _ := s.bc.Balance(wlt.Address)
		req.Nonce = acct.Nonce + 1
	}

	tx := blockchain.Transaction{
		Type:       blockchain.TxTransfer,
		ChainID:    s.bc.ChainID,
		FromPubKey: wlt.PublicKey,
		ToAddress:  req.ToAddress,
		Amount:     req.Amount,
		Fee:        req.Fee,
		Nonce:      req.Nonce,
		Data:       req.Data,
	}

	nextHeight := s.bc.LatestBlock().Height + 1
	h, err := blockchain.TxSigningHashForConsensus(tx, s.bc.Consensus(), nextHeight)
	if err != nil {
		_ = writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	tx.Signature = ed25519.Sign(wlt.PrivateKey, h)

	txid, _ := blockchain.TxIDHex(tx)

	txJSON, _ := json.Marshal(tx)

	_ = writeJSON(w, http.StatusOK, map[string]any{
		"tx":      tx,
		"txJson":  string(txJSON),
		"txid":    txid,
		"signed":  true,
		"from":    wlt.Address,
		"nonce":   tx.Nonce,
		"chainId": tx.ChainID,
	})
}

func (s *Server) handleP2PGetAddr(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	type peerAddr struct {
		IP        string `json:"ip"`
		Port      int    `json:"port"`
		Timestamp int64  `json:"timestamp"`
	}
	var peerAddrs []peerAddr
	if s.peerManager != nil {
		for _, addr := range s.peerManager.Peers() {
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
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"addresses": peerAddrs})
}

func (s *Server) handleP2PAddr(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	type addrMsg struct {
		Addresses []struct {
			IP   string `json:"ip"`
			Port int    `json:"port"`
		} `json:"addresses"`
	}
	var msg addrMsg
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if s.peerManager != nil {
		for _, a := range msg.Addresses {
			addr := fmt.Sprintf("%s:%d", a.IP, a.Port)
			if addr != "" && addr != ":" {
				s.peerManager.AddPeer(addr)
			}
		}
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type AirdropClaimRequest struct {
	Address string `json:"address"`
	Email   string `json:"email,omitempty"`
}

func (s *Server) handleAirdropClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AirdropClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Address == "" {
		http.Error(w, "address required", http.StatusBadRequest)
		return
	}

	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	amount, reason, err := s.airdroper.Claim(req.Address, ip, req.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if amount == 0 {
		http.Error(w, reason, http.StatusBadRequest)
		return
	}

	_ = writeJSON(w, http.StatusOK, map[string]any{
		"status":  "claimed",
		"address": req.Address,
		"amount":  amount,
	})
}

func (s *Server) handleAirdropStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.airdroper.GetStats()
	_ = writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleAirdropStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "address required", http.StatusBadRequest)
		return
	}

	claim := s.airdroper.GetClaim(address)
	if claim == nil {
		_ = writeJSON(w, http.StatusOK, map[string]any{"claimed": false})
		return
	}

	_ = writeJSON(w, http.StatusOK, map[string]any{
		"claimed":   true,
		"amount":    claim.Amount,
		"claimTime": claim.ClaimTime,
	})
}
