package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

type SyncLoop struct {
	pm PeerAPI
	bc *Blockchain

	interval time.Duration
	window   uint64
}

func NewSyncLoop(pm PeerAPI, bc *Blockchain, interval time.Duration) *SyncLoop {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	return &SyncLoop{
		pm:       pm,
		bc:       bc,
		interval: interval,
		window:   200,
	}
}

func (s *SyncLoop) Run(ctx context.Context) {
	t := time.NewTicker(s.interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.SyncOnce(ctx)
		}
	}
}

func (s *SyncLoop) SyncOnce(ctx context.Context) {
	if s.pm == nil {
		return
	}
	localHeight := s.bc.LatestBlock().Height
	localRulesHash := s.bc.RulesHashHex()
	localGenesisHash := ""
	if genesis, ok := s.bc.BlockByHeight(0); ok {
		localGenesisHash = fmt.Sprintf("%x", genesis.Hash)
	}
	strictIdentity := envBool("STRICT_PEER_IDENTITY", true)

	for _, peer := range s.pm.Peers() {
		info, err := s.pm.FetchChainInfo(ctx, peer)
		if err != nil {
			continue
		}
		if info.ChainID != s.bc.ChainID {
			continue
		}
		if strictIdentity && (info.RulesHash == "" || info.GenesisHash == "") {
			continue
		}
		if info.RulesHash != "" && localRulesHash != "" && info.RulesHash != localRulesHash {
			continue
		}
		if info.GenesisHash != "" && localGenesisHash != "" && info.GenesisHash != localGenesisHash {
			continue
		}
		if info.Height <= localHeight {
			continue
		}

		var from uint64
		if info.Height > s.window {
			from = info.Height - s.window
		}

		headers, err := s.pm.FetchHeadersFrom(ctx, peer, from, int(s.window))
		if err != nil {
			continue
		}
		for _, h := range headers {
			if _, ok := s.bc.BlockByHash(h.HashHex); ok {
				continue
			}
			b, err := s.pm.FetchBlockByHash(ctx, peer, h.HashHex)
			if err != nil {
				continue
			}
			_, err = s.bc.AddBlock(b)
			if err != nil && errors.Is(err, ErrUnknownParent) {
				// fetch parent chain then retry once
				if ferr := s.pm.EnsureAncestors(ctx, s.bc, h.PrevHashHex); ferr == nil {
					_, _ = s.bc.AddBlock(b)
				}
				continue
			}
			if err != nil {
				log.Printf("sync: add block failed: %v", err)
			}
		}
	}
}
