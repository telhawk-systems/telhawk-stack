package server

import (
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/query/internal/handlers"
)

// NewRouter constructs a ServeMux with query API routes registered.
func NewRouter(h *handlers.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/search", h.Search)
	mux.HandleFunc("/api/v1/query", h.Query)
	mux.HandleFunc("/api/v1/alerts", h.Alerts)
	mux.HandleFunc("/api/v1/alerts/", h.AlertByID)
	mux.HandleFunc("/api/v1/dashboards", h.Dashboards)
	mux.HandleFunc("/api/v1/dashboards/", h.DashboardByID)
	mux.HandleFunc("/api/v1/export", h.Export)
	mux.HandleFunc("/healthz", h.Health)
	return mux
}
