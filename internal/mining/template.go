package mining

import (
	"time"

	"github.com/Neo4717/NeoCoin/internal/blockchain"
	"github.com/Neo4717/NeoCoin/internal/consensus"
)

func CreateBlockTemplate(bc interface {
	LatestBlock() *blockchain.Block
	GetTransactions(max int) ([]blockchain.Transaction, error)
}, minerAddress string) (*blockchain.Block, error) {
	if minerAddress == "" {
		minerAddress = getDefaultMinerAddress()
	}

	prevBlock := bc.LatestBlock()
	if prevBlock == nil {
		return nil, nil
	}

	height := prevBlock.Height + 1
	difficulty := prevBlock.DifficultyBits

	transactions := []blockchain.Transaction{}

	coinbaseTx := blockchain.Transaction{
		Type:      blockchain.TxCoinbase,
		ChainID:   1,
		ToAddress: minerAddress,
		Amount:    uint64(consensus.BlockReward(int64(height))),
		Fee:       0,
		Nonce:     0,
		Data:      "coinbase",
		Signature: nil,
	}
	transactions = append(transactions, coinbaseTx)

	memTxs, _ := bc.GetTransactions(100)
	for _, tx := range memTxs {
		if tx.Type != blockchain.TxCoinbase {
			transactions = append(transactions, tx)
		}
	}

	block := &blockchain.Block{
		Version:        1,
		Height:         height,
		TimestampUnix:  time.Now().Unix(),
		PrevHash:       prevBlock.Hash,
		Nonce:          0,
		DifficultyBits: difficulty,
		MinerAddress:   minerAddress,
		Transactions:   transactions,
		Hash:           nil,
	}

	return block, nil
}

func getDefaultMinerAddress() string {
	return ""
}
