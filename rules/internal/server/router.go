package server

import (
	"github.com/telhawk-systems/telhawk-stack/common/middleware"
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/rules/internal/handlers"
)

// NewRouter constructs a ServeMux with rules API routes registered.
func NewRouter(h *handlers.Handler) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/healthz", h.HealthCheck)

	// Correlation types API
	mux.HandleFunc("/correlation/types", h.GetCorrelationTypes)

	// API routes (proxied from /api/rules/schemas via web backend)
	mux.HandleFunc("/schemas", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.CreateSchema(w, r)
		} else if r.Method == http.MethodGet {
			h.ListSchemas(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Note: These are simplified routes. In production, use a proper router like chi or gorilla/mux
	mux.HandleFunc("/schemas/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// GET /schemas/:id/versions
		if len(path) > len("/versions") && path[len(path)-len("/versions"):] == "/versions" {
			h.GetVersionHistory(w, r)
			// PUT /schemas/:id/parameters
		} else if len(path) > len("/parameters") && path[len(path)-len("/parameters"):] == "/parameters" {
			h.SetActiveParameterSet(w, r)
			// PUT /schemas/:id/disable
		} else if len(path) > len("/disable") && path[len(path)-len("/disable"):] == "/disable" {
			h.DisableSchema(w, r)
			// PUT /schemas/:id/enable
		} else if len(path) > len("/enable") && path[len(path)-len("/enable"):] == "/enable" {
			h.EnableSchema(w, r)
			// DELETE /schemas/:id
		} else if r.Method == http.MethodDelete {
			h.HideSchema(w, r)
			// PUT /schemas/:id (update = create new version)
		} else if r.Method == http.MethodPut {
			h.UpdateSchema(w, r)
			// GET /schemas/:id
		} else if r.Method == http.MethodGet {
			h.GetSchema(w, r)
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	})

	return middleware.RequestID(mux)
}
