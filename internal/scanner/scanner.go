package scanner

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Transaction struct {
	Hash   string `json:"hash"`
	Inputs []struct {
		Script string `json:"script"`
	} `json:"inputs"`
}

type AddressResponse struct {
	Transactions []Transaction `json:"txs"`
}

type ScanResult struct {
	Address     string
	FoundReuse  bool
	FoundRValue string
	Error       error
}

func ScanAddress(address string) ScanResult {
	url := fmt.Sprintf("https://blockchain.info/rawaddr/%s", address)
	
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return ScanResult{Address: address, Error: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return ScanResult{Address: address, Error: fmt.Errorf("rate limited (429)")}
	}
	if resp.StatusCode != 200 {
		return ScanResult{Address: address, Error: fmt.Errorf("HTTP status %d", resp.StatusCode)}
	}

	var data AddressResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		// If decoding failed, it might be a text response like "Rate limited" even with 200 OK
		return ScanResult{Address: address, Error: fmt.Errorf("decode error (check if rate limited): %v", err)}
	}

	rValues := make(map[string]bool)
	for _, tx := range data.Transactions {
		for _, in := range tx.Inputs {
			r := extractRValue(in.Script)
			if r != "" {
				if rValues[r] {
					return ScanResult{Address: address, FoundReuse: true, FoundRValue: r}
				}
				rValues[r] = true
			}
		}
	}

	return ScanResult{Address: address, FoundReuse: false}
}

// extractRValue implements DER signature parsing for R extraction.
func extractRValue(scriptHex string) string {
	script, err := hex.DecodeString(scriptHex)
	if err != nil {
		return ""
	}

	// DER signature: 0x30 <len> 0x02 <r_len> <r> 0x02 <s_len> <s> <sighash>
	// Standard P2PKH script usually has the signature after a push op.
	for i := 0; i < len(script)-6; i++ {
		if script[i] == 0x30 {
			rLen := int(script[i+3])
			if i+4+rLen < len(script) && script[i+2] == 0x02 && script[i+4+rLen] == 0x02 {
				r := script[i+4 : i+4+rLen]
				// Strip leading 0x00 padding if it exists
				if len(r) > 0 && r[0] == 0x00 {
					r = r[1:]
				}
				return hex.EncodeToString(r)
			}
		}
	}
	return ""
}
