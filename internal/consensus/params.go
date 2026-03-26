package consensus

import (
	"time"
)

const (
	RewardInterval       = 210000
	InitialReward        = 50000000000
	MaxSupply            = 2100000000000000
	TargetSeconds        = 600
	MaxBlockWeight       = 4000000
	MaxBlockSize         = 1000000
	MedianTimePastWindow = 11
	MaxTimeDrift         = 7200
)

type Params struct {
	DifficultyEnable bool

	TargetBlockTime   time.Duration
	DifficultyWindow  int
	DifficultyMaxStep uint32

	MinDifficultyBits uint32
	MaxDifficultyBits uint32

	GenesisDifficultyBits uint32

	MedianTimePastWindow int
	MaxTimeDrift         int64
	MaxBlockSize         uint64

	MerkleEnable           bool
	MerkleActivationHeight uint64

	BinaryEncodingEnable           bool
	BinaryEncodingActivationHeight uint64

	MonetaryPolicy MonetaryPolicy
}

type MonetaryPolicy struct {
	InitialBlockReward uint64
	HalvingInterval    uint64
	MinerFeeShare      uint8
	TailEmission       uint64
}

func DefaultParams() Params {
	p := Params{
		DifficultyEnable:               false,
		TargetBlockTime:                10 * time.Second,
		DifficultyWindow:               20,
		DifficultyMaxStep:              1,
		MinDifficultyBits:              1,
		MaxDifficultyBits:              255,
		GenesisDifficultyBits:          defaultDifficultyBits,
		MedianTimePastWindow:           MedianTimePastWindow,
		MaxTimeDrift:                   MaxTimeDrift,
		MaxBlockSize:                   MaxBlockSize,
		MerkleEnable:                   false,
		MerkleActivationHeight:         0,
		BinaryEncodingEnable:           false,
		BinaryEncodingActivationHeight: 0,
		MonetaryPolicy: MonetaryPolicy{
			InitialBlockReward: InitialReward,
			HalvingInterval:    RewardInterval,
			MinerFeeShare:      100,
			TailEmission:       0,
		},
	}

	if p.TargetBlockTime <= 0 {
		p.TargetBlockTime = 10 * time.Second
	}
	if p.DifficultyWindow <= 0 {
		p.DifficultyWindow = 20
	}
	if p.DifficultyMaxStep == 0 {
		p.DifficultyMaxStep = 1
	}
	if p.MinDifficultyBits == 0 {
		p.MinDifficultyBits = 1
	}
	if p.MaxDifficultyBits == 0 {
		p.MaxDifficultyBits = 255
	}
	if p.MaxDifficultyBits > maxDifficultyBits {
		p.MaxDifficultyBits = maxDifficultyBits
	}
	if p.MinDifficultyBits > p.MaxDifficultyBits {
		p.MinDifficultyBits = p.MaxDifficultyBits
	}
	if p.GenesisDifficultyBits < p.MinDifficultyBits {
		p.GenesisDifficultyBits = p.MinDifficultyBits
	}
	if p.GenesisDifficultyBits > p.MaxDifficultyBits {
		p.GenesisDifficultyBits = p.MaxDifficultyBits
	}
	if p.MedianTimePastWindow <= 0 {
		p.MedianTimePastWindow = 11
	}
	if p.MaxTimeDrift <= 0 {
		p.MaxTimeDrift = 7200
	}
	if p.MaxBlockSize == 0 {
		p.MaxBlockSize = MaxBlockSize
	}
	return p
}

func (p Params) BinaryEncodingActive(height uint64) bool {
	return p.BinaryEncodingEnable && height >= p.BinaryEncodingActivationHeight
}

func (p Params) BlockReward(height uint64) uint64 {
	return uint64(CalcBlockSubsidy(int64(height)))
}
