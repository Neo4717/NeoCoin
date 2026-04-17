package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type SeedDiscovery struct {
	seeds     []Seed
	resolver  *net.Resolver
	mu        sync.RWMutex
	peerAddrs []string
}

type Seed struct {
	Host string
	Port int
	Type string // "onion" or "ip"
}

func NewSeedDiscovery(seeds []string) *SeedDiscovery {
	sd := &SeedDiscovery{
		seeds: make([]Seed, 0),
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, "udp", "8.8.8.8:53")
			},
		},
	}

	// Parse seed strings
	for _, s := range seeds {
		seed := Seed{}
		if len(s) > 6 && s[len(s)-6:] == ".onion" {
			seed.Type = "onion"
			seed.Host = s[:len(s)-6]
			seed.Port = 9090
		} else {
			seed.Type = "ip"
			if host, port, err := net.SplitHostPort(s); err == nil {
				seed.Host = host
				fmt.Sscanf(port, "%d", &seed.Port)
			} else {
				seed.Host = s
				seed.Port = 9090
			}
		}
		sd.seeds = append(sd.seeds, seed)
	}

	return sd
}

func (sd *SeedDiscovery) GetPeers() []string {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.peerAddrs
}

func (sd *SeedDiscovery) Start(ctx context.Context) error {
	// Initial lookup
	if err := sd.lookup(); err != nil {
		fmt.Printf("Seed discovery initial lookup failed: %v\n", err)
	}

	// Periodic refresh
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sd.lookup()
			}
		}
	}()

	return nil
}

func (sd *SeedDiscovery) lookup() error {
	var addrs []string

	for _, seed := range sd.seeds {
		if seed.Type == "onion" {
			// For onion, just add directly (Tor handles resolution)
			addrs = append(addrs, fmt.Sprintf("%s.onion:%d", seed.Host, seed.Port))
			continue
		}

		// DNS lookup for A/AAAA records
		ips, err := sd.resolver.LookupIP(context.Background(), "ip", seed.Host)
		if err != nil {
			continue
		}

		for _, ip := range ips {
			addrs = append(addrs, fmt.Sprintf("%s:%d", ip.String(), seed.Port))
		}
	}

	sd.mu.Lock()
	sd.peerAddrs = addrs
	sd.mu.Unlock()

	return nil
}

// DNSSeedsFromConfig loads seeds from config file or environment
func LoadDNSSeeds() []string {
	// Check for config file
	if data, err := os.ReadFile("../config/seeds.json"); err == nil {
		var config struct {
			Seeds []struct {
				Host string `json:"host"`
				Port int    `json:"port"`
			} `json:"seeds"`
		}
		if json.Unmarshal(data, &config) == nil {
			seeds := make([]string, len(config.Seeds))
			for i, s := range config.Seeds {
				seeds[i] = fmt.Sprintf("%s:%d", s.Host, s.Port)
			}
			return seeds
		}
	}

	// Environment variable override
	if envPeers := os.Getenv("DNS_SEEDS"); envPeers != "" {
		return []string{envPeers}
	}

	// Default empty (will use P2P_PEERS)
	return nil
}
