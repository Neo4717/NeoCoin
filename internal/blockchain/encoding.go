package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/Neo4717/NeoCoin/internal/crypto"
)

func (t Transaction) signingHashLegacyJSON() ([]byte, error) {
	type signingView struct {
		Type      TransactionType `json:"type"`
		ChainID   uint64          `json:"chainId"`
		FromAddr  string          `json:"fromAddr,omitempty"`
		ToAddress string          `json:"toAddress"`
		Amount    uint64          `json:"amount"`
		Fee       uint64          `json:"fee"`
		Nonce     uint64          `json:"nonce,omitempty"`
		Data      string          `json:"data,omitempty"`
	}

	v := signingView{
		Type:      t.Type,
		ChainID:   t.ChainID,
		ToAddress: t.ToAddress,
		Amount:    t.Amount,
		Fee:       t.Fee,
		Nonce:     t.Nonce,
		Data:      t.Data,
	}

	if t.Type == TxTransfer {
		fromAddr, err := t.FromAddress()
		if err != nil {
			return nil, err
		}
		v.FromAddr = fromAddr
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(b)
	return sum[:], nil
}

func (t Transaction) SigningHash() ([]byte, error) {
	return t.signingHashLegacyJSON()
}

func (t Transaction) verifyWithSigningHash(h []byte) error {
	if t.Type != TxTransfer {
		return errors.New("signature verification only applies to transfer transactions")
	}
	if len(t.FromPubKey) != 32 {
		return errors.New("invalid fromPubKey length")
	}
	if len(t.Signature) != 64 {
		return errors.New("invalid signature length")
	}
	pubKey := make([]byte, 32)
	copy(pubKey, t.FromPubKey)
	if !crypto.Verify(pubKey, h, t.Signature) {
		return errors.New("signature verification failed")
	}
	return nil
}

func (t Transaction) Verify() error {
	switch t.Type {
	case TxCoinbase:
		if t.ChainID == 0 {
			return errors.New("chainId must be set")
		}
		if t.Amount == 0 {
			return errors.New("coinbase amount must be > 0")
		}
		if err := ValidateAddress(t.ToAddress); err != nil {
			return err
		}
		if t.FromPubKey != nil || t.Signature != nil || t.Nonce != 0 || t.Fee != 0 {
			return errors.New("coinbase must not include fromPubKey/signature/nonce/fee")
		}
		return nil
	case TxTransfer:
		if t.Amount == 0 {
			return errors.New("amount must be > 0")
		}
		if err := ValidateAddress(t.ToAddress); err != nil {
			return err
		}
		if len(t.FromPubKey) != 32 {
			return errors.New("invalid fromPubKey length")
		}
		if len(t.Signature) != 64 {
			return errors.New("invalid signature length")
		}
		if t.Nonce == 0 {
			return errors.New("nonce must be > 0")
		}
		if t.ChainID == 0 {
			return errors.New("chainId must be set")
		}
		h, err := t.signingHashLegacyJSON()
		if err != nil {
			return err
		}
		return t.verifyWithSigningHash(h)
	default:
		return errors.New("unknown transaction type")
	}
}

func (t Transaction) VerifyForConsensus(p ConsensusParams, height uint64) error {
	switch t.Type {
	case TxCoinbase:
		return t.Verify()
	case TxTransfer:
		if t.Amount == 0 {
			return errors.New("amount must be > 0")
		}
		if err := ValidateAddress(t.ToAddress); err != nil {
			return err
		}
		if len(t.FromPubKey) != 32 {
			return errors.New("invalid fromPubKey length")
		}
		if len(t.Signature) != 64 {
			return errors.New("invalid signature length")
		}
		if t.Nonce == 0 {
			return errors.New("nonce must be > 0")
		}
		if t.ChainID == 0 {
			return errors.New("chainId must be set")
		}
		h, err := txSigningHashForConsensus(t, p, height)
		if err != nil {
			return err
		}
		return t.verifyWithSigningHash(h)
	default:
		return errors.New("unknown transaction type")
	}
}

func (b *Block) TxRootLegacy() ([]byte, error) {
	return b.TxRootLegacyForConsensus(defaultConsensusParamsFromEnv())
}

func (b *Block) TxRootLegacyForConsensus(p ConsensusParams) ([]byte, error) {
	h := sha256.New()
	for _, tx := range b.Transactions {
		th, err := txSigningHashForConsensus(tx, p, b.Height)
		if err != nil {
			return nil, err
		}
		h.Write(th)
	}
	return h.Sum(nil), nil
}

func (b *Block) MerkleRootV2() ([]byte, error) {
	return b.MerkleRootV2ForConsensus(defaultConsensusParamsFromEnv())
}

func (b *Block) MerkleRootV2ForConsensus(p ConsensusParams) ([]byte, error) {
	leaves := make([][]byte, 0, len(b.Transactions))
	for _, tx := range b.Transactions {
		th, err := txSigningHashForConsensus(tx, p, b.Height)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, th)
	}
	return MerkleRoot(leaves)
}

func (b *Block) HeaderBytes(nonce uint64) ([]byte, error) {
	return b.HeaderBytesForConsensus(defaultConsensusParamsFromEnv(), nonce)
}

func (b *Block) HeaderBytesForConsensus(p ConsensusParams, nonce uint64) ([]byte, error) {
	if p.BinaryEncodingActive(b.Height) {
		return blockHeaderPreimageBinaryV1(b, nonce, p)
	}
	switch b.Version {
	case 2:
		root, err := b.MerkleRootV2ForConsensus(p)
		if err != nil {
			return nil, err
		}
		type headerV2 struct {
			Version        uint32 `json:"version"`
			Height         uint64 `json:"height"`
			TimestampUnix  int64  `json:"timestampUnix"`
			PrevHashB64    string `json:"prevHashB64"`
			MerkleRootHex  string `json:"merkleRootHex"`
			DifficultyBits uint32 `json:"difficultyBits"`
			MinerAddress   string `json:"minerAddress"`
			Nonce          uint64 `json:"nonce"`
		}
		v := headerV2{
			Version:        b.Version,
			Height:         b.Height,
			TimestampUnix:  b.TimestampUnix,
			PrevHashB64:    base64.StdEncoding.EncodeToString(b.PrevHash),
			MerkleRootHex:  hex.EncodeToString(root),
			DifficultyBits: b.DifficultyBits,
			MinerAddress:   b.MinerAddress,
			Nonce:          nonce,
		}
		return json.Marshal(v)
	default:
		root, err := b.TxRootLegacyForConsensus(p)
		if err != nil {
			return nil, err
		}
		type headerV1 struct {
			Version        uint32 `json:"version"`
			Height         uint64 `json:"height"`
			TimestampUnix  int64  `json:"timestampUnix"`
			PrevHashB64    string `json:"prevHashB64"`
			TxRootHex      string `json:"txRootHex"`
			DifficultyBits uint32 `json:"difficultyBits"`
			MinerAddress   string `json:"minerAddress"`
			Nonce          uint64 `json:"nonce"`
		}
		v := headerV1{
			Version:        b.Version,
			Height:         b.Height,
			TimestampUnix:  b.TimestampUnix,
			PrevHashB64:    base64.StdEncoding.EncodeToString(b.PrevHash),
			TxRootHex:      hex.EncodeToString(root),
			DifficultyBits: b.DifficultyBits,
			MinerAddress:   b.MinerAddress,
			Nonce:          nonce,
		}
		return json.Marshal(v)
	}
}

const binaryEncodingVersionV1 = uint8(1)

func writeULEB128(buf *bytes.Buffer, n uint64) {
	for {
		b := byte(n & 0x7F)
		n >>= 7
		if n != 0 {
			b |= 0x80
		}
		_ = buf.WriteByte(b)
		if n == 0 {
			return
		}
	}
}

func decodeHex32(addrHex string) ([32]byte, error) {
	var out [32]byte
	b, err := hex.DecodeString(addrHex)
	if err != nil {
		return out, err
	}
	if len(b) != 32 {
		return out, errors.New("expected 32 bytes")
	}
	copy(out[:], b)
	return out, nil
}

func txSigningPreimageBinaryV1(tx Transaction) ([]byte, error) {
	var buf bytes.Buffer
	_ = buf.WriteByte(binaryEncodingVersionV1)

	var txType byte
	switch tx.Type {
	case TxCoinbase:
		txType = 0
	case TxTransfer:
		txType = 1
	default:
		return nil, errors.New("unknown tx type")
	}
	_ = buf.WriteByte(txType)

	if tx.ChainID == 0 {
		return nil, errors.New("missing chainId")
	}
	_ = binary.Write(&buf, binary.LittleEndian, tx.ChainID)

	switch tx.Type {
	case TxCoinbase:
		to, err := decodeHex32(tx.ToAddress)
		if err != nil {
			return nil, errors.New("invalid toAddress")
		}
		_, _ = buf.Write(to[:])
		_ = binary.Write(&buf, binary.LittleEndian, tx.Amount)
		data := []byte(tx.Data)
		writeULEB128(&buf, uint64(len(data)))
		_, _ = buf.Write(data)
	case TxTransfer:
		if len(tx.FromPubKey) != 32 {
			return nil, errors.New("invalid fromPubKey length")
		}
		to, err := decodeHex32(tx.ToAddress)
		if err != nil {
			return nil, errors.New("invalid toAddress")
		}
		_, _ = buf.Write(tx.FromPubKey)
		_, _ = buf.Write(to[:])
		_ = binary.Write(&buf, binary.LittleEndian, tx.Amount)
		_ = binary.Write(&buf, binary.LittleEndian, tx.Nonce)
		_ = binary.Write(&buf, binary.LittleEndian, tx.Fee)
		data := []byte(tx.Data)
		writeULEB128(&buf, uint64(len(data)))
		_, _ = buf.Write(data)
	default:
		return nil, errors.New("unknown tx type")
	}

	return buf.Bytes(), nil
}

func blockHeaderPreimageBinaryV1(b *Block, nonce uint64, p ConsensusParams) ([]byte, error) {
	if b == nil {
		return nil, errors.New("nil block")
	}
	if b.TimestampUnix <= 0 {
		return nil, errors.New("invalid timestamp")
	}
	if len(b.PrevHash) != 0 && len(b.PrevHash) != 32 {
		return nil, errors.New("invalid prevHash length")
	}
	if len(b.Hash) != 0 && len(b.Hash) != 32 {
		return nil, errors.New("invalid hash length")
	}
	miner, err := decodeHex32(b.MinerAddress)
	if err != nil {
		return nil, errors.New("invalid minerAddress")
	}

	var root [32]byte
	switch b.Version {
	case 2:
		r, err := b.MerkleRootV2ForConsensus(p)
		if err != nil {
			return nil, err
		}
		copy(root[:], r)
	default:
		r, err := b.TxRootLegacyForConsensus(p)
		if err != nil {
			return nil, err
		}
		copy(root[:], r)
	}

	var prev [32]byte
	if len(b.PrevHash) == 32 {
		copy(prev[:], b.PrevHash)
	}

	var buf bytes.Buffer
	_ = buf.WriteByte(binaryEncodingVersionV1)
	_ = binary.Write(&buf, binary.LittleEndian, b.Version)
	_ = binary.Write(&buf, binary.LittleEndian, b.Height)
	_ = binary.Write(&buf, binary.LittleEndian, b.TimestampUnix)
	_, _ = buf.Write(prev[:])
	_, _ = buf.Write(root[:])
	_ = binary.Write(&buf, binary.LittleEndian, b.DifficultyBits)
	_, _ = buf.Write(miner[:])
	_ = binary.Write(&buf, binary.LittleEndian, nonce)
	return buf.Bytes(), nil
}

func txSigningHashLegacyJSON(tx Transaction) ([]byte, error) {
	return tx.signingHashLegacyJSON()
}

func txSigningHashBinaryV1(tx Transaction) ([]byte, error) {
	pre, err := txSigningPreimageBinaryV1(tx)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(pre)
	return sum[:], nil
}

func TxSigningHashForConsensus(tx Transaction, p ConsensusParams, height uint64) ([]byte, error) {
	return txSigningHashForConsensus(tx, p, height)
}

func txSigningHashForConsensus(tx Transaction, p ConsensusParams, height uint64) ([]byte, error) {
	if p.BinaryEncodingActive(height) {
		return txSigningHashBinaryV1(tx)
	}
	return txSigningHashLegacyJSON(tx)
}

func TxIDHex(tx Transaction) (string, error) {
	h, err := tx.SigningHash()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h), nil
}

func TxIDHexForConsensus(tx Transaction, p ConsensusParams, height uint64) (string, error) {
	h, err := txSigningHashForConsensus(tx, p, height)
	if err != nil {
		return "", err
	}
	if len(h) != 32 {
		return "", errors.New("expected 32 bytes")
	}
	return hex.EncodeToString(h), nil
}
