package main

import (
	"crypto/sha256"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
)

type TMValidator struct {
	Address string
	PubKey  []byte
	Stake   uint64
	Power   uint64
	Online  bool
	Jailed  bool
}

type Tendermint struct {
	mu    sync.RWMutex
	vals  map[string]*TMValidator
	state TMState
	votes map[uint64]map[uint64]map[string][]byte
}

type TMState struct {
	Height      uint64
	Round       uint64
	Step        int
	Locked      bool
	LockedRound int64
}

func NewTendermint() *Tendermint {
	return &Tendermint{
		vals:  make(map[string]*TMValidator),
		state: TMState{Height: 1},
		votes: make(map[uint64]map[uint64]map[string][]byte),
	}
}

func (t *Tendermint) AddValidator(pubKey []byte, stake uint64) string {
	addr := GenerateAddress(pubKey)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.vals[addr] == nil {
		t.vals[addr] = &TMValidator{Address: addr, PubKey: pubKey, Stake: stake, Power: stake, Online: true}
		log.Printf("[TM] Validator: %s stake=%d", addr[:16], stake)
	}
	return addr
}

func (t *Tendermint) GetProposer(height, round uint64) string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	list := t.sortedVals()
	if len(list) == 0 {
		return ""
	}
	return list[int(height+round)%len(list)].Address
}

func (t *Tendermint) sortedVals() []*TMValidator {
	var list []*TMValidator
	for _, v := range t.vals {
		if v.Online && !v.Jailed {
			list = append(list, v)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Stake > list[j].Stake })
	return list
}

func (t *Tendermint) quorum() uint64 {
	var total uint64
	for _, v := range t.vals {
		if v.Online && !v.Jailed {
			total += v.Power
		}
	}
	if total == 0 {
		return 0
	}
	return (total * 2) / 3
}

func (t *Tendermint) Vote(addr string, blockHash []byte) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	v, ok := t.vals[addr]
	if !ok || !v.Online || v.Jailed {
		return false
	}
	h, r := t.state.Height, t.state.Round
	if t.votes[h] == nil {
		t.votes[h] = make(map[uint64]map[string][]byte)
	}
	if t.votes[h][r] == nil {
		t.votes[h][r] = make(map[string][]byte)
	}
	t.votes[h][r][addr] = blockHash
	return t.countVotes(h, r, blockHash) >= int(t.quorum())
}

func (t *Tendermint) countVotes(height, round uint64, blockHash []byte) int {
	if t.votes[height] == nil || t.votes[height][round] == nil {
		return 0
	}
	c := 0
	for _, h := range t.votes[height][round] {
		if string(h) == string(blockHash) {
			c++
		}
	}
	return c
}

func (t *Tendermint) AdvanceRound() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state.Round++
	t.state.Locked = false
}

func (t *Tendermint) AdvanceHeight() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state.Height++
	t.state.Round = 0
}

func (t *Tendermint) GetHeight() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state.Height
}

func TMFromEnv() *Tendermint {
	t := NewTendermint()
	for _, a := range strings.Split(os.Getenv("TM_VALIDATORS"), ",") {
		if a = strings.TrimSpace(a); a != "" {
			h := sha256.Sum256([]byte(a))
			hh := make([]byte, 32)
			copy(hh, h[:])
			t.AddValidator(hh, 1000)
		}
	}
	return t
}
