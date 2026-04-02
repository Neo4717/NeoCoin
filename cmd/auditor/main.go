package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"b3-research/internal/scanner"
)

func main() {
	address := "1GSMG1JC9wtdSwfwApgj2xcmJPAwx7pr"
	fmt.Printf("--- Auditing Transaction History for %s ---\n", address)

	res := scanner.ScanAddress(address)
	if res.Error != nil {
		log.Fatalf("Audit failed: %v", res.Error)
	}

	// For detailed view, let's fetch raw data manually to save it
	dataDir := "data"
	os.MkdirAll(dataDir, 0755)
	
	// Re-fetch or just use the scan logic to export
	// Since ScanAddress doesn't return the raw list, let's just output the findings.
	
	if res.FoundReuse {
		fmt.Printf("\n[!] SUCCESS: Nonce reuse detected!\nDuplicate R-Value: %s\n", res.FoundRValue)
	} else {
		fmt.Println("\n[i] No nonce reuse detected in the fetched transaction history.")
	}

	// Create a log of the activity
	logPath := filepath.Join(dataDir, "audit_log.json")
	logData := map[string]interface{}{
		"address":     address,
		"timestamp":   scanner.ScanResult{Address: address}.FoundRValue, // Just a placeholder
		"status":      "complete",
		"found_reuse": res.FoundReuse,
		"r_value":     res.FoundRValue,
	}
	
	f, _ := os.Create(logPath)
	json.NewEncoder(f).Encode(logData)
	f.Close()
	
	fmt.Printf("Audit summary saved to %s\n", logPath)
}
