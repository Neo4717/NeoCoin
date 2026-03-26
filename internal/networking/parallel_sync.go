package networking

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Neo4717/NeoCoin/config"
	"github.com/Neo4717/NeoCoin/internal/blockchain"
)

type ParallelSync struct {
	pool       *PoolManager
	pm         *PeerManager
	chain      *blockchain.Blockchain
	workers    int
	maxBatch   int
	downloadMu sync.Mutex
	inFlight   atomic.Int32
}

func NewParallelSync(pool *PoolManager, chain *blockchain.Blockchain, pm *PeerManager, cfg *config.Config) *ParallelSync {
	workers := 8
	if cfg != nil && cfg.SyncWorkers > 0 {
		workers = cfg.SyncWorkers
	}

	return &ParallelSync{
		pool:     pool,
		pm:       pm,
		chain:    chain,
		workers:  workers,
		maxBatch: 100,
	}
}

func (ps *ParallelSync) DownloadBlocks(ctx context.Context, startHeight, endHeight int64) error {
	if endHeight <= startHeight {
		return fmt.Errorf("invalid range: %d to %d", startHeight, endHeight)
	}

	tasks := make(chan BlockRange, ps.workers)
	results := make(chan BlockDownloadResult, ps.workers*2)
	errors := make(chan error, ps.workers)

	var wg sync.WaitGroup
	for i := 0; i < ps.workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ps.downloadWorker(ctx, workerID, tasks, results, errors)
		}(i)
	}

	go func() {
		defer close(tasks)
		for height := startHeight; height <= endHeight; {
			batchEnd := height + int64(ps.maxBatch) - 1
			if batchEnd > endHeight {
				batchEnd = endHeight
			}
			select {
			case tasks <- BlockRange{Start: height, End: batchEnd}:
				height = batchEnd + 1
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	blocksAdded := 0
	var lastErr error

	for result := range results {
		if result.Err != nil {
			lastErr = result.Err
			continue
		}

		for _, block := range result.Blocks {
			if _, err := ps.chain.AddBlock(block); err != nil {
				lastErr = err
			} else {
				blocksAdded++
			}
		}
	}

	for err := range errors {
		lastErr = err
	}

	if blocksAdded == 0 && lastErr != nil {
		return lastErr
	}

	return nil
}

type BlockRange struct {
	Start int64
	End   int64
}

type BlockDownloadResult struct {
	PeerID  string
	Blocks  []*blockchain.Block
	Err     error
	Latency int64
}

func (ps *ParallelSync) downloadWorker(
	ctx context.Context,
	workerID int,
	tasks <-chan BlockRange,
	results chan<- BlockDownloadResult,
	errors chan<- error,
) {
	for task := range tasks {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ps.inFlight.Add(1)
		defer ps.inFlight.Add(-1)

		peer := ps.selectBestPeer(task.Start)
		if peer == nil {
			errors <- fmt.Errorf("no available peers for height %d", task.Start)
			continue
		}

		blocks, latency, err := ps.downloadRangeFromPeer(ctx, peer, task.Start, task.End)

		results <- BlockDownloadResult{
			PeerID:  peer.ID,
			Blocks:  blocks,
			Err:     err,
			Latency: latency,
		}

		if err == nil && len(blocks) > 0 {
			ps.updatePeerScore(peer, true, latency)
		} else {
			ps.updatePeerScore(peer, false, latency)
		}
	}
}

func (ps *ParallelSync) downloadRangeFromPeer(
	ctx context.Context,
	peer *Peer,
	startHeight, endHeight int64,
) ([]*blockchain.Block, int64, error) {
	start := time.Now()

	headers, err := peer.RequestHeaders(startHeight, endHeight)
	if err != nil {
		return nil, 0, err
	}

	if len(headers) == 0 {
		return nil, 0, fmt.Errorf("no headers returned")
	}

	blockChan := make(chan *blockchain.Block, 50)
	errChan := make(chan error, 10)

	go func() {
		defer close(blockChan)

		for i := 0; i < len(headers); i += 50 {
			batchEnd := i + 50
			if batchEnd > len(headers) {
				batchEnd = len(headers)
			}

			hashes := make([]string, batchEnd-i)
			for j := i; j < batchEnd; j++ {
				hashes[j-i] = hex.EncodeToString(headers[j].Hash)
			}

			blocks, err := peer.RequestBlocks(hashes)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
				return
			}

			for _, block := range blocks {
				select {
				case blockChan <- block:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	var blocks []*blockchain.Block
	for {
		select {
		case block := <-blockChan:
			if block == nil {
				goto done
			}
			blocks = append(blocks, block)
		case err := <-errChan:
			return blocks, time.Since(start).Milliseconds(), err
		case <-ctx.Done():
			return blocks, time.Since(start).Milliseconds(), ctx.Err()
		}
	}

done:
	return blocks, time.Since(start).Milliseconds(), nil
}

func (ps *ParallelSync) selectBestPeer(height int64) *Peer {
	peers := ps.pm.GetHealthyPeers()
	if len(peers) == 0 {
		return nil
	}

	sort.Slice(peers, func(i, j int) bool {
		return peers[i].Score > peers[j].Score
	})

	for _, p := range peers {
		if p.LatencyMs() < 5000 && p.InFlight() < 10 && !p.RecentlyFailed() {
			return p
		}
	}

	return peers[0]
}

func (ps *ParallelSync) updatePeerScore(peer *Peer, success bool, latency int64) {
	if peer == nil {
		return
	}
	if success {
		ps.pm.RecordSuccess(peer.Address, latency)
	} else {
		ps.pm.RecordFailure(peer.Address)
		peer.RecordFailure()
	}
}
