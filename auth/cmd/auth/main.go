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

	"github.com/telhawk-systems/telhawk-stack/auth/internal/config"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/audit"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/service"
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

	log.Printf("Starting Auth service on port %d", cfg.Server.Port)
	if *configPath != "" {
		log.Printf("Loaded config from: %s", *configPath)
	}

	// Initialize repository based on config
	var repo repository.Repository
	if cfg.Database.Type == "postgres" {
		connString := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Database.Postgres.User,
			cfg.Database.Postgres.Password,
			cfg.Database.Postgres.Host,
			cfg.Database.Postgres.Port,
			cfg.Database.Postgres.Database,
			cfg.Database.Postgres.SSLMode,
		)
		
		log.Printf("Connecting to PostgreSQL at %s:%d/%s",
			cfg.Database.Postgres.Host,
			cfg.Database.Postgres.Port,
			cfg.Database.Postgres.Database,
		)
		
		pgRepo, err := repository.NewPostgresRepository(context.Background(), connString)
		if err != nil {
			log.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer pgRepo.Close()
		repo = pgRepo
		log.Println("Connected to PostgreSQL")
	} else {
		log.Println("Using in-memory repository (development only)")
		repo = repository.NewInMemoryRepository()
	}

	// Initialize service layer
	var ingestClient *audit.IngestClient
	if cfg.Ingest.Enabled && cfg.Ingest.URL != "" && cfg.Ingest.HECToken != "" {
		log.Printf("Enabling auth event forwarding to ingest service at %s", cfg.Ingest.URL)
		ingestClient = audit.NewIngestClient(cfg.Ingest.URL, cfg.Ingest.HECToken)
	}

	authService := service.NewAuthService(repo, ingestClient)

	// Initialize HTTP handlers
	handler := handlers.NewAuthHandler(authService)

	// Setup HTTP router
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/register", handler.Register)
	mux.HandleFunc("/api/v1/auth/login", handler.Login)
	mux.HandleFunc("/api/v1/auth/refresh", handler.RefreshToken)
	mux.HandleFunc("/api/v1/auth/validate", handler.ValidateToken)
	mux.HandleFunc("/api/v1/auth/validate-hec", handler.ValidateHECToken)
	mux.HandleFunc("/api/v1/auth/revoke", handler.RevokeToken)
	
	// User management endpoints
	mux.HandleFunc("/api/v1/users", handler.ListUsers)
	mux.HandleFunc("/api/v1/users/get", handler.GetUser)
	mux.HandleFunc("/api/v1/users/update", handler.UpdateUser)
	mux.HandleFunc("/api/v1/users/delete", handler.DeleteUser)
	mux.HandleFunc("/api/v1/users/reset-password", handler.ResetPassword)
	
	mux.HandleFunc("/healthz", handler.HealthCheck)

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
		log.Printf("Auth service listening on %s", srv.Addr)
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
