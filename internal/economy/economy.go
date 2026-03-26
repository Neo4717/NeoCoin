package economy

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Neo4717/NeoCoin/internal/blockchain"
)

type FeeMarket struct {
	mu                   sync.RWMutex
	baseFee              uint64
	minPriorityFee       uint64
	maxPriorityFee       uint64
	feeHistory           []FeeHistoryEntry
	maxHistorySize       int
	elasticityMultiplier uint64
	baseFeeChangeRate    int64
}

type FeeHistoryEntry struct {
	BlockHeight       uint64
	BaseFee           uint64
	GasUsed           uint64
	GasLimit          uint64
	PriorityFeePerGas uint64
	Timestamp         time.Time
}

type FeeEstimate struct {
	Low      uint64
	Medium   uint64
	High     uint64
	BaseFee  uint64
	GasUsed  uint64
	GasLimit uint64
}

func NewFeeMarket() *FeeMarket {
	return &FeeMarket{
		baseFee:              1000000000,
		minPriorityFee:       1000000,
		maxPriorityFee:       1000000000,
		feeHistory:           make([]FeeHistoryEntry, 0, 100),
		maxHistorySize:       100,
		elasticityMultiplier: 2,
		baseFeeChangeRate:    8,
	}
}

func (fm *FeeMarket) SetBaseFee(baseFee uint64) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.baseFee = baseFee
}

func (fm *FeeMarket) GetBaseFee() uint64 {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.baseFee
}

func (fm *FeeMarket) CalculateBaseFee(parentGasUsed, parentGasLimit, height uint64) uint64 {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if parentGasLimit == 0 {
		return fm.baseFee
	}

	gasUsedRatio := float64(parentGasUsed) / float64(parentGasLimit)

	multiplier := int64(fm.elasticityMultiplier)
	changeRate := fm.baseFeeChangeRate

	if gasUsedRatio > 0.5 {
		excess := gasUsedRatio - 0.5
		factor := 1 + (excess * float64(multiplier) / float64(changeRate))
		newBaseFee := uint64(float64(fm.baseFee) * factor)

		minFee := fm.baseFee / uint64(changeRate)
		if minFee < 1 {
			minFee = 1
		}
		if newBaseFee < minFee {
			newBaseFee = minFee
		}

		maxFee := fm.baseFee * uint64(changeRate)
		if newBaseFee > maxFee {
			newBaseFee = maxFee
		}

		fm.baseFee = newBaseFee
	} else if gasUsedRatio < 0.5 {
		deficit := 0.5 - gasUsedRatio
		factor := 1 - (deficit * float64(multiplier) / float64(changeRate))
		newBaseFee := uint64(float64(fm.baseFee) * factor)

		minFee := fm.baseFee / uint64(changeRate)
		if minFee < 1 {
			minFee = 1
		}
		if newBaseFee < minFee {
			newBaseFee = minFee
		}

		fm.baseFee = newBaseFee
	}

	return fm.baseFee
}

func (fm *FeeMarket) PriorityFee(senderFee uint64, userTipMax uint64) uint64 {
	if userTipMax < fm.minPriorityFee {
		return fm.minPriorityFee
	}
	if userTipMax > fm.maxPriorityFee {
		return fm.maxPriorityFee
	}
	if senderFee < userTipMax {
		return senderFee
	}
	return userTipMax
}

func (fm *FeeMarket) CalculateEffectiveFee(gasPrice, gasLimit uint64) uint64 {
	baseFee := fm.GetBaseFee()
	return (baseFee + gasPrice) * gasLimit
}

func (fm *FeeMarket) AddBlockToHistory(entry FeeHistoryEntry) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.feeHistory = append(fm.feeHistory, entry)
	if len(fm.feeHistory) > fm.maxHistorySize {
		fm.feeHistory = fm.feeHistory[len(fm.feeHistory)-fm.maxHistorySize:]
	}
}

func (fm *FeeMarket) GetFeeHistory(count int) []FeeHistoryEntry {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if count > len(fm.feeHistory) {
		count = len(fm.feeHistory)
	}
	if count == 0 {
		return nil
	}

	result := make([]FeeHistoryEntry, count)
	copy(result, fm.feeHistory[len(fm.feeHistory)-count:])
	return result
}

func (fm *FeeMarket) EstimateFee() FeeEstimate {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	history := fm.feeHistory
	if len(history) == 0 {
		return FeeEstimate{
			Low:     fm.baseFee + fm.minPriorityFee,
			Medium:  fm.baseFee + fm.minPriorityFee*10,
			High:    fm.baseFee + fm.minPriorityFee*100,
			BaseFee: fm.baseFee,
		}
	}

	var totalBaseFee, totalPriorityFee, totalGasUsed, totalGasLimit uint64
	recentBlocks := 10
	if len(history) < recentBlocks {
		recentBlocks = len(history)
	}

	for i := len(history) - recentBlocks; i < len(history); i++ {
		totalBaseFee += history[i].BaseFee
		totalPriorityFee += history[i].PriorityFeePerGas
		totalGasUsed += history[i].GasUsed
		totalGasLimit += history[i].GasLimit
	}

	avgBaseFee := totalBaseFee / uint64(recentBlocks)
	avgPriorityFee := totalPriorityFee / uint64(recentBlocks)
	avgGasUsed := totalGasUsed / uint64(recentBlocks)
	avgGasLimit := totalGasLimit / uint64(recentBlocks)

	return FeeEstimate{
		Low:      avgBaseFee + avgPriorityFee/2,
		Medium:   avgBaseFee + avgPriorityFee,
		High:     avgBaseFee + avgPriorityFee*2,
		BaseFee:  avgBaseFee,
		GasUsed:  avgGasUsed,
		GasLimit: avgGasLimit,
	}
}

func (fm *FeeMarket) EstimatePriorityFee(urgency PriorityLevel) uint64 {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if len(fm.feeHistory) == 0 {
		return fm.minPriorityFee
	}

	var totalPriorityFee uint64
	recentCount := 3
	if len(fm.feeHistory) < recentCount {
		recentCount = len(fm.feeHistory)
	}

	for i := len(fm.feeHistory) - recentCount; i < len(fm.feeHistory); i++ {
		totalPriorityFee += fm.feeHistory[i].PriorityFeePerGas
	}

	avgPriorityFee := totalPriorityFee / uint64(recentCount)

	switch urgency {
	case PriorityLow:
		return avgPriorityFee / 2
	case PriorityMedium:
		return avgPriorityFee
	case PriorityHigh:
		return avgPriorityFee * 2
	case PriorityVeryHigh:
		return avgPriorityFee * 5
	default:
		return avgPriorityFee
	}
}

type PriorityLevel int

const (
	PriorityLow PriorityLevel = iota
	PriorityMedium
	PriorityHigh
	PriorityVeryHigh
)

type MiningIncentives struct {
	mu             sync.RWMutex
	policy         blockchain.MonetaryPolicy
	blockRewards   []uint64
	rewardHistory  []RewardHistoryEntry
	maxHistorySize int
}

type RewardHistoryEntry struct {
	Height      uint64
	BlockReward uint64
	FeesBurned  uint64
	FeesToMiner uint64
	TotalReward uint64
	Timestamp   time.Time
}

func NewMiningIncentives(policy blockchain.MonetaryPolicy) *MiningIncentives {
	return &MiningIncentives{
		policy:         policy,
		blockRewards:   make([]uint64, 0, 1000),
		rewardHistory:  make([]RewardHistoryEntry, 0, 100),
		maxHistorySize: 100,
	}
}

func (mi *MiningIncentives) SetPolicy(policy blockchain.MonetaryPolicy) {
	mi.mu.Lock()
	defer mi.mu.Unlock()
	mi.policy = policy
}

func (mi *MiningIncentives) GetPolicy() blockchain.MonetaryPolicy {
	mi.mu.RLock()
	defer mi.mu.RUnlock()
	return mi.policy
}

func (mi *MiningIncentives) CalculateBlockReward(height uint64) uint64 {
	mi.mu.RLock()
	defer mi.mu.RUnlock()

	policy := mi.policy
	if policy.HalvingInterval == 0 {
		return policy.InitialBlockReward
	}

	halvings := height / policy.HalvingInterval
	if halvings >= 64 {
		return policy.TailEmission
	}

	reward := policy.InitialBlockReward >> halvings
	if reward == 0 {
		return policy.TailEmission
	}
	return reward
}

func (mi *MiningIncentives) CalculateTotalFees(txs []blockchain.Transaction) uint64 {
	var total uint64
	for _, tx := range txs {
		total += tx.Fee
	}
	return total
}

func (mi *MiningIncentives) CalculateMinerReward(height uint64, txs []blockchain.Transaction) (minerReward, burnedFees uint64) {
	mi.mu.Lock()
	defer mi.mu.Unlock()

	blockReward := mi.policy.BlockReward(height)
	totalFees := mi.CalculateTotalFees(txs)

	minerFeeShare := mi.policy.MinerFeeAmount(totalFees)
	burnedFees = totalFees - minerFeeShare

	totalReward := blockReward + minerFeeShare

	mi.rewardHistory = append(mi.rewardHistory, RewardHistoryEntry{
		Height:      height,
		BlockReward: blockReward,
		FeesBurned:  burnedFees,
		FeesToMiner: minerFeeShare,
		TotalReward: totalReward,
		Timestamp:   time.Now(),
	})

	if len(mi.rewardHistory) > mi.maxHistorySize {
		mi.rewardHistory = mi.rewardHistory[len(mi.rewardHistory)-mi.maxHistorySize:]
	}

	return totalReward, burnedFees
}

func (mi *MiningIncentives) GetRewardHistory(count int) []RewardHistoryEntry {
	mi.mu.RLock()
	defer mi.mu.RUnlock()

	if count > len(mi.rewardHistory) {
		count = len(mi.rewardHistory)
	}
	if count == 0 {
		return nil
	}

	result := make([]RewardHistoryEntry, count)
	copy(result, mi.rewardHistory[len(mi.rewardHistory)-count:])
	return result
}

func (mi *MiningIncentives) CalculateUncleReward(uncleHeight, nephewHeight uint64) uint64 {
	nephewReward := mi.CalculateBlockReward(nephewHeight)
	age := nephewHeight - uncleHeight
	if age > 8 {
		age = 8
	}
	return nephewReward * uint64(8-age) / 8
}

func (mi *MiningIncentives) GetSubsidySchedule(height, numPeriods uint64) []SubsidyPeriod {
	mi.mu.RLock()
	defer mi.mu.RUnlock()

	periods := make([]SubsidyPeriod, 0, numPeriods)
	halvingInterval := mi.policy.HalvingInterval

	for i := uint64(0); i < numPeriods && (height+i*halvingInterval) < math.MaxUint64; i++ {
		periodHeight := height + i*halvingInterval
		reward := mi.policy.BlockReward(periodHeight)
		periods = append(periods, SubsidyPeriod{
			StartHeight: periodHeight,
			EndHeight:   periodHeight + halvingInterval,
			Reward:      reward,
		})
	}

	return periods
}

type SubsidyPeriod struct {
	StartHeight uint64
	EndHeight   uint64
	Reward      uint64
}

func (mi *MiningIncentives) CalculateSustainabilityIndex(lookbackBlocks uint64) float64 {
	mi.mu.RLock()
	defer mi.mu.RUnlock()

	if len(mi.rewardHistory) == 0 {
		return 0
	}

	recentCount := int(lookbackBlocks)
	if len(mi.rewardHistory) < recentCount {
		recentCount = len(mi.rewardHistory)
	}

	var totalBlockReward, totalFees uint64
	startIdx := len(mi.rewardHistory) - recentCount
	for i := startIdx; i < len(mi.rewardHistory); i++ {
		totalBlockReward += mi.rewardHistory[i].BlockReward
		totalFees += mi.rewardHistory[i].FeesToMiner
	}

	if totalBlockReward == 0 {
		return 0
	}

	return float64(totalFees) / float64(totalBlockReward)
}

type EconomicSecurity struct {
	mu                     sync.RWMutex
	minerHashRate          uint64
	coinMarketCap          uint64
	hardwareCostPerHash    float64
	electricityCostPerHour float64
	attackWindow           time.Duration
	selfishMiningThreshold float64
	minerPerformances      map[string]MinerPerformance
	blockWindow            int
	feeSnipingWindow       uint64
}

type MinerPerformance struct {
	MinerAddress   string
	BlocksMined    uint64
	TotalHashes    uint64
	FirstBlockTime time.Time
	LastBlockTime  time.Time
	Revenue        uint64
	SelfishMining  bool
}

func NewEconomicSecurity(hashRate uint64, marketCap uint64) *EconomicSecurity {
	return &EconomicSecurity{
		minerHashRate:          hashRate,
		coinMarketCap:          marketCap,
		hardwareCostPerHash:    0.00000001,
		electricityCostPerHour: 0.05,
		attackWindow:           10 * time.Minute,
		selfishMiningThreshold: 0.25,
		minerPerformances:      make(map[string]MinerPerformance),
		blockWindow:            100,
		feeSnipingWindow:       6,
	}
}

func (es *EconomicSecurity) Calculate51AttackCost(durationHours float64) uint64 {
	es.mu.RLock()
	defer es.mu.RUnlock()

	requiredHashRate := es.minerHashRate * 51 / 100

	hoursPerBlock := 15 * 60.0
	blocksNeeded := int(durationHours * 3600 / hoursPerBlock)
	if blocksNeeded < 1 {
		blocksNeeded = 1
	}

	totalHashes := requiredHashRate * uint64(blocksNeeded)

	hardwareCost := float64(totalHashes) * es.hardwareCostPerHash
	electricityCost := durationHours * es.electricityCostPerHour

	totalCost := hardwareCost + electricityCost

	return uint64(totalCost)
}

func (es *EconomicSecurity) CalculateAttackROI(attackerShare float64, durationHours float64) float64 {
	cost := es.Calculate51AttackCost(durationHours)
	if cost == 0 {
		return 0
	}

	blockReward := uint64(100)
	blocksPerHour := 3600 / 15
	totalBlocks := uint64(durationHours) * uint64(blocksPerHour)

	revenue := blockReward * totalBlocks * uint64(attackerShare) / 100

	return float64(revenue) / float64(cost)
}

func (es *EconomicSecurity) Is51AttackProfitable(durationHours float64) bool {
	roi := es.CalculateAttackROI(51, durationHours)
	return roi > 1.0
}

func (es *EconomicSecurity) DetectSelfishMining(miner string, blocksMined, totalBlocks uint64) bool {
	es.mu.Lock()
	defer es.mu.Unlock()

	if totalBlocks == 0 {
		return false
	}

	miningRatio := float64(blocksMined) / float64(totalBlocks)

	if perf, ok := es.minerPerformances[miner]; ok {
		prevRatio := float64(perf.BlocksMined) / float64(es.blockWindow)
		if prevRatio > es.selfishMiningThreshold || miningRatio > es.selfishMiningThreshold {
			perf.SelfishMining = true
			es.minerPerformances[miner] = perf
			return true
		}
	}

	return miningRatio > es.selfishMiningThreshold
}

func (es *EconomicSecurity) RecordMinerBlock(miner string, timestamp time.Time) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if perf, ok := es.minerPerformances[miner]; ok {
		perf.BlocksMined++
		perf.LastBlockTime = timestamp
		if perf.FirstBlockTime.IsZero() {
			perf.FirstBlockTime = timestamp
		}
		es.minerPerformances[miner] = perf
	} else {
		es.minerPerformances[miner] = MinerPerformance{
			MinerAddress:   miner,
			BlocksMined:    1,
			FirstBlockTime: timestamp,
			LastBlockTime:  timestamp,
		}
	}
}

func (es *EconomicSecurity) GetMinerPerformance(miner string) (MinerPerformance, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	perf, ok := es.minerPerformances[miner]
	return perf, ok
}

func (es *EconomicSecurity) GetSelfishMiners() []string {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var selfishMiners []string
	for miner, perf := range es.minerPerformances {
		if perf.SelfishMining {
			selfishMiners = append(selfishMiners, miner)
		}
	}
	return selfishMiners
}

func (es *EconomicSecurity) PreventFeeSniping(blockHeight, txBlockHeight uint64) bool {
	es.mu.RLock()
	defer es.mu.RUnlock()

	if blockHeight <= txBlockHeight {
		return false
	}

	window := es.feeSnipingWindow
	if blockHeight-txBlockHeight > window {
		return false
	}

	return true
}

func (es *EconomicSecurity) SetFeeSnipingWindow(window uint64) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.feeSnipingWindow = window
}

func (es *EconomicSecurity) ValidateTransactionOrdering(txs []blockchain.Transaction) error {
	if len(txs) <= 1 {
		return nil
	}

	for i := 1; i < len(txs); i++ {
		if txs[i].Nonce < txs[i-1].Nonce {
			return errors.New("transaction nonce ordering violation")
		}
	}

	return nil
}

func (es *EconomicSecurity) AntiFrontRunCheck(tx blockchain.Transaction, poolTxs []blockchain.Transaction) bool {
	for _, poolTx := range poolTxs {
		if tx.ToAddress == poolTx.ToAddress && tx.Amount == poolTx.Amount {
			if tx.Fee <= poolTx.Fee {
				return true
			}
		}
	}
	return false
}

func (es *EconomicSecurity) UpdateHashRate(hashRate uint64) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.minerHashRate = hashRate
}

func (es *EconomicSecurity) UpdateMarketCap(marketCap uint64) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.coinMarketCap = marketCap
}

func (es *EconomicSecurity) GetNetworkSecurity() NetworkSecurity {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return NetworkSecurity{
		TotalHashRate:    es.minerHashRate,
		MarketCap:        es.coinMarketCap,
		AttackCost1Hour:  es.Calculate51AttackCost(1),
		AttackCost24Hour: es.Calculate51AttackCost(24),
	}
}

type NetworkSecurity struct {
	TotalHashRate    uint64
	MarketCap        uint64
	AttackCost1Hour  uint64
	AttackCost24Hour uint64
}

type DynamicBlockSize struct {
	mu                sync.RWMutex
	baseSize          uint64
	minSize           uint64
	maxSize           uint64
	currentSize       uint64
	targetUtilization float64
	windowSize        int
	gasUsedHistory    []uint64
}

func NewDynamicBlockSize(baseSize uint64) *DynamicBlockSize {
	return &DynamicBlockSize{
		baseSize:          baseSize,
		minSize:           baseSize / 2,
		maxSize:           baseSize * 2,
		currentSize:       baseSize,
		targetUtilization: 0.5,
		windowSize:        50,
		gasUsedHistory:    make([]uint64, 0, 50),
	}
}

func (dbs *DynamicBlockSize) UpdateBlockSize(gasUsed uint64) {
	dbs.mu.Lock()
	defer dbs.mu.Unlock()

	dbs.gasUsedHistory = append(dbs.gasUsedHistory, gasUsed)
	if len(dbs.gasUsedHistory) > dbs.windowSize {
		dbs.gasUsedHistory = dbs.gasUsedHistory[len(dbs.gasUsedHistory)-dbs.windowSize:]
	}

	var total uint64
	for _, g := range dbs.gasUsedHistory {
		total += g
	}
	avgGasUsed := total / uint64(len(dbs.gasUsedHistory))

	utilization := float64(avgGasUsed) / float64(dbs.currentSize)

	if utilization > dbs.targetUtilization {
		factor := 1 + (utilization - dbs.targetUtilization)
		newSize := uint64(float64(dbs.currentSize) * (1 + factor*0.1))
		if newSize > dbs.maxSize {
			newSize = dbs.maxSize
		}
		dbs.currentSize = newSize
	} else if utilization < dbs.targetUtilization {
		factor := dbs.targetUtilization - utilization
		newSize := uint64(float64(dbs.currentSize) * (1 - factor*0.1))
		if newSize < dbs.minSize {
			newSize = dbs.minSize
		}
		dbs.currentSize = newSize
	}
}

func (dbs *DynamicBlockSize) GetCurrentSize() uint64 {
	dbs.mu.RLock()
	defer dbs.mu.RUnlock()
	return dbs.currentSize
}

func ValidateTransaction(tx blockchain.Transaction) error {
	if tx.ToAddress == "" {
		return errors.New("transaction must have a recipient address")
	}
	if tx.Amount == 0 && tx.Fee == 0 {
		return errors.New("transaction must have amount or fee")
	}
	return nil
}

func EstimateTransactionSize(tx blockchain.Transaction) int {
	baseSize := 40
	if tx.FromPubKey != nil {
		baseSize += len(tx.FromPubKey)
	}
	baseSize += len(tx.ToAddress)
	baseSize += len(tx.Data)
	baseSize += len(tx.Signature)
	return baseSize
}

type EconomyManager struct {
	FeeMarket        *FeeMarket
	MiningIncentives *MiningIncentives
	EconomicSecurity *EconomicSecurity
	DynamicBlockSize *DynamicBlockSize
}

func NewEconomyManager(policy blockchain.MonetaryPolicy, hashRate, marketCap uint64) *EconomyManager {
	return &EconomyManager{
		FeeMarket:        NewFeeMarket(),
		MiningIncentives: NewMiningIncentives(policy),
		EconomicSecurity: NewEconomicSecurity(hashRate, marketCap),
		DynamicBlockSize: NewDynamicBlockSize(1000000),
	}
}

func (em *EconomyManager) ProcessBlock(block *blockchain.Block, parentGasUsed, parentGasLimit uint64) error {
	baseFee := em.FeeMarket.CalculateBaseFee(parentGasUsed, parentGasLimit, block.Height)
	em.FeeMarket.SetBaseFee(baseFee)

	entry := FeeHistoryEntry{
		BlockHeight:       block.Height,
		BaseFee:           baseFee,
		GasUsed:           em.CalculateGasUsed(block.Transactions),
		GasLimit:          parentGasLimit,
		PriorityFeePerGas: em.FeeMarket.EstimatePriorityFee(PriorityMedium),
		Timestamp:         time.Now(),
	}
	em.FeeMarket.AddBlockToHistory(entry)

	_, burnedFees := em.MiningIncentives.CalculateMinerReward(block.Height, block.Transactions)
	_ = burnedFees

	em.DynamicBlockSize.UpdateBlockSize(entry.GasUsed)

	em.EconomicSecurity.RecordMinerBlock(block.MinerAddress, time.Now())

	return nil
}

func (em *EconomyManager) CalculateGasUsed(txs []blockchain.Transaction) uint64 {
	var total uint64
	for _, tx := range txs {
		total += tx.Fee
	}
	return total
}

func (em *EconomyManager) GetFeeEstimate() FeeEstimate {
	return em.FeeMarket.EstimateFee()
}

func (em *EconomyManager) ValidateBlockEconomics(block *blockchain.Block, parentBlock *blockchain.Block) error {
	if err := em.EconomicSecurity.ValidateTransactionOrdering(block.Transactions); err != nil {
		return fmt.Errorf("transaction ordering validation failed: %w", err)
	}

	for _, tx := range block.Transactions {
		if err := ValidateTransaction(tx); err != nil {
			return fmt.Errorf("transaction validation failed: %w", err)
		}
	}

	currentSize := em.DynamicBlockSize.GetCurrentSize()
	blockSize := em.EstimateBlockSize(block)
	if blockSize > int(currentSize) {
		return fmt.Errorf("block size %d exceeds limit %d", blockSize, currentSize)
	}

	return nil
}

func (em *EconomyManager) EstimateBlockSize(block *blockchain.Block) int {
	var total int
	for _, tx := range block.Transactions {
		total += EstimateTransactionSize(tx)
	}
	return total
}

func (em *EconomyManager) GetSecurityReport() SecurityReport {
	netSec := em.EconomicSecurity.GetNetworkSecurity()
	selfishMiners := em.EconomicSecurity.GetSelfishMiners()

	return SecurityReport{
		NetworkSecurity:    netSec,
		SelfishMinersCount: len(selfishMiners),
		SelfishMiners:      selfishMiners,
		FeeEstimate:        em.FeeMarket.EstimateFee(),
		CurrentBlockSize:   em.DynamicBlockSize.GetCurrentSize(),
	}
}

type SecurityReport struct {
	NetworkSecurity    NetworkSecurity
	SelfishMinersCount int
	SelfishMiners      []string
	FeeEstimate        FeeEstimate
	CurrentBlockSize   uint64
}
