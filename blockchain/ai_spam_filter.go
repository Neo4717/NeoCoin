package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	DefaultAIURL        = "http://localhost:8001"
	DefaultTimeout      = 10 * time.Second
	DefaultMaxSpamScore = 80
)

var (
	ErrAIUnavailable = fmt.Errorf("AI auditor unavailable")
	ErrSpamDetected  = fmt.Errorf("transaction rejected by AI spam filter")
)

type AISpamConfig struct {
	URL          string
	Enabled      bool
	MaxSpamScore int
	Timeout      time.Duration
	FailOpen     bool
}

type AISpamRequest struct {
	FromAddress string `json:"fromAddress"`
	ToAddress   string `json:"toAddress"`
	Amount      uint64 `json:"amount"`
	Fee         uint64 `json:"fee"`
	Nonce       uint64 `json:"nonce"`
	ChainID     uint64 `json:"chainId"`
	Data        string `json:"data,omitempty"`
}

type AISpamResponse struct {
	SpamScore      int      `json:"spamScore"`
	RiskFactors    []string `json:"riskFactors"`
	Recommendation string   `json:"recommendation"`
}

type AISpamFilter struct {
	client *http.Client
	config AISpamConfig
	mu     sync.RWMutex
	stats  AISpamStats
}

type AISpamStats struct {
	mu           sync.Mutex
	TotalChecked int64
	AIUnavail    int64
	LastCheck    time.Time
}

func NewAISpamFilter(url string, enabled bool) *AISpamFilter {
	if url == "" {
		url = DefaultAIURL
	}

	return &AISpamFilter{
		client: &http.Client{Timeout: DefaultTimeout},
		config: AISpamConfig{
			URL:          url,
			Enabled:      enabled,
			MaxSpamScore: DefaultMaxSpamScore,
			FailOpen:     true,
		},
	}
}

func (a *AISpamFilter) Configure(enabled bool, maxScore int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.config.Enabled = enabled
	a.config.MaxSpamScore = maxScore
}

func (a *AISpamFilter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Enabled
}

func (a *AISpamFilter) URL() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.URL
}

func (a *AISpamFilter) AnalyzeTransaction(ctx context.Context, fromAddr, toAddr string, amount, fee, nonce, chainID uint64, data string) (int, []string, error) {
	a.mu.RLock()
	enabled := a.config.Enabled
	url := a.config.URL
	maxScore := a.config.MaxSpamScore
	failOpen := a.config.FailOpen
	a.mu.RUnlock()

	if !enabled {
		return 0, nil, nil
	}

	req := AISpamRequest{
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Amount:      amount,
		Fee:         fee,
		Nonce:       nonce,
		ChainID:     chainID,
		Data:        data,
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url+"/analyze", bytes.NewReader(body))
	if err != nil {
		a.recordStats(false)
		if failOpen {
			return 0, nil, nil
		}
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		a.recordStats(false)
		if failOpen {
			return 0, nil, nil
		}
		return 0, nil, ErrAIUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.recordStats(false)
		if failOpen {
			return 0, nil, nil
		}
		return 0, nil, fmt.Errorf("AI status: %d", resp.StatusCode)
	}

	var result AISpamResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, nil, fmt.Errorf("parse response: %w", err)
	}

	a.recordStats(true)

	if result.SpamScore > maxScore {
		return result.SpamScore, result.RiskFactors, ErrSpamDetected
	}

	return result.SpamScore, result.RiskFactors, nil
}

func (a *AISpamFilter) recordStats(connected bool) {
	a.stats.mu.Lock()
	defer a.stats.mu.Unlock()
	a.stats.TotalChecked++
	if !connected {
		a.stats.AIUnavail++
	} else {
		a.stats.LastCheck = time.Now()
	}
}

func (a *AISpamFilter) GetStats() (total, unavailable int64, last time.Time) {
	a.stats.mu.Lock()
	defer a.stats.mu.Unlock()
	return a.stats.TotalChecked, a.stats.AIUnavail, a.stats.LastCheck
}

func (a *AISpamFilter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	total, unavail, last := a.GetStats()
	json.NewEncoder(w).Encode(map[string]any{
		"enabled":       a.IsEnabled(),
		"aiUrl":         a.URL(),
		"totalChecked":  total,
		"aiUnavailable": unavail,
		"lastCheck":     last.Format(time.RFC3339),
	})
}
