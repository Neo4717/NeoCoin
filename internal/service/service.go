package service

import (
	"context"

	"github.com/Neo4717/NeoCoin/internal/blockchain"
)

type NodeService interface {
	GetBlock(ctx context.Context, height int64) (*blockchain.Block, error)
	GetBlockByHash(ctx context.Context, hash string) (*blockchain.Block, error)
	GetBalance(ctx context.Context, addr string) (int64, error)
	GetTransaction(ctx context.Context, txid string) (*blockchain.Transaction, error)
	SubmitTransaction(ctx context.Context, tx *blockchain.Transaction) error
	GetChainInfo(ctx context.Context) (*ChainInfo, error)
}

type ChainInfo struct {
	Height     int64
	Hash       string
	Difficulty int64
	Supply     int64
}
