package server

import (
	"fmt"
	"github.com/telhawk-systems/telhawk-stack/common/middleware"
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/auth"
	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/proxy"
)

// RouterConfig holds dependencies needed to configure routes
type RouterConfig struct {
	AuthHandler      *handlers.AuthHandler
	DashboardHandler *handlers.DashboardHandler
	AuthMiddleware   *auth.Middleware
	AuthProxy        *proxy.Proxy
	QueryProxy       *proxy.Proxy
	CoreProxy        *proxy.Proxy
	RulesProxy       *proxy.Proxy
	AlertingProxy    *proxy.Proxy
	StaticDir        string
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

	// User management endpoints (protected)
	mux.Handle("/api/auth/", cfg.AuthMiddleware.Protect(
		http.StripPrefix("/api/auth", cfg.AuthProxy.Handler()),
	))

	// Query service proxy (protected)
	mux.Handle("/api/query/", cfg.AuthMiddleware.Protect(
		http.StripPrefix("/api/query", cfg.QueryProxy.Handler()),
	))

	// Core service proxy (protected)
	mux.Handle("/api/core/", cfg.AuthMiddleware.Protect(
		http.StripPrefix("/api/core", cfg.CoreProxy.Handler()),
	))

	// Rules service proxy (protected)
	mux.Handle("/api/rules/", cfg.AuthMiddleware.Protect(
		http.StripPrefix("/api/rules", cfg.RulesProxy.Handler()),
	))

	// Alerting service proxy (protected)
	mux.Handle("/api/alerting/", cfg.AuthMiddleware.Protect(
		http.StripPrefix("/api/alerting", cfg.AlertingProxy.Handler()),
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
