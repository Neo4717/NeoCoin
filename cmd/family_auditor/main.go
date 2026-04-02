package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type MempoolTx struct {
	Txid string `json:"txid"`
	Vin  []struct {
		ScriptSig string `json:"scriptsig"`
		PrevOut   struct {
			ScriptPubKey string `json:"scriptpubkey"`
			Address      string `json:"address"`
		} `json:"prevout"`
	} `json:"vin"`
}

type SigData struct {
	Txid      string
	R         string
	Address   string
	ScriptSig string
}

func main() {
	targetPubKey := "022038071e200488f3999bdd959fd2774038092cb513285a157b33f6a6c92cf3fc"
	familyAddresses := []string{
		"1P9fAFAsSLRmMu2P7wZ5CXDPRfLSWTy9N8",
		"18zuLTKQnLjp987LdxuYvjekYnNAvXif2b",
		"1HoDPH3wCSCiyGmSXX7xiadW2DayqaNaCo",
		"15HiQkbvQMoAzXyKdQbuCKTGDxTswYBUf5",
		"1AenFm1zSRkhtPHwZmP2UuRQbWpakD8cVZ",
		"13KYdPnzGh5H8exFY3FhUo9Rvvs6kKAcL8",
		"1EUJKGm3FB65rr5W9anAEoWA3m71WpDayZ",
		"18cKGtwdQHmnDXD6w6AhBhHsmxnK8gsVHf",
		"19DdkMxutkLGY67REFPLu51imfxG9CUJLD",
	}

	fmt.Printf("--- Deep Family Audit: PubKey %s... ---\n", targetPubKey[:16])
	fmt.Printf("Auditing %d linked addresses for cross-transaction nonce reuse...\n\n", len(familyAddresses))

	rValueMap := make(map[string]SigData) // R -> Signature Metadata
	collisionFound := false

	client := &http.Client{Timeout: 30 * time.Second}

	for _, addr := range familyAddresses {
		fmt.Printf("Fetching TXs for %s...\n", addr)
		
		url := fmt.Sprintf("https://mempool.space/api/address/%s/txs", addr)
		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("  Error fetching %s: %v\n", addr, err)
			continue
		}
		
		if resp.StatusCode != 200 {
			fmt.Printf("  Error: HTTP %d for %s\n", resp.StatusCode, addr)
			resp.Body.Close()
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var txs []MempoolTx
		if err := json.Unmarshal(body, &txs); err != nil {
			fmt.Printf("  Error parsing JSON for %s\n", addr)
			continue
		}

		for _, tx := range txs {
			for _, in := range tx.Vin {
				// Only analyze inputs from this specific address/pubkey
				if in.PrevOut.Address == addr {
					r := extractR(in.ScriptSig)
					if r != "" {
						if existing, found := rValueMap[r]; found && existing.Txid != tx.Txid {
							fmt.Printf("\n[🔥] CRITICAL: CROSS-ADDRESS NONCE REUSE DETECTED!\n")
							fmt.Printf("R-Value: %s\n", r)
							fmt.Printf("Collision between:\n")
							fmt.Printf("  1. Address: %s | TXID: %s\n", existing.Address, existing.Txid)
							fmt.Printf("  2. Address: %s | TXID: %s\n", addr, tx.Txid)
							collisionFound = true
						}
						rValueMap[r] = SigData{
							Txid:      tx.Txid,
							R:         r,
							Address:   addr,
							ScriptSig: in.ScriptSig,
						}
					}
				}
			}
		}
		time.Sleep(1 * time.Second) // API courtesy
	}

	if !collisionFound {
		fmt.Printf("\n[✓] Audit complete. No nonce reuse found across %d signatures in this family.\n", len(rValueMap))
	} else {
		fmt.Printf("\n[!] Found potential exploits. Save the R-values for private key recovery.\n")
	}

	// Save the signature database for offline analysis
	os.MkdirAll("data", 0755)
	f, _ := os.Create("data/family_signatures.json")
	json.NewEncoder(f).Encode(rValueMap)
	f.Close()
	fmt.Println("Signature database saved to data/family_signatures.json")
}

func extractR(scriptHex string) string {
	script, _ := hex.DecodeString(scriptHex)
	// DER parser: 0x30 <len> 0x02 <r_len> <r>
	for i := 0; i < len(script)-5; i++ {
		if script[i] == 0x30 && script[i+2] == 0x02 {
			rLen := int(script[i+3])
			if i+4+rLen < len(script) {
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
