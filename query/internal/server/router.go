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
	// Events (JSON:API resource-centric endpoints)
	mux.HandleFunc("/api/v1/events", h.Events)
	mux.HandleFunc("/api/v1/events/", h.EventsByAction)
	mux.HandleFunc("/api/v1/alerts", h.Alerts)
	mux.HandleFunc("/api/v1/alerts/", h.AlertByID)
	mux.HandleFunc("/api/v1/dashboards", h.Dashboards)
	mux.HandleFunc("/api/v1/dashboards/", h.DashboardByID)
	mux.HandleFunc("/api/v1/saved-searches", h.SavedSearches)
	mux.HandleFunc("/api/v1/saved-searches/", h.SavedSearchByID)
	mux.HandleFunc("/api/v1/export", h.Export)
	mux.HandleFunc("/healthz", h.Health)
	return mux
}
