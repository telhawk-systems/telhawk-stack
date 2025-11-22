package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/search/internal/service"
)

// Dashboards handles GET /api/v1/dashboards.
func (h *Handler) Dashboards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowedJSONAPI(w, http.MethodGet)
		return
	}
	if !acceptJSONAPI(r) {
		h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
		return
	}
	resp, err := h.svc.ListDashboards(r.Context())
	if err != nil {
		h.writeJSONAPIError(w, http.StatusInternalServerError, "dashboards_unavailable", err.Error())
		return
	}
	items := make([]jsonAPIResource, 0, len(resp.Dashboards))
	for _, d := range resp.Dashboards {
		attrs := map[string]interface{}{"name": d.Name, "description": d.Description, "widgets": d.Widgets}
		items = append(items, jsonAPIResource{Type: "dashboard", ID: d.ID, Attributes: attrs})
	}
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": items, "links": map[string]interface{}{"self": r.URL.RequestURI()}})
}

// DashboardByID handles GET /api/v1/dashboards/{dashboardId}.
func (h *Handler) DashboardByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowedJSONAPI(w, http.MethodGet)
		return
	}
	if !acceptJSONAPI(r) {
		h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/dashboards/")
	if id == "" || strings.ContainsRune(id, '/') {
		h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_dashboard_id", "dashboard id must be provided")
		return
	}
	dashboard, err := h.svc.GetDashboard(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrDashboardNotFound) {
			h.writeJSONAPIError(w, http.StatusNotFound, "dashboard_not_found", "dashboard not found")
			return
		}
		h.writeJSONAPIError(w, http.StatusInternalServerError, "dashboard_lookup_failed", err.Error())
		return
	}
	attrs := map[string]interface{}{"name": dashboard.Name, "description": dashboard.Description, "widgets": dashboard.Widgets}
	h.writeJSONAPIResourceGeneric(w, http.StatusOK, "dashboard", dashboard.ID, attrs, nil)
}
