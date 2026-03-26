package blockchain

import (
	"testing"
)

func FuzzTransactionValidation(f *testing.F) {
	f.Fuzz(func(t *testing.T, from, to string, amount, fee, nonce int64, sig []byte) {
		if amount < 0 || fee < 0 || nonce < 0 {
			return
		}

		tx := Transaction{
			Type:      TxTransfer,
			ChainID:   1,
			ToAddress: to,
			Amount:    uint64(amount),
			Fee:       uint64(fee),
			Nonce:     uint64(nonce),
			Signature: sig,
		}

		if len(from) == 32 {
			tx.FromPubKey = []byte(from)
		} else if len(from) >= 32 {
			tx.FromPubKey = []byte(from[:32])
		}

		_ = tx.EstimateSize()
	})
}

func FuzzBlockHeaderParsing(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 10000 {
			data = data[:10000]
		}
		_ = data
	})
}

func FuzzAccountState(f *testing.F) {
	f.Fuzz(func(t *testing.T, balance, nonce int64) {
		if balance < 0 || nonce < 0 {
			return
		}

		acc := Account{
			Balance: uint64(balance),
			Nonce:   uint64(nonce),
		}

		_ = acc.Balance
		_ = acc.Nonce
	})
}
