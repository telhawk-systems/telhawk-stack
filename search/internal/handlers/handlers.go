package handlers

import (
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/common/httputil"
	"github.com/telhawk-systems/telhawk-stack/common/messaging"
	"github.com/telhawk-systems/telhawk-stack/search/internal/models"
	searchnats "github.com/telhawk-systems/telhawk-stack/search/internal/nats"
	"github.com/telhawk-systems/telhawk-stack/search/internal/service"
)

// Handler wires HTTP routes to the query service.
type Handler struct {
	svc       *service.SearchService
	scheduler interface {
		GetMetrics() map[string]interface{}
	}
	natsHandler *searchnats.Handler
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

// WithNATSHandler sets the NATS handler for health reporting.
func (h *Handler) WithNATSHandler(natsHandler *searchnats.Handler) *Handler {
	h.natsHandler = natsHandler
	return h
}

// Health handles GET /healthz for liveness probes.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
		return
	}
	health := h.svc.Health(r.Context())
	if h.scheduler != nil {
		health.Scheduler = h.scheduler.GetMetrics()
	}
	// Add NATS health status
	if h.natsHandler != nil && h.natsHandler.Client() != nil {
		natsStatus := messaging.CheckClientHealth(r.Context(), h.natsHandler.Client())
		health.NATS = &models.NATSHealthStatus{
			Connected: natsStatus.Connected,
			Latency:   natsStatus.Latency.Milliseconds(),
			Error:     natsStatus.Error,
		}
	}
	httputil.WriteJSON(w, http.StatusOK, health)
}
