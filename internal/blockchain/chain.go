package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Neo4717/NeoCoin/config"
	"github.com/Neo4717/NeoCoin/internal/cache"
)

const (
	minFee = uint64(1)
)

type Blockchain struct {
	ChainID      uint64
	MinerAddress string

	consensus ConsensusParams
	rulesHash [32]byte

	events       EventSink
	onBlockMined func(lastBlockTime int64)

	mu             sync.RWMutex
	accountManager *AccountManager

	blocks []*Block
	state  map[string]Account
	store  ChainStore

	blocksByHash  map[string]*Block
	bestTipHash   string
	canonicalWork *big.Int

	txIndex map[string]TxLocation

	addressIndex map[string][]AddressTxEntry

	cache *cache.Cache

	pruner *Pruner
	cfg    *config.Config

	orphanPool *orphanPool
}

func (bc *Blockchain) Close() error {
	if bc.store != nil {
		if closer, ok := bc.store.(interface{ Close() error }); ok {
			return closer.Close()
		}
	}
	return nil
}

func LoadBlockchain(chainID uint64, minerAddress string, store ChainStore, genesisSupply uint64, cfg *config.Config) (*Blockchain, error) {
	envConsensus := defaultConsensusParamsFromEnv()
	genesisPath, err := GenesisPathFromEnv(chainID)
	if err != nil {
		return nil, err
	}
	genesisCfg, err := LoadGenesisConfig(genesisPath)
	if err != nil {
		return nil, err
	}
	if chainID != 0 && genesisCfg.ChainID != chainID {
		return nil, fmt.Errorf("genesis chainId mismatch: env=%d genesis=%d", chainID, genesisCfg.ChainID)
	}
	chainID = genesisCfg.ChainID

	if cfg == nil {
		cfg = config.LoadConfig()
	}

	bc := &Blockchain{
		ChainID:       chainID,
		MinerAddress:  minerAddress,
		consensus:     genesisCfg.ConsensusParams,
		state:         map[string]Account{},
		store:         store,
		blocksByHash:  map[string]*Block{},
		txIndex:       map[string]TxLocation{},
		addressIndex:  map[string][]AddressTxEntry{},
		canonicalWork: big.NewInt(0),
		cfg:           cfg,
	}
	bc.accountManager = NewAccountManager(256)

	maxBlocks := 10000
	maxBalances := 100000
	maxProofs := 10000
	if cfg != nil {
		if cfg.CacheMaxBlocks > 0 {
			maxBlocks = cfg.CacheMaxBlocks
		}
		if cfg.CacheMaxBalances > 0 {
			maxBalances = cfg.CacheMaxBalances
		}
		if cfg.CacheMaxProofs > 0 {
			maxProofs = cfg.CacheMaxProofs
		}
	}
	bc.cache = cache.NewCache(maxBlocks, maxBalances, maxProofs)
	bc.orphanPool = newOrphanPool(100)

	if minerAddress != "" {
		if !strings.HasPrefix(minerAddress, "NEO") {
			if _, err := hex.DecodeString(minerAddress); err != nil {
				return nil, fmt.Errorf("invalid MINER_ADDRESS (not hex or NEO00): %w", err)
			}
		}
	}

	envConsensus.MonetaryPolicy = bc.consensus.MonetaryPolicy
	if consensusEnvOverridesSet() && envConsensus != bc.consensus {
		log.Print("WARNING: consensus env vars are ignored because genesis.json is authoritative")
	}

	blocks, err := store.ReadCanonical()
	if err != nil {
		return nil, err
	}
	bc.blocks = blocks
	allBlocks, err := store.ReadAllBlocks()
	if err != nil {
		return nil, err
	}
	if len(allBlocks) > 0 {
		bc.blocksByHash = allBlocks
	}

	curRulesHash := bc.consensus.MustRulesHash()
	bc.rulesHash = curRulesHash

	ignoreRulesHash := envBool("IGNORE_RULES_HASH_CHECK", false)
	if !ignoreRulesHash && envBool("UNSAFE_IGNORE_RULES_HASH_CHECK", false) {
		ignoreRulesHash = true
		log.Print("WARNING: UNSAFE_IGNORE_RULES_HASH_CHECK is deprecated; use IGNORE_RULES_HASH_CHECK=true instead")
	}
	if ignoreRulesHash {
		log.Print("WARNING: IGNORE_RULES_HASH_CHECK=true; running with consensus params that do not match stored rules hash")
	}

	if stored, ok, err := store.GetRulesHash(); err != nil {
		return nil, err
	} else if ok {
		if len(stored) != 32 {
			return nil, fmt.Errorf("invalid stored rules hash length: %d", len(stored))
		}
		var storedHash [32]byte
		copy(storedHash[:], stored)
		if storedHash != curRulesHash {
			if ignoreRulesHash {
				log.Printf("WARNING: rules hash mismatch ignored: stored=%x current=%x", storedHash, curRulesHash)
			} else {
				return nil, fmt.Errorf("consensus params mismatch: stored rulesHash=%x current rulesHash=%x (set IGNORE_RULES_HASH_CHECK=true to bypass, or delete data/ to reinit)", storedHash, curRulesHash)
			}
		}
	} else {
		if len(bc.blocks) > 0 {
			log.Print("WARNING: initializing rules hash on an existing chain; ensure all nodes use identical consensus env vars")
		}
		if err := store.PutRulesHash(curRulesHash[:]); err != nil {
			return nil, err
		}
	}

	if len(bc.blocks) == 0 {
		genesis, err := BuildGenesisBlock(genesisCfg, bc.consensus)
		if err != nil {
			return nil, err
		}
		if err := bc.store.AppendCanonical(genesis); err != nil {
			return nil, err
		}
		_ = bc.store.PutBlock(genesis)
		bc.blocks = append(bc.blocks, genesis)
	} else {
		if err := ValidateGenesisBlock(bc.blocks[0], genesisCfg, bc.consensus); err != nil {
			return nil, err
		}
	}

	if len(bc.blocks) == 0 {
		return nil, errors.New("missing genesis block")
	}
	genesisHash, err := ensureBlockHash(bc.blocks[0], bc.consensus)
	if err != nil {
		return nil, err
	}
	if stored, ok, err := store.GetGenesisHash(); err != nil {
		return nil, err
	} else if ok {
		if !bytes.Equal(stored, genesisHash) {
			return nil, fmt.Errorf("genesis hash mismatch: stored=%x current=%x", stored, genesisHash)
		}
	} else {
		if err := store.PutGenesisHash(genesisHash); err != nil {
			return nil, err
		}
	}

	if err := bc.recomputeStateLocked(cfg); err != nil {
		return nil, err
	}
	bc.initCanonicalIndexesLocked()
	return bc, nil
}

func (bc *Blockchain) Consensus() ConsensusParams {
	return bc.consensus
}

func (bc *Blockchain) Mu() *sync.RWMutex {
	return &bc.mu
}

func (bc *Blockchain) Blocks() []*Block {
	return bc.blocks
}

func (bc *Blockchain) BlocksByHash() map[string]*Block {
	return bc.blocksByHash
}

func (bc *Blockchain) RulesHashHex() string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if bc.rulesHash == ([32]byte{}) {
		return ""
	}
	return hex.EncodeToString(bc.rulesHash[:])
}

func (bc *Blockchain) SetEventSink(sink EventSink) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.events = sink
}

func (bc *Blockchain) SetBlockMinedCallback(fn func(lastBlockTime int64)) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.onBlockMined = fn
}

func (bc *Blockchain) LatestBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.blocks[len(bc.blocks)-1]
}

func (bc *Blockchain) CanonicalWork() *big.Int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if bc.canonicalWork == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(bc.canonicalWork)
}

func (bc *Blockchain) BlockByHeight(height uint64) (*Block, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if height >= uint64(len(bc.blocks)) {
		return nil, false
	}
	return bc.blocks[int(height)], true
}

func (bc *Blockchain) CanonicalTxCount() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	total := 0
	for _, b := range bc.blocks {
		total += len(b.Transactions)
	}
	return total
}

func (bc *Blockchain) Balance(address string) (Account, bool) {
	unlock := bc.accountManager.RLockAccount(address)
	defer unlock()

	if bc.cache != nil {
		if bal, ok := bc.cache.GetCachedBalance(address); ok {
			return Account{Balance: uint64(bal)}, true
		}
	}

	acct, ok := bc.state[address]
	if ok && bc.cache != nil {
		bc.cache.CacheBalance(address, int64(acct.Balance))
	}
	return acct, ok
}

func (bc *Blockchain) TotalSupply() uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	var total uint64
	for _, acct := range bc.state {
		total += acct.Balance
	}
	return total
}

func (bc *Blockchain) CacheStats() string {
	if bc.cache == nil {
		return "cache not initialized"
	}
	s := bc.cache.Stats()
	return fmt.Sprintf("blocks: hits=%d miss=%d | balances: hits=%d miss=%d",
		s.BlockHits, s.BlockMisses, s.BalanceHits, s.BalanceMisses)
}

func (bc *Blockchain) SubmitTransfer(tx Transaction, requireAIAudit bool, aiApproved bool) (*Block, error) {
	if requireAIAudit && !aiApproved {
		return nil, errors.New("transaction rejected by AI auditor")
	}
	return bc.MineTransfers([]Transaction{tx})
}

func (bc *Blockchain) MineTransfers(transfers []Transaction) (*Block, error) {
	for attempt := 0; attempt < 100; attempt++ {
		block, err := bc.mineOneBlock(transfers)
		if err != nil {
			return nil, err
		}
		if block != nil {
			return block, nil
		}
	}
	return nil, errors.New("mining failed after 100 retries")
}

func (bc *Blockchain) mineOneBlock(transfers []Transaction) (*Block, error) {
	bc.mu.Lock()

	latest := bc.blocks[len(bc.blocks)-1]
	prevHash := append([]byte(nil), latest.Hash...)
	height := latest.Height + 1
	now := time.Now().Unix()
	ts := now
	if ts <= latest.TimestampUnix {
		ts = latest.TimestampUnix + 1
	}

	var fees uint64
	for _, tx := range transfers {
		if tx.Type != TxTransfer {
			bc.mu.Unlock()
			return nil, errors.New("only transfer txs can be mined")
		}
		if tx.ChainID == 0 {
			tx.ChainID = bc.ChainID
		}
		if tx.ChainID != bc.ChainID {
			bc.mu.Unlock()
			return nil, fmt.Errorf("wrong chainId: %d", tx.ChainID)
		}
		if err := tx.VerifyForConsensus(bc.consensus, height); err != nil {
			bc.mu.Unlock()
			return nil, err
		}
		if tx.Fee < minFee {
			bc.mu.Unlock()
			return nil, fmt.Errorf("fee too low: minFee=%d", minFee)
		}
		fees += tx.Fee
	}

	policy := bc.consensus.MonetaryPolicy
	reward := policy.BlockReward(height)
	minerFees := policy.MinerFeeAmount(fees)
	coinbase := Transaction{
		Type:      TxCoinbase,
		ChainID:   bc.ChainID,
		ToAddress: bc.MinerAddress,
		Amount:    reward + minerFees,
		Data:      fmt.Sprintf("block reward + fees (height=%d)", height),
	}

	txs := make([]Transaction, 0, 1+len(transfers))
	txs = append(txs, coinbase)
	txs = append(txs, transfers...)

	diffBits := nextDifficultyBitsFromPath(bc.consensus, bc.blocks)

	bc.mu.Unlock()

	newBlock := &Block{
		Version:        blockVersionForHeight(bc.consensus, height),
		Height:         height,
		TimestampUnix:  ts,
		PrevHash:       prevHash,
		DifficultyBits: diffBits,
		MinerAddress:   bc.MinerAddress,
		Transactions:   txs,
	}
	pow := NewProofOfWork(bc.consensus, newBlock)
	nonce, hash, err := pow.Run()
	if err != nil {
		return nil, err
	}
	newBlock.Nonce = nonce
	newBlock.Hash = hash

	bc.mu.Lock()

	if len(bc.blocks) == 0 || bc.blocks[len(bc.blocks)-1].Height >= height {
		bc.mu.Unlock()
		return nil, nil
	}
	if !bytes.Equal(bc.blocks[len(bc.blocks)-1].Hash, prevHash) {
		bc.mu.Unlock()
		return nil, nil
	}

	accountLocks := bc.lockAccountsForBlock(newBlock)

	if err := applyBlockToState(bc.consensus, bc.state, newBlock); err != nil {
		for _, unlock := range accountLocks {
			unlock()
		}
		bc.mu.Unlock()
		return nil, err
	}
	if err := bc.store.AppendCanonical(newBlock); err != nil {
		for _, unlock := range accountLocks {
			unlock()
		}
		bc.mu.Unlock()
		return nil, err
	}
	bc.blocks = append(bc.blocks, newBlock)
	bc.addToIndexLocked(newBlock)
	bc.indexTxsForBlockLocked(newBlock)
	bc.indexAddressTxsForBlockLocked(newBlock)
	bc.bestTipHash = hex.EncodeToString(newBlock.Hash)
	if bc.canonicalWork == nil {
		bc.canonicalWork = big.NewInt(0)
	}
	bc.canonicalWork.Add(bc.canonicalWork, WorkForDifficultyBits(newBlock.DifficultyBits))

	for _, unlock := range accountLocks {
		unlock()
	}

	onBlockMinedFn := bc.onBlockMined
	latestTimestamp := latest.TimestampUnix

	if bc.cache != nil {
		bc.cache.CacheBlock(bc.bestTipHash, newBlock)
		for _, tx := range newBlock.Transactions {
			if tx.Type == TxTransfer {
				fromAddr, _ := tx.FromAddress()
				if acct, ok := bc.state[fromAddr]; ok {
					bc.cache.CacheBalance(fromAddr, int64(acct.Balance))
				}
				if acct, ok := bc.state[tx.ToAddress]; ok {
					bc.cache.CacheBalance(tx.ToAddress, int64(acct.Balance))
				}
			}
		}
	}

	storeMode := bc.cfg
	pruneInterval := int64(0)
	if storeMode != nil && storeMode.StoreMode == "pruned" {
		pruneInterval = storeMode.CheckpointInterval
		if pruneInterval == 0 {
			pruneInterval = 100
		}
	}
	cfg := bc.cfg

	bc.mu.Unlock()

	if onBlockMinedFn != nil {
		onBlockMinedFn(latestTimestamp)
	}

	if pruneInterval > 0 && int64(newBlock.Height)%pruneInterval == 0 {
		pruner := NewPruner(bc.store, cfg)
		_ = pruner.StateCheckpoint(int64(newBlock.Height))
	}

	if ShouldCommit(newBlock.Height) {
		if err := bc.ComputeAndStoreCommitment(newBlock.Height); err != nil {
			log.Printf("WARNING: failed to store commitment at height %d: %v", newBlock.Height, err)
		}
	}

	go func() {
		bc.mu.RLock()
		eventSink := bc.events
		bc.mu.RUnlock()
		if eventSink != nil {
			eventSink.Publish(WSEvent{
				Type: "new_block",
				Data: map[string]any{
					"height":         newBlock.Height,
					"hash":           hex.EncodeToString(newBlock.Hash),
					"prevHash":       hex.EncodeToString(newBlock.PrevHash),
					"difficultyBits": newBlock.DifficultyBits,
					"txCount":        len(newBlock.Transactions),
					"addresses":      addressesForBlock(newBlock),
				},
			})
		}
	}()

	return newBlock, nil
}

func (bc *Blockchain) lockAccountsForBlock(b *Block) []func() {
	accounts := collectAccountsForBlock(b)
	unlocks := make([]func(), 0, len(accounts))
	for addr := range accounts {
		unlocks = append(unlocks, bc.accountManager.LockAccount(addr))
	}
	return unlocks
}

func collectAccountsForBlock(b *Block) map[string]struct{} {
	accounts := make(map[string]struct{})
	for _, tx := range b.Transactions {
		accounts[tx.ToAddress] = struct{}{}
		if tx.Type == TxTransfer {
			if from, err := tx.FromAddress(); err == nil {
				accounts[from] = struct{}{}
			}
		}
	}
	return accounts
}

func (bc *Blockchain) AuditChain() error {
	bc.mu.RLock()
	blocks := append([]*Block(nil), bc.blocks...)
	consensus := bc.consensus
	bc.mu.RUnlock()
	if len(blocks) == 0 {
		return errors.New("empty chain")
	}
	for i, b := range blocks {
		if i == 0 {
			if b.Height != 0 || len(b.PrevHash) != 0 {
				return errors.New("invalid genesis header")
			}
			if b.Version != blockVersionForHeight(consensus, 0) {
				return fmt.Errorf("bad block version at %d: expected %d got %d", b.Height, blockVersionForHeight(consensus, 0), b.Version)
			}
		} else {
			prev := blocks[i-1]
			if b.Height != prev.Height+1 {
				return fmt.Errorf("bad height at %d", b.Height)
			}
			if string(b.PrevHash) != string(prev.Hash) {
				return fmt.Errorf("bad prev hash at %d", b.Height)
			}
			if err := validateBlockTime(consensus, blocks, i); err != nil {
				return err
			}
			if consensus.DifficultyEnable {
				expected := expectedDifficultyBitsForBlockIndex(consensus, blocks, i)
				if b.DifficultyBits != expected {
					return fmt.Errorf("bad difficulty at %d: expected %d got %d", b.Height, expected, b.DifficultyBits)
				}
			}
			if b.Version != blockVersionForHeight(consensus, b.Height) {
				return fmt.Errorf("bad block version at %d: expected %d got %d", b.Height, blockVersionForHeight(consensus, b.Height), b.Version)
			}
		}
		if b.DifficultyBits == 0 || b.DifficultyBits > maxDifficultyBits {
			return fmt.Errorf("difficultyBits out of range at %d: %d", b.Height, b.DifficultyBits)
		}
		ok, err := NewProofOfWork(consensus, b).Validate()
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("invalid pow at height %d", b.Height)
		}
		for _, tx := range b.Transactions {
			if tx.ChainID == 0 {
				return fmt.Errorf("missing chainId at height %d", b.Height)
			}
			if err := tx.VerifyForConsensus(consensus, b.Height); err != nil {
				return fmt.Errorf("invalid tx at height %d: %w", b.Height, err)
			}
		}
	}
	return bc.recomputeState()
}

func (bc *Blockchain) recomputeState() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.recomputeStateLocked(bc.cfg)
}

func (bc *Blockchain) recomputeStateLocked(cfg *config.Config) error {
	bc.state = map[string]Account{}

	storeMode := StoreModeFull
	if cfg != nil && cfg.StoreMode == "pruned" {
		storeMode = StoreModePruned
	}

	if storeMode == StoreModePruned && len(bc.blocks) > 0 {
		checkpoints, err := bc.store.GetCheckpointHeights()
		if err == nil && len(checkpoints) > 0 {
			latestCheckpoint := checkpoints[len(checkpoints)-1]
			var replayFrom int64
			for _, b := range bc.blocks {
				if int64(b.Height) == latestCheckpoint {
					replayFrom = latestCheckpoint + 1
					break
				}
			}

			if replayFrom == 0 && len(bc.blocks) > 0 {
				replayFrom = int64(bc.blocks[0].Height)
			}

			if replayFrom > 0 {
				for _, b := range bc.blocks {
					if int64(b.Height) == latestCheckpoint {
						state, err := bc.store.ReadCheckpoint(latestCheckpoint)
						if err == nil && state != nil {
							bc.state = state
						}
						break
					}
				}
			}

			for _, b := range bc.blocks {
				if int64(b.Height) >= replayFrom {
					if err := applyBlockToState(bc.consensus, bc.state, b); err != nil {
						return fmt.Errorf("apply block %d: %w", b.Height, err)
					}
				}
			}
			return nil
		}
	}

	for _, b := range bc.blocks {
		if err := applyBlockToState(bc.consensus, bc.state, b); err != nil {
			return fmt.Errorf("apply block %d: %w", b.Height, err)
		}
	}
	return nil
}

func (bc *Blockchain) TxByID(txid string) (Transaction, TxLocation, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	loc, ok := bc.txIndex[txid]
	if !ok {
		return Transaction{}, TxLocation{}, false
	}
	if loc.Height >= uint64(len(bc.blocks)) || loc.Index < 0 {
		return Transaction{}, TxLocation{}, false
	}
	b := bc.blocks[int(loc.Height)]
	if loc.Index >= len(b.Transactions) {
		return Transaction{}, TxLocation{}, false
	}
	if hex.EncodeToString(b.Hash) != loc.BlockHashHex {
		return Transaction{}, TxLocation{}, false
	}
	return b.Transactions[loc.Index], loc, true
}

func (bc *Blockchain) indexTxsForBlockLocked(b *Block) {
	if bc.txIndex == nil {
		bc.txIndex = map[string]TxLocation{}
	}
	hashHex := hex.EncodeToString(b.Hash)
	for i, tx := range b.Transactions {
		if tx.Type != TxTransfer {
			continue
		}
		txid, err := TxIDHexForConsensus(tx, bc.consensus, b.Height)
		if err != nil {
			continue
		}
		bc.txIndex[txid] = TxLocation{Height: b.Height, BlockHashHex: hashHex, Index: i}
	}
}

func (bc *Blockchain) indexAddressTxsForBlockLocked(b *Block) {
	if bc.addressIndex == nil {
		bc.addressIndex = map[string][]AddressTxEntry{}
	}
	hashHex := hex.EncodeToString(b.Hash)
	for i, tx := range b.Transactions {
		if tx.Type != TxTransfer {
			continue
		}
		txid, err := TxIDHexForConsensus(tx, bc.consensus, b.Height)
		if err != nil {
			continue
		}
		from, err := tx.FromAddress()
		if err != nil {
			continue
		}
		entry := AddressTxEntry{
			TxID: txid,
			Location: TxLocation{
				Height:       b.Height,
				BlockHashHex: hashHex,
				Index:        i,
			},
			FromAddr:  from,
			ToAddress: tx.ToAddress,
			Amount:    tx.Amount,
			Fee:       tx.Fee,
			Nonce:     tx.Nonce,
		}
		bc.addressIndex[from] = append(bc.addressIndex[from], entry)
		if tx.ToAddress != from {
			bc.addressIndex[tx.ToAddress] = append(bc.addressIndex[tx.ToAddress], entry)
		}
	}
}

func (bc *Blockchain) reindexAllTxsLocked() {
	bc.txIndex = map[string]TxLocation{}
	for _, b := range bc.blocks {
		bc.indexTxsForBlockLocked(b)
	}
}

func (bc *Blockchain) reindexAllAddressTxsLocked() {
	bc.addressIndex = map[string][]AddressTxEntry{}
	for _, b := range bc.blocks {
		bc.indexAddressTxsForBlockLocked(b)
	}
}

func (bc *Blockchain) AddressTxs(address string, limit int, cursor int) ([]AddressTxEntry, int, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if bc.addressIndex == nil {
		return nil, 0, false
	}
	all := bc.addressIndex[address]
	if len(all) == 0 {
		return []AddressTxEntry{}, 0, false
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if cursor < 0 {
		cursor = 0
	}
	start := len(all) - 1 - cursor
	if start < 0 {
		return []AddressTxEntry{}, cursor, false
	}
	out := make([]AddressTxEntry, 0, limit)
	i := start
	for i >= 0 && len(out) < limit {
		out = append(out, all[i])
		i--
	}
	nextCursor := cursor + len(out)
	more := (len(all) - 1 - nextCursor) >= 0
	return out, nextCursor, more
}

func applyBlockToState(p ConsensusParams, state map[string]Account, b *Block) error {
	if p.MaxBlockSize > 0 {
		size, err := blockSizeForConsensus(b)
		if err != nil {
			return err
		}
		if uint64(size) > p.MaxBlockSize {
			return fmt.Errorf("block too large: %d bytes (max %d)", size, p.MaxBlockSize)
		}
	}
	if len(b.Transactions) == 0 {
		return errors.New("block has no transactions")
	}
	if b.Transactions[0].Type != TxCoinbase {
		return errors.New("first tx must be coinbase")
	}

	if b.Height > 0 {
		if err := ValidateAddress(b.MinerAddress); err != nil {
			return fmt.Errorf("invalid minerAddress: %w", err)
		}
		var fees uint64
		for _, tx := range b.Transactions[1:] {
			if tx.Type != TxTransfer {
				continue
			}
			fees += tx.Fee
		}
		cb := b.Transactions[0]
		if cb.ToAddress != b.MinerAddress {
			return errors.New("coinbase toAddress must match minerAddress")
		}
		policy := p.MonetaryPolicy
		expected := policy.BlockReward(b.Height) + policy.MinerFeeAmount(fees)
		if cb.Amount != expected {
			return fmt.Errorf("bad coinbase amount: expected %d got %d", expected, cb.Amount)
		}
	}

	for i, tx := range b.Transactions {
		switch tx.Type {
		case TxCoinbase:
			if i != 0 {
				return errors.New("coinbase must be first")
			}
			if err := tx.VerifyForConsensus(p, b.Height); err != nil {
				return err
			}
			acct := state[tx.ToAddress]
			acct.Balance += tx.Amount
			state[tx.ToAddress] = acct
		case TxTransfer:
			if err := tx.VerifyForConsensus(p, b.Height); err != nil {
				return err
			}
			fromAddr, err := tx.FromAddress()
			if err != nil {
				return err
			}
			from := state[fromAddr]
			if from.Nonce+1 != tx.Nonce {
				return fmt.Errorf("bad nonce for %s: expected %d got %d", fromAddr, from.Nonce+1, tx.Nonce)
			}
			totalDebit := tx.Amount + tx.Fee
			if from.Balance < totalDebit {
				return fmt.Errorf("insufficient funds for %s", fromAddr)
			}
			from.Balance -= totalDebit
			from.Nonce = tx.Nonce
			state[fromAddr] = from

			to := state[tx.ToAddress]
			to.Balance += tx.Amount
			state[tx.ToAddress] = to
		default:
			return fmt.Errorf("unknown tx type: %q", tx.Type)
		}
	}
	return nil
}

type TxEntry interface {
	Tx() Transaction
	TxIDHex() string
	FromAddress() (string, error)
}

func (bc *Blockchain) SelectMempoolTxs(entries []TxEntry, max int) ([]Transaction, []string, error) {
	if max <= 0 {
		max = 100
	}

	bc.mu.RLock()
	baseState := make(map[string]Account, len(bc.state))
	for k, v := range bc.state {
		baseState[k] = v
	}
	nextHeight := bc.LatestBlock().Height + 1
	bc.mu.RUnlock()

	bySender := make(map[string][]SelectedTxEntry)
	for _, e := range entries {
		if e.Tx().Type != TxTransfer {
			continue
		}
		fromAddr, err := e.FromAddress()
		if err != nil {
			continue
		}
		entry := SelectedTxEntry{
			tx:      e.Tx(),
			txIDHex: e.TxIDHex(),
		}
		bySender[fromAddr] = append(bySender[fromAddr], entry)
	}

	for sender := range bySender {
		sort.Slice(bySender[sender], func(i, j int) bool {
			return bySender[sender][i].tx.Nonce < bySender[sender][j].tx.Nonce
		})
	}

	var wg sync.WaitGroup
	results := make(chan senderResult, len(bySender))
	maxWorkers := runtime.NumCPU()
	if maxWorkers > len(bySender) {
		maxWorkers = len(bySender)
	}
	semaphore := make(chan struct{}, maxWorkers)

	for sender, txs := range bySender {
		wg.Add(1)
		go func(s string, txs []SelectedTxEntry) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			valid := bc.validateTxsForSender(baseState, s, txs, bc.ChainID, bc.consensus, nextHeight)
			results <- senderResult{sender: s, valid: valid}
		}(sender, txs)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allValid []SelectedTxEntry
	for sr := range results {
		allValid = append(allValid, sr.valid...)
	}

	sort.Slice(allValid, func(i, j int) bool {
		return allValid[i].tx.Fee > allValid[j].tx.Fee
	})

	if len(allValid) > max {
		allValid = allValid[:max]
	}

	var picked []Transaction
	var pickedIDs []string
	for _, e := range allValid {
		picked = append(picked, e.tx)
		pickedIDs = append(pickedIDs, e.txIDHex)
	}

	return picked, pickedIDs, nil
}

type SelectedTxEntry struct {
	tx      Transaction
	txIDHex string
}

type senderResult struct {
	sender string
	valid  []SelectedTxEntry
}

func (bc *Blockchain) validateTxsForSender(baseState map[string]Account, addr string, txs []SelectedTxEntry, chainID uint64, consensus ConsensusParams, height uint64) []SelectedTxEntry {
	unlock := bc.accountManager.LockAccount(addr)
	defer unlock()

	state := make(map[string]Account, len(baseState))
	for k, v := range baseState {
		state[k] = v
	}

	nonce := state[addr].Nonce
	var valid []SelectedTxEntry
	for _, entry := range txs {
		tx := entry.tx
		if tx.ChainID == 0 {
			tx.ChainID = chainID
		}
		if err := tx.VerifyForConsensus(consensus, height); err != nil {
			continue
		}
		if tx.Nonce != nonce+1 {
			continue
		}
		from := state[addr]
		totalDebit := tx.Amount + tx.Fee
		if from.Balance < totalDebit {
			continue
		}
		from.Balance -= totalDebit
		from.Nonce = tx.Nonce
		state[addr] = from

		to := state[tx.ToAddress]
		to.Balance += tx.Amount
		state[tx.ToAddress] = to

		valid = append(valid, entry)
		nonce = tx.Nonce
	}
	return valid
}

func (bc *Blockchain) initCanonicalIndexesLocked() {
	if bc.blocksByHash == nil {
		bc.blocksByHash = map[string]*Block{}
	}
	for _, b := range bc.blocks {
		bc.addToIndexLocked(b)
	}
	bc.bestTipHash = hex.EncodeToString(bc.blocks[len(bc.blocks)-1].Hash)
	bc.reindexAllTxsLocked()
	bc.reindexAllAddressTxsLocked()
	bc.canonicalWork = big.NewInt(0)
	for _, b := range bc.blocks {
		bc.canonicalWork.Add(bc.canonicalWork, WorkForDifficultyBits(b.DifficultyBits))
	}
}

func (bc *Blockchain) addToIndexLocked(b *Block) {
	if len(b.Hash) == 0 {
		return
	}
	bc.blocksByHash[hex.EncodeToString(b.Hash)] = b
}

func addressesForBlock(b *Block) []string {
	if b == nil {
		return nil
	}
	set := map[string]struct{}{}
	for _, tx := range b.Transactions {
		if tx.ToAddress != "" {
			set[strings.ToLower(tx.ToAddress)] = struct{}{}
		}
		if tx.Type == TxTransfer {
			from, err := tx.FromAddress()
			if err == nil && from != "" {
				set[strings.ToLower(from)] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for addr := range set {
		out = append(out, addr)
	}
	sort.Strings(out)
	return out
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func SHA256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
