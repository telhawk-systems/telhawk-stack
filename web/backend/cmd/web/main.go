package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/telhawk-systems/telhawk-stack/common/config"
	"github.com/telhawk-systems/telhawk-stack/common/hecstats"
	"github.com/telhawk-systems/telhawk-stack/common/messaging"
	"github.com/telhawk-systems/telhawk-stack/common/messaging/nats"
	"github.com/telhawk-systems/telhawk-stack/common/middleware"
	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/auth"
	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/handlers"
	webmiddleware "github.com/telhawk-systems/telhawk-stack/web/backend/internal/middleware"
	webnats "github.com/telhawk-systems/telhawk-stack/web/backend/internal/nats"
	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/proxy"
	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/server"
)

type Config struct {
	Port                   string
	StaticDir              string
	AuthenticateServiceURL string
	SearchServiceURL       string
	CoreServiceURL         string
	RespondServiceURL      string // Handles rules, alerts, and cases (merged from rules + alerting)
	CookieDomain           string
	CookieSecure           bool
	DevMode                bool
	NATSURL                string
	RedisURL               string
}

func loadConfig() *Config {
	cfg := &Config{
		Port:                   getEnv("WEB_PORT", "3000"),
		StaticDir:              getEnv("STATIC_DIR", "./static"),
		AuthenticateServiceURL: getEnv("AUTHENTICATE_SERVICE_URL", "http://authenticate:8080"),
		SearchServiceURL:       getEnv("SEARCH_SERVICE_URL", "http://search:8082"),
		CoreServiceURL:         getEnv("CORE_SERVICE_URL", "http://core:8090"),
		RespondServiceURL:      getEnv("RESPOND_SERVICE_URL", "http://respond:8086"),
		CookieDomain:           getEnv("COOKIE_DOMAIN", ""),
		CookieSecure:           getEnv("COOKIE_SECURE", "true") == "true",
		DevMode:                getEnv("DEV_MODE", "false") == "true",
		NATSURL:                getEnv("NATS_URL", "nats://nats:4222"),
		RedisURL:               getEnv("REDIS_URL", "redis://redis:6379"),
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
	// Parse command line flags (for backward compatibility, not used)
	_ = flag.String("config", "", "Path to config file (deprecated, use TELHAWK_CONFIG_DIR)")
	flag.Parse()

	// Load configuration using common config package
	config.MustLoad("web")
	globalCfg := config.GetConfig()

	// Load web-specific config from environment (until WebConfig is fully implemented)
	cfg := loadConfig()

	authClient := auth.NewClient(cfg.AuthenticateServiceURL)
	authMiddleware := auth.NewMiddleware(authClient, cfg.CookieDomain, cfg.CookieSecure)

	searchProxy := proxy.NewProxy(cfg.SearchServiceURL, authClient)
	coreProxy := proxy.NewProxy(cfg.CoreServiceURL, authClient)
	authenticateProxy := proxy.NewProxy(cfg.AuthenticateServiceURL, authClient)
	respondProxy := proxy.NewProxy(cfg.RespondServiceURL, authClient)

	authHandler := handlers.NewAuthHandler(authClient, cfg.CookieDomain, cfg.CookieSecure)
	dashboardHandler := handlers.NewDashboardHandler(cfg.SearchServiceURL, cfg.RespondServiceURL)

	// Initialize Redis client for HEC stats (read-only for web backend)
	// Use Redis URL from common config if available, otherwise fall back to env var
	var hecStatsHandler *handlers.HECStatsHandler
	redisURL := globalCfg.Redis.URL
	if redisURL == "" {
		redisURL = cfg.RedisURL
	}
	if redisURL != "" && globalCfg.Redis.Enabled {
		statsClient, err := hecstats.NewClient(redisURL, "web-backend")
		if err != nil {
			log.Printf("Warning: Failed to connect to Redis at %s: %v", redisURL, err)
			log.Printf("HEC token stats will be unavailable")
		} else {
			log.Printf("Connected to Redis for HEC stats")
			hecStatsHandler = handlers.NewHECStatsHandler(statsClient)
		}
	}

	// Initialize NATS client for async query support
	// Use NATS URL from common config if available, otherwise fall back to env var
	var natsClient messaging.Client
	var asyncQueryHandler *handlers.AsyncQueryHandler
	var resultSubscriber *webnats.ResultSubscriber

	natsURL := globalCfg.NATS.URL
	if natsURL == "" {
		natsURL = cfg.NATSURL
	}
	if natsURL != "" && globalCfg.NATS.Enabled {
		natsCfg := nats.DefaultConfig()
		natsCfg.URL = natsURL
		natsCfg.Name = "telhawk-web"

		var err error
		natsClient, err = nats.NewClient(natsCfg)
		if err != nil {
			log.Printf("Warning: Failed to connect to NATS at %s: %v", natsURL, err)
			log.Printf("Async query support will be disabled")
		} else {
			log.Printf("Connected to NATS at %s", natsURL)
			asyncQueryHandler = handlers.NewAsyncQueryHandler(natsClient, messaging.SubjectSearchJobsQuery)

			// Start result subscriber to receive search results
			resultSubscriber = webnats.NewResultSubscriber(natsClient, asyncQueryHandler)
			if err := resultSubscriber.Start(); err != nil {
				log.Printf("Warning: Failed to start result subscriber: %v", err)
			} else {
				log.Printf("Started search result subscriber")
			}
		}
	}

	mux := server.NewRouter(server.RouterConfig{
		AuthHandler:       authHandler,
		DashboardHandler:  dashboardHandler,
		AsyncQueryHandler: asyncQueryHandler,
		HECStatsHandler:   hecStatsHandler,
		AuthMiddleware:    authMiddleware,
		AuthenticateProxy: authenticateProxy,
		SearchProxy:       searchProxy,
		CoreProxy:         coreProxy,
		RespondProxy:      respondProxy,
		StaticDir:         cfg.StaticDir,
	})

	// CORS configuration
	corsConfig := middleware.CORSConfig{
		AllowedOrigins:   []string{"http://localhost:5173"}, // Vite dev server
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}

	// Apply security middleware
	securityConfig := webmiddleware.SecurityConfig{
		CookieSecure: cfg.CookieSecure,
	}

	// Chain middleware: CORS -> Security Headers -> CSRF -> Routes
	handler := middleware.CORS(corsConfig)(mux)
	handler = webmiddleware.SecurityHeaders(securityConfig)(handler)
	csrfMiddleware := webmiddleware.CSRF(cfg.CookieSecure)
	handler = csrfMiddleware(handler)

	// Use server timeouts from common config if available, otherwise use defaults
	readTimeout := globalCfg.Server.ReadTimeoutDuration()
	if readTimeout == 0 {
		readTimeout = 15 * time.Second
	}
	writeTimeout := globalCfg.Server.WriteTimeoutDuration()
	if writeTimeout == 0 {
		writeTimeout = 15 * time.Second
	}
	idleTimeout := globalCfg.Server.IdleTimeoutDuration()
	if idleTimeout == 0 {
		idleTimeout = 60 * time.Second
	}

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	log.Printf("TelHawk Web UI starting on :%s", cfg.Port)
	log.Printf("Authenticate Service: %s", cfg.AuthenticateServiceURL)
	log.Printf("Search Service: %s", cfg.SearchServiceURL)
	log.Printf("Respond Service: %s", cfg.RespondServiceURL)
	log.Printf("Static Dir: %s", cfg.StaticDir)
	log.Printf("Dev Mode: %v", cfg.DevMode)
	log.Printf("NATS URL: %s (async queries: %v)", cfg.NATSURL, natsClient != nil)

	// Ensure NATS resources are cleaned up on shutdown
	if resultSubscriber != nil {
		defer func() {
			if err := resultSubscriber.Stop(); err != nil {
				log.Printf("Error stopping result subscriber: %v", err)
			}
		}()
	}
	if natsClient != nil {
		defer func() {
			if err := natsClient.Drain(); err != nil {
				log.Printf("Error draining NATS connection: %v", err)
			}
		}()
	}

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
