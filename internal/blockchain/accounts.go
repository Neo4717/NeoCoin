package blockchain

import (
	"sync"
	"sync/atomic"
)

type AccountManager struct {
	locks       sync.Map
	shardMask   uint64
	globalMu    sync.RWMutex
	globalEpoch atomic.Int64
}

type accountLock struct {
	mu       sync.Mutex
	refCount int32
}

const DefaultShards = 256

func NewAccountManager(numShards uint64) *AccountManager {
	if numShards == 0 {
		numShards = DefaultShards
	}
	return &AccountManager{
		shardMask: numShards - 1,
	}
}

func (am *AccountManager) getLock(addr string) *accountLock {
	shard := hashToShard(addr, am.shardMask)
	lockI, _ := am.locks.LoadOrStore(shard, &accountLock{})
	return lockI.(*accountLock)
}

func hashToShard(addr string, mask uint64) uint64 {
	h := fnvHash(addr)
	return uint64(h) & mask
}

func fnvHash(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func (am *AccountManager) LockAccount(addr string) func() {
	lock := am.getLock(addr)
	lock.mu.Lock()
	return lock.mu.Unlock
}

func (am *AccountManager) RLockAccount(addr string) func() {
	lock := am.getLock(addr)
	lock.mu.Lock()
	return lock.mu.Unlock
}

func (am *AccountManager) LockAccounts(accounts map[string]struct{}) []func() {
	unlocks := make([]func(), 0, len(accounts))
	for addr := range accounts {
		unlocks = append(unlocks, am.LockAccount(addr))
	}
	return unlocks
}

func (am *AccountManager) GlobalWrite() func() {
	am.globalMu.Lock()
	am.globalEpoch.Add(1)
	return am.globalMu.Unlock
}

func (am *AccountManager) GlobalRead() func() {
	am.globalMu.RLock()
	return am.globalMu.RUnlock
}

func (am *AccountManager) HasExclusive() bool {
	return true
}
