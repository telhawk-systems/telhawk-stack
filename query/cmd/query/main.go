package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	qauth "github.com/telhawk-systems/telhawk-stack/query/internal/auth"
	"github.com/telhawk-systems/telhawk-stack/query/internal/client"
	"github.com/telhawk-systems/telhawk-stack/query/internal/config"
	"github.com/telhawk-systems/telhawk-stack/query/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/query/internal/notification"
	"github.com/telhawk-systems/telhawk-stack/query/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/query/internal/scheduler"
	"github.com/telhawk-systems/telhawk-stack/query/internal/server"
	"github.com/telhawk-systems/telhawk-stack/query/internal/service"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config file")
	addr := flag.String("addr", "", "override listen address")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	listenAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	if *addr != "" {
		listenAddr = *addr
	}

	osClient, err := client.NewOpenSearchClient(cfg.OpenSearch)
	if err != nil {
		log.Fatalf("create opensearch client: %v", err)
	}
	log.Printf("connected to opensearch at %s", cfg.OpenSearch.URL)

	// Run DB migrations if configured
	if cfg.DatabaseURL != "" {
		log.Println("running database migrations...")
		m, err := migrate.New("file://migrations", cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("init migrations: %v", err)
		}
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("run migrations: %v", err)
		}
		log.Println("database migrations completed")
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
	authClient := qauth.NewClient(cfg.AuthURL)

	svc := service.NewQueryService("0.1.0", osClient).WithDependencies(repo, authClient)
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
		log.Printf("query service listening on %s", listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-shutdownCtx.Done()
	log.Println("shutdown signal received")

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
