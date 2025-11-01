package server

import (
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/core/internal/handlers"
)

// NewRouter wires HTTP routes for the core service.
func NewRouter(h *handlers.ProcessorHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/normalize", h.Normalize)
	mux.HandleFunc("/healthz", h.Health)
	return mux
}
