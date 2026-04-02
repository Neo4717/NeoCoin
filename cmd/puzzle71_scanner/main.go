package main

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

func main() {
	targetAddress := "1PWo3JeB9jrGwfHDNpdGK54CRas7fsVzXU"
	fmt.Printf("--- Specialized Puzzle 71 Scanner ---\n")
	fmt.Printf("Target: %s\n", targetAddress)
	fmt.Printf("Range: 2^70 to 2^71-1\n\n")

	// Range boundaries
	start := new(big.Int).Exp(big.NewInt(2), big.NewInt(70), nil)
	end := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(71), nil), big.NewInt(1))

	fmt.Printf("Start Hex: %x\n", start)
	fmt.Printf("End Hex:   %x\n", end)

	numCPU := runtime.NumCPU()
	var wg sync.WaitGroup
	
	// Chunking for workers
	totalKeys := new(big.Int).Sub(end, start)
	chunkSize := new(big.Int).Div(totalKeys, big.NewInt(int64(numCPU)))

	for i := 0; i < numCPU; i++ {
		workerStart := new(big.Int).Add(start, new(big.Int).Mul(big.NewInt(int64(i)), chunkSize))
		workerEnd := new(big.Int).Add(workerStart, chunkSize)
		if i == numCPU-1 {
			workerEnd = end
		}

		wg.Add(1)
		go func(id int, s, e *big.Int) {
			defer wg.Done()
			current := new(big.Int).Set(s)
			count := 0
			
			for current.Cmp(e) <= 0 {
				privBytes := current.Bytes()
				// Ensure 32 bytes
				fullPriv := make([]byte, 32)
				copy(fullPriv[32-len(privBytes):], privBytes)
				
				_, pub := btcec.PrivKeyFromBytes(fullPriv)
				
				// Standard P2PKH (Uncompressed for early puzzles, but usually compressed)
				// We check compressed first as it's standard now
				addr, _ := btcutil.NewAddressPubKeyHash(btcutil.Hash160(pub.SerializeCompressed()), &chaincfg.MainNetParams)
				if addr.EncodeAddress() == targetAddress {
					fmt.Printf("\n[🚨] SUCCESS! PRIVATE KEY FOUND FOR PUZZLE 71!\n")
					fmt.Printf("Private Key (Hex): %x\n", fullPriv)
					return
				}

				current.Add(current, big.NewInt(1))
				count++
				if count % 100000 == 0 {
					// Progress heartbeat for worker
				}
			}
		}(i, workerStart, workerEnd)
	}

	wg.Wait()
	fmt.Println("Scan finished.")
}
