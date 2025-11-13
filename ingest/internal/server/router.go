package server

import (
	"github.com/telhawk-systems/telhawk-stack/common/middleware"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/handlers"
)

// NewRouter constructs a ServeMux with ingest API routes registered.
func NewRouter(h *handlers.HECHandler) http.Handler {
	mux := http.NewServeMux()

	// Splunk HEC endpoints
	mux.HandleFunc("/services/collector/event", h.HandleEvent)
	mux.HandleFunc("/services/collector/raw", h.HandleRaw)
	mux.HandleFunc("/services/collector/health", h.Health)
	mux.HandleFunc("/services/collector/ack", h.Ack)

	// Health endpoints
	mux.HandleFunc("/healthz", h.Health)
	mux.HandleFunc("/readyz", h.Ready)

	// Prometheus metrics
	mux.Handle("/metrics", promhttp.Handler())

	return middleware.RequestID(mux)
}
