package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type MempoolTx struct {
	Txid string `json:"txid"`
	Vin  []struct {
		ScriptSig string `json:"scriptsig"`
	} `json:"vin"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <address>", os.Args[0])
	}
	address := os.Args[1]
	allTxs := []MempoolTx{}
	lastTxid := ""

	fmt.Printf("--- Deep Audit for %s ---\n", address)

	for {
		url := fmt.Sprintf("https://mempool.space/api/address/%s/txs", address)
		if lastTxid != "" {
			url = fmt.Sprintf("%s/chain/%s", url, lastTxid)
		}

		fmt.Printf("Fetching page (after %s)...\n", lastTxid)
		
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			log.Fatalf("failed to fetch: %v", err)
		}
		
		if resp.StatusCode == 429 {
			fmt.Println("Rate limited. Waiting 10s...")
			time.Sleep(10 * time.Second)
			resp.Body.Close()
			continue
		}
		
		if resp.StatusCode != 200 {
			log.Fatalf("HTTP Status %d", resp.StatusCode)
		}

		var page []MempoolTx
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		if err := json.Unmarshal(body, &page); err != nil {
			log.Fatalf("failed to parse: %v", err)
		}

		if len(page) == 0 {
			break
		}

		allTxs = append(allTxs, page...)
		lastTxid = page[len(page)-1].Txid
		
		if len(page) < 25 {
			break // Last page
		}
		
		time.Sleep(1 * time.Second) // API courtesy
	}

	fmt.Printf("\nTotal Transactions Found: %d\n", len(allTxs))
	
	// Save full data
	_ = os.MkdirAll("data", 0755)
	path := "data/full_audit_1GSMG1.json"
	file, _ := os.Create(path)
	json.NewEncoder(file).Encode(allTxs)
	file.Close()
	fmt.Printf("Saved full transaction history to %s\n", path)

	rValues := make(map[string]string)
	foundCount := 0
	for _, tx := range allTxs {
		for _, in := range tx.Vin {
			r := extractRValue(in.ScriptSig)
			if r != "" {
				if oldTxid, exists := rValues[r]; exists && oldTxid != tx.Txid {
					fmt.Printf("\n[!] CRITICAL: NONCE REUSE DETECTED!\n")
					fmt.Printf("R-Value: %s\n", r)
					fmt.Printf("TX1: %s\n", oldTxid)
					fmt.Printf("TX2: %s\n", tx.Txid)
					foundCount++
				}
				rValues[r] = tx.Txid
			}
		}
	}

	if foundCount == 0 {
		fmt.Println("\n[i] Deep audit complete. No nonce reuse found in any transactions.")
	} else {
		fmt.Printf("\n[i] Audit complete. Found %d nonce reuse instances.\n", foundCount)
	}
}

func extractRValue(scriptHex string) string {
	script, err := hex.DecodeString(scriptHex)
	if err != nil {
		return ""
	}
	for i := 0; i < len(script)-6; i++ {
		if script[i] == 0x30 {
			rLen := int(script[i+3])
			if i+4+rLen < len(script) && script[i+2] == 0x02 && script[i+4+rLen] == 0x02 {
				r := script[i+4 : i+4+rLen]
				if len(r) > 0 && r[0] == 0x00 {
					r = r[1:]
				}
				return hex.EncodeToString(r)
			}
		}
	}
	return ""
}
