package consensus

import "time"

const maxDifficultyBits = uint32(256)

func NextDifficultyBits(p Params, blocks []BlockInfo) uint32 {
	if len(blocks) == 0 {
		return p.GenesisDifficultyBits
	}
	parentIdx := len(blocks) - 1
	parent := blocks[parentIdx]

	if !p.DifficultyEnable {
		if parent.DifficultyBits == 0 {
			return p.GenesisDifficultyBits
		}
		return clampDifficultyBits(p, parent.DifficultyBits)
	}

	if parentIdx < p.DifficultyWindow {
		return clampDifficultyBits(p, parent.DifficultyBits)
	}

	older := blocks[parentIdx-p.DifficultyWindow]
	actualSpanSec := parent.TimestampUnix - older.TimestampUnix
	if actualSpanSec <= 0 {
		actualSpanSec = 1
	}

	targetSec := int64(p.TargetBlockTime / time.Second)
	if targetSec <= 0 {
		targetSec = 1
	}
	expectedSpanSec := int64(p.DifficultyWindow) * targetSec
	if expectedSpanSec <= 0 {
		expectedSpanSec = 1
	}
	if actualSpanSec < expectedSpanSec/4 {
		actualSpanSec = expectedSpanSec / 4
	}
	if actualSpanSec > expectedSpanSec*4 {
		actualSpanSec = expectedSpanSec * 4
	}

	next := int64(parent.DifficultyBits)
	if actualSpanSec < expectedSpanSec/2 {
		next += int64(p.DifficultyMaxStep)
	} else if actualSpanSec > expectedSpanSec*2 {
		next -= int64(p.DifficultyMaxStep)
	}

	if next < 1 {
		next = 1
	}
	return clampDifficultyBits(p, uint32(next))
}

func ExpectedDifficultyBitsForBlockIndex(p Params, blocks []BlockInfo, idx int) uint32 {
	if idx <= 0 || idx >= len(blocks) {
		return 0
	}
	parentPath := blocks[:idx]
	return NextDifficultyBits(p, parentPath)
}

func clampDifficultyBits(p Params, bits uint32) uint32 {
	if bits < 1 {
		bits = 1
	}
	if bits > maxDifficultyBits {
		bits = maxDifficultyBits
	}
	if bits < p.MinDifficultyBits {
		return p.MinDifficultyBits
	}
	if bits > p.MaxDifficultyBits {
		return p.MaxDifficultyBits
	}
	return bits
}

type BlockInfo struct {
	Height         uint64
	DifficultyBits uint32
	TimestampUnix  int64
}
