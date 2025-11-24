package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/telhawk-systems/telhawk-stack/respond/internal/auth"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/models"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/storage"
)

// AlertsHandler provides HTTP handlers for alerts endpoints.
type AlertsHandler struct {
	storage    *storage.OpenSearchStorage
	authClient *auth.Client
}

// NewAlertsHandler creates a new AlertsHandler instance.
func NewAlertsHandler(storage *storage.OpenSearchStorage) *AlertsHandler {
	return &AlertsHandler{storage: storage}
}

// WithAuthClient sets the auth client for token validation.
func (h *AlertsHandler) WithAuthClient(client *auth.Client) *AlertsHandler {
	h.authClient = client
	return h
}

// requireUserContext extracts and validates bearer token, returning user context.
func (h *AlertsHandler) requireUserContext(r *http.Request) (*auth.UserContext, bool) {
	if h.authClient == nil {
		return nil, false
	}

	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return nil, false
	}

	token := strings.TrimSpace(authz[len("Bearer "):])
	uc, err := h.authClient.Validate(r.Context(), token)
	if err != nil {
		return nil, false
	}

	return uc, true
}

// ListAlerts handles GET /api/v1/alerts
func (h *AlertsHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAlertsError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Require authentication for data isolation
	uc, ok := h.requireUserContext(r)
	if !ok {
		writeAlertsError(w, http.StatusUnauthorized, "unauthorized", "Valid authentication required")
		return
	}

	// Parse query parameters
	req := &models.ListAlertsRequest{
		Page:     1,
		Limit:    20,
		ClientID: uc.ClientID, // CRITICAL: Data isolation filter
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			req.Page = p
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			req.Limit = l
		}
	}

	if severity := r.URL.Query().Get("severity"); severity != "" {
		req.Severity = severity
	}

	if status := r.URL.Query().Get("status"); status != "" {
		req.Status = status
	}

	if priority := r.URL.Query().Get("priority"); priority != "" {
		req.Priority = priority
	}

	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			req.From = &t
		}
	}

	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			req.To = &t
		}
	}

	if schemaID := r.URL.Query().Get("detection_schema_id"); schemaID != "" {
		req.DetectionSchemaID = schemaID
	}

	if caseID := r.URL.Query().Get("case_id"); caseID != "" {
		req.CaseID = caseID
	}

	// Query OpenSearch
	resp, err := h.storage.ListAlerts(r.Context(), req)
	if err != nil {
		writeAlertsError(w, http.StatusInternalServerError, "storage_error", err.Error())
		return
	}

	// Return response in the format expected by the frontend
	writeAlertsJSON(w, http.StatusOK, map[string]interface{}{
		"alerts":      resp.Alerts,
		"page":        resp.Pagination.Page,
		"limit":       resp.Pagination.Limit,
		"total":       resp.Pagination.Total,
		"total_pages": resp.Pagination.TotalPages,
	})
}

// GetAlert handles GET /api/v1/alerts/{id}
func (h *AlertsHandler) GetAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAlertsError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Require authentication for data isolation
	uc, ok := h.requireUserContext(r)
	if !ok {
		writeAlertsError(w, http.StatusUnauthorized, "unauthorized", "Valid authentication required")
		return
	}

	// Extract alert ID from path
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/")
	if id == "" {
		writeAlertsError(w, http.StatusBadRequest, "missing_id", "Alert ID is required")
		return
	}

	alert, err := h.storage.GetAlertByID(r.Context(), id, uc.ClientID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeAlertsError(w, http.StatusNotFound, "not_found", "Alert not found")
			return
		}
		writeAlertsError(w, http.StatusInternalServerError, "storage_error", err.Error())
		return
	}

	writeAlertsJSON(w, http.StatusOK, alert)
}

// AlertsRoute handles routing for /api/v1/alerts and /api/v1/alerts/{id}
func (h *AlertsHandler) AlertsRoute(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /api/v1/alerts - list all alerts
	if path == "/api/v1/alerts" || path == "/api/v1/alerts/" {
		h.ListAlerts(w, r)
		return
	}

	// /api/v1/alerts/{id} - get single alert
	if strings.HasPrefix(path, "/api/v1/alerts/") {
		h.GetAlert(w, r)
		return
	}

	writeAlertsError(w, http.StatusNotFound, "not_found", "Endpoint not found")
}

// Helper functions for alerts handlers
func writeAlertsJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := encodeJSON(w, data); err != nil {
		// Can't return error since headers already sent
		return
	}
}

func writeAlertsError(w http.ResponseWriter, status int, errCode, message string) {
	writeAlertsJSON(w, status, models.ErrorResponse{
		Error:   errCode,
		Message: message,
	})
}

func encodeJSON(w io.Writer, data interface{}) error {
	return json.NewEncoder(w).Encode(data)
}
