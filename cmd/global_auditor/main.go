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

type SigRecord struct {
	R       string
	Txid    string
	Address string
}

func main() {
	// 1. Load all targets
	data, _ := os.ReadFile("bitcoin_crack/dormant_list.json")
	var addresses []string
	json.Unmarshal(data, &addresses)

	fmt.Printf("--- Global Collision & Nonce-Reuse Auditor ---\n")
	fmt.Printf("Auditing %d dormant addresses collectively...\n", len(addresses))

	pubkeyToAddress := make(map[string][]string)
	rValueToSig := make(map[string]SigRecord)
	
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	limit := make(chan struct{}, 3) // Low concurrency to avoid 429

	for i, addr := range addresses {
		if i >= 50 { break } // Limit for demo
		
		wg.Add(1)
		go func(address string) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()

			txs := fetchTXs(address)
			
			mu.Lock()
			for _, tx := range txs {
				for _, in := range tx.Vin {
					if in.PrevOut.Address == address {
						pk := extractPK(in.ScriptSig)
						if pk != "" {
							pubkeyToAddress[pk] = appendUnique(pubkeyToAddress[pk], address)
						}
						
						r := extractR(in.ScriptSig)
						if r != "" {
							if existing, found := rValueToSig[r]; found && existing.Txid != tx.Txid {
								fmt.Printf("\n[🔥] CRITICAL COLLISION: R=%s\n", r)
								fmt.Printf("  TX1: %s (%s)\n", existing.Txid, existing.Address)
								fmt.Printf("  TX2: %s (%s)\n", tx.Txid, address)
							}
							rValueToSig[r] = SigRecord{R: r, Txid: tx.Txid, Address: address}
						}
					}
				}
			}
			mu.Unlock()
			time.Sleep(1 * time.Second)
		}(addr)
	}

	wg.Wait()

	fmt.Println("\n--- Shared Key Summary ---")
	for pk, addrs := range pubkeyToAddress {
		if len(addrs) > 1 {
			fmt.Printf("Key %s... controls: %v\n", pk[:16], addrs)
		}
	}
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

func extractR(script string) string {
	b, _ := hex.DecodeString(script)
	for i := 0; i < len(b)-5; i++ {
		if b[i] == 0x30 && b[i+2] == 0x02 {
			rLen := int(b[i+3])
			if i+4+rLen < len(b) {
				r := b[i+4 : i+4+rLen]
				if len(r) > 0 && r[0] == 0x00 { r = r[1:] }
				return hex.EncodeToString(r)
			}
		}
	}
	return ""
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice { if s == val { return slice } }
	return append(slice, val)
}
