package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/service"
)

func main() {
	// Initialize ingestion service
	ingestService := service.NewIngestService()

	// Initialize HTTP handlers
	handler := handlers.NewHECHandler(ingestService)

	// Setup HTTP router
	mux := http.NewServeMux()
	
	// Splunk HEC endpoints
	mux.HandleFunc("/services/collector/event", handler.HandleEvent)
	mux.HandleFunc("/services/collector/raw", handler.HandleRaw)
	mux.HandleFunc("/services/collector/health", handler.Health)
	mux.HandleFunc("/services/collector/ack", handler.Ack)
	
	// Health endpoints
	mux.HandleFunc("/healthz", handler.Health)
	mux.HandleFunc("/readyz", handler.Ready)

	// Create server
	srv := &http.Server{
		Addr:         ":8088",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Ingest service starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
