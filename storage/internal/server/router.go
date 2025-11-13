package server

import (
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/storage/internal/handlers"
)

// NewRouter constructs a ServeMux with storage API routes registered.
func NewRouter(h *handlers.StorageHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/ingest", h.Ingest)
	mux.HandleFunc("/api/v1/bulk", h.BulkIngest)
	mux.HandleFunc("/healthz", h.Health)
	mux.HandleFunc("/readyz", h.Ready)
	return mux
}
