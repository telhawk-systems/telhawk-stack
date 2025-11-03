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

	// Initialize repository (will be PostgreSQL in production)
	repo := repository.NewInMemoryRepository()

	// Initialize service layer
	authService := service.NewAuthService(repo)

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
