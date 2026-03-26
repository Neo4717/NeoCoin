package ai

import (
	"math"
	"regexp"
	"sync"
	"time"
)

type WalletSecurityAnalyzer struct {
	mu               sync.RWMutex
	addressHistory   map[string]*WalletHistory
	globalStats      *WalletGlobalStats
	knownCompromised map[string]time.Time
}

type WalletHistory struct {
	mu               sync.RWMutex
	Address          string
	FirstSeen        time.Time
	LastActivity     time.Time
	TransactionCount int64
	TotalSent        int64
	TotalReceived    int64
	FailedTxCount    int64
	VolumeChanges    []VolumeChange
	PatternFlags     []string
	RiskScore        float64
	IsCompromised    bool
}

type VolumeChange struct {
	Timestamp time.Time
	Amount    int64
	Direction string
}

type WalletGlobalStats struct {
	mu               sync.RWMutex
	TotalWallets     int64
	CompromisedCount int64
	WarningCount     int64
}

type WalletSecurityResult struct {
	Address         string
	IsSafe          bool
	RiskLevel       string
	RiskScore       float64
	Issues          []WalletIssue
	Recommendations []string
	IsCompromised   bool
	Confidence      float64
}

type WalletIssue struct {
	Type        string
	Severity    string
	Description string
}

var (
	suspiciousPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)verify.*wallet|connect.*wallet|sign.*transaction`),
		regexp.MustCompile(`(?i)private.*key.*leak|private.*key.*compromised`),
		regexp.MustCompile(`(?i)seed.*phrase.*leak|mnemonic.*compromised`),
		regexp.MustCompile(`(?i)wallet.*drain|balance.*drain|transfer.*all`),
		regexp.MustCompile(`(?i)unusual.*withdrawal|large.*withdraw.*sudden`),
	}

	legitimatePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)regular.*payment|salary|income|transfer.*friend`),
		regexp.MustCompile(`(?i)exchange.*deposit|exchange.*withdraw`),
		regexp.MustCompile(`(?i)staking.*reward|mining.*reward`),
	}
)

func NewWalletSecurityAnalyzer() *WalletSecurityAnalyzer {
	return &WalletSecurityAnalyzer{
		addressHistory:   make(map[string]*WalletHistory),
		globalStats:      &WalletGlobalStats{},
		knownCompromised: make(map[string]time.Time),
	}
}

func (wsa *WalletSecurityAnalyzer) AnalyzeTransaction(address string, amount int64, direction string, txData string) *WalletSecurityResult {
	result := &WalletSecurityResult{
		Address: address,
		IsSafe:  true,
		Issues:  []WalletIssue{},
	}

	wsa.mu.RLock()
	history, exists := wsa.addressHistory[address]
	wsa.mu.RUnlock()

	if !exists {
		history = &WalletHistory{
			Address:      address,
			FirstSeen:    time.Now(),
			LastActivity: time.Now(),
		}
		wsa.mu.Lock()
		wsa.addressHistory[address] = history
		wsa.mu.Unlock()
	}

	history.mu.Lock()
	history.LastActivity = time.Now()
	history.TransactionCount++

	if direction == "sent" {
		history.TotalSent += amount
	} else {
		history.TotalReceived += amount
	}

	history.VolumeChanges = append(history.VolumeChanges, VolumeChange{
		Timestamp: time.Now(),
		Amount:    amount,
		Direction: direction,
	})
	if len(history.VolumeChanges) > 100 {
		history.VolumeChanges = history.VolumeChanges[len(history.VolumeChanges)-100:]
	}
	history.mu.Unlock()

	dataLower := ""
	if txData != "" {
		dataLower = txData
	}

	for _, pattern := range suspiciousPatterns {
		if pattern.MatchString(dataLower) {
			result.Issues = append(result.Issues, WalletIssue{
				Type:        "suspicious_activity",
				Severity:    "high",
				Description: "Suspicious transaction pattern detected",
			})
			result.RiskScore += 30
			history.mu.Lock()
			history.PatternFlags = append(history.PatternFlags, "suspicious_activity")
			history.mu.Unlock()
		}
	}

	if history.TransactionCount > 0 {
		sentRatio := float64(history.TotalSent) / float64(history.TotalSent+history.TotalReceived)
		if sentRatio > 0.95 && history.TotalSent > 1000000 {
			result.Issues = append(result.Issues, WalletIssue{
				Type:        "drain_pattern",
				Severity:    "high",
				Description: "Wallet is draining almost all received funds",
			})
			result.RiskScore += 40
		}
	}

	if history.TransactionCount > 10 {
		history.mu.RLock()
		recentTxs := history.VolumeChanges[len(history.VolumeChanges)-10:]
		history.mu.RUnlock()

		var recentTotal int64
		var directionChanges int
		var lastDirection string

		for _, vc := range recentTxs {
			recentTotal += vc.Amount
			if lastDirection != "" && vc.Direction != lastDirection {
				directionChanges++
			}
			lastDirection = vc.Direction
		}

		if directionChanges > 7 {
			result.Issues = append(result.Issues, WalletIssue{
				Type:        "rapid_direction_change",
				Severity:    "medium",
				Description: "Rapid changes in transaction direction",
			})
			result.RiskScore += 20
		}
	}

	if history.TotalReceived > 10000000 && history.FirstSeen.After(time.Now().Add(-24*time.Hour)) {
		result.Issues = append(result.Issues, WalletIssue{
			Type:        "new_whale",
			Severity:    "medium",
			Description: "New wallet with very large volume",
		})
		result.RiskScore += 15
	}

	if wsa.isKnownCompromised(address) {
		result.Issues = append(result.Issues, WalletIssue{
			Type:        "known_compromised",
			Severity:    "critical",
			Description: "Wallet address is in compromised list",
		})
		result.RiskScore += 50
		result.IsCompromised = true
	}

	if result.RiskScore >= 70 {
		result.RiskLevel = "critical"
		result.IsSafe = false
	} else if result.RiskScore >= 40 {
		result.RiskLevel = "high"
		result.IsSafe = false
	} else if result.RiskScore >= 20 {
		result.RiskLevel = "medium"
	} else {
		result.RiskLevel = "low"
	}

	history.mu.Lock()
	history.RiskScore = result.RiskScore
	history.mu.Unlock()

	result.Recommendations = wsa.generateRecommendations(result.Issues)
	result.Confidence = calculateWalletConfidence(history)

	return result
}

func (wsa *WalletSecurityAnalyzer) MarkCompromised(address string) {
	wsa.mu.Lock()
	defer wsa.mu.Unlock()

	wsa.knownCompromised[address] = time.Now()

	if history, exists := wsa.addressHistory[address]; exists {
		history.mu.Lock()
		history.IsCompromised = true
		history.mu.Unlock()
	}

	wsa.globalStats.mu.Lock()
	wsa.globalStats.CompromisedCount++
	wsa.globalStats.mu.Unlock()
}

func (wsa *WalletSecurityAnalyzer) GetWalletHistory(address string) *WalletHistory {
	wsa.mu.RLock()
	defer wsa.mu.RUnlock()
	return wsa.addressHistory[address]
}

func (wsa *WalletSecurityAnalyzer) GetGlobalStats() map[string]interface{} {
	wsa.mu.RLock()
	defer wsa.mu.RUnlock()

	wsa.globalStats.mu.RLock()
	defer wsa.globalStats.mu.RUnlock()

	return map[string]interface{}{
		"total_wallets":     len(wsa.addressHistory),
		"compromised_count": wsa.globalStats.CompromisedCount,
	}
}

func (wsa *WalletSecurityAnalyzer) generateRecommendations(issues []WalletIssue) []string {
	recs := []string{}

	for _, issue := range issues {
		switch issue.Type {
		case "suspicious_activity":
			recs = append(recs, "Verify all transactions before signing")
			recs = append(recs, "Never share private keys or seed phrases")
		case "drain_pattern":
			recs = append(recs, "Consider moving funds to a new wallet")
			recs = append(recs, "Enable additional wallet security")
		case "new_whale":
			recs = append(recs, "Split large holdings across multiple wallets")
			recs = append(recs, "Use hardware wallet for large amounts")
		case "known_compromised":
			recs = append(recs, "IMMEDIATELY move funds to new wallet")
			recs = append(recs, "Do not use this address again")
		}
	}

	if len(recs) == 0 {
		recs = append(recs, "Continue following security best practices")
		recs = append(recs, "Monitor wallet activity regularly")
	}

	return recs
}

func (wsa *WalletSecurityAnalyzer) isKnownCompromised(address string) bool {
	wsa.mu.RLock()
	_, compromised := wsa.knownCompromised[address]
	wsa.mu.RUnlock()
	return compromised
}

func calculateWalletConfidence(history *WalletHistory) float64 {
	confidence := 50.0

	if history == nil {
		return 10.0
	}

	history.mu.RLock()
	txCount := history.TransactionCount
	firstSeen := history.FirstSeen
	history.mu.RUnlock()

	ageDays := time.Since(firstSeen).Hours() / 24
	if ageDays > 365 {
		confidence += 30
	} else if ageDays > 180 {
		confidence += 20
	} else if ageDays > 30 {
		confidence += 10
	}

	if txCount > 100 {
		confidence += 15
	} else if txCount > 10 {
		confidence += 5
	}

	return math.Min(100, confidence)
}

func (wsa *WalletSecurityAnalyzer) CleanupOldData(maxAge time.Duration) {
	wsa.mu.Lock()
	defer wsa.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for addr, history := range wsa.addressHistory {
		history.mu.RLock()
		if history.LastActivity.Before(cutoff) {
			delete(wsa.addressHistory, addr)
		}
		history.mu.RUnlock()
	}
}
