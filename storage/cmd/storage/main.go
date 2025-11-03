package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/telhawk-systems/telhawk-stack/storage/internal/client"
	"github.com/telhawk-systems/telhawk-stack/storage/internal/config"
	"github.com/telhawk-systems/telhawk-stack/storage/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/storage/internal/indexmgr"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting Storage service on port %d", cfg.Server.Port)
	if *configPath != "" {
		log.Printf("Loaded config from: %s", *configPath)
	}
	log.Printf("OpenSearch URL: %s", cfg.OpenSearch.URL)

	ctx := context.Background()

	osClient, err := client.NewOpenSearchClient(cfg.OpenSearch)
	if err != nil {
		log.Fatalf("Failed to create OpenSearch client: %v", err)
	}

	indexManager := indexmgr.NewIndexManager(osClient, cfg.IndexManagement)
	if err := indexManager.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize index manager: %v", err)
	}

	handler := handlers.NewStorageHandler(osClient, indexManager)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/ingest", handler.Ingest)
	mux.HandleFunc("/api/v1/bulk", handler.BulkIngest)
	mux.HandleFunc("/healthz", handler.Health)
	mux.HandleFunc("/readyz", handler.Ready)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Printf("Storage service listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.WriteTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
