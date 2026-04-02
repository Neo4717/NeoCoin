package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
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

type PubKeyMap struct {
	Address string   `json:"address"`
	PubKeys []string `json:"pubkeys"`
}

func main() {
	// List of high-value dormant targets (from dormant_list.json)
	data, _ := os.ReadFile("bitcoin_crack/dormant_list.json")
	var addresses []string
	json.Unmarshal(data, &addresses)

	fmt.Printf("--- BIP32/44 Key Collision Hunter (BVRS v3.1) ---\n")
	fmt.Printf("Scanning %d addresses for shared public keys...\n", len(addresses))

	pubkeyToAddress := make(map[string][]string)

	// Scan first 30 addresses to avoid aggressive rate limiting (for demo)
	for i, addr := range addresses {
		if i >= 30 {
			break
		}
		
		fmt.Printf("[%d/%d] Scanning %s...\n", i+1, 30, addr)
		pks := fetchPubKeys(addr)
		for _, pk := range pks {
			pubkeyToAddress[pk] = append(pubkeyToAddress[pk], addr)
		}
		time.Sleep(1500 * time.Millisecond) // Respect API limits
	}

	fmt.Println("\n--- Collision Report ---")
	found := false
	for pk, addrs := range pubkeyToAddress {
		if len(addrs) > 1 {
			fmt.Printf("\n[!] CRITICAL: SAME PUBLIC KEY FOUND ACROSS DIFFERENT ADDRESSES!\n")
			fmt.Printf("PubKey: %s\n", pk)
			for _, a := range addrs {
				fmt.Printf("  - %s\n", a)
			}
			found = true
		}
	}

	if !found {
		fmt.Println("No cross-address key collisions found in this batch.")
	}

	// Save the mapping for future analysis
	f, _ := os.Create("data/pubkey_collision_map.json")
	json.NewEncoder(f).Encode(pubkeyToAddress)
	f.Close()
	fmt.Println("\nMapping saved to data/pubkey_collision_map.json")
}

func fetchPubKeys(address string) []string {
	url := fmt.Sprintf("https://mempool.space/api/address/%s/txs", address)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return nil
	}
	defer resp.Body.Close()

	var txs []MempoolTx
	json.NewDecoder(resp.Body).Decode(&txs)

	uniquePks := make(map[string]bool)
	for _, tx := range txs {
		for _, in := range tx.Vin {
			pk := extractPubKey(in.ScriptSig)
			if pk != "" {
				uniquePks[pk] = true
			}
		}
	}

	var pks []string
	for pk := range uniquePks {
		pks = append(pks, pk)
	}
	return pks
}

func extractPubKey(scriptHex string) string {
	script, _ := hex.DecodeString(scriptHex)
	for i := 0; i < len(script); i++ {
		// Uncompressed
		if script[i] == 0x04 && i+64 < len(script) {
			return hex.EncodeToString(script[i : i+65])
		}
		// Compressed
		if (script[i] == 0x02 || script[i] == 0x03) && i+32 < len(script) {
			return hex.EncodeToString(script[i : i+33])
		}
	}
	return ""
}
