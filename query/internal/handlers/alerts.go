package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
	"github.com/telhawk-systems/telhawk-stack/query/internal/service"
)

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
	if !acceptJSONAPI(r) {
		h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
		return
	}
	resp, err := h.svc.ListAlerts(r.Context())
	if err != nil {
		h.writeJSONAPIError(w, http.StatusInternalServerError, "alerts_unavailable", err.Error())
		return
	}
	items := make([]jsonAPIResource, 0, len(resp.Alerts))
	for _, a := range resp.Alerts {
		attrs := map[string]interface{}{
			"name": a.Name, "description": a.Description, "query": a.Query, "severity": a.Severity,
			"schedule": a.Schedule, "status": a.Status, "last_triggered_at": a.LastTriggeredAt, "owner": a.Owner,
		}
		items = append(items, jsonAPIResource{Type: "alert", ID: a.ID, Attributes: attrs})
	}
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": items, "links": map[string]interface{}{"self": r.URL.RequestURI()}})
}

func (h *Handler) handleUpsertAlert(w http.ResponseWriter, r *http.Request) {
	if !acceptJSONAPI(r) {
		h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
		return
	}
	if !strings.Contains(r.Header.Get("Content-Type"), "application/vnd.api+json") {
		h.writeJSONAPIError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/vnd.api+json")
		return
	}
	var req models.AlertRequest
	typ, _, err := h.decodeJSONAPIResource(r.Body, &req)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if typ != "alert" {
		h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_type", "data.type must be 'alert'")
		return
	}
	alert, created, err := h.svc.UpsertAlert(r.Context(), &req)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusInternalServerError, "alert_upsert_failed", err.Error())
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	attrs := map[string]interface{}{
		"name": alert.Name, "description": alert.Description, "query": alert.Query, "severity": alert.Severity,
		"schedule": alert.Schedule, "status": alert.Status, "last_triggered_at": alert.LastTriggeredAt, "owner": alert.Owner,
	}
	h.writeJSONAPIResourceGeneric(w, status, "alert", alert.ID, attrs, nil)
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
	if !acceptJSONAPI(r) {
		h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
		return
	}
	alert, err := h.svc.GetAlert(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrAlertNotFound) {
			h.writeJSONAPIError(w, http.StatusNotFound, "alert_not_found", "alert not found")
			return
		}
		h.writeJSONAPIError(w, http.StatusInternalServerError, "alert_lookup_failed", err.Error())
		return
	}
	attrs := map[string]interface{}{
		"name": alert.Name, "description": alert.Description, "query": alert.Query, "severity": alert.Severity,
		"schedule": alert.Schedule, "status": alert.Status, "last_triggered_at": alert.LastTriggeredAt, "owner": alert.Owner,
	}
	h.writeJSONAPIResourceGeneric(w, http.StatusOK, "alert", alert.ID, attrs, nil)
}

func (h *Handler) patchAlert(w http.ResponseWriter, r *http.Request, id string) {
	if !acceptJSONAPI(r) {
		h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
		return
	}
	if !strings.Contains(r.Header.Get("Content-Type"), "application/vnd.api+json") {
		h.writeJSONAPIError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/vnd.api+json")
		return
	}
	var req models.AlertPatchRequest
	// Allow partial attributes; decode using plain JSON helper
	if err := decodeJSON(r.Body, &req); err != nil {
		h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	alert, err := h.svc.PatchAlert(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, service.ErrAlertNotFound) {
			h.writeJSONAPIError(w, http.StatusNotFound, "alert_not_found", "alert not found")
			return
		}
		h.writeJSONAPIError(w, http.StatusInternalServerError, "alert_patch_failed", err.Error())
		return
	}
	attrs := map[string]interface{}{
		"name": alert.Name, "description": alert.Description, "query": alert.Query, "severity": alert.Severity,
		"schedule": alert.Schedule, "status": alert.Status, "last_triggered_at": alert.LastTriggeredAt, "owner": alert.Owner,
	}
	h.writeJSONAPIResourceGeneric(w, http.StatusOK, "alert", alert.ID, attrs, nil)
}
