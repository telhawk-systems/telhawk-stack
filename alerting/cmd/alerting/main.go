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

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/config"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/evaluator"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/service"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Build PostgreSQL connection string
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.Postgres.User,
		cfg.Database.Postgres.Password,
		cfg.Database.Postgres.Host,
		cfg.Database.Postgres.Port,
		cfg.Database.Postgres.Database,
		cfg.Database.Postgres.SSLMode,
	)

	// Run database migrations
	log.Println("Running database migrations...")
	m, err := migrate.New("file://migrations", connString)
	if err != nil {
		log.Fatalf("Failed to initialize migrations: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Initialize repository
	repo, err := repository.NewPostgresRepository(context.Background(), connString)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer repo.Close()

	// Initialize service
	svc := service.NewService(repo)

	// Initialize handlers
	handler := handlers.NewHandler(svc)

	// Initialize evaluation engine
	rulesClient := evaluator.NewHTTPRulesClient("http://rules:8084")
	storageClient := evaluator.NewHTTPStorageClient(
		cfg.Storage.URL,
		cfg.Storage.Username,
		cfg.Storage.Password,
		cfg.Storage.Insecure,
	)
	eval := evaluator.NewEvaluator(rulesClient, storageClient)

	// Start evaluation engine in background
	evalCtx, evalCancel := context.WithCancel(context.Background())
	defer evalCancel()
	go eval.Run(evalCtx, 1*time.Minute) // Evaluate every minute

	// Setup HTTP router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/healthz", handler.HealthCheck)

	// API routes
	mux.HandleFunc("/api/v1/cases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.CreateCase(w, r)
		} else if r.Method == http.MethodGet {
			handler.ListCases(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Note: These are simplified routes. In production, use a proper router like chi or gorilla/mux
	mux.HandleFunc("/api/v1/cases/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// POST /api/v1/cases/:id/alerts
		if r.Method == http.MethodPost && len(path) > len("/alerts") && path[len(path)-len("/alerts"):] == "/alerts" {
			handler.AddAlertsToCase(w, r)
			// GET /api/v1/cases/:id/alerts
		} else if r.Method == http.MethodGet && len(path) > len("/alerts") && path[len(path)-len("/alerts"):] == "/alerts" {
			handler.GetCaseAlerts(w, r)
			// PUT /api/v1/cases/:id/close
		} else if len(path) > len("/close") && path[len(path)-len("/close"):] == "/close" {
			handler.CloseCase(w, r)
			// PUT /api/v1/cases/:id/reopen
		} else if len(path) > len("/reopen") && path[len(path)-len("/reopen"):] == "/reopen" {
			handler.ReopenCase(w, r)
			// PUT /api/v1/cases/:id
		} else if r.Method == http.MethodPut {
			handler.UpdateCase(w, r)
			// GET /api/v1/cases/:id
		} else if r.Method == http.MethodGet {
			handler.GetCase(w, r)
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Alerting service listening on %s", srv.Addr)
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

	log.Println("Server stopped gracefully")
}
