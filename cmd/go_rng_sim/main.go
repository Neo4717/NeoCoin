package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

func getAddressFromPrivKey(privHex string) (string, error) {
	privBytes, err := hex.DecodeString(privHex)
	if err != nil {
		return "", err
	}
	_, pub := btcec.PrivKeyFromBytes(privBytes)
	
	// Legacy P2PKH address
	addr, err := btcutil.NewAddressPubKeyHash(btcutil.Hash160(pub.SerializeUncompressed()), &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}
	return addr.EncodeAddress(), nil
}

func main() {
	fmt.Println("--- High-Speed Go RNG Simulator (BVRS v5.0) ---")

	// Load targets
	data, _ := os.ReadFile("data/puzzle_addresses.json")
	var puzzleAddresses []string
	json.Unmarshal(data, &puzzleAddresses)
	
	dormantData, _ := os.ReadFile("bitcoin_crack/dormant_list.json")
	var dormantAddresses []string
	json.Unmarshal(dormantData, &dormantAddresses)

	allTargets := make(map[string]bool)
	for _, a := range puzzleAddresses { allTargets[a] = true }
	for _, a := range dormantAddresses { allTargets[a] = true }

	fmt.Printf("Monitoring %d target addresses...\n", len(allTargets))

	years := []int{2009, 2010, 2011, 2012, 2013}
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)

	var wg sync.WaitGroup

	for _, year := range years {
		startTs := int(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
		endTs := int(time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
		
		fmt.Printf("Launching workers for year %d...\n", year)
		
		chunkSize := (endTs - startTs) / numCPU
		
		for i := 0; i < numCPU; i++ {
			wg.Add(1)
			go func(start, end int) {
				defer wg.Done()
				for ts := start; ts < end; ts++ {
					seed := []byte(fmt.Sprintf("%d", ts))
					hash := sha256.Sum256(seed)
					privHex := hex.EncodeToString(hash[:])
					
					// Use address check
					addr, err := getAddressFromPrivKey(privHex)
					if err == nil {
						if allTargets[addr] {
							fmt.Printf("\n[🚨] MATCH FOUND! Seed: %d -> Address: %s\n", ts, addr)
							fmt.Printf("Private Key (hex): %s\n", privHex)
						}
					}
				}
			}(startTs + (i * chunkSize), startTs + ((i + 1) * chunkSize))
		}
	}

	wg.Wait()
	fmt.Println("\nSimulation complete.")
}
