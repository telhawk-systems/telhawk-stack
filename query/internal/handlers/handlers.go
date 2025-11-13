package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/telhawk-systems/telhawk-stack/common/httputil"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
	"github.com/telhawk-systems/telhawk-stack/query/internal/service"
	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

// Handler wires HTTP routes to the query service.
type Handler struct {
	svc       *service.QueryService
	scheduler interface {
		GetMetrics() map[string]interface{}
	}
}

// New creates a Handler instance.
func New(svc *service.QueryService) *Handler {
	return &Handler{svc: svc, scheduler: nil}
}

// WithScheduler sets the scheduler for metrics reporting.
func (h *Handler) WithScheduler(scheduler interface{ GetMetrics() map[string]interface{} }) *Handler {
	h.scheduler = scheduler
	return h
}

// Search handles POST /api/v1/search requests.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
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
	var req models.SearchRequest
	typ, _, err := h.decodeJSONAPIResource(r.Body, &req)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if typ != "search" {
		h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_type", "data.type must be 'search'")
		return
	}
	resp, err := h.svc.ExecuteSearch(r.Context(), &req)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusInternalServerError, "search_failed", err.Error())
		return
	}
	attrs := map[string]interface{}{
		"request_id":    resp.RequestID,
		"latency_ms":    resp.LatencyMS,
		"result_count":  resp.ResultCount,
		"total_matches": resp.TotalMatches,
		"results":       resp.Results,
	}
	if resp.SearchAfter != nil {
		attrs["search_after"] = resp.SearchAfter
	}
	h.writeJSONAPIResourceGeneric(w, http.StatusOK, "search-result", resp.RequestID, attrs, nil)
}

// Query handles POST /api/v1/query requests with canonical JSON query format.
func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
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
	var q model.Query
	typ, _, err := h.decodeJSONAPIResource(r.Body, &q)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if typ != "query" {
		h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_type", "data.type must be 'query'")
		return
	}
	resp, err := h.svc.ExecuteQuery(r.Context(), &q)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}
	attrs := map[string]interface{}{
		"request_id":       resp.RequestID,
		"latency_ms":       resp.LatencyMS,
		"result_count":     resp.ResultCount,
		"total_matches":    resp.TotalMatches,
		"results":          resp.Results,
		"opensearch_query": resp.OpenSearchQuery,
	}
	if resp.SearchAfter != nil {
		attrs["search_after"] = resp.SearchAfter
	}
	h.writeJSONAPIResourceGeneric(w, http.StatusOK, "search-result", resp.RequestID, attrs, nil)
}

// SavedSearches handles collection routes: GET list, POST create.
func (h *Handler) SavedSearches(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !acceptJSONAPI(r) {
			h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
			return
		}
		showAll := parseShowAll(r.URL)
		// If cursor provided, use cursor-based pagination; else use page/size
		if curStr := r.URL.Query().Get("page[cursor]"); curStr != "" {
			ca, cv, size := parseCursor(r.URL)
			results, err := h.svc.ListSavedSearchesAfter(r.Context(), showAll, ca, cv, size)
			if err != nil {
				h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_list_failed", "Failed to list saved searches")
				return
			}
			meta := map[string]interface{}{"page": map[string]interface{}{"size": size}}
			if len(results) == size {
				last := results[len(results)-1]
				meta["next_cursor"] = buildCursor(&last)
			}
			// Build links
			links := map[string]interface{}{"self": r.URL.RequestURI()}
			if meta["next_cursor"] != nil {
				links["next"] = fmt.Sprintf("%s?page[cursor]=%v", r.URL.Path, meta["next_cursor"])
			}
			h.writeJSONAPIListWithMeta(w, http.StatusOK, results, meta, links)
		} else {
			page, size := parsePage(r.URL)
			var (
				searches []models.SavedSearch
				total    int
				err      error
			)
			searches, total, err = h.svc.ListSavedSearchesPaged(r.Context(), showAll, page, size)
			if err != nil {
				h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_list_failed", "Failed to list saved searches")
				return
			}
			links := map[string]interface{}{"self": r.URL.RequestURI()}
			h.writeJSONAPIListWithMeta(w, http.StatusOK, searches, map[string]interface{}{"page": map[string]int{"number": page, "size": size}, "total": total}, links)
		}
	case http.MethodPost:
		if !acceptJSONAPI(r) {
			h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
			return
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "application/vnd.api+json") {
			h.writeJSONAPIError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/vnd.api+json")
			return
		}
		userID, ok := h.requireUser(r)
		if !ok {
			h.writeJSONAPIUnauthorized(w)
			return
		}
		var req models.SavedSearchCreateRequest
		typ, _, err := h.decodeJSONAPIResource(r.Body, &req)
		if err != nil {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if typ != "saved-search" {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_type", "data.type must be 'saved-search'")
			return
		}
		req.CreatedBy = userID
		saved, err := h.svc.CreateSavedSearch(r.Context(), &req)
		if err != nil {
			log.Printf("Failed to create saved search: %v", err)
			// Return validation errors with 400, other errors with 500
			if strings.Contains(err.Error(), "validation failed") ||
				strings.Contains(err.Error(), "cannot be empty") ||
				strings.Contains(err.Error(), "invalid query") {
				h.writeJSONAPIError(w, http.StatusBadRequest, "validation_failed", err.Error())
			} else {
				h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_create_failed", err.Error())
			}
			return
		}
		w.Header().Set("Location", "/api/v1/saved-searches/"+saved.ID)
		h.writeJSONAPIResourceWithLinks(w, http.StatusCreated, saved, map[string]interface{}{"self": "/api/v1/saved-searches/" + saved.ID})
	default:
		h.methodNotAllowedJSONAPI(w, http.MethodGet, http.MethodPost)
	}
}

// SavedSearchByID handles item routes: GET, PUT, DELETE, and POST /run.
func (h *Handler) SavedSearchByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/saved-searches/")
	if path == "" {
		h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_saved_search_id", "id required")
		return
	}
	if strings.HasSuffix(path, "/run") {
		id := strings.TrimSuffix(path, "/run")
		if id == "" || strings.ContainsRune(id, '/') {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_saved_search_id", "id required")
			return
		}
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
		resp, err := h.svc.RunSavedSearch(r.Context(), id)
		if err != nil {
			log.Printf("Failed to run saved search %s: %v", id, err)
			if err.Error() == "search_disabled" {
				h.writeJSONAPIConflict(w, "search_disabled", "Saved search is disabled")
				return
			}
			h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_run_failed", err.Error())
			return
		}
		attrs := map[string]interface{}{
			"request_id":    resp.RequestID,
			"latency_ms":    resp.LatencyMS,
			"result_count":  resp.ResultCount,
			"total_matches": resp.TotalMatches,
			"results":       resp.Results,
		}
		if resp.SearchAfter != nil {
			attrs["search_after"] = resp.SearchAfter
		}
		h.writeJSONAPIResourceGeneric(w, http.StatusOK, "search-result", resp.RequestID, attrs, nil)
		return
	}
	if strings.HasSuffix(path, "/disable") {
		id := strings.TrimSuffix(path, "/disable")
		if id == "" || strings.ContainsRune(id, '/') {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_saved_search_id", "id required")
			return
		}
		if r.Method != http.MethodPost {
			h.methodNotAllowed(w, http.MethodPost)
			return
		}
		userID, ok := h.requireUser(r)
		if !ok {
			h.writeJSONAPIUnauthorized(w)
			return
		}
		saved, err := h.svc.DisableSavedSearch(r.Context(), id, userID)
		if err != nil {
			h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_disable_failed", "Failed to disable")
			return
		}
		h.writeJSONAPIResource(w, http.StatusOK, saved)
		return
	}
	if strings.HasSuffix(path, "/enable") {
		id := strings.TrimSuffix(path, "/enable")
		if id == "" || strings.ContainsRune(id, '/') {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_saved_search_id", "id required")
			return
		}
		if r.Method != http.MethodPost {
			h.methodNotAllowed(w, http.MethodPost)
			return
		}
		userID, ok := h.requireUser(r)
		if !ok {
			h.writeJSONAPIUnauthorized(w)
			return
		}
		saved, err := h.svc.EnableSavedSearch(r.Context(), id, userID)
		if err != nil {
			h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_enable_failed", "Failed to enable")
			return
		}
		h.writeJSONAPIResource(w, http.StatusOK, saved)
		return
	}
	if strings.HasSuffix(path, "/hide") {
		id := strings.TrimSuffix(path, "/hide")
		if id == "" || strings.ContainsRune(id, '/') {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_saved_search_id", "id required")
			return
		}
		if r.Method != http.MethodPost {
			h.methodNotAllowed(w, http.MethodPost)
			return
		}
		userID, ok := h.requireUser(r)
		if !ok {
			h.writeJSONAPIUnauthorized(w)
			return
		}
		saved, err := h.svc.HideSavedSearch(r.Context(), id, userID)
		if err != nil {
			h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_hide_failed", "Failed to hide")
			return
		}
		h.writeJSONAPIResource(w, http.StatusOK, saved)
		return
	}
	if strings.HasSuffix(path, "/unhide") {
		id := strings.TrimSuffix(path, "/unhide")
		if id == "" || strings.ContainsRune(id, '/') {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_saved_search_id", "id required")
			return
		}
		if r.Method != http.MethodPost {
			h.methodNotAllowed(w, http.MethodPost)
			return
		}
		userID, ok := h.requireUser(r)
		if !ok {
			h.writeJSONAPIUnauthorized(w)
			return
		}
		saved, err := h.svc.UnhideSavedSearch(r.Context(), id, userID)
		if err != nil {
			h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_unhide_failed", "Failed to unhide")
			return
		}
		h.writeJSONAPIResource(w, http.StatusOK, saved)
		return
	}
	id := path
	switch r.Method {
	case http.MethodGet:
		if !acceptJSONAPI(r) {
			h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
			return
		}
		saved, err := h.svc.GetSavedSearch(r.Context(), id)
		if err != nil {
			h.writeJSONAPIError(w, http.StatusNotFound, "saved_search_not_found", "Saved search not found")
			return
		}
		h.writeJSONAPIResource(w, http.StatusOK, saved)
	case http.MethodPatch:
		if !acceptJSONAPI(r) {
			h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
			return
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "application/vnd.api+json") {
			h.writeJSONAPIError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/vnd.api+json")
			return
		}
		userID, ok := h.requireUser(r)
		if !ok {
			h.writeJSONAPIUnauthorized(w)
			return
		}
		var req models.SavedSearchUpdateRequest
		typ, rid, err := h.decodeJSONAPIResource(r.Body, &req)
		if err != nil {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if typ != "saved-search" {
			h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_type", "data.type must be 'saved-search'")
			return
		}
		if rid != "" && rid != id {
			h.writeJSONAPIError(w, http.StatusConflict, "id_mismatch", "data.id must match URL id")
			return
		}
		req.CreatedBy = userID
		saved, err := h.svc.UpdateSavedSearch(r.Context(), id, &req)
		if err != nil {
			log.Printf("Failed to update saved search: %v", err)
			// Return validation errors with 400, other errors with 500
			if strings.Contains(err.Error(), "validation failed") ||
				strings.Contains(err.Error(), "cannot be empty") ||
				strings.Contains(err.Error(), "invalid query") {
				h.writeJSONAPIError(w, http.StatusBadRequest, "validation_failed", err.Error())
			} else {
				h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_update_failed", err.Error())
			}
			return
		}
		h.writeJSONAPIResourceWithLinks(w, http.StatusOK, saved, map[string]interface{}{"self": "/api/v1/saved-searches/" + saved.ID})
	case http.MethodDelete:
		if !acceptJSONAPI(r) {
			h.writeJSONAPIError(w, http.StatusNotAcceptable, "not_acceptable", "Accept must allow application/vnd.api+json")
			return
		}
		if _, ok := h.requireUser(r); !ok {
			h.writeJSONAPIUnauthorized(w)
			return
		}
		if err := h.svc.DeleteSavedSearch(r.Context(), id); err != nil {
			h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_delete_failed", "Failed to delete saved search")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		h.methodNotAllowedJSONAPI(w, http.MethodGet, http.MethodPatch, http.MethodDelete)
	}
}

// State transition endpoints for saved searches
func (h *Handler) SavedSearchDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	userID, ok := h.requireUser(r)
	if !ok {
		h.writeJSONAPIUnauthorized(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/saved-searches/")
	id = strings.TrimSuffix(id, "/disable")
	saved, err := h.svc.DisableSavedSearch(r.Context(), id, userID)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_disable_failed", "Failed to disable")
		return
	}
	h.writeJSONAPIResource(w, http.StatusOK, saved)
}

func (h *Handler) SavedSearchEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	userID, ok := h.requireUser(r)
	if !ok {
		h.writeJSONAPIUnauthorized(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/saved-searches/")
	id = strings.TrimSuffix(id, "/enable")
	saved, err := h.svc.EnableSavedSearch(r.Context(), id, userID)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_enable_failed", "Failed to enable")
		return
	}
	h.writeJSONAPIResource(w, http.StatusOK, saved)
}

func (h *Handler) SavedSearchHide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	userID, ok := h.requireUser(r)
	if !ok {
		h.writeJSONAPIUnauthorized(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/saved-searches/")
	id = strings.TrimSuffix(id, "/hide")
	saved, err := h.svc.HideSavedSearch(r.Context(), id, userID)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_hide_failed", "Failed to hide")
		return
	}
	h.writeJSONAPIResource(w, http.StatusOK, saved)
}

func (h *Handler) SavedSearchUnhide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w, http.MethodPost)
		return
	}
	userID, ok := h.requireUser(r)
	if !ok {
		h.writeJSONAPIUnauthorized(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/saved-searches/")
	id = strings.TrimSuffix(id, "/unhide")
	saved, err := h.svc.UnhideSavedSearch(r.Context(), id, userID)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusInternalServerError, "saved_search_unhide_failed", "Failed to unhide")
		return
	}
	h.writeJSONAPIResource(w, http.StatusOK, saved)
}

// --- JSON:API helpers ---
type jsonAPIResource struct {
	Type          string                 `json:"type"`
	ID            string                 `json:"id"`
	Attributes    map[string]interface{} `json:"attributes"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
}

// Events (JSON:API) -----------------------------------------------------------

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
			if err.Error() == "search_disabled" {
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

func (h *Handler) writeJSONAPIList(w http.ResponseWriter, status int, searches []models.SavedSearch) {
	items := make([]jsonAPIResource, 0, len(searches))
	for _, s := range searches {
		items = append(items, toJSONAPI(s))
	}
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": items})
}

func (h *Handler) writeJSONAPIListWithMeta(w http.ResponseWriter, status int, searches []models.SavedSearch, meta map[string]interface{}, links map[string]interface{}) {
	items := make([]jsonAPIResource, 0, len(searches))
	for _, s := range searches {
		items = append(items, toJSONAPI(s))
	}
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	resp := map[string]interface{}{"data": items}
	if meta != nil {
		resp["meta"] = meta
	}
	if links != nil {
		resp["links"] = links
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) writeJSONAPIResource(w http.ResponseWriter, status int, s *models.SavedSearch) {
	h.writeJSONAPIResourceWithLinks(w, status, s, nil)
}

func (h *Handler) writeJSONAPIResourceWithLinks(w http.ResponseWriter, status int, s *models.SavedSearch, links map[string]interface{}) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	resp := map[string]interface{}{"data": toJSONAPI(*s)}
	if links != nil {
		resp["links"] = links
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// writeJSONAPIResourceGeneric writes a JSON:API single resource with provided type/id/attributes.
func (h *Handler) writeJSONAPIResourceGeneric(w http.ResponseWriter, status int, typ, id string, attrs map[string]interface{}, rels map[string]interface{}) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	res := jsonAPIResource{Type: typ, ID: id, Attributes: attrs, Relationships: rels}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": res})
}

func (h *Handler) writeJSONAPIError(w http.ResponseWriter, status int, code, title string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"errors": []map[string]string{{
		"status": fmt.Sprintf("%d", status),
		"code":   code,
		"title":  title,
	}}})
}

func (h *Handler) writeJSONAPIUnauthorized(w http.ResponseWriter) {
	h.writeJSONAPIError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
}

func (h *Handler) writeJSONAPIConflict(w http.ResponseWriter, code, title string) {
	h.writeJSONAPIError(w, http.StatusConflict, code, title)
}

func toJSONAPI(s models.SavedSearch) jsonAPIResource {
	attrs := map[string]interface{}{
		"version_id":  s.VersionID,
		"name":        s.Name,
		"query":       s.Query,
		"filters":     s.Filters,
		"is_global":   s.IsGlobal,
		"created_at":  s.CreatedAt,
		"disabled_at": s.DisabledAt,
		"hidden_at":   s.HiddenAt,
	}
	rels := map[string]interface{}{}
	if s.OwnerID != nil {
		rels["owner"] = map[string]interface{}{
			"data": map[string]interface{}{"type": "user", "id": *s.OwnerID},
		}
	}
	if s.CreatedBy != "" {
		rels["created_by"] = map[string]interface{}{
			"data": map[string]interface{}{"type": "user", "id": s.CreatedBy},
		}
	}
	return jsonAPIResource{Type: "saved-search", ID: s.ID, Attributes: attrs, Relationships: rels}
}

func (h *Handler) decodeJSONAPI(body io.ReadCloser, dst interface{}) error {
	defer body.Close()
	var payload struct {
		Data struct{ Attributes json.RawMessage }
	}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return err
	}
	return json.Unmarshal(payload.Data.Attributes, dst)
}

func (h *Handler) decodeJSONAPIResource(body io.ReadCloser, dst interface{}) (string, string, error) {
	defer body.Close()
	var payload struct {
		Data struct {
			Type       string          `json:"type"`
			ID         string          `json:"id"`
			Attributes json.RawMessage `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return "", "", err
	}
	if err := json.Unmarshal(payload.Data.Attributes, dst); err != nil {
		return "", "", err
	}
	return payload.Data.Type, payload.Data.ID, nil
}

func parseShowAll(u *url.URL) bool {
	// JSON:API filter[show_all]=true
	if vals, ok := u.Query()["filter[show_all]"]; ok && len(vals) > 0 {
		v := strings.ToLower(vals[0])
		return v == "true" || v == "1"
	}
	return false
}

func parsePage(u *url.URL) (int, int) {
	q := u.Query()
	pn := 1
	ps := 20
	if v := q.Get("page[number]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pn = n
		}
	}
	if v := q.Get("page[size]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			ps = n
		}
	}
	return pn, ps
}

// Cursor helpers: encode/decode as "<unix_ns>|<version_id>"
func parseCursor(u *url.URL) (*time.Time, string, int) {
	q := u.Query()
	cur := q.Get("page[cursor]")
	size := 20
	if v := q.Get("page[size]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			size = n
		}
	}
	if cur == "" {
		return nil, "", size
	}
	parts := strings.SplitN(cur, "|", 2)
	if len(parts) != 2 {
		return nil, "", size
	}
	// parse unix ns
	ns, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, "", size
	}
	t := time.Unix(0, ns).UTC()
	return &t, parts[1], size
}

func buildCursor(s *models.SavedSearch) string {
	return fmt.Sprintf("%d|%s", s.CreatedAt.UnixNano(), s.VersionID)
}

// Accept header validation for JSON:API endpoints
func acceptJSONAPI(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" || accept == "*/*" {
		return true
	}
	return strings.Contains(accept, "application/vnd.api+json")
}

// Authentication: extract and validate bearer token via auth service.
func (h *Handler) requireUser(r *http.Request) (string, bool) {
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return "", false
	}
	token := strings.TrimSpace(authz[len("Bearer "):])
	userID, err := h.svc.ValidateToken(r.Context(), token)
	if err != nil {
		return "", false
	}
	return userID, true
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

// Export handles POST /api/v1/export requests.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
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
	var req models.ExportRequest
	typ, _, err := h.decodeJSONAPIResource(r.Body, &req)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if typ != "export" {
		h.writeJSONAPIError(w, http.StatusBadRequest, "invalid_type", "data.type must be 'export'")
		return
	}
	resp, err := h.svc.RequestExport(r.Context(), &req)
	if err != nil {
		h.writeJSONAPIError(w, http.StatusInternalServerError, "export_failed", err.Error())
		return
	}
	attrs := map[string]interface{}{"status": resp.Status, "expires_at": resp.ExpiresAt}
	h.writeJSONAPIResourceGeneric(w, http.StatusAccepted, "export-job", resp.ExportID, attrs, nil)
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

func (h *Handler) methodNotAllowedJSONAPI(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	h.writeJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
}

// svcIDFallback generates a simple fallback ID string when an event lacks an ID field.
func (h *Handler) svcIDFallback() string {
	// Use time-based fallback; not stable but avoids violating JSON:API schema.
	return fmt.Sprintf("evt-%d", time.Now().UnixNano())
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
