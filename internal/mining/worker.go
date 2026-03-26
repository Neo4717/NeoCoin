package mining

import (
	"crypto/sha256"
	"math/big"
	"sync"
	"time"

	"github.com/Neo4717/NeoCoin/internal/blockchain"
)

func WorkForDifficultyBits(bits uint32) *big.Int {
	if bits == 0 {
		return big.NewInt(0)
	}
	if bits > 256 {
		bits = 256
	}
	target := big.NewInt(1)
	target.Lsh(target, uint(256-bits))
	return target
}

type Worker struct {
	ID int
	bc interface {
		LatestBlock() *blockchain.Block
		AddBlock(block *blockchain.Block) error
	}
	workCh  chan *blockchain.Block
	stopCh  chan struct{}
	stopped bool
	mu      sync.Mutex

	hashCount    uint64
	lastHashTime time.Time
	hashrate     float64
}

func NewWorker(id int, bc interface {
	LatestBlock() *blockchain.Block
	AddBlock(block *blockchain.Block) error
}) *Worker {
	return &Worker{
		ID:      id,
		bc:      bc,
		workCh:  make(chan *blockchain.Block),
		stopCh:  make(chan struct{}),
		stopped: false,
	}
}

func (w *Worker) SetWork(block *blockchain.Block) {
	select {
	case w.workCh <- block:
	default:
	}
}

func (w *Worker) Run() {
	for {
		select {
		case <-w.stopCh:
			return
		case block := <-w.workCh:
			if block != nil {
				w.mineBlock(block)
			}
		}
	}
}

func (w *Worker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		w.stopped = true
		close(w.stopCh)
	}
}

func (w *Worker) mineBlock(block *blockchain.Block) {
	target := WorkForDifficultyBits(block.DifficultyBits)
	if target == nil {
		return
	}

	var nonce uint64
	startTime := time.Now()

	for {
		select {
		case <-w.stopCh:
			return
		default:
		}

		header := w.prepareHeader(block, nonce)
		hash := sha256.Sum256(header)

		var hashInt big.Int
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(target) <= 0 {
			block.Hash = hash[:]
			block.Nonce = nonce

			w.UpdateHashrate(nonce+1, time.Since(startTime))

			err := w.SubmitBlock(block)
			if err == nil {
			}
			return
		}

		nonce++

		if nonce%1000000 == 0 {
			w.UpdateHashrate(1000000, time.Since(startTime))
		}
	}
}

func (w *Worker) prepareHeader(block *blockchain.Block, nonce uint64) []byte {
	header := make([]byte, 0)
	header = append(header, varint(block.Version)...)
	header = append(header, block.PrevHash...)
	header = append(header, varint(block.TimestampUnix)...)
	header = append(header, varint(block.DifficultyBits)...)
	header = append(header, varint(nonce)...)
	return header
}

func varint(v interface{}) []byte {
	switch val := v.(type) {
	case uint64:
		if val < 0xFD {
			return []byte{byte(val)}
		}
		return []byte{0xFD, byte(val), byte(val >> 8)}
	case uint32:
		if val < 0xFD {
			return []byte{byte(val)}
		}
		return []byte{0xFD, byte(val), byte(val >> 8)}
	default:
		return nil
	}
}

func (w *Worker) SubmitBlock(block *blockchain.Block) error {
	if block == nil {
		return nil
	}

	prevBlock := w.bc.LatestBlock()
	if prevBlock != nil && block.Height != prevBlock.Height+1 {
		return nil
	}

	err := w.bc.AddBlock(block)
	if err != nil {
		return err
	}

	return nil
}

func (w *Worker) UpdateHashrate(hashCount uint64, duration time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.hashCount += hashCount
	elapsed := time.Since(w.lastHashTime)

	if elapsed > 0 {
		w.hashrate = float64(w.hashCount) / elapsed.Seconds()
	}

	w.hashCount = 0
	w.lastHashTime = time.Now()
}

func (w *Worker) Hashrate() float64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.hashrate
}
