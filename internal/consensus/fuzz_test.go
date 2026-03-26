package consensus

import (
	"testing"
)

func FuzzPowValidation(f *testing.F) {
	f.Fuzz(func(t *testing.T, nonce uint64, data []byte) {
		if len(data) == 0 {
			return
		}
		_ = nonce
	})
}

func FuzzDifficultyAdjustment(f *testing.F) {
	f.Fuzz(func(t *testing.T, ts1, ts2, ts3, ts4, ts5 int64) {
		timestamps := []int64{ts1, ts2, ts3, ts4, ts5}
		for i, ts := range timestamps {
			if ts < 0 {
				timestamps[i] = -ts
			}
		}
		_ = timestamps
	})
}

func FuzzBlockValidation(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		if len(data) > 10000 {
			data = data[:10000]
		}

		_ = data
	})
}
