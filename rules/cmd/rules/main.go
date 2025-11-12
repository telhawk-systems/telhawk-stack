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

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/config"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/service"
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

	// Setup HTTP router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/healthz", handler.HealthCheck)

	// API routes (proxied from /api/rules/schemas via web backend)
	mux.HandleFunc("/schemas", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.CreateSchema(w, r)
		} else if r.Method == http.MethodGet {
			handler.ListSchemas(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Note: These are simplified routes. In production, use a proper router like chi or gorilla/mux
	mux.HandleFunc("/schemas/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// GET /schemas/:id/versions
		if len(path) > len("/versions") && path[len(path)-len("/versions"):] == "/versions" {
			handler.GetVersionHistory(w, r)
			// PUT /schemas/:id/disable
		} else if len(path) > len("/disable") && path[len(path)-len("/disable"):] == "/disable" {
			handler.DisableSchema(w, r)
			// PUT /schemas/:id/enable
		} else if len(path) > len("/enable") && path[len(path)-len("/enable"):] == "/enable" {
			handler.EnableSchema(w, r)
			// DELETE /schemas/:id
		} else if r.Method == http.MethodDelete {
			handler.HideSchema(w, r)
			// PUT /schemas/:id (update = create new version)
		} else if r.Method == http.MethodPut {
			handler.UpdateSchema(w, r)
			// GET /schemas/:id
		} else if r.Method == http.MethodGet {
			handler.GetSchema(w, r)
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
		log.Printf("Rules service listening on %s", srv.Addr)
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
