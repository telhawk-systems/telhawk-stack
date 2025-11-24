package server

import (
	"fmt"
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/common/middleware"

	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/auth"
	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/proxy"
)

// RouterConfig holds dependencies needed to configure routes
type RouterConfig struct {
	AuthHandler       *handlers.AuthHandler
	DashboardHandler  *handlers.DashboardHandler
	AsyncQueryHandler *handlers.AsyncQueryHandler // Optional: nil if NATS unavailable
	AuthMiddleware    *auth.Middleware
	AuthenticateProxy *proxy.Proxy
	SearchProxy       *proxy.Proxy
	CoreProxy         *proxy.Proxy
	RespondProxy      *proxy.Proxy // Handles rules, alerts, and cases (merged from rules + alerting)
	StaticDir         string
}

// NewRouter constructs a ServeMux with web backend routes registered.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	// Auth endpoints
	mux.HandleFunc("GET /api/auth/csrf-token", cfg.AuthHandler.GetCSRFToken)
	mux.HandleFunc("POST /api/auth/login", cfg.AuthHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", cfg.AuthHandler.Logout)
	mux.Handle("GET /api/auth/me", cfg.AuthMiddleware.Protect(http.HandlerFunc(cfg.AuthHandler.Me)))

	// Dashboard metrics endpoint with caching (protected)
	mux.Handle("GET /api/dashboard/metrics", cfg.AuthMiddleware.Protect(http.HandlerFunc(cfg.DashboardHandler.GetMetrics)))

	// Async query endpoints (protected, only if NATS is available)
	if cfg.AsyncQueryHandler != nil {
		mux.Handle("POST /api/async-query/submit", cfg.AuthMiddleware.Protect(http.HandlerFunc(cfg.AsyncQueryHandler.SubmitQuery)))
		mux.Handle("GET /api/async-query/status/{id}", cfg.AuthMiddleware.Protect(http.HandlerFunc(cfg.AsyncQueryHandler.GetQueryStatus)))
	}

	// User management endpoints (protected)
	mux.Handle("/api/auth/", cfg.AuthMiddleware.Protect(
		http.StripPrefix("/api/auth", cfg.AuthenticateProxy.Handler()),
	))

	// Search service proxy (protected)
	mux.Handle("/api/search/", cfg.AuthMiddleware.Protect(
		http.StripPrefix("/api/search", cfg.SearchProxy.Handler()),
	))

	// Core service proxy (protected)
	mux.Handle("/api/core/", cfg.AuthMiddleware.Protect(
		http.StripPrefix("/api/core", cfg.CoreProxy.Handler()),
	))

	// Respond service proxy (handles rules, alerts, cases) - protected
	// /api/rules/ -> respond service for detection schemas
	mux.Handle("/api/rules/", cfg.AuthMiddleware.Protect(
		http.StripPrefix("/api/rules", cfg.RespondProxy.Handler()),
	))
	// /api/alerting/ -> respond service for alerts and cases
	mux.Handle("/api/alerting/", cfg.AuthMiddleware.Protect(
		http.StripPrefix("/api/alerting", cfg.RespondProxy.Handler()),
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

	return middleware.RequestID(mux)
}
