package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/common/httputil"
	"github.com/telhawk-systems/telhawk-stack/search/internal/models"
	"github.com/telhawk-systems/telhawk-stack/search/internal/service"
)

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
				links["next"] = buildNextLink(r.URL.Path, meta["next_cursor"])
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
			if errors.Is(err, service.ErrValidationFailed) {
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
		uc, ok := h.requireUserContext(r)
		if !ok {
			h.writeJSONAPIUnauthorized(w)
			return
		}
		resp, err := h.svc.RunSavedSearch(r.Context(), id, uc.ClientID)
		if err != nil {
			log.Printf("Failed to run saved search %s: %v", id, err)
			if errors.Is(err, service.ErrSearchDisabled) {
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
			w.Header().Set("Allow", http.MethodPost)
			httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
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
			w.Header().Set("Allow", http.MethodPost)
			httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
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
			w.Header().Set("Allow", http.MethodPost)
			httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
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
			w.Header().Set("Allow", http.MethodPost)
			httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
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
			if errors.Is(err, service.ErrValidationFailed) {
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
		w.Header().Set("Allow", http.MethodPost)
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
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
		w.Header().Set("Allow", http.MethodPost)
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
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
		w.Header().Set("Allow", http.MethodPost)
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
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
		w.Header().Set("Allow", http.MethodPost)
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
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
