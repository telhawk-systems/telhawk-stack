package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/search/internal/models"
	"github.com/telhawk-systems/telhawk-stack/search/internal/service"
	"github.com/telhawk-systems/telhawk-stack/search/pkg/model"
)

// Events handles GET /api/v1/events and POST /api/v1/events (not used).
// GET supports simple filtering via filter[query], sort, page[number], page[size].
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !acceptJSONAPI(r) {
			h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
			return
		}
		if _, ok := h.requireUser(r); !ok {
			h.writeJSONAPIUnauthorized(w)
			return
		}
		// Build search request from params
		q := r.URL.Query().Get("filter[query]")
		page, size := parsePage(r.URL)
		sortField := r.URL.Query().Get("sort")
		var sortOpt *models.SortOptions
		if sortField != "" {
			order := "asc"
			if strings.HasPrefix(sortField, "-") {
				order = "desc"
				sortField = strings.TrimPrefix(sortField, "-")
			}
			sortOpt = &models.SortOptions{Field: sortField, Order: order}
		}
		req := models.SearchRequest{Query: q, Limit: size, Sort: sortOpt}
		resp, err := h.svc.ExecuteSearch(r.Context(), &req)
		if err != nil {
			h.writeJSONAPIError(w, http.StatusInternalServerError, "events_query_failed", err.Error())
			return
		}
		// Build data
		data := make([]jsonAPIResource, 0, len(resp.Results))
		for _, ev := range resp.Results {
			id := ""
			if v, ok := ev["_id"].(string); ok && v != "" {
				id = v
			} else if v2, ok2 := ev["id"].(string); ok2 {
				id = v2
			} else {
				id = h.svcIDFallback()
			}
			data = append(data, jsonAPIResource{Type: "event", ID: id, Attributes: ev})
		}
		meta := map[string]interface{}{"total": resp.TotalMatches, "latency_ms": resp.LatencyMS, "page": map[string]int{"number": page, "size": size}}
		links := map[string]interface{}{"self": r.URL.RequestURI()}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": data, "meta": meta, "links": links})
	default:
		h.methodNotAllowedJSONAPI(w, http.MethodGet)
	}
}

// EventsByAction handles POST /api/v1/events/query and /api/v1/events/run/{id}
func (h *Handler) EventsByAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/events/")
	if strings.HasPrefix(path, "query") {
		if r.Method != http.MethodPost {
			h.methodNotAllowedJSONAPI(w, http.MethodPost)
			return
		}
		if !acceptJSONAPI(r) {
			h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
			return
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "application/vnd.api+json") {
			h.writeJSONAPIError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/vnd.api+json")
			return
		}
		if _, ok := h.requireUser(r); !ok {
			h.writeJSONAPIUnauthorized(w)
			return
		}
		// Decode canonical query
		var q model.Query
		typ, _, err := h.decodeJSONAPIResource(r.Body, &q)
		if err != nil {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if typ != "event-query" {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_type", "data.type must be 'event-query'")
			return
		}
		resp, err := h.svc.ExecuteQuery(r.Context(), &q)
		if err != nil {
			h.writeJSONAPIError(w, http.StatusInternalServerError, "events_query_failed", err.Error())
			return
		}
		data := make([]jsonAPIResource, 0, len(resp.Results))
		for _, ev := range resp.Results {
			id := ""
			if v, ok := ev["_id"].(string); ok && v != "" {
				id = v
			} else if v2, ok2 := ev["id"].(string); ok2 {
				id = v2
			} else {
				id = h.svcIDFallback()
			}
			data = append(data, jsonAPIResource{Type: "event", ID: id, Attributes: ev})
		}
		meta := map[string]interface{}{"total": resp.TotalMatches, "latency_ms": resp.LatencyMS}
		if resp.Aggregations != nil {
			meta["aggregations"] = resp.Aggregations
		}
		links := map[string]interface{}{"self": r.URL.RequestURI()}
		if resp.SearchAfter != nil && len(resp.Results) > 0 {
			meta["next_cursor"] = resp.SearchAfter
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": data, "meta": meta, "links": links})
		return
	}
	if strings.HasPrefix(path, "run/") {
		if r.Method != http.MethodPost {
			h.methodNotAllowedJSONAPI(w, http.MethodPost)
			return
		}
		if !acceptJSONAPI(r) {
			h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
			return
		}
		if _, ok := h.requireUser(r); !ok {
			h.writeJSONAPIUnauthorized(w)
			return
		}
		id := strings.TrimPrefix(path, "run/")
		if id == "" || strings.ContainsRune(id, '/') {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_saved_search_id", "id required")
			return
		}
		resp, err := h.svc.RunSavedSearch(r.Context(), id)
		if err != nil {
			if errors.Is(err, service.ErrSearchDisabled) {
				h.writeJSONAPIConflict(w, "search_disabled", "Saved search is disabled")
				return
			}
			h.writeJSONAPIError(w, http.StatusInternalServerError, "events_run_failed", err.Error())
			return
		}
		data := make([]jsonAPIResource, 0, len(resp.Results))
		for _, ev := range resp.Results {
			id := ""
			if v, ok := ev["_id"].(string); ok && v != "" {
				id = v
			} else if v2, ok2 := ev["id"].(string); ok2 {
				id = v2
			} else {
				id = h.svcIDFallback()
			}
			data = append(data, jsonAPIResource{Type: "event", ID: id, Attributes: ev})
		}
		meta := map[string]interface{}{"total": resp.TotalMatches, "latency_ms": resp.LatencyMS}
		links := map[string]interface{}{"self": r.URL.RequestURI()}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": data, "meta": meta, "links": links})
		return
	}
	h.writeJSONAPIError(w, http.StatusNotFound, "not_found", "unknown events action")
}
