package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/telhawk-systems/telhawk-stack/common/logging"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/ack"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/authclient"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/config"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/dlq"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/normalizer"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/normalizer/generated"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/pipeline"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/ratelimit"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/server"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/service"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/storage"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/validator"
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

	// Initialize structured logging
	logger := logging.New(
		logging.ParseLevel(cfg.Logging.Level),
		cfg.Logging.Format,
	).With(logging.Service("ingest"))
	logging.SetDefault(logger)

	slog.Info("Starting Ingest service",
		slog.Int("port", cfg.Server.Port),
		slog.String("log_level", cfg.Logging.Level),
		slog.String("log_format", cfg.Logging.Format),
	)
	if *configPath != "" {
		slog.Info("Loaded configuration", slog.String("config_path", *configPath))
	}
	slog.Info("Service URLs configured",
		slog.String("authenticate_url", cfg.Authenticate.URL),
		slog.String("opensearch_url", cfg.OpenSearch.URL),
	)

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

	// Initialize Dead Letter Queue
	var dlqWriter *dlq.Queue
	if cfg.DLQ.Enabled {
		var err error
		dlqWriter, err = dlq.NewQueue(cfg.DLQ.BasePath)
		if err != nil {
			log.Fatalf("Failed to initialize DLQ: %v", err)
		}
		log.Printf("Dead Letter Queue enabled at: %s", cfg.DLQ.BasePath)
	} else {
		log.Println("Dead Letter Queue disabled")
	}

	// Initialize normalization pipeline
	// Create normalizer registry with OCSF passthrough (highest priority)
	normalizers := []normalizer.Normalizer{
		&normalizer.OCSFPassthroughNormalizer{},
	}

	// Add all generated normalizers (77 normalizers for OCSF event classes)
	normalizers = append(normalizers, generated.AllNormalizers()...)

	// Add HEC fallback normalizer (lowest priority)
	normalizers = append(normalizers, &normalizer.HECNormalizer{})

	normalizerRegistry := normalizer.NewRegistry(normalizers...)

	// Initialize validator chain with basic validator
	validators := []validator.Validator{
		&validator.BasicValidator{},
	}

	// Add all generated validators
	validators = append(validators, generated.AllValidators()...)

	validatorChain := validator.NewChain(validators...)

	// Create normalization pipeline
	normalizationPipeline := pipeline.New(normalizerRegistry, validatorChain)
	log.Printf("Normalization pipeline initialized with %d normalizers and %d validators", len(normalizers), len(validators))

	// Initialize clients
	authClient := authclient.New(cfg.Authenticate.URL, 5*time.Second, cfg.Authenticate.TokenValidationCacheTTL)

	// Initialize direct OpenSearch client (replaces storage service)
	openSearchConfig := storage.Config{
		URL:             cfg.OpenSearch.URL,
		Username:        cfg.OpenSearch.Username,
		Password:        cfg.OpenSearch.Password,
		TLSSkipVerify:   cfg.OpenSearch.TLSSkipVerify,
		IndexPrefix:     cfg.OpenSearch.IndexPrefix,
		ShardCount:      cfg.OpenSearch.ShardCount,
		ReplicaCount:    cfg.OpenSearch.ReplicaCount,
		RefreshInterval: cfg.OpenSearch.RefreshInterval,
		RetentionDays:   cfg.OpenSearch.RetentionDays,
		RolloverSizeGB:  cfg.OpenSearch.RolloverSizeGB,
		RolloverAge:     cfg.OpenSearch.RolloverAge,
	}

	storageClient, err := storage.NewClient(openSearchConfig)
	if err != nil {
		log.Fatalf("Failed to create OpenSearch client: %v", err)
	}

	// Initialize OpenSearch indices, templates, and policies
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	if err := storageClient.Initialize(ctx); err != nil {
		log.Printf("WARNING: Failed to initialize OpenSearch: %v", err)
		log.Println("Events may fail to index until OpenSearch is properly configured")
	}
	cancel()

	// Initialize ingestion service with pipeline
	ingestService := service.NewIngestService(normalizationPipeline, dlqWriter, storageClient, authClient)

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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.WriteTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
