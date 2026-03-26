package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Neo4717/NeoCoin/config"
	"github.com/Neo4717/NeoCoin/internal/blockchain"
	"github.com/Neo4717/NeoCoin/internal/logger"
	"github.com/Neo4717/NeoCoin/internal/mempool"
	"github.com/Neo4717/NeoCoin/internal/networking"
)

func main() {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Config validation: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.LogLevel, logger.TextEncoding, "", "", false)
	logger.Info("NeoCoin P2P service starting...")

	store, err := blockchain.OpenChainStoreFromEnv()
	if err != nil {
		logger.Error("Failed to open chain store: %v", err)
		os.Exit(1)
	}
	defer func() {
		if s, ok := store.(*blockchain.BoltChainStore); ok {
			s.Close()
		}
	}()

	bc, err := blockchain.LoadBlockchain(cfg.ChainID, cfg.MinerAddress, store, 0, cfg)
	if err != nil {
		logger.Error("Failed to load blockchain: %v", err)
		os.Exit(1)
	}
	defer bc.Close()

	mp := mempool.NewMempool(cfg)

	pm := networking.NewPeerManager(networking.DefaultPeerManagerConfig(), cfg.MinerAddress)

	serverCfg := networking.DefaultServerConfig()
	serverCfg.ListenAddr = fmt.Sprintf(":%d", cfg.P2PPort)
	serverCfg.NodeID = cfg.MinerAddress
	serverCfg.ChainID = cfg.ChainID
	serverCfg.RulesHash = bc.RulesHashHex()

	bcAdapter := &blockchainAdapter{bc: bc}
	mpAdapter := &mempoolAdapter{mp: mp}

	p2pServer := networking.NewServer(serverCfg, bcAdapter, mpAdapter, pm)

	p2pCtx, p2pCancel := context.WithCancel(context.Background())

	go func() {
		if err := p2pServer.Serve(p2pCtx); err != nil {
			logger.Warn("P2P server error: %v", err)
		}
	}()

	logger.Info("P2P service listening on :%d", cfg.P2PPort)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down P2P service...")
	p2pCancel()
	p2pServer.Close()
	mp.Stop()
	bc.Close()
	logger.Info("P2P service stopped")
}

type blockchainAdapter struct {
	bc *blockchain.Blockchain
}

func (a *blockchainAdapter) ChainID() uint64 {
	return a.bc.ChainID
}

func (a *blockchainAdapter) RulesHashHex() string {
	return a.bc.RulesHashHex()
}

func (a *blockchainAdapter) LatestBlock() networking.BlockInterface {
	return blockToBlockData(a.bc.LatestBlock())
}

func (a *blockchainAdapter) BlockByHeight(height uint64) (networking.BlockInterface, bool) {
	b, ok := a.bc.BlockByHeight(height)
	if !ok {
		return nil, false
	}
	return blockToBlockData(b), true
}

func (a *blockchainAdapter) BlockByHash(hashHex string) (networking.BlockInterface, bool) {
	b, ok := a.bc.BlockByHash(hashHex)
	if !ok {
		return nil, false
	}
	return blockToBlockData(b), true
}

func (a *blockchainAdapter) HeadersFrom(from uint64, count int) []networking.BlockHeader {
	headers := a.bc.HeadersFrom(from, count)
	result := make([]networking.BlockHeader, len(headers))
	for i, h := range headers {
		result[i] = networking.BlockHeader{
			Version:        1,
			Height:         h.Height,
			TimestampUnix:  h.TimestampUnix,
			PrevHash:       []byte(h.PrevHashHex),
			Hash:           []byte(h.HashHex),
			DifficultyBits: h.DifficultyBits,
		}
	}
	return result
}

func (a *blockchainAdapter) AddBlock(block *networking.BlockData) (interface{}, error) {
	b := blockDataToBlock(block)
	ok, err := a.bc.AddBlock(b)
	return ok, err
}

type mempoolAdapter struct {
	mp *mempool.Mempool
}

func (a *mempoolAdapter) Add(tx interface{}) (interface{}, error) {
	return nil, nil
}

func blockToBlockData(b *blockchain.Block) *networking.BlockData {
	txs := make([]networking.TransactionData, len(b.Transactions))
	for i, tx := range b.Transactions {
		txs[i] = networking.TransactionData{
			Type:       string(tx.Type),
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
	return &networking.BlockData{
		Version:        b.Version,
		Height:         b.Height,
		TimestampUnix:  b.TimestampUnix,
		PrevHash:       b.PrevHash,
		Nonce:          b.Nonce,
		DifficultyBits: b.DifficultyBits,
		MinerAddress:   b.MinerAddress,
		Transactions:   txs,
		Hash:           b.Hash,
	}
}

func blockDataToBlock(bd *networking.BlockData) *blockchain.Block {
	txs := make([]blockchain.Transaction, len(bd.Transactions))
	for i, tx := range bd.Transactions {
		txs[i] = blockchain.Transaction{
			Type:       blockchain.TransactionType(tx.Type),
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
	return &blockchain.Block{
		Version:        bd.Version,
		Height:         bd.Height,
		TimestampUnix:  bd.TimestampUnix,
		PrevHash:       bd.PrevHash,
		Nonce:          bd.Nonce,
		DifficultyBits: bd.DifficultyBits,
		MinerAddress:   bd.MinerAddress,
		Transactions:   txs,
		Hash:           bd.Hash,
	}
}
