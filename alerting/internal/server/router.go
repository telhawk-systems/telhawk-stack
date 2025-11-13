package server

import (
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/alerting/internal/handlers"
)

// NewRouter constructs a ServeMux with alerting API routes registered.
func NewRouter(h *handlers.Handler) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/healthz", h.HealthCheck)

	// Alerts API routes
	mux.HandleFunc("/api/v1/alerts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.ListAlerts(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/alerts/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetAlert(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Cases API routes
	mux.HandleFunc("/api/v1/cases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.CreateCase(w, r)
		} else if r.Method == http.MethodGet {
			h.ListCases(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Note: These are simplified routes. In production, use a proper router like chi or gorilla/mux
	mux.HandleFunc("/api/v1/cases/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// POST /api/v1/cases/:id/alerts
		if r.Method == http.MethodPost && len(path) > len("/alerts") && path[len(path)-len("/alerts"):] == "/alerts" {
			h.AddAlertsToCase(w, r)
			// GET /api/v1/cases/:id/alerts
		} else if r.Method == http.MethodGet && len(path) > len("/alerts") && path[len(path)-len("/alerts"):] == "/alerts" {
			h.GetCaseAlerts(w, r)
			// PUT /api/v1/cases/:id/close
		} else if len(path) > len("/close") && path[len(path)-len("/close"):] == "/close" {
			h.CloseCase(w, r)
			// PUT /api/v1/cases/:id/reopen
		} else if len(path) > len("/reopen") && path[len(path)-len("/reopen"):] == "/reopen" {
			h.ReopenCase(w, r)
			// PUT /api/v1/cases/:id
		} else if r.Method == http.MethodPut {
			h.UpdateCase(w, r)
			// GET /api/v1/cases/:id
		} else if r.Method == http.MethodGet {
			h.GetCase(w, r)
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	})

	return mux
}
