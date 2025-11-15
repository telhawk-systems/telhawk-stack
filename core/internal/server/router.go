package server

import (
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/common/middleware"

	"github.com/telhawk-systems/telhawk-stack/core/internal/handlers"
)

// NewRouter wires HTTP routes for the core service.
func NewRouter(h *handlers.ProcessorHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/normalize", h.Normalize)
	mux.HandleFunc("/api/v1/dlq", h.ListDLQ)
	mux.HandleFunc("/api/v1/dlq/purge", h.PurgeDLQ)
	mux.HandleFunc("/healthz", h.Health)
	return middleware.RequestID(mux)
}
