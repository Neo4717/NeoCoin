package consensus

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidHeader     = errors.New("invalid block header")
	ErrInvalidDifficulty = errors.New("difficulty target mismatch")
	ErrInvalidPoW        = errors.New("proof of work validation failed")
	ErrInvalidTimestamp  = errors.New("invalid timestamp")
	ErrInvalidBlock      = errors.New("invalid block")
	ErrFutureTimestamp   = errors.New("timestamp too far in the future")
	ErrMedianTimePast    = errors.New("timestamp before median time past")
	ErrNoTransactions    = errors.New("block must have at least one transaction")
)

type Header struct {
	Version        uint32
	Height         uint64
	TimestampUnix  int64
	PrevHash       []byte
	DifficultyBits uint32
	MinerAddress   string
	Nonce          uint64
	Hash           []byte
}

func ValidateHeader(header *Header) error {
	if header == nil {
		return fmt.Errorf("%w: nil header", ErrInvalidHeader)
	}
	if header.Version == 0 {
		return fmt.Errorf("%w: version is zero", ErrInvalidHeader)
	}
	if header.Height == 0 && len(header.PrevHash) != 0 {
		return fmt.Errorf("%w: genesis block cannot have prevHash", ErrInvalidHeader)
	}
	if header.Height > 0 && len(header.PrevHash) != 32 {
		return fmt.Errorf("%w: invalid prevHash length", ErrInvalidHeader)
	}
	if header.MinerAddress == "" {
		return fmt.Errorf("%w: miner address is empty", ErrInvalidHeader)
	}
	if header.DifficultyBits == 0 {
		return fmt.Errorf("%w: difficulty bits is zero", ErrInvalidHeader)
	}
	return nil
}

func ValidateDifficulty(header *Header, chain []BlockInfo, p Params) error {
	expectedBits := NextDifficultyBits(p, chain)
	if header.DifficultyBits != expectedBits {
		return fmt.Errorf("%w: expected %d, got %d", ErrInvalidDifficulty, expectedBits, header.DifficultyBits)
	}
	return nil
}

func ValidatePoWHeader(header *Header, headerBytes []byte) (bool, error) {
	return ValidatePoW(headerBytes, header.Hash, header.DifficultyBits)
}

func ValidateTimestamp(header *Header, chain []BlockInfo, p Params) error {
	if header.TimestampUnix <= 0 {
		return fmt.Errorf("%w: timestamp must be positive", ErrInvalidTimestamp)
	}

	now := time.Now().Unix()
	if header.TimestampUnix > now+int64(p.MaxTimeDrift) {
		return fmt.Errorf("%w: timestamp %d is too far in the future (max drift: %d)",
			ErrFutureTimestamp, header.TimestampUnix, p.MaxTimeDrift)
	}

	if len(chain) == 0 {
		return nil
	}

	windowSize := p.MedianTimePastWindow
	if windowSize > len(chain) {
		windowSize = len(chain)
	}

	var mtp int64
	startIdx := len(chain) - windowSize
	for i := startIdx; i < len(chain); i++ {
		mtp += chain[i].TimestampUnix
	}
	mtp /= int64(windowSize)

	if header.TimestampUnix <= mtp {
		return fmt.Errorf("%w: timestamp %d <= MTP %d", ErrMedianTimePast, header.TimestampUnix, mtp)
	}

	prevBlock := chain[len(chain)-1]
	if header.TimestampUnix <= prevBlock.TimestampUnix {
		return fmt.Errorf("%w: timestamp must be greater than parent timestamp", ErrInvalidTimestamp)
	}

	return nil
}

func ValidateBlock(block *Header, txs []Transaction, chain []BlockInfo, p Params) error {
	if err := ValidateHeader(block); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidBlock, err)
	}

	if len(txs) == 0 {
		return fmt.Errorf("%w: no transactions", ErrInvalidBlock)
	}

	if err := ValidateDifficulty(block, chain, p); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidBlock, err)
	}

	return nil
}

type Transaction struct {
	Type      string
	ChainID   uint64
	ToAddress string
	Amount    uint64
	Fee       uint64
	Nonce     uint64
	Data      string
}
