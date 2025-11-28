package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/audit"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/middleware"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/server"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/service"
	"github.com/telhawk-systems/telhawk-stack/common/config"
	"github.com/telhawk-systems/telhawk-stack/common/logging"
)

func main() {
	// Parse command line flags (for backward compatibility, not used)
	_ = flag.String("config", "", "path to config file (deprecated, use TELHAWK_CONFIG_DIR)")
	flag.Parse()

	// Load configuration using common config package
	config.MustLoad("authenticate")
	cfg := config.GetConfig()

	// Initialize structured logging
	logger := logging.New(
		logging.ParseLevel(cfg.Logging.Level),
		cfg.Logging.Format,
	).With(logging.Service("authenticate"))
	logging.SetDefault(logger)

	slog.Info("Starting Authenticate service",
		slog.Int("port", cfg.Authenticate.Server.Port),
		slog.String("log_level", cfg.Logging.Level),
		slog.String("log_format", cfg.Logging.Format),
	)

	// Initialize repository based on config
	var repo repository.Repository
	if cfg.Database.Type == "postgres" {
		connString := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.Authenticate.Database.Postgres.User,
			cfg.Authenticate.Database.Postgres.Password,
			cfg.Database.Postgres.Host,
			cfg.Database.Postgres.Port,
			cfg.Authenticate.Database.Postgres.Database,
			cfg.Database.Postgres.SSLMode,
		)

		slog.Info("Connecting to PostgreSQL",
			slog.String("host", cfg.Database.Postgres.Host),
			slog.Int("port", cfg.Database.Postgres.Port),
			slog.String("database", cfg.Authenticate.Database.Postgres.Database),
		)

		pgRepo, err := repository.NewPostgresRepository(context.Background(), connString)
		if err != nil {
			slog.Error("Failed to connect to PostgreSQL", slog.String("error", err.Error()))
			os.Exit(1)
		}
		defer pgRepo.Close()
		repo = pgRepo
		slog.Info("Connected to PostgreSQL")

		// Run database migrations
		slog.Info("Running database migrations")
		m, err := migrate.New(
			"file://migrations",
			connString,
		)
		if err != nil {
			slog.Error("Failed to initialize migrations", slog.String("error", err.Error()))
			os.Exit(1)
		}

		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			slog.Error("Failed to run migrations", slog.String("error", err.Error()))
			os.Exit(1)
		}

		version, dirty, err := m.Version()
		if err != nil {
			slog.Warn("Could not get migration version", slog.String("error", err.Error()))
		} else {
			slog.Info("Database migration complete",
				slog.Uint64("version", uint64(version)),
				slog.Bool("dirty", dirty),
			)
		}
	} else {
		slog.Warn("Using in-memory repository (development only)")
		repo = repository.NewInMemoryRepository()
	}

	// Initialize service layer
	var ingestClient *audit.IngestClient
	if cfg.Authenticate.Ingest.Enabled && cfg.Authenticate.Ingest.URL != "" && cfg.Authenticate.Ingest.HECToken != "" {
		slog.Info("Enabling auth event forwarding",
			slog.String("ingest_url", cfg.Authenticate.Ingest.URL),
		)
		ingestClient = audit.NewIngestClient(cfg.Authenticate.Ingest.URL, cfg.Authenticate.Ingest.HECToken)
	}

	authService := service.NewAuthService(repo, ingestClient)

	// Initialize HTTP handlers and middleware
	handler := handlers.NewAuthHandler(authService)
	authMiddleware := middleware.NewAuthMiddleware(authService)
	router := server.NewRouter(handler, authMiddleware)

	// Create server with config values
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Authenticate.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Authenticate.Server.ReadTimeoutDuration(),
		WriteTimeout: cfg.Authenticate.Server.WriteTimeoutDuration(),
		IdleTimeout:  cfg.Authenticate.Server.IdleTimeoutDuration(),
	}

	// Start server in goroutine
	go func() {
		slog.Info("Authenticate service listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.WriteTimeoutDuration())
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("Server stopped gracefully")
}
