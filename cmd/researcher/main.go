package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"b3-research/internal/scanner"
)

func main() {
	fmt.Println("--- BVRS v3 (High-Performance Go Edition) ---")

	// Load dormant targets
	targetsPath := "bitcoin_crack/dormant_list.json"
	data, err := os.ReadFile(targetsPath)
	if err != nil {
		log.Fatalf("failed to read targets: %v", err)
	}

	var addresses []string
	if err := json.Unmarshal(data, &addresses); err != nil {
		log.Fatalf("failed to parse targets: %v", err)
	}

	fmt.Printf("Starting concurrent scan of %d addresses...\n", len(addresses))

	results := make(chan scanner.ScanResult)
	var wg sync.WaitGroup

	// Concurrency limiter
	limit := make(chan struct{}, 5) 

	for _, addr := range addresses {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()

			res := scanner.ScanAddress(address)
			results <- res
			time.Sleep(2 * time.Second) // Rate limiting
		}(addr)
	}

	// Closer
	go func() {
		wg.Wait()
		close(results)
	}()

	// Monitor results
	successCount := 0
	errorCount := 0
	for res := range results {
		if res.Error != nil {
			errorCount++
		} else {
			successCount++
			if res.FoundReuse {
				fmt.Printf("\n[!!!] CRITICAL: Nonce reuse found at %s (R: %s)\n", res.Address, res.FoundRValue)
			}
		}
		
		if (successCount+errorCount)%10 == 0 {
			fmt.Printf("Progress: %d/%d scanned...\n", successCount+errorCount, len(addresses))
		}
	}

	fmt.Printf("\nScan complete. Success: %d, Errors: %d\n", successCount, errorCount)
}
