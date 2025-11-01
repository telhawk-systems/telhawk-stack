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
	"time"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/config"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/coreclient"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/service"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting Ingest service on port %d", cfg.Server.Port)
	if *configPath != "" {
		log.Printf("Loaded config from: %s", *configPath)
	}
	log.Printf("Auth URL: %s", cfg.Auth.URL)
	log.Printf("Core URL: %s", cfg.Core.URL)
	log.Printf("OpenSearch URL: %s", cfg.OpenSearch.URL)

	// Initialize ingestion service
	coreClient := coreclient.New(cfg.Core.URL, 10*time.Second)
	ingestService := service.NewIngestService(coreClient)

	// Initialize HTTP handlers
	handler := handlers.NewHECHandler(ingestService)

	// Setup HTTP router
	mux := http.NewServeMux()

	// Splunk HEC endpoints
	mux.HandleFunc("/services/collector/event", handler.HandleEvent)
	mux.HandleFunc("/services/collector/raw", handler.HandleRaw)
	mux.HandleFunc("/services/collector/health", handler.Health)
	mux.HandleFunc("/services/collector/ack", handler.Ack)

	// Health endpoints
	mux.HandleFunc("/healthz", handler.Health)
	mux.HandleFunc("/readyz", handler.Ready)

	// Create server with config values
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Ingest service listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
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
