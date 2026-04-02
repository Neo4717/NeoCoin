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
	fmt.Println("--- RELENTLESS AUDITOR (BVRS v6.0) ---")
	
	// Target set: dormant + puzzle
	data, _ := os.ReadFile("data/puzzle_addresses.json")
	var puzzleAddresses []string
	json.Unmarshal(data, &puzzleAddresses)
	
	dormantData, _ := os.ReadFile("bitcoin_crack/dormant_list.json")
	var dormantAddresses []string
	json.Unmarshal(dormantData, &dormantAddresses)

	allTargets := append(puzzleAddresses, dormantAddresses...)
	fmt.Printf("Relentless loop starting for %d targets...\n", len(allTargets))

	pubkeyMap := make(map[string][]string)
	rValueMap := make(map[string]string)
	mu := sync.Mutex{}

	// Endless cycle
	for {
		for i, addr := range allTargets {
			fmt.Printf("[%s] Auditing %s (%d/%d)...\n", time.Now().Format("15:04:05"), addr, i+1, len(allTargets))
			
			txs := fetchTXs(addr)
			mu.Lock()
			for _, tx := range txs {
				for _, in := range tx.Vin {
					if in.PrevOut.Address == addr {
						pk := extractPK(in.ScriptSig)
						if pk != "" {
							pubkeyMap[pk] = appendUnique(pubkeyMap[pk], addr)
							if len(pubkeyMap[pk]) > 1 {
								fmt.Printf("\n[🔥] COLLISION: Key %s controls %v\n", pk[:16], pubkeyMap[pk])
							}
						}
						
						r := extractR(in.ScriptSig)
						if r != "" {
							if oldTxid, found := rValueMap[r]; found && oldTxid != tx.Txid {
								fmt.Printf("\n[🚨] NONCE REUSE! R: %s | TX1: %s | TX2: %s\n", r, oldTxid, tx.Txid)
							}
							rValueMap[r] = tx.Txid
						}
					}
				}
			}
			mu.Unlock()
			
			time.Sleep(2 * time.Second) // API courtesy
		}
		fmt.Println("Full cycle complete. Restarting in 60s...")
		time.Sleep(60 * time.Second)
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
