package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rs/cors"
	"github.com/telhawk/web/internal/auth"
	"github.com/telhawk/web/internal/handlers"
	"github.com/telhawk/web/internal/middleware"
	"github.com/telhawk/web/internal/proxy"
	"github.com/telhawk/web/internal/server"
)

type Config struct {
	Port               string
	StaticDir          string
	AuthServiceURL     string
	QueryServiceURL    string
	CoreServiceURL     string
	RulesServiceURL    string
	AlertingServiceURL string
	CookieDomain       string
	CookieSecure       bool
	DevMode            bool
}

func loadConfig() *Config {
	cfg := &Config{
		Port:               getEnv("WEB_PORT", "3000"),
		StaticDir:          getEnv("STATIC_DIR", "./static"),
		AuthServiceURL:     getEnv("AUTH_SERVICE_URL", "http://auth:8080"),
		QueryServiceURL:    getEnv("QUERY_SERVICE_URL", "http://query:8082"),
		CoreServiceURL:     getEnv("CORE_SERVICE_URL", "http://core:8090"),
		RulesServiceURL:    getEnv("RULES_SERVICE_URL", "http://rules:8084"),
		AlertingServiceURL: getEnv("ALERTING_SERVICE_URL", "http://alerting:8085"),
		CookieDomain:       getEnv("COOKIE_DOMAIN", ""),
		CookieSecure:       getEnv("COOKIE_SECURE", "true") == "true",
		DevMode:            getEnv("DEV_MODE", "false") == "true",
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
	authProxy := proxy.NewProxy(cfg.AuthServiceURL, authClient)
	rulesProxy := proxy.NewProxy(cfg.RulesServiceURL, authClient)
	alertingProxy := proxy.NewProxy(cfg.AlertingServiceURL, authClient)

	authHandler := handlers.NewAuthHandler(authClient, cfg.CookieDomain, cfg.CookieSecure)
	dashboardHandler := handlers.NewDashboardHandler(cfg.QueryServiceURL)

	mux := server.NewRouter(server.RouterConfig{
		AuthHandler:      authHandler,
		DashboardHandler: dashboardHandler,
		AuthMiddleware:   authMiddleware,
		AuthProxy:        authProxy,
		QueryProxy:       queryProxy,
		CoreProxy:        coreProxy,
		RulesProxy:       rulesProxy,
		AlertingProxy:    alertingProxy,
		StaticDir:        cfg.StaticDir,
	})

	// CORS configuration
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, // Vite dev server
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
		Debug:            cfg.DevMode,
	})

	// Apply security middleware
	securityConfig := middleware.SecurityConfig{
		CookieSecure: cfg.CookieSecure,
	}

	// Chain middleware: CORS -> Security Headers -> CSRF -> Routes
	handler := corsHandler.Handler(mux)
	handler = middleware.SecurityHeaders(securityConfig)(handler)
	csrfMiddleware := middleware.CSRF(cfg.CookieSecure)
	handler = csrfMiddleware(handler)

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
	log.Printf("Rules Service: %s", cfg.RulesServiceURL)
	log.Printf("Alerting Service: %s", cfg.AlertingServiceURL)
	log.Printf("Static Dir: %s", cfg.StaticDir)
	log.Printf("Dev Mode: %v", cfg.DevMode)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
