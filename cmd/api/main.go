package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpapi "github.com/Neo4717/NeoCoin/api/http"
	"github.com/Neo4717/NeoCoin/api/websocket"
	"github.com/Neo4717/NeoCoin/config"
	"github.com/Neo4717/NeoCoin/internal/blockchain"
	"github.com/Neo4717/NeoCoin/internal/logger"
	"github.com/Neo4717/NeoCoin/internal/mempool"
)

func main() {
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Config validation: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.LogLevel, logger.TextEncoding, "", "", false)
	logger.Info("NeoCoin API service starting...")

	store, err := blockchain.OpenChainStoreFromEnv()
	if err != nil {
		logger.Error("Failed to open chain store: %v", err)
		os.Exit(1)
	}
	defer func() {
		if s, ok := store.(*blockchain.BoltChainStore); ok {
			s.Close()
		}
	}()

	bc, err := blockchain.LoadBlockchain(cfg.ChainID, cfg.MinerAddress, store, 0, cfg)
	if err != nil {
		logger.Error("Failed to load blockchain: %v", err)
		os.Exit(1)
	}

	mp := mempool.NewMempool(cfg)
	wsHub := websocket.NewHub(100)

	bc.SetEventSink(&eventSinkAdapter{hub: wsHub})

	apiServer := httpapi.NewServer(bc, cfg.AIAuditorURL, mp, cfg.AdminToken, nil, cfg.TrustProxy, cfg.WSEnable, wsHub)

	handler := apiServer.Handler()

	if cfg.WSEnable {
		mux := http.NewServeMux()
		mux.Handle("/", handler)
		mux.HandleFunc("/ws", wsHub.ServeWS)
		handler = mux
	}

	srv := &http.Server{
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("API server listening on :%d", cfg.NodePort)
		if err := srv.Serve(nil); err != nil && err != http.ErrServerClosed {
			logger.Error("API server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down API service...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Shutdown error: %v", err)
	}

	mp.Stop()
	logger.Info("API service stopped")
}

type eventSinkAdapter struct {
	hub *websocket.Hub
}

func (a *eventSinkAdapter) Publish(e blockchain.WSEvent) {
	a.hub.Publish(websocket.WSEvent{
		Type: e.Type,
		Data: e.Data,
	})
}
