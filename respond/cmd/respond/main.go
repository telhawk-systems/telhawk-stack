package main

import (
	"context"
	"errors"
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
	natsclient "github.com/telhawk-systems/telhawk-stack/common/messaging/nats"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/config"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/handlers"
	respondnats "github.com/telhawk-systems/telhawk-stack/respond/internal/nats"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/scheduler"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/server"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/service"
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
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Initialize repository
	repo, err := repository.NewPostgresRepository(context.Background(), connString)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer repo.Close()

	// Initialize service layer
	svc := service.NewService(repo)

	// TODO: Initialize Redis for state management (correlation state, suppression cache)
	// TODO: Initialize query executor for correlation rules (OpenSearch client)
	// TODO: Initialize evaluation engine (correlation rule evaluator)
	// TODO: Initialize rule importer (load rules from alerting/dist/rules/)

	// Initialize NATS client (optional - service works without it)
	var natsClient *natsclient.Client
	var natsPublisher *respondnats.Publisher
	var natsHandler *respondnats.Handler
	var correlationScheduler *scheduler.Scheduler

	if cfg.NATS.Enabled {
		natsCfg := natsclient.Config{
			URL:           cfg.NATS.URL,
			Name:          "respond-service",
			MaxReconnects: cfg.NATS.MaxReconnects,
			ReconnectWait: cfg.NATS.ReconnectWait,
		}

		var err error
		natsClient, err = natsclient.NewClient(natsCfg)
		if err != nil {
			log.Printf("Warning: Failed to connect to NATS: %v (continuing without NATS)", err)
		} else {
			log.Printf("Connected to NATS at %s", cfg.NATS.URL)

			// Create publisher
			natsPublisher = respondnats.NewPublisher(natsClient)

			// Create and start handler
			natsHandler = respondnats.NewHandler(natsClient, repo, natsPublisher)
			if err := natsHandler.Start(context.Background()); err != nil {
				log.Printf("Warning: Failed to start NATS handler: %v", err)
				natsHandler = nil
			}

			// Start correlation scheduler (every 1 minute)
			correlationScheduler = scheduler.NewScheduler(repo, natsPublisher, 1*time.Minute)
			go correlationScheduler.Start(context.Background())
		}
	} else {
		log.Println("NATS is disabled, running in HTTP-only mode")
	}

	// Initialize handlers with service
	handler := handlers.NewHandler(svc)

	// Setup HTTP router
	router := server.NewRouter(handler)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Respond service listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Stop NATS components first
	if correlationScheduler != nil {
		log.Println("Stopping correlation scheduler...")
		correlationScheduler.Stop()
	}
	if natsHandler != nil {
		log.Println("Stopping NATS handler...")
		if err := natsHandler.Stop(); err != nil {
			log.Printf("Warning: Error stopping NATS handler: %v", err)
		}
	}
	if natsClient != nil {
		log.Println("Closing NATS connection...")
		if err := natsClient.Close(); err != nil {
			log.Printf("Warning: Error closing NATS connection: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.WriteTimeout)

	if err := srv.Shutdown(ctx); err != nil {
		cancel()
		repo.Close()                                     //nolint:errcheck // closing on fatal
		log.Fatalf("Server forced to shutdown: %v", err) //nolint:gocritic // repo.Close() called explicitly above
	}
	cancel()

	log.Println("Server stopped gracefully")
}
