package blockchain

import (
	"container/list"
	"encoding/hex"
	"sync"
	"time"
)

type orphanBlock struct {
	block   *Block
	addedAt time.Time
}

type orphanPool struct {
	mu         sync.RWMutex
	blocks     map[string]*orphanBlock
	heightList *list.List
	maxOrphans int
}

func newOrphanPool(maxOrphans int) *orphanPool {
	if maxOrphans <= 0 {
		maxOrphans = 100
	}
	return &orphanPool{
		blocks:     make(map[string]*orphanBlock),
		heightList: list.New(),
		maxOrphans: maxOrphans,
	}
}

func (op *orphanPool) AddBlock(b *Block) {
	op.mu.Lock()
	defer op.mu.Unlock()

	hashHex := hex.EncodeToString(b.Hash)

	if _, exists := op.blocks[hashHex]; exists {
		return
	}

	if len(op.blocks) >= op.maxOrphans {
		op.removeOldestLocked()
	}

	ob := &orphanBlock{
		block:   b,
		addedAt: time.Now(),
	}
	op.blocks[hashHex] = ob

	op.heightList.PushFront(hashHex)
}

func (op *orphanPool) removeOldestLocked() {
	el := op.heightList.Back()
	if el == nil {
		return
	}
	hashHex := el.Value.(string)
	delete(op.blocks, hashHex)
	op.heightList.Remove(el)
}

func (op *orphanPool) GetBlock(hash []byte) *Block {
	op.mu.RLock()
	defer op.mu.RUnlock()

	hashHex := hex.EncodeToString(hash)
	if ob, ok := op.blocks[hashHex]; ok {
		return ob.block
	}
	return nil
}

func (op *orphanPool) GetBlockByHash(hashHex string) *Block {
	op.mu.RLock()
	defer op.mu.RUnlock()

	if ob, ok := op.blocks[hashHex]; ok {
		return ob.block
	}
	return nil
}

func (op *orphanPool) GetOrphansByParent(prevHash []byte) []*Block {
	op.mu.RLock()
	defer op.mu.RUnlock()

	var result []*Block
	prevHashHex := hex.EncodeToString(prevHash)

	for _, ob := range op.blocks {
		if hex.EncodeToString(ob.block.PrevHash) == prevHashHex {
			result = append(result, ob.block)
		}
	}
	return result
}

func (op *orphanPool) RemoveBlock(hash []byte) {
	op.mu.Lock()
	defer op.mu.Unlock()

	hashHex := hex.EncodeToString(hash)
	delete(op.blocks, hashHex)
}

func (op *orphanPool) Size() int {
	op.mu.RLock()
	defer op.mu.RUnlock()
	return len(op.blocks)
}

func (op *orphanPool) GetAllOrphans() []*Block {
	op.mu.RLock()
	defer op.mu.RUnlock()

	result := make([]*Block, 0, len(op.blocks))
	for _, ob := range op.blocks {
		result = append(result, ob.block)
	}
	return result
}

func (op *orphanPool) LimitSize(maxOrphans int) {
	op.mu.Lock()
	defer op.mu.Unlock()

	for len(op.blocks) > maxOrphans {
		op.removeOldestLocked()
	}
}

func (op *orphanPool) RemoveBlocksBelowHeight(height uint64) {
	op.mu.Lock()
	defer op.mu.Unlock()

	var toRemove []string
	for hashHex, ob := range op.blocks {
		if ob.block.Height < height {
			toRemove = append(toRemove, hashHex)
		}
	}
	for _, hashHex := range toRemove {
		delete(op.blocks, hashHex)
	}
}

func (op *orphanPool) Stats() map[string]any {
	op.mu.RLock()
	defer op.mu.RUnlock()

	stats := map[string]any{
		"count":      len(op.blocks),
		"maxOrphans": op.maxOrphans,
	}

	if len(op.blocks) > 0 {
		var oldestTime time.Time
		var newestTime time.Time
		for _, ob := range op.blocks {
			if oldestTime.IsZero() || ob.addedAt.Before(oldestTime) {
				oldestTime = ob.addedAt
			}
			if newestTime.IsZero() || ob.addedAt.After(newestTime) {
				newestTime = ob.addedAt
			}
		}
		stats["oldestAge"] = time.Since(oldestTime).Seconds()
		stats["newestAge"] = time.Since(newestTime).Seconds()
	}

	return stats
}
