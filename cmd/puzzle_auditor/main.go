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
	// Load the 160 puzzle addresses
	data, err := os.ReadFile("data/puzzle_addresses.json")
	if err != nil {
		fmt.Printf("Error reading data/puzzle_addresses.json: %v\n", err)
		return
	}
	var puzzleAddresses []string
	if err := json.Unmarshal(data, &puzzleAddresses); err != nil {
		fmt.Printf("Error parsing puzzle addresses: %v\n", err)
		return
	}

	fmt.Println("--- Mass Puzzle Address Collision Auditor (BVRS v3.2) ---")
	fmt.Printf("Auditing %d puzzle targets for shared keys...\n", len(puzzleAddresses))

	pubkeyMap := make(map[string][]string)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	
	// Use limited concurrency to avoid 429
	limit := make(chan struct{}, 2)

	for i, addr := range puzzleAddresses {
		wg.Add(1)
		go func(index int, address string) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()

			fmt.Printf("[%d/%d] Auditing %s...\n", index+1, len(puzzleAddresses), address)
			txs := fetchTXs(address)
			
			mu.Lock()
			for _, tx := range txs {
				for _, in := range tx.Vin {
					if in.PrevOut.Address == address {
						pk := extractPK(in.ScriptSig)
						if pk != "" {
							pubkeyMap[pk] = appendUnique(pubkeyMap[pk], address)
						}
					}
				}
			}
			mu.Unlock()
			time.Sleep(1 * time.Second)
		}(i, addr)
	}

	wg.Wait()

	fmt.Println("\n--- Final Shared Key Report ---")
	found := false
	for pk, addrs := range pubkeyMap {
		if len(addrs) > 1 {
			fmt.Printf("\n[🔥] SHARED KEY FOUND: %s\n", pk)
			for _, a := range addrs {
				fmt.Printf("  - %s\n", a)
			}
			found = true
		}
	}

	if !found {
		fmt.Println("Audit complete. No shared keys found among the 160 puzzle addresses.")
	}

	// Save the results
	f, _ := os.Create("data/puzzle_pubkey_mapping.json")
	json.NewEncoder(f).Encode(pubkeyMap)
	f.Close()
	fmt.Println("\nFull mapping saved to data/puzzle_pubkey_mapping.json")
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
		if b[i] == 0x04 && i+64 < len(b) { return hex.EncodeToString(b[i : i+65]) }
		if (b[i] == 0x02 || b[i] == 0x03) && i+32 < len(b) { return hex.EncodeToString(b[i : i+33]) }
	}
	return ""
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice { if s == val { return slice } }
	return append(slice, val)
}
