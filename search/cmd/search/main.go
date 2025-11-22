package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/telhawk-systems/telhawk-stack/common/logging"
	natsclient "github.com/telhawk-systems/telhawk-stack/common/messaging/nats"
	sauth "github.com/telhawk-systems/telhawk-stack/search/internal/auth"
	"github.com/telhawk-systems/telhawk-stack/search/internal/client"
	"github.com/telhawk-systems/telhawk-stack/search/internal/config"
	"github.com/telhawk-systems/telhawk-stack/search/internal/handlers"
	searchnats "github.com/telhawk-systems/telhawk-stack/search/internal/nats"
	"github.com/telhawk-systems/telhawk-stack/search/internal/notification"
	"github.com/telhawk-systems/telhawk-stack/search/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/search/internal/scheduler"
	"github.com/telhawk-systems/telhawk-stack/search/internal/server"
	"github.com/telhawk-systems/telhawk-stack/search/internal/service"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config file")
	addr := flag.String("addr", "", "override listen address")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Initialize structured logging
	logger := logging.New(
		logging.ParseLevel(cfg.Logging.Level),
		cfg.Logging.Format,
	).With(logging.Service("search"))
	logging.SetDefault(logger)

	slog.Info("Starting Search service",
		slog.Int("port", cfg.Server.Port),
		slog.String("log_level", cfg.Logging.Level),
		slog.String("log_format", cfg.Logging.Format),
	)

	listenAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	if *addr != "" {
		listenAddr = *addr
	}

	osClient, err := client.NewOpenSearchClient(cfg.OpenSearch)
	if err != nil {
		slog.Error("Failed to create OpenSearch client", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("Connected to OpenSearch", slog.String("url", cfg.OpenSearch.URL))

	// Run DB migrations if configured
	if cfg.DatabaseURL != "" {
		slog.Info("Running database migrations")
		m, err := migrate.New("file://migrations", cfg.DatabaseURL)
		if err != nil {
			slog.Error("Failed to initialize migrations", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			slog.Error("Failed to run migrations", slog.String("error", err.Error()))
			os.Exit(1)
		}
		slog.Info("Database migrations completed")
	}

	// Initialize repo + auth client
	var repo *repository.PostgresRepository
	if cfg.DatabaseURL != "" {
		repo, err = repository.NewPostgresRepository(context.Background(), cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("connect postgres: %v", err)
		}
		defer repo.Close()
	}
	authClient := sauth.NewClient(cfg.AuthURL)

	svc := service.NewSearchService("0.1.0", osClient).WithDependencies(repo, authClient)
	h := handlers.New(svc)

	var alertScheduler *scheduler.Scheduler
	schedulerCtx, schedulerStop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	if cfg.Alerting.Enabled {
		notifChannel := buildNotificationChannel(cfg)
		schedulerCfg := scheduler.Config{
			CheckInterval: time.Duration(cfg.Alerting.CheckIntervalSeconds) * time.Second,
		}
		alertScheduler = scheduler.NewScheduler(svc, svc, notifChannel, schedulerCfg)

		if err := alertScheduler.Start(schedulerCtx); err != nil {
			log.Fatalf("failed to start alert scheduler: %v", err)
		}
		h.WithScheduler(alertScheduler)
		log.Printf("alert scheduler enabled")
	} else {
		log.Printf("alert scheduler disabled")
	}

	// Initialize NATS client (optional - service works without it)
	var natsHandler *searchnats.Handler
	if cfg.NATS.Enabled {
		natsCfg := natsclient.Config{
			URL:           cfg.NATS.URL,
			Name:          "search-service",
			MaxReconnects: cfg.NATS.MaxReconnects,
			ReconnectWait: cfg.NATS.ReconnectWaitDuration(),
			Timeout:       5 * time.Second,
		}

		natsClient, err := natsclient.NewClient(natsCfg)
		if err != nil {
			slog.Warn("Failed to connect to NATS (continuing without NATS)",
				slog.String("url", cfg.NATS.URL),
				slog.String("error", err.Error()))
		} else {
			slog.Info("Connected to NATS", slog.String("url", cfg.NATS.URL))

			natsHandler = searchnats.NewHandler(natsClient, svc)
			if err := natsHandler.Start(context.Background()); err != nil {
				slog.Warn("Failed to start NATS handler",
					slog.String("error", err.Error()))
				natsClient.Close()
				natsHandler = nil
			} else {
				// Set NATS handler on HTTP handler for health checks
				h.WithNATSHandler(natsHandler)
			}
		}
	} else {
		slog.Info("NATS messaging disabled")
	}

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      server.NewRouter(h),
		ReadTimeout:  cfg.Server.ReadTimeout(),
		WriteTimeout: cfg.Server.WriteTimeout(),
		IdleTimeout:  cfg.Server.IdleTimeout(),
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer schedulerStop()

	go func() {
		log.Printf("search service listening on %s", listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-shutdownCtx.Done()
	log.Println("shutdown signal received")

	// Stop NATS handler first
	if natsHandler != nil {
		log.Println("stopping NATS handler")
		if err := natsHandler.Stop(); err != nil {
			log.Printf("NATS handler shutdown error: %v", err)
		}
		// Close NATS client
		if natsClient := natsHandler.Client(); natsClient != nil {
			natsClient.Close()
		}
	}

	if alertScheduler != nil {
		log.Println("stopping alert scheduler")
		if err := alertScheduler.Stop(); err != nil {
			log.Printf("alert scheduler shutdown error: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func buildNotificationChannel(cfg *config.Config) notification.Channel {
	channels := []notification.Channel{
		notification.NewLogChannel(log.Printf),
	}

	timeout := time.Duration(cfg.Alerting.NotificationTimeout) * time.Second

	if cfg.Alerting.WebhookURL != "" {
		channels = append(channels, notification.NewWebhookChannel(cfg.Alerting.WebhookURL, timeout))
		log.Printf("webhook notifications enabled: %s", cfg.Alerting.WebhookURL)
	}

	if cfg.Alerting.SlackWebhookURL != "" {
		channels = append(channels, notification.NewSlackChannel(cfg.Alerting.SlackWebhookURL, timeout))
		log.Printf("slack notifications enabled")
	}

	if len(channels) == 1 {
		return channels[0]
	}

	return notification.NewMultiChannel(channels...)
}
