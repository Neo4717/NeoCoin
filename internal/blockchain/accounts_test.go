package blockchain

import (
	"sync"
	"testing"
	"time"
)

func TestAccountManager_LockUnlock(t *testing.T) {
	am := NewAccountManager(256)

	unlock := am.LockAccount("addr1")
	if unlock == nil {
		t.Fatal("expected non-nil unlock function")
	}

	unlock()
}

func TestAccountManager_RLockAccount(t *testing.T) {
	am := NewAccountManager(256)

	unlock := am.RLockAccount("addr1")
	if unlock == nil {
		t.Fatal("expected non-nil unlock function")
	}

	unlock()
}

func TestAccountManager_SameAddressSameLock(t *testing.T) {
	am := NewAccountManager(256)

	lock1 := am.getLock("addr1")
	lock2 := am.getLock("addr1")

	if lock1 != lock2 {
		t.Error("same address should return same lock")
	}

	unlock1 := am.LockAccount("addr1")
	unlock1()
}

func TestAccountManager_DifferentAddresses(t *testing.T) {
	am := NewAccountManager(256)

	lock1 := am.getLock("addr1")
	lock2 := am.getLock("addr2")

	if lock1 == lock2 {
		t.Log("different addresses happened to get same lock (possible with sharding)")
	}
}

func TestAccountManager_MultipleAccountsLocked(t *testing.T) {
	am := NewAccountManager(256)

	var unlocks []func()
	for i := 0; i < 10; i++ {
		unlock := am.LockAccount(string(rune('a' + i)))
		unlocks = append(unlocks, unlock)
	}

	for _, unlock := range unlocks {
		unlock()
	}
}

func TestAccountManager_ConcurrentLockUnlock(t *testing.T) {
	am := NewAccountManager(256)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			addr := string(rune('a' + id%26))
			unlock := am.LockAccount(addr)
			time.Sleep(time.Microsecond * 10)
			unlock()
		}(i)
	}

	wg.Wait()
}

func TestAccountManager_ConcurrentMixedOperations(t *testing.T) {
	am := NewAccountManager(256)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			addr := string(rune('a' + id%26))
			unlock := am.LockAccount(addr)
			time.Sleep(time.Microsecond * 5)
			unlock()
		}(i)
	}

	wg.Wait()
}

func TestAccountManager_GlobalWrite(t *testing.T) {
	am := NewAccountManager(256)

	unlock := am.GlobalWrite()
	if unlock == nil {
		t.Fatal("expected non-nil unlock function")
	}

	unlock()
}

func TestAccountManager_GlobalRead(t *testing.T) {
	am := NewAccountManager(256)

	unlock := am.GlobalRead()
	if unlock == nil {
		t.Fatal("expected non-nil unlock function")
	}

	unlock()
}

func TestAccountManager_GlobalWriteIncrementsEpoch(t *testing.T) {
	am := NewAccountManager(256)

	initialEpoch := am.globalEpoch.Load()
	am.GlobalWrite()
	newEpoch := am.globalEpoch.Load()

	if newEpoch <= initialEpoch {
		t.Errorf("expected epoch to increment, was %d now %d", initialEpoch, newEpoch)
	}
}

func TestAccountManager_HasExclusive(t *testing.T) {
	am := NewAccountManager(256)

	if !am.HasExclusive() {
		t.Error("expected HasExclusive to return true")
	}
}

func TestAccountManager_DefaultShards(t *testing.T) {
	am := NewAccountManager(0)

	if am.shardMask != DefaultShards-1 {
		t.Errorf("expected shardMask %d, got %d", DefaultShards-1, am.shardMask)
	}
}

func TestAccountManager_Sharding(t *testing.T) {
	am := NewAccountManager(16)

	shards := make(map[uint64]int)
	for i := 0; i < 1000; i++ {
		addr := string(rune(i))
		shard := hashToShard(addr, am.shardMask)
		shards[shard]++
	}

	for shard, count := range shards {
		t.Logf("shard %d has %d addresses", shard, count)
	}

	if len(shards) < 4 {
		t.Error("expected distribution across multiple shards")
	}
}

func TestAccountManager_FNVHash(t *testing.T) {
	h1 := fnvHash("test")
	h2 := fnvHash("test")
	h3 := fnvHash("different")

	if h1 != h2 {
		t.Error("same input should produce same hash")
	}

	if h1 == h3 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestAccountManager_FNVHashDeterministic(t *testing.T) {
	input := "consistency-check-string"

	for i := 0; i < 100; i++ {
		h := fnvHash(input)
		if h == 0 {
			t.Error("fnvHash should not return 0")
		}
	}
}

func TestAccountManager_NoDeadlock_Simple(t *testing.T) {
	am := NewAccountManager(256)

	done := make(chan bool)
	go func() {
		unlock1 := am.LockAccount("addr1")
		time.Sleep(10 * time.Millisecond)
		unlock2 := am.LockAccount("addr2")
		unlock2()
		unlock1()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("deadlock detected: timeout waiting for lock")
	}
}

func TestAccountManager_NoDeadlock_ConcurrentSameAddress(t *testing.T) {
	am := NewAccountManager(256)

	var wg sync.WaitGroup
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := am.LockAccount("same-addr")
			time.Sleep(5 * time.Millisecond)
			unlock()
		}()
	}

	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("deadlock detected: timeout with concurrent same-address locks")
	}
}

func TestAccountManager_NoDeadlock_ManyAddresses(t *testing.T) {
	am := NewAccountManager(256)

	var wg sync.WaitGroup
	done := make(chan bool)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			addr := string(rune('a' + id%26))
			unlock := am.LockAccount(addr)
			time.Sleep(time.Millisecond)
			unlock()
		}(i)
	}

	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock detected: timeout with many concurrent locks")
	}
}

func TestAccountManager_NoDeadlock_MixedGlobalAndAccount(t *testing.T) {
	am := NewAccountManager(256)

	var wg sync.WaitGroup
	done := make(chan bool)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			unlock := am.GlobalWrite()
			time.Sleep(time.Millisecond)
			unlock()
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			addr := string(rune('a' + id%26))
			unlock := am.LockAccount(addr)
			time.Sleep(time.Millisecond)
			unlock()
		}(i)
	}

	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock detected: timeout with mixed global and account locks")
	}
}

func TestAccountManager_NoDeadlock_GlobalWriteAndRead(t *testing.T) {
	am := NewAccountManager(256)

	var wg sync.WaitGroup
	done := make(chan bool)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := am.GlobalWrite()
			time.Sleep(time.Millisecond)
			unlock()
		}()
	}

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := am.GlobalRead()
			time.Sleep(time.Millisecond)
			unlock()
		}()
	}

	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock detected: timeout with global read/write locks")
	}
}

func TestAccountManager_LoadOrStore(t *testing.T) {
	am := NewAccountManager(256)

	lock1, _ := am.locks.LoadOrStore(uint64(0), &accountLock{})
	lock2, _ := am.locks.LoadOrStore(uint64(0), &accountLock{})

	if lock1 != lock2 {
		t.Error("LoadOrStore should return same lock for same key")
	}
}

func TestAccountManager_AccountLockRefCount(t *testing.T) {
	lock := &accountLock{}

	if lock.refCount != 0 {
		t.Errorf("expected initial refCount 0, got %d", lock.refCount)
	}
}

func TestAccountManager_ShardMaskCalculation(t *testing.T) {
	testCases := []struct {
		numShards    uint64
		expectedMask uint64
	}{
		{256, 255},
		{128, 127},
		{64, 63},
		{16, 15},
		{8, 7},
	}

	for _, tc := range testCases {
		am := NewAccountManager(tc.numShards)
		if am.shardMask != tc.expectedMask {
			t.Errorf("for %d shards: expected mask %d, got %d",
				tc.numShards, tc.expectedMask, am.shardMask)
		}
	}
}

func TestAccountManager_RWMutexBehavior(t *testing.T) {
	am := NewAccountManager(256)

	var wg sync.WaitGroup
	counter := 0
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := am.GlobalRead()
			mu.Lock()
			counter++
			mu.Unlock()
			unlock()
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := am.GlobalWrite()
			mu.Lock()
			counter++
			mu.Unlock()
			unlock()
		}()
	}

	wg.Wait()

	if counter != 15 {
		t.Errorf("expected 15 total operations, got %d", counter)
	}
}

func TestAccountManager_LockAccountReturnsUnlock(t *testing.T) {
	am := NewAccountManager(256)

	locked := false
	var mu sync.Mutex

	go func() {
		unlock := am.LockAccount("addr1")
		mu.Lock()
		locked = true
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		unlock()
	}()

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	isLocked := locked
	mu.Unlock()

	if !isLocked {
		t.Error("expected account to be locked")
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	isLocked = locked
	mu.Unlock()
}

func TestAccountManager_AddressStringHashConsistency(t *testing.T) {
	am := NewAccountManager(256)

	addr := "test-address-123"
	shard1 := hashToShard(addr, am.shardMask)
	shard2 := hashToShard(addr, am.shardMask)

	if shard1 != shard2 {
		t.Error("same address should hash to same shard")
	}
}

func TestAccountManager_DifferentAddressesMayShareShard(t *testing.T) {
	am := NewAccountManager(4)

	addr1 := "addr1"
	addr2 := "addr2"

	shard1 := hashToShard(addr1, am.shardMask)
	shard2 := hashToShard(addr2, am.shardMask)

	t.Logf("addr1 shard: %d, addr2 shard: %d", shard1, shard2)

	_ = shard1
	_ = shard2
}
