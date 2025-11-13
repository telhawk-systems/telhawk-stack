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

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/ack"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/authclient"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/config"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/coreclient"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/ratelimit"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/server"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/service"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/storageclient"
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
	log.Printf("Storage URL: %s", cfg.Storage.URL)
	log.Printf("OpenSearch URL: %s", cfg.OpenSearch.URL)

	// Initialize rate limiter
	var rateLimiter ratelimit.RateLimiter
	if cfg.Redis.Enabled && cfg.Ingestion.RateLimitEnabled {
		log.Printf("Initializing Redis rate limiter: %s", cfg.Redis.URL)
		limiter, err := ratelimit.NewRedisRateLimiter(
			cfg.Redis.URL,
			cfg.Ingestion.RateLimitRequests,
			cfg.Ingestion.RateLimitWindow,
			false,
		)
		if err != nil {
			log.Printf("WARNING: Failed to initialize Redis rate limiter: %v", err)
			log.Println("Continuing without rate limiting")
			rateLimiter = &ratelimit.NoOpRateLimiter{}
		} else {
			rateLimiter = limiter
			log.Printf("Rate limiting enabled: %d requests per %s", cfg.Ingestion.RateLimitRequests, cfg.Ingestion.RateLimitWindow)
		}
	} else {
		rateLimiter = &ratelimit.NoOpRateLimiter{}
		if !cfg.Redis.Enabled {
			log.Println("Redis disabled - rate limiting not available")
		}
		if !cfg.Ingestion.RateLimitEnabled {
			log.Println("Rate limiting disabled in configuration")
		}
	}
	defer rateLimiter.Close()

	// Initialize ack manager
	var ackManager *ack.Manager
	if cfg.Ack.Enabled {
		ackManager = ack.NewManager(cfg.Ack.TTL)
		log.Printf("HEC acknowledgement channel enabled (TTL: %s)", cfg.Ack.TTL)
		defer ackManager.Close()
	} else {
		log.Println("HEC acknowledgement channel disabled")
	}

	// Initialize ingestion service
	authClient := authclient.New(cfg.Auth.URL, 5*time.Second, cfg.Auth.TokenValidationCacheTTL)
	coreClient := coreclient.New(cfg.Core.URL, 10*time.Second)
	storageClient := storageclient.New(cfg.Storage.URL, 30*time.Second)
	ingestService := service.NewIngestService(coreClient, storageClient, authClient)

	// Configure ack manager if enabled
	if ackManager != nil {
		ingestService.SetAckManager(ackManager)
	}

	// Initialize HTTP handlers
	handler := handlers.NewHECHandler(ingestService, rateLimiter)
	router := server.NewRouter(handler)

	// Create server with config values
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
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
