package server

import (
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/common/middleware"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/handlers"
)

// NewRouter constructs a ServeMux with respond API routes registered.
func NewRouter(h *handlers.Handler) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/healthz", h.HealthCheck)
	mux.HandleFunc("/readyz", h.ReadyCheck)

	// TODO: Migrate rules API routes from rules service
	// - /schemas (POST, GET)
	// - /schemas/{id} (GET, PUT, DELETE)
	// - /schemas/{id}/versions (GET)
	// - /schemas/{id}/parameters (PUT)
	// - /schemas/{id}/disable (PUT)
	// - /schemas/{id}/enable (PUT)
	// - /correlation/types (GET)

	// TODO: Migrate alerts API routes from alerting service
	// - /api/v1/alerts (GET)
	// - /api/v1/alerts/{id} (GET)
	// - /api/v1/cases (POST, GET)
	// - /api/v1/cases/{id} (GET, PUT)
	// - /api/v1/cases/{id}/close (PUT)
	// - /api/v1/cases/{id}/reopen (PUT)
	// - /api/v1/cases/{id}/alerts (POST, GET)

	return middleware.RequestID(mux)
}
