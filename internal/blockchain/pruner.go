package blockchain

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/Neo4717/NeoCoin/config"
)

type StoreMode int

const (
	StoreModeFull StoreMode = iota
	StoreModePruned
	StoreModeArchive
)

type Pruner struct {
	store              ChainStore
	pruneDepth         int64
	lastPrune          int64
	pruneLock          sync.Mutex
	checkpointInterval int64
}

func NewPruner(store ChainStore, cfg *config.Config) *Pruner {
	pruneDepth := cfg.PruneDepth
	if pruneDepth == 0 {
		pruneDepth = 1000
	}
	interval := cfg.CheckpointInterval
	if interval == 0 {
		interval = 100
	}
	return &Pruner{
		store:              store,
		pruneDepth:         pruneDepth,
		lastPrune:          0,
		checkpointInterval: interval,
	}
}

func (p *Pruner) PruneDepth() int64 {
	return p.pruneDepth
}

func (p *Pruner) CheckpointInterval() int64 {
	return p.checkpointInterval
}

func (p *Pruner) ShouldPrune() bool {
	p.pruneLock.Lock()
	defer p.pruneLock.Unlock()

	currentHeight, err := p.getCurrentHeight()
	if err != nil {
		return false
	}

	return currentHeight > p.pruneDepth && currentHeight-p.lastPrune >= p.pruneDepth
}

func (p *Pruner) getCurrentHeight() (int64, error) {
	heights, err := p.store.GetCheckpointHeights()
	if err != nil {
		return 0, err
	}
	if len(heights) == 0 {
		return 0, nil
	}
	maxH := heights[len(heights)-1]

	canonical, err := p.store.ReadCanonical()
	if err != nil {
		return maxH, nil
	}
	if len(canonical) > 0 {
		last := int64(canonical[len(canonical)-1].Height)
		if last > maxH {
			return last, nil
		}
	}
	return maxH, nil
}

func (p *Pruner) PruneToHeight(height int64) error {
	p.pruneLock.Lock()
	defer p.pruneLock.Unlock()

	if err := p.store.PruneBelow(height); err != nil {
		return fmt.Errorf("prune below: %w", err)
	}

	p.lastPrune = height
	return nil
}

func (p *Pruner) GetStateAt(height int64) (map[string]Account, error) {
	heights, err := p.store.GetCheckpointHeights()
	if err != nil {
		return nil, fmt.Errorf("get checkpoint heights: %w", err)
	}

	if len(heights) == 0 {
		return nil, errors.New("no checkpoints available")
	}

	var targetHeight int64
	for i := len(heights) - 1; i >= 0; i-- {
		if heights[i] <= height {
			targetHeight = heights[i]
			break
		}
	}

	if targetHeight == 0 && len(heights) > 0 {
		targetHeight = heights[0]
	}

	if targetHeight == 0 && (len(heights) == 0 || heights[0] != 0) {
		return nil, errors.New("no checkpoint found at or below target height")
	}

	state, err := p.store.ReadCheckpoint(targetHeight)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint at %d: %w", targetHeight, err)
	}

	if targetHeight < height {
		canonical, err := p.store.ReadCanonical()
		if err != nil {
			return nil, err
		}
		for _, b := range canonical {
			if int64(b.Height) <= targetHeight {
				continue
			}
			if int64(b.Height) > height {
				break
			}
			if err := applyBlockToStateFromPruner(state, b); err != nil {
				return nil, fmt.Errorf("apply block %d: %w", b.Height, err)
			}
		}
	}

	return state, nil
}

func (p *Pruner) StateCheckpoint(height int64) error {
	if p.store == nil {
		return errors.New("no store configured")
	}

	if height%p.checkpointInterval != 0 {
		return nil
	}

	var stateCopy map[string]Account

	if getter, ok := p.store.(interface{ GetState() map[string]Account }); ok {
		state := getter.GetState()
		stateCopy = make(map[string]Account, len(state))
		for k, v := range state {
			stateCopy[k] = v
		}
	} else {
		canonical, err := p.store.ReadCanonical()
		if err != nil {
			return fmt.Errorf("read canonical: %w", err)
		}

		state := map[string]Account{}
		for _, b := range canonical {
			if int64(b.Height) > height {
				break
			}
			if err := applyBlockToStateFromPruner(state, b); err != nil {
				return fmt.Errorf("apply block %d: %w", b.Height, err)
			}
		}
		stateCopy = make(map[string]Account, len(state))
		for k, v := range state {
			stateCopy[k] = v
		}
	}

	if err := p.store.WriteCheckpoint(height, stateCopy); err != nil {
		return fmt.Errorf("write checkpoint: %w", err)
	}

	log.Printf("checkpoint saved at height %d", height)
	return nil
}

func (p *Pruner) WriteCheckpoint(height int64, state map[string]Account) error {
	return p.store.WriteCheckpoint(height, state)
}

func (p *Pruner) FastSyncInit(targetHeight int64) (map[string]Account, error) {
	heights, err := p.store.GetCheckpointHeights()
	if err != nil {
		return nil, fmt.Errorf("get checkpoint heights: %w", err)
	}

	if len(heights) == 0 {
		return nil, errors.New("no checkpoints available for fast sync")
	}

	var checkpointHeight int64
	for i := len(heights) - 1; i >= 0; i-- {
		if heights[i] <= targetHeight {
			checkpointHeight = heights[i]
			break
		}
	}

	if checkpointHeight == 0 {
		checkpointHeight = heights[0]
	}

	state, err := p.store.ReadCheckpoint(checkpointHeight)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint at %d: %w", checkpointHeight, err)
	}

	log.Printf("fast sync: loaded checkpoint at height %d, target %d", checkpointHeight, targetHeight)
	return state, nil
}

func (p *Pruner) AutoPrune() error {
	if !p.ShouldPrune() {
		return nil
	}

	canonical, err := p.store.ReadCanonical()
	if err != nil {
		return err
	}

	if len(canonical) == 0 {
		return nil
	}

	currentHeight := int64(canonical[len(canonical)-1].Height)
	pruneTo := currentHeight - p.pruneDepth

	checkpoints, err := p.store.GetCheckpointHeights()
	if err != nil {
		return err
	}

	var nearestCheckpoint int64
	for _, h := range checkpoints {
		if h < pruneTo {
			nearestCheckpoint = h
		}
	}

	if nearestCheckpoint > 0 && nearestCheckpoint >= p.pruneDepth {
		pruneTo = nearestCheckpoint
	}

	return p.PruneToHeight(pruneTo)
}

func applyBlockToStateFromPruner(state map[string]Account, b *Block) error {
	for _, tx := range b.Transactions {
		switch tx.Type {
		case TxCoinbase:
			acct := state[tx.ToAddress]
			acct.Balance += tx.Amount
			state[tx.ToAddress] = acct
		case TxTransfer:
			fromAddr, err := tx.FromAddress()
			if err != nil {
				continue
			}
			from := state[fromAddr]
			totalDebit := tx.Amount + tx.Fee
			if from.Balance < totalDebit {
				continue
			}
			from.Balance -= totalDebit
			state[fromAddr] = from

			to := state[tx.ToAddress]
			to.Balance += tx.Amount
			state[tx.ToAddress] = to
		}
	}
	return nil
}

func SerializeState(state map[string]Account) ([]byte, error) {
	return json.Marshal(state)
}

func DeserializeState(data []byte) (map[string]Account, error) {
	var state map[string]Account
	if err := json.Unmarshal(data, &state); err != nil {
		var legacy []AccountRecord
		if err2 := gob.NewDecoder(bytes.NewReader(data)).Decode(&legacy); err2 == nil {
			state = make(map[string]Account)
			for _, rec := range legacy {
				state[rec.Address] = rec.Account
			}
			return state, nil
		}
		return nil, err
	}
	return state, nil
}

type AccountRecord struct {
	Address string
	Account Account
}

func init() {
	gob.Register(&Account{})
}
