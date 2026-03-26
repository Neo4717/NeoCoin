package blockchain

import "errors"

type TransactionType string

const (
	TxCoinbase TransactionType = "coinbase"
	TxTransfer TransactionType = "transfer"
)

var (
	errFromAddressOnlyForTransfer = errors.New("from address only exists for transfer transactions")
	errInvalidFromPubKeyLength    = errors.New("invalid fromPubKey length: expected 32 bytes")
)

type Transaction struct {
	Type       TransactionType `json:"type"`
	ChainID    uint64          `json:"chainId"`
	FromPubKey []byte          `json:"fromPubKey,omitempty"`
	ToAddress  string          `json:"toAddress"`
	Amount     uint64          `json:"amount"`
	Fee        uint64          `json:"fee"`
	Nonce      uint64          `json:"nonce,omitempty"`
	Data       string          `json:"data,omitempty"`
	Signature  []byte          `json:"signature,omitempty"`
}

type Block struct {
	Version        uint32        `json:"version"`
	Height         uint64        `json:"height"`
	TimestampUnix  int64         `json:"timestampUnix"`
	PrevHash       []byte        `json:"prevHash"`
	Nonce          uint64        `json:"nonce"`
	DifficultyBits uint32        `json:"difficultyBits"`
	MinerAddress   string        `json:"minerAddress"`
	Transactions   []Transaction `json:"transactions"`
	Hash           []byte        `json:"hash"`
	StateRoot      []byte        `json:"stateRoot,omitempty"`
	UTXORoot       []byte        `json:"utxoRoot,omitempty"`
}

type Account struct {
	Balance uint64 `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

type TxLocation struct {
	Height       uint64 `json:"height"`
	BlockHashHex string `json:"blockHashHex"`
	Index        int    `json:"index"`
}

type AddressTxEntry struct {
	TxID      string     `json:"txId"`
	Location  TxLocation `json:"location"`
	FromAddr  string     `json:"fromAddr"`
	ToAddress string     `json:"toAddress"`
	Amount    uint64     `json:"amount"`
	Fee       uint64     `json:"fee"`
	Nonce     uint64     `json:"nonce"`
}

type WSEvent struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type EventSink interface {
	Publish(WSEvent)
}

func (t Transaction) FromAddress() (string, error) {
	if t.Type != TxTransfer {
		return "", errFromAddressOnlyForTransfer
	}
	if len(t.FromPubKey) != 32 {
		return "", errInvalidFromPubKeyLength
	}
	return GenerateAddress(t.FromPubKey), nil
}

func (t Transaction) FromAddressSafe() string {
	addr, _ := t.FromAddress()
	return addr
}

func (t Transaction) EstimateSize() int {
	return 80 + len(t.Signature) + len(t.Data)
}
