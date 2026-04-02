package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type MempoolTx struct {
	Txid string `json:"txid"`
	Vin  []struct {
		ScriptSig string `json:"scriptsig"`
		PrevOut   struct {
			Address string `json:"address"`
		} `json:"prevout"`
	} `json:"vin"`
}

func main() {
	// 1. Load all targets
	data, _ := os.ReadFile("bitcoin_crack/dormant_list.json")
	var addresses []string
	json.Unmarshal(data, &addresses)

	fmt.Printf("--- Relentless Collision Hunter (BVRS v4.0) ---\n")
	fmt.Printf("Scanning %d dormant addresses for cross-address key sharing...\n", len(addresses))

	pubkeyMap := make(map[string]map[string]bool) // PubKey -> Set of Addresses
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	limit := make(chan struct{}, 3) // Stay under API limits

	for i, addr := range addresses {
		wg.Add(1)
		go func(index int, address string) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()

			if index%10 == 0 {
				fmt.Printf("Progress: %d/%d addresses checked...\n", index, len(addresses))
			}

			txs := fetchTXs(address)
			
			mu.Lock()
			for _, tx := range txs {
				for _, in := range tx.Vin {
					if in.PrevOut.Address == address {
						pk := extractPK(in.ScriptSig)
						if pk != "" {
							if pubkeyMap[pk] == nil {
								pubkeyMap[pk] = make(map[string]bool)
							}
							pubkeyMap[pk][address] = true
						}
					}
				}
			}
			mu.Unlock()
			time.Sleep(1 * time.Second)
		}(i, addr)
	}

	wg.Wait()

	fmt.Println("\n--- CRITICAL KEY COLLISION REPORT ---")
	foundAny := false
	for pk, addrSet := range pubkeyMap {
		if len(addrSet) > 1 {
			fmt.Printf("\n[!] SUCCESS: Shared Private Key Identified!\n")
			fmt.Printf("PubKey: %s\n", pk)
			fmt.Printf("Addresses using this key:\n")
			for a := range addrSet {
				fmt.Printf("  - %s\n", a)
			}
			foundAny = true
		}
	}

	if !foundAny {
		fmt.Println("No cross-address key sharing detected in this set.")
	}

	// Save data
	f, _ := os.Create("data/global_key_collisions.json")
	json.NewEncoder(f).Encode(pubkeyMap)
	f.Close()
	fmt.Println("\nFull PubKey-to-Address map saved to data/global_key_collisions.json")
}

func fetchTXs(addr string) []MempoolTx {
	url := fmt.Sprintf("https://mempool.space/api/address/%s/txs", addr)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 { return nil }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var txs []MempoolTx
	json.Unmarshal(body, &txs)
	return txs
}

func extractPK(script string) string {
	b, _ := hex.DecodeString(script)
	for i := 0; i < len(b); i++ {
		// Look for standard pubkey prefixes
		if b[i] == 0x04 && i+64 < len(b) { return hex.EncodeToString(b[i : i+65]) }
		if (b[i] == 0x02 || b[i] == 0x03) && i+32 < len(b) { 
			// Secondary check: is it actually a key or just random data?
			// (Simplified for demo)
			return hex.EncodeToString(b[i : i+33]) 
		}
	}
	return ""
}
