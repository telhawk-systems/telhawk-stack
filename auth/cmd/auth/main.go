package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/audit"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/config"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/handlers"
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

		// Run database migrations
		log.Println("Running database migrations...")
		m, err := migrate.New(
			"file://migrations",
			connString,
		)
		if err != nil {
			log.Fatalf("Failed to initialize migrations: %v", err)
		}

		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		version, dirty, _ := m.Version()
		log.Printf("Database migration complete (version: %d, dirty: %v)", version, dirty)
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
	mux.HandleFunc("/api/v1/auth/login", handler.Login)
	mux.HandleFunc("/api/v1/auth/refresh", handler.RefreshToken)
	mux.HandleFunc("/api/v1/auth/validate", handler.ValidateToken)
	mux.HandleFunc("/api/v1/auth/validate-hec", handler.ValidateHECToken)
	mux.HandleFunc("/api/v1/auth/revoke", handler.RevokeToken)

	// User management endpoints (admin-only, requires authentication)
	// Use Go 1.22+ method routing for explicit path matching
	mux.HandleFunc("POST /api/v1/users/create", handler.CreateUser)
	mux.HandleFunc("GET /api/v1/users/get", handler.GetUser)
	mux.HandleFunc("PUT /api/v1/users/update", handler.UpdateUser)
	mux.HandleFunc("PATCH /api/v1/users/update", handler.UpdateUser)
	mux.HandleFunc("DELETE /api/v1/users/delete", handler.DeleteUser)
	mux.HandleFunc("POST /api/v1/users/reset-password", handler.ResetPassword)
	mux.HandleFunc("GET /api/v1/users", handler.ListUsers)

	// HEC token management endpoints
	mux.HandleFunc("/api/v1/hec/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.CreateHECToken(w, r)
		} else if r.Method == http.MethodGet {
			handler.ListHECTokens(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/hec/tokens/revoke", handler.RevokeHECTokenHandler)

	// RESTful endpoint for revoking specific token by ID: /api/v1/hec/tokens/{id}/revoke
	mux.HandleFunc("/api/v1/hec/tokens/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Check if path matches /api/v1/hec/tokens/{id}/revoke
		if strings.HasPrefix(path, "/api/v1/hec/tokens/") && strings.HasSuffix(path, "/revoke") {
			if r.Method == http.MethodDelete || r.Method == http.MethodPost {
				handler.RevokeHECTokenByIDHandler(w, r)
				return
			}
		}
		http.Error(w, "Not found", http.StatusNotFound)
	})

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
