package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

func main() {
	address := "1PWo3JeB9jrGwfHDNpdGK54CRas7fsVzXU"
	fmt.Printf("--- GHOST HUNTER: Searching for Leaked PubKey for %s ---\n", address)

	apis := []string{
		"https://blockchain.info/rawaddr/%s",
		"https://mempool.space/api/address/%s/txs",
		"https://api.blockcypher.com/v1/btc/main/addrs/%s/full",
		"https://blockstream.info/api/address/%s/txs",
	}

	for _, api := range apis {
		url := fmt.Sprintf(api, address)
		fmt.Printf("Checking %s...\n", url)
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if findPubKeyTrace(string(body)) {
			fmt.Printf("\n[🚨] POTENTIAL PUBKEY TRACE FOUND IN API RESPONSE!\n")
			fmt.Printf("Source: %s\n", url)
		} else {
			fmt.Println("  No pubkey trace found.")
		}
		
		time.Sleep(1 * time.Second)
	}
}

func findPubKeyTrace(body string) bool {
	// Match compressed (02/03 + 64 hex) or uncompressed (04 + 128 hex)
	re := regexp.MustCompile(`(0[23][0-9a-fA-F]{64}|04[0-9a-fA-F]{128})`)
	matches := re.FindAllString(body, -1)
	if len(matches) > 0 {
		for _, m := range matches {
			fmt.Printf("\n[!] Candidate PubKey found: %s\n", m)
		}
		return true
	}
	return false
}
