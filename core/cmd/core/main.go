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

	"github.com/telhawk-systems/telhawk-stack/core/internal/config"
	"github.com/telhawk-systems/telhawk-stack/core/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer"
	"github.com/telhawk-systems/telhawk-stack/core/internal/pipeline"
	"github.com/telhawk-systems/telhawk-stack/core/internal/server"
	"github.com/telhawk-systems/telhawk-stack/core/internal/service"
	"github.com/telhawk-systems/telhawk-stack/core/internal/validator"
)

func main() {
	configPath := flag.String("config", "", "path to JSON config file")
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

	registry := normalizer.NewRegistry(normalizer.HECNormalizer{})
	validators := validator.NewChain(validator.BasicValidator{})
	pipe := pipeline.New(registry, validators)
	processor := service.NewProcessor(pipe)
	handler := handlers.NewProcessorHandler(processor)

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      server.NewRouter(handler),
		ReadTimeout:  cfg.Server.ReadTimeout(),
		WriteTimeout: cfg.Server.WriteTimeout(),
		IdleTimeout:  cfg.Server.IdleTimeout(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("core service listening on %s", listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
