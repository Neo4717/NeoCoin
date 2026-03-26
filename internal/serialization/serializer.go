package serialization

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/Neo4717/NeoCoin/internal/blockchain"
	np "github.com/Neo4717/NeoCoin/proto"
	"google.golang.org/protobuf/proto"
)

type Serializer struct{}

func NewSerializer() *Serializer {
	return &Serializer{}
}

func (s *Serializer) BlockToProto(b *blockchain.Block) *np.Block {
	if b == nil {
		return nil
	}
	txs := make([]*np.Transaction, len(b.Transactions))
	for i, tx := range b.Transactions {
		txs[i] = s.TxToProto(tx)
	}
	return &np.Block{
		Header:       s.HeaderToProto(b),
		Transactions: txs,
	}
}

func (s *Serializer) ProtoToBlock(pb *np.Block) *blockchain.Block {
	if pb == nil {
		return nil
	}
	txs := make([]blockchain.Transaction, len(pb.Transactions))
	for i, tx := range pb.Transactions {
		txs[i] = *s.ProtoToTx(tx)
	}
	return &blockchain.Block{
		Version:        uint32(pb.Header.GetVersion()),
		Height:         uint64(pb.Header.GetHeight()),
		TimestampUnix:  pb.Header.GetTimestamp(),
		PrevHash:       decodeHex(pb.Header.GetPrevBlock()),
		Nonce:          decodeUint64(pb.Header.GetNonce()),
		DifficultyBits: uint32(pb.Header.GetDifficulty()),
		MinerAddress:   pb.Header.GetMiner(),
		Transactions:   txs,
		Hash:           nil,
	}
}

func (s *Serializer) HeaderToProto(b *blockchain.Block) *np.Header {
	if b == nil {
		return nil
	}
	return &np.Header{
		Version:    int64(b.Version),
		PrevBlock:  encodeHex(b.PrevHash),
		MerkleRoot: encodeHex(b.Hash),
		Timestamp:  b.TimestampUnix,
		Height:     int64(b.Height),
		Difficulty: int64(b.DifficultyBits),
		Nonce:      encodeUint64(b.Nonce),
		Miner:      b.MinerAddress,
	}
}

func (s *Serializer) TxToProto(tx blockchain.Transaction) *np.Transaction {
	txid, _ := blockchain.TxIDHex(tx)
	return &np.Transaction{
		Txid:       txid,
		From:       tx.FromAddressSafe(),
		To:         tx.ToAddress,
		Amount:     int64(tx.Amount),
		Fee:        int64(tx.Fee),
		Nonce:      int64(tx.Nonce),
		Signature:  tx.Signature,
		IsCoinbase: tx.Type == blockchain.TxCoinbase,
	}
}

func (s *Serializer) ProtoToTx(pb *np.Transaction) *blockchain.Transaction {
	if pb == nil {
		return nil
	}
	tx := blockchain.Transaction{
		Type:      blockchain.TxTransfer,
		ChainID:   0,
		ToAddress: pb.GetTo(),
		Amount:    uint64(pb.GetAmount()),
		Fee:       uint64(pb.GetFee()),
		Nonce:     uint64(pb.GetNonce()),
	}
	if pb.GetIsCoinbase() {
		tx.Type = blockchain.TxCoinbase
	}
	return &tx
}

func (s *Serializer) SerializeBlock(b *blockchain.Block) ([]byte, error) {
	pb := s.BlockToProto(b)
	return proto.Marshal(pb)
}

func (s *Serializer) DeserializeBlock(data []byte) (*blockchain.Block, error) {
	var pb np.Block
	if err := proto.Unmarshal(data, &pb); err != nil {
		return nil, err
	}
	return s.ProtoToBlock(&pb), nil
}

func (s *Serializer) BlockHash(b *blockchain.Block) []byte {
	data, _ := s.SerializeBlock(b)
	hash := sha256.Sum256(data)
	return hash[:]
}

func encodeHex(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return hex.EncodeToString(b)
}

func decodeHex(s string) []byte {
	if s == "" {
		return nil
	}
	b, _ := hex.DecodeString(s)
	return b
}

func encodeUint64(n uint64) string {
	return hex.EncodeToString([]byte{byte(n >> 56), byte(n >> 48), byte(n >> 40), byte(n >> 32), byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)})
}

func decodeUint64(s string) uint64 {
	if s == "" {
		return 0
	}
	b, _ := hex.DecodeString(s)
	if len(b) == 0 {
		return 0
	}
	var n uint64
	for _, b := range b {
		n = n<<8 | uint64(b)
	}
	return n
}
