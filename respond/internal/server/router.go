// Package server provides HTTP server setup for the respond service.
package server

import (
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/common/middleware"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/handlers"
)

// NewRouter constructs a ServeMux with respond API routes registered.
func NewRouter(h *handlers.Handler) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/healthz", h.HealthCheck)
	mux.HandleFunc("/readyz", h.ReadyCheck)

	// Detection Schema routes
	mux.HandleFunc("/schemas", h.SchemasHandler)
	mux.HandleFunc("/schemas/", schemaRouteHandler(h))

	// Case routes (under /api/v1/ prefix)
	mux.HandleFunc("/api/v1/cases", h.CasesHandler)
	mux.HandleFunc("/api/v1/cases/", caseRouteHandler(h))

	return middleware.RequestID(mux)
}

// schemaRouteHandler routes /schemas/{id}/* requests to appropriate handlers
func schemaRouteHandler(h *handlers.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Check for sub-routes
		switch {
		case strings.HasSuffix(path, "/disable"):
			h.DisableSchemaHandler(w, r)
		case strings.HasSuffix(path, "/enable"):
			h.EnableSchemaHandler(w, r)
		case strings.HasSuffix(path, "/versions"):
			h.VersionHistoryHandler(w, r)
		case strings.HasSuffix(path, "/parameters"):
			h.SetParameterSetHandler(w, r)
		default:
			// Handle /schemas/{id} directly
			h.SchemaHandler(w, r)
		}
	}
}

// caseRouteHandler routes /api/v1/cases/{id}/* requests to appropriate handlers
func caseRouteHandler(h *handlers.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Check for sub-routes
		switch {
		case strings.HasSuffix(path, "/close"):
			h.CloseCaseHandler(w, r)
		case strings.HasSuffix(path, "/reopen"):
			h.ReopenCaseHandler(w, r)
		case strings.HasSuffix(path, "/alerts"):
			h.CaseAlertsHandler(w, r)
		default:
			// Handle /api/v1/cases/{id} directly
			h.CaseHandler(w, r)
		}
	}
}
