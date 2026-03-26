package ai

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

type SpamFilter struct {
	mu             sync.RWMutex
	senderStats    map[string]*SenderMetrics
	globalStats    *GlobalMempoolStats
	minScore       float64
	windowDuration time.Duration
}

type SenderMetrics struct {
	mu               sync.RWMutex
	Address          string
	FirstSeen        time.Time
	TransactionCount int64
	TotalAmount      int64
	FailedCount      int64
	RecentTxs        []time.Time
	AvgFeeRate       float64
}

type GlobalMempoolStats struct {
	mu            sync.RWMutex
	TotalTxs      int64
	TotalAmount   int64
	AvgFeeRate    float64
	SendersCount  int64
	HighRiskCount int64
	LastUpdate    time.Time
}

type SpamResult struct {
	IsSpam       bool
	SpamScore    float64
	Reasons      []string
	Severity     string
	ShouldReject bool
}

var (
	spamPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)airdrop|claim.*free|free.*token|Claim|get.*now`),
		regexp.MustCompile(`(?i)100[xx]|1000[xx]|10000[xx]|moon|lambo|pump|dump`),
		regexp.MustCompile(`(?i)urgent|immediately|act.*now|limited.*time|don't.*miss`),
		regexp.MustCompile(`(?i)verify.*wallet|connect.*wallet|sign.*message.*danger`),
		regexp.MustCompile(`(?i)gas.*free|no.*fee|zero.*fee|0.*fee`),
		regexp.MustCompile(`(?i)double.*your|mul.*2x|guaranteed.*return|no.*risk`),
		regexp.MustCompile(`(?i)discord.*admin|telegram.*admin|support.*chat`),
		regexp.MustCompile(`(?i)private.*key|seed.*phrase|recovery.*phrase`),
		regexp.MustCompile(`(?i)nft.*mint.*free|free.*nft|mint.*now.*free`),
		regexp.MustCompile(`(?i)update.*wallet|security.*alert|unlock.*account`),
	}

	suspiciousDataPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)http.*bitcoin|eth.*giveaway|send.*receive.*double`),
		regexp.MustCompile(`(?i).*\.xyz|.*\.top|.*\.work|.*\.click`),
		regexp.MustCompile(`(?i)referral|invite.*code|link.*ref`),
	}

	legitimatePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)contract.*call|execute|swap|transfer|approve`),
		regexp.MustCompile(`(?i)stake|unstake|claim.*reward|withdraw`),
		regexp.MustCompile(`(?i)mint.*nft|buy.*nft|list.*nft|transfer.*nft`),
	}
)

func NewSpamFilter(windowDuration time.Duration) *SpamFilter {
	return &SpamFilter{
		senderStats:    make(map[string]*SenderMetrics),
		globalStats:    &GlobalMempoolStats{},
		minScore:       50.0,
		windowDuration: windowDuration,
	}
}

func (sf *SpamFilter) AnalyzeTransaction(sender string, amount int64, fee int64, data string, nonce uint64) *SpamResult {
	result := &SpamResult{
		IsSpam:  false,
		Reasons: []string{},
	}

	dataLower := strings.ToLower(data)

	spamMatches := 0
	for _, pattern := range spamPatterns {
		if pattern.MatchString(dataLower) {
			spamMatches++
			result.Reasons = append(result.Reasons, "spam_keyword_detected")
		}
	}

	suspiciousMatches := 0
	for _, pattern := range suspiciousDataPatterns {
		if pattern.MatchString(dataLower) {
			suspiciousMatches++
			result.Reasons = append(result.Reasons, "suspicious_link_detected")
		}
	}

	legitimateMatches := 0
	for _, pattern := range legitimatePatterns {
		if pattern.MatchString(dataLower) {
			legitimateMatches++
			result.Reasons = append(result.Reasons, "legitimate_contract_call")
		}
	}

	score := float64(spamMatches*30 + suspiciousMatches*15)

	sf.mu.RLock()
	senderMetrics, exists := sf.senderStats[sender]
	sf.mu.RUnlock()

	if exists {
		senderMetrics.mu.RLock()
		txCount := senderMetrics.TransactionCount
		failedRatio := float64(senderMetrics.FailedCount) / float64(max(1, int(txCount)))
		senderMetrics.mu.RUnlock()

		if txCount > 100 {
			score += 20
			result.Reasons = append(result.Reasons, "high_volume_sender")
		}
		if failedRatio > 0.5 {
			score += 30
			result.Reasons = append(result.Reasons, "high_failure_rate")
		}
	}

	if nonce == 0 {
		score -= 15
		result.Reasons = append(result.Reasons, "first_transaction")
	}

	if amount > 0 && fee > 0 {
		feeRate := float64(fee) / float64(amount)
		if feeRate > 0.1 {
			score -= 20
			result.Reasons = append(result.Reasons, "high_fee_rate")
		}
		if feeRate < 0.0001 && amount > 1000 {
			score += 25
			result.Reasons = append(result.Reasons, "suspiciously_low_fee")
		}
	}

	if legitimateMatches > spamMatches {
		score -= float64(legitimateMatches * 10)
	}

	result.SpamScore = clampFloat(0, 100, score)

	if result.SpamScore >= 70 {
		result.Severity = "high"
		result.ShouldReject = true
		result.IsSpam = true
	} else if result.SpamScore >= 40 {
		result.Severity = "medium"
		result.ShouldReject = false
	} else {
		result.Severity = "low"
		result.ShouldReject = false
	}

	if len(result.Reasons) == 0 {
		result.Reasons = append(result.Reasons, "no_issues_detected")
	}

	return result
}

func (sf *SpamFilter) RecordTransaction(sender string, amount int64, fee int64, accepted bool) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if _, exists := sf.senderStats[sender]; !exists {
		sf.senderStats[sender] = &SenderMetrics{
			Address:   sender,
			FirstSeen: time.Now(),
		}
	}

	senderMetrics := sf.senderStats[sender]
	senderMetrics.mu.Lock()
	senderMetrics.TransactionCount++
	senderMetrics.TotalAmount += amount
	senderMetrics.RecentTxs = append(senderMetrics.RecentTxs, time.Now())

	if fee > 0 && amount > 0 {
		feeRate := float64(fee) / float64(amount)
		senderMetrics.AvgFeeRate = (senderMetrics.AvgFeeRate*float64(senderMetrics.TransactionCount-1) + feeRate) / float64(senderMetrics.TransactionCount)
	}

	if !accepted {
		senderMetrics.FailedCount++
	}

	windowStart := time.Now().Add(-sf.windowDuration)
	validTxs := 0
	for i := len(senderMetrics.RecentTxs) - 1; i >= 0; i-- {
		if senderMetrics.RecentTxs[i].Before(windowStart) {
			senderMetrics.RecentTxs = senderMetrics.RecentTxs[i+1:]
			break
		}
		validTxs++
	}

	if validTxs > 50 {
		senderMetrics.TransactionCount = int64(validTxs)
	}
	senderMetrics.mu.Unlock()

	sf.globalStats.mu.Lock()
	sf.globalStats.TotalTxs++
	sf.globalStats.TotalAmount += amount
	sf.globalStats.LastUpdate = time.Now()
	sf.globalStats.mu.Unlock()
}

func (sf *SpamFilter) GetSenderMetrics(sender string) *SenderMetrics {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return sf.senderStats[sender]
}

func (sf *SpamFilter) GetGlobalStats() GlobalMempoolStats {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return GlobalMempoolStats{
		TotalTxs:     sf.globalStats.TotalTxs,
		TotalAmount:  sf.globalStats.TotalAmount,
		SendersCount: int64(len(sf.senderStats)),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampFloat(minVal, maxVal float64, val float64) float64 {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
