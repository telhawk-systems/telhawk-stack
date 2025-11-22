package handlers

import (
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/search/internal/models"
	"github.com/telhawk-systems/telhawk-stack/search/pkg/model"
)

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
