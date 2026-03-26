package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

type MonetaryPolicy struct {
	InitialBlockReward uint64
	HalvingInterval    uint64
	MinerFeeShare      uint8
	TailEmission       uint64
}

func (p MonetaryPolicy) BlockReward(height uint64) uint64 {
	if p.HalvingInterval == 0 {
		return p.InitialBlockReward
	}
	halvings := height / p.HalvingInterval
	if halvings >= 64 {
		return p.TailEmission
	}
	reward := p.InitialBlockReward >> halvings
	if reward == 0 && p.TailEmission == 0 {
		return 1
	}
	if reward == 0 {
		return p.TailEmission
	}
	return reward
}

func (p MonetaryPolicy) MinerFeeAmount(totalFees uint64) uint64 {
	if p.MinerFeeShare == 0 || totalFees == 0 {
		return 0
	}
	return totalFees * uint64(p.MinerFeeShare) / 100
}

func (p MonetaryPolicy) Validate() error {
	if p.InitialBlockReward == 0 && p.TailEmission == 0 {
		return errors.New("monetaryPolicy.initialBlockReward or tailEmission must be > 0")
	}
	if p.HalvingInterval == 0 {
		return errors.New("monetaryPolicy.halvingInterval must be > 0")
	}
	if p.MinerFeeShare > 100 {
		return errors.New("monetaryPolicy.minerFeeShare must be <= 100")
	}
	return nil
}

type monetaryPolicyJSON struct {
	InitialBlockReward *Uint64String `json:"initialBlockReward"`
	HalvingInterval    *Uint64String `json:"halvingInterval"`
	MinerFeeShare      *uint8        `json:"minerFeeShare"`
	TailEmission       *Uint64String `json:"tailEmission,omitempty"`
}

func parseMonetaryPolicy(raw json.RawMessage) (MonetaryPolicy, error) {
	if len(raw) == 0 {
		return MonetaryPolicy{}, errors.New("genesis monetaryPolicy is required")
	}
	var aux monetaryPolicyJSON
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&aux); err != nil {
		return MonetaryPolicy{}, fmt.Errorf("parse monetaryPolicy: %w", err)
	}
	if aux.InitialBlockReward == nil {
		return MonetaryPolicy{}, errors.New("monetaryPolicy.initialBlockReward is required")
	}
	if aux.HalvingInterval == nil {
		return MonetaryPolicy{}, errors.New("monetaryPolicy.halvingInterval is required")
	}
	if aux.MinerFeeShare == nil {
		return MonetaryPolicy{}, errors.New("monetaryPolicy.minerFeeShare is required")
	}

	p := MonetaryPolicy{
		InitialBlockReward: aux.InitialBlockReward.Uint64(),
		HalvingInterval:    aux.HalvingInterval.Uint64(),
		MinerFeeShare:      *aux.MinerFeeShare,
	}
	if aux.TailEmission != nil {
		p.TailEmission = aux.TailEmission.Uint64()
	}
	if err := p.Validate(); err != nil {
		return MonetaryPolicy{}, err
	}
	return p, nil
}
