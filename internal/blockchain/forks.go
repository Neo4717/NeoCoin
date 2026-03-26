package blockchain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sort"
)

var ErrUnknownParent = errors.New("unknown parent")

func (bc *Blockchain) AddBlock(b *Block) (bool, error) {
	if b == nil {
		return false, errors.New("nil block")
	}
	if b.Height == 0 {
		return false, errors.New("external genesis blocks not accepted")
	}
	if expected := blockVersionForHeight(bc.consensus, b.Height); b.Version != expected {
		return false, fmt.Errorf("bad block version at %d: expected %d got %d", b.Height, expected, b.Version)
	}
	if len(b.Hash) == 0 {
		return false, errors.New("missing block hash")
	}
	if b.DifficultyBits == 0 {
		return false, errors.New("missing difficultyBits")
	}
	if b.DifficultyBits > maxDifficultyBits {
		return false, fmt.Errorf("difficultyBits out of range: %d", b.DifficultyBits)
	}

	ok, err := NewProofOfWork(bc.consensus, b).Validate()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, errors.New("invalid proof-of-work")
	}

	for _, tx := range b.Transactions {
		if tx.ChainID == 0 {
			tx.ChainID = bc.ChainID
		}
		if tx.ChainID != bc.ChainID {
			return false, fmt.Errorf("wrong chainId: %d", tx.ChainID)
		}
		if err := tx.VerifyForConsensus(bc.consensus, b.Height); err != nil {
			return false, err
		}
	}

	bc.mu.Lock()
	var events EventSink
	var toPublish []WSEvent
	defer func() {
		bc.mu.Unlock()
		if events == nil {
			return
		}
		for _, e := range toPublish {
			events.Publish(e)
		}
	}()

	parentHashHex := hex.EncodeToString(b.PrevHash)
	if _, ok := bc.blocksByHash[parentHashHex]; !ok {
		if bc.orphanPool != nil {
			bc.orphanPool.AddBlock(b)
		}
		return false, ErrUnknownParent
	}

	hashHex := hex.EncodeToString(b.Hash)
	if _, exists := bc.blocksByHash[hashHex]; exists {
		return false, errors.New("duplicate block")
	}
	bc.blocksByHash[hashHex] = b

	path, state, newWork, err := bc.computeCanonicalForTipLocked(hashHex)
	if err != nil {
		delete(bc.blocksByHash, hashHex)
		return false, err
	}

	if err := bc.store.PutBlock(b); err != nil {
		delete(bc.blocksByHash, hashHex)
		return false, err
	}

	currentTip := bc.bestTipHash
	currentWork := bc.canonicalWork
	if currentWork == nil {
		currentWork = big.NewInt(0)
	}

	better := false
	if newWork.Cmp(currentWork) > 0 {
		better = true
	} else if newWork.Cmp(currentWork) == 0 && hashHex < currentTip {
		better = true
	}

	if !better {
		return false, nil
	}

	currentHeight := bc.blocks[len(bc.blocks)-1].Height
	wasExtension := (parentHashHex == currentTip && b.Height == currentHeight+1)
	bc.blocks = path
	bc.state = state
	bc.bestTipHash = hashHex
	bc.reindexAllTxsLocked()
	bc.reindexAllAddressTxsLocked()
	bc.canonicalWork = new(big.Int).Set(newWork)
	if err := bc.store.RewriteCanonical(bc.blocks); err != nil {
		return !wasExtension, err
	}

	if wasExtension && bc.onBlockMined != nil {
		parentBlock, ok := bc.blocksByHash[parentHashHex]
		if ok {
			bc.onBlockMined(parentBlock.TimestampUnix)
		}
	}

	if bc.cfg != nil && bc.cfg.StoreMode == "pruned" {
		interval := bc.cfg.CheckpointInterval
		if interval == 0 {
			interval = 100
		}
		if int64(b.Height)%interval == 0 {
			pruner := NewPruner(bc.store, bc.cfg)
			_ = pruner.StateCheckpoint(int64(b.Height))
		}
	}

	if ShouldCommit(b.Height) {
		if err := bc.ComputeAndStoreCommitment(b.Height); err != nil {
			log.Printf("WARNING: failed to store commitment at height %d: %v", b.Height, err)
		}
	}

	events = bc.events
	toPublish = append(toPublish, WSEvent{
		Type: "new_block",
		Data: map[string]any{
			"height":         b.Height,
			"hash":           hashHex,
			"prevHash":       parentHashHex,
			"difficultyBits": b.DifficultyBits,
			"txCount":        len(b.Transactions),
			"addresses":      addressesForBlock(b),
		},
	})
	if !wasExtension {
		toPublish = append(toPublish, WSEvent{
			Type: "reorg",
			Data: map[string]any{
				"oldTip": currentTip,
				"newTip": hashHex,
			},
		})
	}
	return !wasExtension, nil
}

func (bc *Blockchain) ProcessOrphans() int {
	if bc.orphanPool == nil {
		return 0
	}

	processed := 0
	for {
		bestHash, _ := hex.DecodeString(bc.bestTipHash)
		orphans := bc.orphanPool.GetOrphansByParent(bestHash)
		if len(orphans) == 0 {
			break
		}

		for _, orphan := range orphans {
			_, err := bc.AddBlock(orphan)
			if err != nil {
				break
			}
			bc.orphanPool.RemoveBlock(orphan.Hash)
			processed++
		}
	}

	bc.orphanPool.LimitSize(100)

	return processed
}

func (bc *Blockchain) OrphanStats() map[string]any {
	if bc.orphanPool == nil {
		return map[string]any{"count": 0}
	}
	return bc.orphanPool.Stats()
}

func (bc *Blockchain) GetOrphanCount() int {
	if bc.orphanPool == nil {
		return 0
	}
	return bc.orphanPool.Size()
}

func (bc *Blockchain) computeCanonicalForTipLocked(tipHashHex string) ([]*Block, map[string]Account, *big.Int, error) {
	var rev []*Block
	seen := map[string]struct{}{}
	cur := tipHashHex

	for {
		if _, ok := seen[cur]; ok {
			return nil, nil, nil, errors.New("cycle detected")
		}
		seen[cur] = struct{}{}

		b, ok := bc.blocksByHash[cur]
		if !ok {
			return nil, nil, nil, errors.New("missing ancestor")
		}
		rev = append(rev, b)
		if b.Height == 0 || len(b.PrevHash) == 0 {
			break
		}
		cur = hex.EncodeToString(b.PrevHash)
	}

	path := make([]*Block, 0, len(rev))
	for i := len(rev) - 1; i >= 0; i-- {
		path = append(path, rev[i])
	}

	state := map[string]Account{}
	work := big.NewInt(0)
	for i, b := range path {
		if i == 0 {
			if b.Height != 0 || len(b.PrevHash) != 0 {
				return nil, nil, nil, errors.New("bad genesis header")
			}
			if expected := blockVersionForHeight(bc.consensus, 0); b.Version != expected {
				return nil, nil, nil, errors.New("bad genesis version")
			}
		} else {
			prev := path[i-1]
			if b.Height != prev.Height+1 {
				return nil, nil, nil, errors.New("bad height linkage")
			}
			if hex.EncodeToString(b.PrevHash) != hex.EncodeToString(prev.Hash) {
				return nil, nil, nil, errors.New("bad prevhash linkage")
			}
			if err := validateBlockTime(bc.consensus, path, i); err != nil {
				return nil, nil, nil, err
			}
			if bc.consensus.DifficultyEnable {
				expected := expectedDifficultyBitsForBlockIndex(bc.consensus, path, i)
				if expected == 0 {
					return nil, nil, nil, errors.New("difficulty calc failed")
				}
				if b.DifficultyBits != expected {
					return nil, nil, nil, fmt.Errorf("bad difficulty at height %d: expected %d got %d", b.Height, expected, b.DifficultyBits)
				}
			}
			if expected := blockVersionForHeight(bc.consensus, b.Height); b.Version != expected {
				return nil, nil, nil, fmt.Errorf("bad block version at %d: expected %d got %d", b.Height, expected, b.Version)
			}
		}
		if err := applyBlockToState(bc.consensus, state, b); err != nil {
			return nil, nil, nil, err
		}
		work.Add(work, WorkForDifficultyBits(b.DifficultyBits))
	}
	return path, state, work, nil
}

type BlockHeader struct {
	Height         uint64 `json:"height"`
	TimestampUnix  int64  `json:"timestampUnix"`
	PrevHashHex    string `json:"prevHashHex"`
	HashHex        string `json:"hashHex"`
	Nonce          uint64 `json:"nonce"`
	DifficultyBits uint32 `json:"difficultyBits"`
	MinerAddress   string `json:"minerAddress"`
	TxCount        int    `json:"txCount"`
}

func (bc *Blockchain) HeadersFrom(height uint64, count int) []BlockHeader {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if count <= 0 {
		count = 100
	}
	if height >= uint64(len(bc.blocks)) {
		return nil
	}
	end := height + uint64(count)
	if end > uint64(len(bc.blocks)) {
		end = uint64(len(bc.blocks))
	}
	out := make([]BlockHeader, 0, end-height)
	for _, b := range bc.blocks[height:end] {
		out = append(out, BlockHeader{
			Height:         b.Height,
			TimestampUnix:  b.TimestampUnix,
			PrevHashHex:    hex.EncodeToString(b.PrevHash),
			HashHex:        hex.EncodeToString(b.Hash),
			Nonce:          b.Nonce,
			DifficultyBits: b.DifficultyBits,
			MinerAddress:   b.MinerAddress,
			TxCount:        len(b.Transactions),
		})
	}
	return out
}

func (bc *Blockchain) BlocksFrom(height uint64, count int) []*Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if count <= 0 {
		count = 100
	}
	if height >= uint64(len(bc.blocks)) {
		return nil
	}
	end := height + uint64(count)
	if end > uint64(len(bc.blocks)) {
		end = uint64(len(bc.blocks))
	}
	return append([]*Block(nil), bc.blocks[height:end]...)
}

func (bc *Blockchain) BlockByHash(hashHex string) (*Block, bool) {
	if bc.cache != nil {
		if block, ok := bc.cache.GetCachedBlock(hashHex); ok {
			return block.(*Block), true
		}
	}

	bc.mu.RLock()
	defer bc.mu.RUnlock()
	b, ok := bc.blocksByHash[hashHex]
	if ok && bc.cache != nil {
		bc.cache.CacheBlock(hashHex, b)
	}
	return b, ok
}

func (bc *Blockchain) KnownTips() []string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	var tips []string
	isParent := map[string]struct{}{}
	for _, b := range bc.blocksByHash {
		if len(b.PrevHash) > 0 {
			isParent[hex.EncodeToString(b.PrevHash)] = struct{}{}
		}
	}
	for h := range bc.blocksByHash {
		if _, ok := isParent[h]; !ok {
			tips = append(tips, h)
		}
	}
	sort.Strings(tips)
	return tips
}
