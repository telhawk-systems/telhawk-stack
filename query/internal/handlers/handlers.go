package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
	"github.com/telhawk-systems/telhawk-stack/query/internal/service"
)

// Handler wires HTTP routes to the query service.
type Handler struct {
	svc *service.QueryService
}

// New creates a Handler instance.
func New(svc *service.QueryService) *Handler {
	return &Handler{svc: svc}
}

// Search handles POST /api/v1/search requests.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	var req models.SearchRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	resp, err := h.svc.ExecuteSearch(r.Context(), &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "search_failed", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// Alerts handles GET/POST /api/v1/alerts.
func (h *Handler) Alerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListAlerts(w, r)
	case http.MethodPost:
		h.handleUpsertAlert(w, r)
	default:
		h.methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (h *Handler) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListAlerts(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "alerts_unavailable", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleUpsertAlert(w http.ResponseWriter, r *http.Request) {
	var req models.AlertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	alert, created, err := h.svc.UpsertAlert(r.Context(), &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "alert_upsert_failed", err.Error())
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	h.writeJSON(w, status, alert)
}

// AlertByID handles GET/PATCH /api/v1/alerts/{alertId}.
func (h *Handler) AlertByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/")
	if id == "" || strings.ContainsRune(id, '/') {
		h.writeError(w, http.StatusBadRequest, "invalid_alert_id", "alert id must be provided")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getAlert(w, r, id)
	case http.MethodPatch:
		h.patchAlert(w, r, id)
	default:
		h.methodNotAllowed(w, http.MethodGet, http.MethodPatch)
	}
}

func (h *Handler) getAlert(w http.ResponseWriter, r *http.Request, id string) {
	alert, err := h.svc.GetAlert(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrAlertNotFound) {
			h.writeError(w, http.StatusNotFound, "alert_not_found", "alert not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "alert_lookup_failed", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, alert)
}

func (h *Handler) patchAlert(w http.ResponseWriter, r *http.Request, id string) {
	var req models.AlertPatchRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	alert, err := h.svc.PatchAlert(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, service.ErrAlertNotFound) {
			h.writeError(w, http.StatusNotFound, "alert_not_found", "alert not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "alert_patch_failed", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, alert)
}

// Dashboards handles GET /api/v1/dashboards.
func (h *Handler) Dashboards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	resp, err := h.svc.ListDashboards(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "dashboards_unavailable", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// DashboardByID handles GET /api/v1/dashboards/{dashboardId}.
func (h *Handler) DashboardByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/dashboards/")
	if id == "" || strings.ContainsRune(id, '/') {
		h.writeError(w, http.StatusBadRequest, "invalid_dashboard_id", "dashboard id must be provided")
		return
	}
	dashboard, err := h.svc.GetDashboard(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrDashboardNotFound) {
			h.writeError(w, http.StatusNotFound, "dashboard_not_found", "dashboard not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "dashboard_lookup_failed", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, dashboard)
}

// Export handles POST /api/v1/export requests.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	var req models.ExportRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	resp, err := h.svc.RequestExport(r.Context(), &req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "export_failed", err.Error())
		return
	}
	h.writeJSON(w, http.StatusAccepted, resp)
}

// Health handles GET /healthz for liveness probes.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w, http.MethodGet)
		return
	}
	h.writeJSON(w, http.StatusOK, h.svc.Health(r.Context()))
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func (h *Handler) methodNotAllowed(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
}

func decodeJSON(body io.ReadCloser, dst interface{}) error {
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	return nil
}
