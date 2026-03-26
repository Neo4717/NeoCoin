package blockchain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type consensusParamsJSON struct {
	DifficultyEnable               *bool   `json:"difficultyEnable"`
	DifficultyTargetMs             *int64  `json:"difficultyTargetMs"`
	DifficultyTargetSpacing        *int64  `json:"difficultyTargetSpacing"`
	DifficultyWindow               *int    `json:"difficultyWindow"`
	DifficultyWindowSize           *int    `json:"difficultyWindowSize"`
	DifficultyAdjustmentInterval   *int    `json:"difficultyAdjustmentInterval"`
	DifficultyMaxStepBits          *uint32 `json:"difficultyMaxStepBits"`
	DifficultyMaxStep              *uint32 `json:"difficultyMaxStep"`
	MinDifficultyBits              *uint32 `json:"difficultyMinBits"`
	MaxDifficultyBits              *uint32 `json:"difficultyMaxBits"`
	GenesisDifficultyBits          *uint32 `json:"genesisDifficultyBits"`
	MedianTimePastWindow           *int    `json:"medianTimePastWindow"`
	MaxTimeDrift                   *int64  `json:"maxTimeDrift"`
	MaxBlockSize                   *uint64 `json:"maxBlockSize"`
	MerkleEnable                   *bool   `json:"merkleEnable"`
	MerkleActivationHeight         *uint64 `json:"merkleActivationHeight"`
	BinaryEncodingEnable           *bool   `json:"binaryEncodingEnable"`
	BinaryEncodingActivationHeight *uint64 `json:"binaryEncodingActivationHeight"`
}

func parseConsensusParams(raw json.RawMessage) (ConsensusParams, error) {
	if len(raw) == 0 {
		return ConsensusParams{}, errors.New("genesis consensusParams is required")
	}
	var aux consensusParamsJSON
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&aux); err != nil {
		return ConsensusParams{}, fmt.Errorf("parse consensusParams: %w", err)
	}

	if aux.DifficultyEnable == nil {
		return ConsensusParams{}, errors.New("consensusParams.difficultyEnable is required")
	}
	if aux.MerkleEnable == nil {
		return ConsensusParams{}, errors.New("consensusParams.merkleEnable is required")
	}
	if aux.BinaryEncodingEnable == nil {
		return ConsensusParams{}, errors.New("consensusParams.binaryEncodingEnable is required")
	}

	targetMs, err := pickInt64("difficultyTarget", aux.DifficultyTargetMs, toMillis(aux.DifficultyTargetSpacing))
	if err != nil {
		return ConsensusParams{}, err
	}
	window, err := pickInt("difficultyWindow", aux.DifficultyWindow, aux.DifficultyWindowSize, aux.DifficultyAdjustmentInterval)
	if err != nil {
		return ConsensusParams{}, err
	}
	maxStep, err := pickUint32("difficultyMaxStepBits", aux.DifficultyMaxStepBits, aux.DifficultyMaxStep)
	if err != nil {
		return ConsensusParams{}, err
	}
	minBits, err := requireUint32("difficultyMinBits", aux.MinDifficultyBits)
	if err != nil {
		return ConsensusParams{}, err
	}
	maxBits, err := requireUint32("difficultyMaxBits", aux.MaxDifficultyBits)
	if err != nil {
		return ConsensusParams{}, err
	}
	genesisBits, err := requireUint32("genesisDifficultyBits", aux.GenesisDifficultyBits)
	if err != nil {
		return ConsensusParams{}, err
	}
	mtpWindow, err := requireInt("medianTimePastWindow", aux.MedianTimePastWindow)
	if err != nil {
		return ConsensusParams{}, err
	}
	maxTimeDrift, err := requireInt64("maxTimeDrift", aux.MaxTimeDrift)
	if err != nil {
		return ConsensusParams{}, err
	}
	maxBlockSize, err := requireUint64("maxBlockSize", aux.MaxBlockSize)
	if err != nil {
		return ConsensusParams{}, err
	}
	merkleHeight, err := requireUint64("merkleActivationHeight", aux.MerkleActivationHeight)
	if err != nil {
		return ConsensusParams{}, err
	}
	binaryHeight, err := requireUint64("binaryEncodingActivationHeight", aux.BinaryEncodingActivationHeight)
	if err != nil {
		return ConsensusParams{}, err
	}

	p := ConsensusParams{
		DifficultyEnable:               *aux.DifficultyEnable,
		TargetBlockTime:                time.Duration(targetMs) * time.Millisecond,
		DifficultyWindow:               window,
		DifficultyMaxStep:              maxStep,
		MinDifficultyBits:              minBits,
		MaxDifficultyBits:              maxBits,
		GenesisDifficultyBits:          genesisBits,
		MedianTimePastWindow:           mtpWindow,
		MaxTimeDrift:                   maxTimeDrift,
		MaxBlockSize:                   maxBlockSize,
		MerkleEnable:                   *aux.MerkleEnable,
		MerkleActivationHeight:         merkleHeight,
		BinaryEncodingEnable:           *aux.BinaryEncodingEnable,
		BinaryEncodingActivationHeight: binaryHeight,
	}
	if err := validateConsensusParams(p); err != nil {
		return ConsensusParams{}, err
	}
	return p, nil
}

func toMillis(v *int64) *int64 {
	if v == nil {
		return nil
	}
	ms := *v * 1000
	return &ms
}

func pickInt(name string, values ...*int) (int, error) {
	var out int
	var set bool
	for _, v := range values {
		if v == nil {
			continue
		}
		if !set {
			out = *v
			set = true
			continue
		}
		if *v != out {
			return 0, fmt.Errorf("consensusParams.%s mismatch: %d vs %d", name, out, *v)
		}
	}
	if !set {
		return 0, fmt.Errorf("consensusParams.%s is required", name)
	}
	return out, nil
}

func pickInt64(name string, values ...*int64) (int64, error) {
	var out int64
	var set bool
	for _, v := range values {
		if v == nil {
			continue
		}
		if !set {
			out = *v
			set = true
			continue
		}
		if *v != out {
			return 0, fmt.Errorf("consensusParams.%s mismatch: %d vs %d", name, out, *v)
		}
	}
	if !set {
		return 0, fmt.Errorf("consensusParams.%s is required", name)
	}
	return out, nil
}

func pickUint32(name string, values ...*uint32) (uint32, error) {
	var out uint32
	var set bool
	for _, v := range values {
		if v == nil {
			continue
		}
		if !set {
			out = *v
			set = true
			continue
		}
		if *v != out {
			return 0, fmt.Errorf("consensusParams.%s mismatch: %d vs %d", name, out, *v)
		}
	}
	if !set {
		return 0, fmt.Errorf("consensusParams.%s is required", name)
	}
	return out, nil
}

func requireInt(name string, v *int) (int, error) {
	if v == nil {
		return 0, fmt.Errorf("consensusParams.%s is required", name)
	}
	return *v, nil
}

func requireInt64(name string, v *int64) (int64, error) {
	if v == nil {
		return 0, fmt.Errorf("consensusParams.%s is required", name)
	}
	return *v, nil
}

func requireUint32(name string, v *uint32) (uint32, error) {
	if v == nil {
		return 0, fmt.Errorf("consensusParams.%s is required", name)
	}
	return *v, nil
}

func requireUint64(name string, v *uint64) (uint64, error) {
	if v == nil {
		return 0, fmt.Errorf("consensusParams.%s is required", name)
	}
	return *v, nil
}

func validateConsensusParams(p ConsensusParams) error {
	if p.TargetBlockTime <= 0 {
		return errors.New("consensusParams.difficultyTarget must be > 0")
	}
	if p.DifficultyWindow <= 0 {
		return errors.New("consensusParams.difficultyWindow must be > 0")
	}
	if p.DifficultyMaxStep == 0 {
		return errors.New("consensusParams.difficultyMaxStepBits must be > 0")
	}
	if p.MinDifficultyBits == 0 {
		return errors.New("consensusParams.difficultyMinBits must be > 0")
	}
	if p.MaxDifficultyBits == 0 {
		return errors.New("consensusParams.difficultyMaxBits must be > 0")
	}
	if p.MaxDifficultyBits > maxDifficultyBits {
		return fmt.Errorf("consensusParams.difficultyMaxBits must be <= %d", maxDifficultyBits)
	}
	if p.MinDifficultyBits > p.MaxDifficultyBits {
		return errors.New("consensusParams.difficultyMinBits must be <= difficultyMaxBits")
	}
	if p.GenesisDifficultyBits < p.MinDifficultyBits || p.GenesisDifficultyBits > p.MaxDifficultyBits {
		return errors.New("consensusParams.genesisDifficultyBits must be within min/max difficulty bits")
	}
	if p.MedianTimePastWindow <= 0 {
		return errors.New("consensusParams.medianTimePastWindow must be > 0")
	}
	if p.MaxTimeDrift <= 0 {
		return errors.New("consensusParams.maxTimeDrift must be > 0")
	}
	if p.MaxBlockSize == 0 {
		return errors.New("consensusParams.maxBlockSize must be > 0")
	}
	return nil
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvUint64(key string, defaultVal uint64) uint64 {
	if val := os.Getenv(key); val != "" {
		if uintVal, err := strconv.ParseUint(val, 10, 64); err == nil {
			return uintVal
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1"
	}
	return defaultVal
}

func getEnvInt64(key string, def int64) int64 {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			return intVal
		}
	}
	return def
}
