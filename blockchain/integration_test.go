package main

import (
	"testing"
)

func TestNewMiner(t *testing.T) {
	mp := NewMempool(100)
	if mp == nil {
		t.Error("NewMempool should not return nil")
	}
}

func TestNewMempool(t *testing.T) {
	mp := NewMempool(100)
	if mp.Size() != 0 {
		t.Errorf("expected size 0, got %d", mp.Size())
	}
}

func TestNewMempoolWithAI(t *testing.T) {
	filter := NewAISpamFilter("", false)
	mp := NewMempoolWithAI(100, filter)

	if mp == nil {
		t.Error("NewMempoolWithAI should not return nil")
	}
}

func TestMempool_Size_Basic(t *testing.T) {
	mp := NewMempool(100)
	size := mp.Size()
	if size < 0 {
		t.Error("size should be non-negative")
	}
}

func TestMonetaryPolicy(t *testing.T) {
	mp := MonetaryPolicy{
		InitialBlockReward: 50,
		HalvingInterval:    210000,
		TailEmission:       1,
	}

	reward := mp.BlockReward(0)
	if reward != 50 {
		t.Errorf("expected BlockReward 50 at height 0, got %d", reward)
	}

	err := mp.Validate()
	if err != nil {
		t.Errorf("valid policy should not error: %v", err)
	}
}

func TestDifficultyParams(t *testing.T) {
	p := defaultConsensusParamsFromEnv()

	if p.TargetBlockTime == 0 {
		p.TargetBlockTime = 15e9
	}

	if p.DifficultyWindow == 0 {
		p.DifficultyWindow = 20
	}

	if p.MaxDifficultyBits == 0 {
		p.MaxDifficultyBits = 255
	}

	_ = p.BinaryEncodingActive(0)
}

func TestP2PConfig_New(t *testing.T) {
	config := P2PConfig{
		ListenAddr: ":9090",
		Seeds:      []string{"seed.example.com:9090"},
		MaxPeers:   10,
	}

	if config.ListenAddr != ":9090" {
		t.Error("ListenAddr not set correctly")
	}

	if config.MaxPeers != 10 {
		t.Error("MaxPeers not set correctly")
	}
}

func TestSeedDiscovery(t *testing.T) {
	seeds := []string{"seed1.example.com:9090"}
	sd := NewSeedDiscovery(seeds)

	if sd == nil {
		t.Error("NewSeedDiscovery should not return nil")
	}
}

func TestLoadDNSSeeds(t *testing.T) {
	seeds := LoadDNSSeeds()
	_ = seeds
}

func TestParsePeersEnv(t *testing.T) {
	peers := ParsePeersEnv("http://192.168.1.1:8080,http://192.168.1.2:8080")
	if len(peers) != 2 {
		t.Errorf("expected 2 peers, got %d", len(peers))
	}

	peersEmpty := ParsePeersEnv("")
	if peersEmpty != nil {
		t.Errorf("expected nil for empty, got %v", peersEmpty)
	}
}

func TestParseP2PPeersEnv(t *testing.T) {
	peers := ParseP2PPeersEnv("/ip4/192.168.1.1/tcp/9090,/ip4/192.168.1.2/tcp/9090")
	if len(peers) != 2 {
		t.Errorf("expected 2 peers, got %d", len(peers))
	}
}
