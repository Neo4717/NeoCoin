package ai

import (
	"math"
	"sync"
	"time"
)

type DifficultyPredictor struct {
	mu              sync.RWMutex
	blockHistory    []BlockData
	maxHistory      int
	targetBlockTime time.Duration
	windowSize      int
}

type BlockData struct {
	Height         uint64
	Timestamp      time.Time
	DifficultyBits uint32
	BlockTime      time.Duration
	Hashrate       float64
}

type PredictionResult struct {
	PredictedDifficultyBits uint32
	Confidence              float64
	RecommendedAdjustment   float64
	Trend                   string
	Reason                  string
	HashrateEstimate        float64
	NextDifficultyEstimate  float32
}

func NewDifficultyPredictor(windowSize int, targetBlockTime time.Duration) *DifficultyPredictor {
	return &DifficultyPredictor{
		blockHistory:    make([]BlockData, 0, windowSize*2),
		maxHistory:      windowSize * 10,
		windowSize:      windowSize,
		targetBlockTime: targetBlockTime,
	}
}

func (dp *DifficultyPredictor) RecordBlock(height uint64, timestamp time.Time, diffBits uint32, blockTime time.Duration) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	blockData := BlockData{
		Height:         height,
		Timestamp:      timestamp,
		DifficultyBits: diffBits,
		BlockTime:      blockTime,
		Hashrate:       calculateHashrate(diffBits, blockTime),
	}

	dp.blockHistory = append(dp.blockHistory, blockData)

	if len(dp.blockHistory) > dp.maxHistory {
		dp.blockHistory = dp.blockHistory[len(dp.blockHistory)-dp.maxHistory:]
	}
}

func (dp *DifficultyPredictor) Predict() *PredictionResult {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	if len(dp.blockHistory) < dp.windowSize {
		return &PredictionResult{
			Confidence: 0.0,
			Reason:     "insufficient_data",
			Trend:      "unknown",
		}
	}

	recentBlocks := dp.blockHistory[len(dp.blockHistory)-dp.windowSize:]

	avgBlockTime := calculateAvgBlockTime(recentBlocks)
	avgHashrate := calculateAvgHashrate(recentBlocks)
	hashrateVolatility := calculateVolatility(recentBlocks)

	targetTime := float64(dp.targetBlockTime.Seconds())
	actualTime := avgBlockTime.Seconds()

	adjustmentRatio := targetTime / actualTime

	if adjustmentRatio > 1.5 {
		adjustmentRatio = 1.5
	} else if adjustmentRatio < 0.67 {
		adjustmentRatio = 0.67
	}

	currentDiffBits := recentBlocks[len(recentBlocks)-1].DifficultyBits
	currentDiff := uint64(math.Pow(2, float64(32-currentDiffBits)))
	predictedDiff := uint64(float64(currentDiff) * adjustmentRatio)

	predictedDiffBits := uint32(32 - math.Log2(float64(predictedDiff)))
	if predictedDiffBits < 1 {
		predictedDiffBits = 1
	}
	if predictedDiffBits > 255 {
		predictedDiffBits = 255
	}

	confidence := calculateConfidence(hashrateVolatility, len(recentBlocks))

	trend := "stable"
	if adjustmentRatio > 1.1 {
		trend = "increasing"
	} else if adjustmentRatio < 0.9 {
		trend = "decreasing"
	}

	reason := "based on hashrate analysis"
	if hashrateVolatility > 0.5 {
		reason = "high hashrate volatility detected"
		confidence *= 0.7
	}

	return &PredictionResult{
		PredictedDifficultyBits: predictedDiffBits,
		Confidence:              confidence,
		RecommendedAdjustment:   adjustmentRatio,
		Trend:                   trend,
		Reason:                  reason,
		HashrateEstimate:        avgHashrate,
		NextDifficultyEstimate:  float32(predictedDiffBits),
	}
}

func (dp *DifficultyPredictor) GetStatistics() map[string]interface{} {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	if len(dp.blockHistory) < 2 {
		return map[string]interface{}{
			"status": "insufficient_data",
		}
	}

	recentBlocks := dp.blockHistory[len(dp.blockHistory)-min(100, len(dp.blockHistory)):]

	avgBlockTime := calculateAvgBlockTime(recentBlocks)
	avgHashrate := calculateAvgHashrate(recentBlocks)
	volatility := calculateVolatility(recentBlocks)

	return map[string]interface{}{
		"total_blocks":      len(dp.blockHistory),
		"avg_block_time":    avgBlockTime.Seconds(),
		"avg_hashrate":      avgHashrate,
		"volatility":        volatility,
		"target_block_time": dp.targetBlockTime.Seconds(),
		"window_size":       dp.windowSize,
	}
}

func (dp *DifficultyPredictor) AnalyzeHashrateTrend() string {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	if len(dp.blockHistory) < 20 {
		return "insufficient_data"
	}

	recent := dp.blockHistory[len(dp.blockHistory)-10:]
	older := dp.blockHistory[len(dp.blockHistory)-20 : len(dp.blockHistory)-10]

	recentAvg := calculateAvgHashrate(recent)
	olderAvg := calculateAvgHashrate(older)

	changeRatio := recentAvg / olderAvg

	if changeRatio > 1.2 {
		return "rapid_increase"
	} else if changeRatio > 1.05 {
		return "gradual_increase"
	} else if changeRatio < 0.8 {
		return "rapid_decrease"
	} else if changeRatio < 0.95 {
		return "gradual_decrease"
	}
	return "stable"
}

func calculateAvgBlockTime(blocks []BlockData) time.Duration {
	if len(blocks) < 2 {
		return 0
	}

	var total time.Duration
	for i := 1; i < len(blocks); i++ {
		total += blocks[i].BlockTime
	}
	return total / time.Duration(len(blocks)-1)
}

func calculateAvgHashrate(blocks []BlockData) float64 {
	if len(blocks) == 0 {
		return 0
	}

	var total float64
	for _, b := range blocks {
		total += b.Hashrate
	}
	return total / float64(len(blocks))
}

func calculateHashrate(diffBits uint32, blockTime time.Duration) float64 {
	if blockTime == 0 {
		return 0
	}
	target := math.Pow(2, float64(256-diffBits))
	return target / blockTime.Seconds()
}

func calculateVolatility(blocks []BlockData) float64 {
	if len(blocks) < 2 {
		return 0
	}

	hashrates := make([]float64, len(blocks))
	for i, b := range blocks {
		hashrates[i] = b.Hashrate
	}

	mean := 0.0
	for _, h := range hashrates {
		mean += h
	}
	mean /= float64(len(hashrates))

	variance := 0.0
	for _, h := range hashrates {
		diff := h - mean
		variance += diff * diff
	}
	variance /= float64(len(hashrates))

	stdDev := math.Sqrt(variance)
	if mean == 0 {
		return 0
	}

	return stdDev / mean
}

func calculateConfidence(volatility float64, sampleCount int) float64 {
	confidence := 1.0

	if volatility > 0.5 {
		confidence *= 0.5
	} else if volatility > 0.3 {
		confidence *= 0.7
	} else if volatility > 0.1 {
		confidence *= 0.9
	}

	if sampleCount < 20 {
		confidence *= float64(sampleCount) / 20.0
	}

	return confidence
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
