package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rs/cors"
	"github.com/telhawk/web/internal/auth"
	"github.com/telhawk/web/internal/handlers"
	"github.com/telhawk/web/internal/proxy"
)

type Config struct {
	Port             string
	StaticDir        string
	AuthServiceURL   string
	QueryServiceURL  string
	CoreServiceURL   string
	CookieDomain     string
	CookieSecure     bool
	DevMode          bool
}

func loadConfig() *Config {
	cfg := &Config{
		Port:             getEnv("WEB_PORT", "3000"),
		StaticDir:        getEnv("STATIC_DIR", "./static"),
		AuthServiceURL:   getEnv("AUTH_SERVICE_URL", "http://auth:8080"),
		QueryServiceURL:  getEnv("QUERY_SERVICE_URL", "http://query:8082"),
		CoreServiceURL:   getEnv("CORE_SERVICE_URL", "http://core:8090"),
		CookieDomain:     getEnv("COOKIE_DOMAIN", ""),
		CookieSecure:     getEnv("COOKIE_SECURE", "true") == "true",
		DevMode:          getEnv("DEV_MODE", "false") == "true",
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	if *configPath != "" {
		log.Printf("Config file support not yet implemented, using env vars")
	}

	cfg := loadConfig()

	authClient := auth.NewClient(cfg.AuthServiceURL)
	authMiddleware := auth.NewMiddleware(authClient, cfg.CookieDomain, cfg.CookieSecure)

	queryProxy := proxy.NewProxy(cfg.QueryServiceURL, authClient)
	coreProxy := proxy.NewProxy(cfg.CoreServiceURL, authClient)

	mux := http.NewServeMux()

	// Auth endpoints
	authHandler := handlers.NewAuthHandler(authClient, cfg.CookieDomain, cfg.CookieSecure)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.Handle("GET /api/auth/me", authMiddleware.Protect(http.HandlerFunc(authHandler.Me)))

	// Query service proxy (protected)
	mux.Handle("/api/query/", authMiddleware.Protect(
		http.StripPrefix("/api/query", queryProxy.Handler()),
	))

	// Core service proxy (protected)
	mux.Handle("/api/core/", authMiddleware.Protect(
		http.StripPrefix("/api/core", coreProxy.Handler()),
	))

	// Health check
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","service":"web"}`)
	})

	// Serve React static files (must be last)
	fs := http.FileServer(http.Dir(cfg.StaticDir))
	mux.Handle("/", handlers.NewSPAHandler(cfg.StaticDir, fs))

	// CORS configuration
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, // Vite dev server
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
		Debug:            cfg.DevMode,
	})

	handler := corsHandler.Handler(mux)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("TelHawk Web UI starting on :%s", cfg.Port)
	log.Printf("Auth Service: %s", cfg.AuthServiceURL)
	log.Printf("Query Service: %s", cfg.QueryServiceURL)
	log.Printf("Static Dir: %s", cfg.StaticDir)
	log.Printf("Dev Mode: %v", cfg.DevMode)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
