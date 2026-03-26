#!/bin/bash
cd "$(dirname "$0")/.."
echo "=== NeoCoin Benchmarks ==="
echo ""
echo "Consensus:"
go test -bench=BenchmarkDifficulty -benchmem -run=^$ ./internal/consensus/
echo ""
echo "Crypto:"
go test -bench=BenchmarkHash -benchmem -run=^$ ./internal/crypto/
echo ""
echo "Blockchain:"
go test -bench=BenchmarkBlock -benchmem -run=^$ ./internal/blockchain/
echo ""
echo "Mempool:"
go test -bench=BenchmarkTX -benchmem -run=^$ ./internal/mempool/
