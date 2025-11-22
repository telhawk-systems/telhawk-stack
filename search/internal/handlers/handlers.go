package handlers

import (
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/search/internal/service"
)

// Handler wires HTTP routes to the query service.
type Handler struct {
	svc       *service.SearchService
	scheduler interface {
		GetMetrics() map[string]interface{}
	}
}

// New creates a Handler instance.
func New(svc *service.SearchService) *Handler {
	return &Handler{svc: svc, scheduler: nil}
}

// WithScheduler sets the scheduler for metrics reporting.
func (h *Handler) WithScheduler(scheduler interface{ GetMetrics() map[string]interface{} }) *Handler {
	h.scheduler = scheduler
	return h
}

// Health handles GET /healthz for liveness probes.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	health := h.svc.Health(r.Context())
	if h.scheduler != nil {
		health.Scheduler = h.scheduler.GetMetrics()
	}
	h.writeJSON(w, http.StatusOK, health)
}
