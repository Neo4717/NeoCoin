package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	binaryEncodingVersionV1 = uint8(1)
)

var ErrBinaryEncoding = errors.New("binary encoding error")

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

func decodeHex32(addr string) ([32]byte, error) {
	var out [32]byte
	addr = strings.TrimPrefix(addr, AddressPrefix)
	b, err := hex.DecodeString(addr)
	if err != nil {
		return out, err
	}
	// Address format: version(1) + hash(32) + checksum(4) = 37 bytes total
	// For mining, we need just the 32-byte hash (skip version byte)
	if len(b) == 37 {
		copy(out[:], b[1:33])
		return out, nil
	}
	if len(b) == 32 {
		copy(out[:], b)
		return out, nil
	}
	return out, fmt.Errorf("expected 32 or 37 bytes, got %d", len(b))
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
