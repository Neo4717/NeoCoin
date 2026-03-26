package mempool

import (
	"testing"
)

func FuzzTXSelection(f *testing.F) {
	f.Fuzz(func(t *testing.T, feeRate, txSize int) {
		if feeRate < 0 || txSize < 0 {
			return
		}

		if feeRate > 1e10 || txSize > 1e6 {
			return
		}
	})
}
