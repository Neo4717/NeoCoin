package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Neo4717/NeoCoin/config"
	"github.com/Neo4717/NeoCoin/internal/blockchain"
	"github.com/Neo4717/NeoCoin/internal/logger"
	"github.com/Neo4717/NeoCoin/internal/mempool"
	"github.com/Neo4717/NeoCoin/internal/mining"
)

func main() {
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Config validation: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.LogLevel, logger.TextEncoding, "", "", false)
	logger.Info("NeoCoin Miner service starting...")

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

	mp := mempool.NewMempool(cfg)

	bcAdapter := &miningBlockchainAdapter{bc: bc}
	mpAdapter := &miningMempoolAdapter{mp: mp}
	miner := mining.NewMiner(bcAdapter, mpAdapter, 100, false, cfg.StratumEnabled, cfg.StratumAddr)

	logger.Info("Miner service started, address: %s", cfg.MinerAddress)

	runCtx, stop := context.WithCancel(context.Background())
	defer stop()

	go func() {
		miner.Run(runCtx, 1*time.Second)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down miner service...")
	stop()
	mp.Stop()
	logger.Info("Miner service stopped")
}

type miningBlockchainAdapter struct {
	bc *blockchain.Blockchain
}

func (a *miningBlockchainAdapter) SelectMempoolTxs(entries []mining.TxEntry, maxTx int) ([]interface{}, [][]byte, error) {
	bcEntries := make([]blockchain.TxEntry, len(entries))
	for i, e := range entries {
		bcEntries[i] = e.(blockchain.TxEntry)
	}
	txs, ids, err := a.bc.SelectMempoolTxs(bcEntries, maxTx)
	if err != nil {
		return nil, nil, err
	}
	result := make([]interface{}, len(txs))
	for i, tx := range txs {
		result[i] = tx
	}
	byteIds := make([][]byte, len(ids))
	for i, id := range ids {
		byteIds[i] = []byte(id)
	}
	return result, byteIds, nil
}

func (a *miningBlockchainAdapter) MineTransfers(txs []interface{}) (*interface{}, error) {
	realTxs := make([]blockchain.Transaction, len(txs))
	for i, tx := range txs {
		realTxs[i] = tx.(blockchain.Transaction)
	}
	b, err := a.bc.MineTransfers(realTxs)
	if err != nil {
		return nil, err
	}
	result := interface{}(b)
	return &result, nil
}

type miningMempoolAdapter struct {
	mp *mempool.Mempool
}

func (a *miningMempoolAdapter) RemoveMany(ids [][]byte) {
	strIds := make([]string, len(ids))
	for i, id := range ids {
		strIds[i] = string(id)
	}
	a.mp.RemoveMany(strIds)
}

func (a *miningMempoolAdapter) EntriesSortedByFeeDesc() []mining.TxEntry {
	entries := a.mp.EntriesSortedByFeeDesc()
	result := make([]mining.TxEntry, len(entries))
	for i := range entries {
		result[i] = &entries[i]
	}
	return result
}

func (a *miningMempoolAdapter) Size() int {
	return a.mp.Size()
}
