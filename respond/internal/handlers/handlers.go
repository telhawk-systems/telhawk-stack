// Package handlers provides HTTP request handlers for the respond service.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/common/httputil"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/auth"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/models"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/service"
)

// Handler provides HTTP handlers for the respond service
type Handler struct {
	svc        *service.Service
	authClient *auth.Client
}

// NewHandler creates a new Handler instance
func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// WithAuthClient sets the auth client for token validation
func (h *Handler) WithAuthClient(client *auth.Client) *Handler {
	h.authClient = client
	return h
}

// =============================================================================
// Helper Methods
// =============================================================================

// getUserIDFromRequest extracts user ID from authenticated request
func (h *Handler) getUserIDFromRequest(r *http.Request) (string, error) {
	if h.authClient == nil {
		return "", errors.New("auth client not configured")
	}

	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return "", errors.New("missing bearer token")
	}

	token := strings.TrimSpace(authz[len("Bearer "):])
	uc, err := h.authClient.Validate(r.Context(), token)
	if err != nil {
		return "", err
	}

	if uc.UserID == "" {
		return "", errors.New("user ID not found in token")
	}

	return uc.UserID, nil
}

// extractIDFromPath extracts an ID from a URL path like /schemas/{id} or /api/v1/cases/{id}
func extractIDFromPath(path, prefix string) string {
	// Remove prefix and get remaining path
	remaining := strings.TrimPrefix(path, prefix)
	remaining = strings.TrimPrefix(remaining, "/")

	// Get the first segment (the ID)
	parts := strings.Split(remaining, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// =============================================================================
// Health Check Handlers
// =============================================================================

// HealthCheck handles GET /healthz
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSONAPI(w, http.StatusOK, models.HealthResponse{
		Status:  "ok",
		Service: "respond",
	})
}

// ReadyCheck handles GET /readyz
func (h *Handler) ReadyCheck(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSONAPI(w, http.StatusOK, models.HealthResponse{
		Status:  "ready",
		Service: "respond",
	})
}

// =============================================================================
// Detection Schema Handlers
// =============================================================================

// SchemasHandler handles /schemas routes
func (h *Handler) SchemasHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListSchemas(w, r)
	case http.MethodPost:
		h.CreateSchema(w, r)
	default:
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
	}
}

// SchemaHandler handles /schemas/{id} routes
func (h *Handler) SchemaHandler(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path, "/schemas")
	if id == "" {
		httputil.WriteJSONAPIValidationError(w, "Schema ID required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.GetSchema(w, r, id)
	case http.MethodPut:
		h.UpdateSchema(w, r, id)
	case http.MethodDelete:
		h.HideSchema(w, r, id)
	default:
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
	}
}

// CreateSchema handles POST /schemas
func (h *Handler) CreateSchema(w http.ResponseWriter, r *http.Request) {
	var req models.CreateSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONAPIValidationError(w, "Invalid request body")
		return
	}

	schema, err := h.svc.CreateSchema(r.Context(), &req)
	if err != nil {
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusCreated, schema)
}

// GetSchema handles GET /schemas/{id}
func (h *Handler) GetSchema(w http.ResponseWriter, r *http.Request, id string) {
	schema, err := h.svc.GetSchema(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrSchemaNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "detection_schema", id)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, schema)
}

// ListSchemas handles GET /schemas
func (h *Handler) ListSchemas(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))   //nolint:errcheck // defaults to 0 on error, handled by service
	limit, _ := strconv.Atoi(q.Get("limit")) //nolint:errcheck // defaults to 0 on error, handled by service

	req := &models.ListSchemasRequest{
		Page:            page,
		Limit:           limit,
		Severity:        q.Get("severity"),
		Title:           q.Get("title"),
		ID:              q.Get("id"),
		IncludeDisabled: q.Get("include_disabled") == "true",
		IncludeHidden:   q.Get("include_hidden") == "true",
	}

	resp, err := h.svc.ListSchemas(r.Context(), req)
	if err != nil {
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, resp)
}

// UpdateSchema handles PUT /schemas/{id}
func (h *Handler) UpdateSchema(w http.ResponseWriter, r *http.Request, id string) {
	var req models.UpdateSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONAPIValidationError(w, "Invalid request body")
		return
	}

	schema, err := h.svc.UpdateSchema(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, repository.ErrSchemaNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "detection_schema", id)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, schema)
}

// HideSchema handles DELETE /schemas/{id}
func (h *Handler) HideSchema(w http.ResponseWriter, r *http.Request, id string) {
	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		httputil.WriteJSONAPIError(w, http.StatusUnauthorized, "auth_required", "Authentication required", "User ID not found in context")
		return
	}

	if err := h.svc.HideSchema(r.Context(), id, userID); err != nil {
		if errors.Is(err, repository.ErrSchemaNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "detection_schema", id)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, map[string]interface{}{
		"id":        id,
		"hidden_at": "now",
	})
}

// DisableSchemaHandler handles PUT /schemas/{id}/disable
func (h *Handler) DisableSchemaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/schemas")

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		httputil.WriteJSONAPIError(w, http.StatusUnauthorized, "auth_required", "Authentication required", "User ID not found in context")
		return
	}

	schema, err := h.svc.DisableSchema(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrSchemaNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "detection_schema", id)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, schema)
}

// EnableSchemaHandler handles PUT /schemas/{id}/enable
func (h *Handler) EnableSchemaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/schemas")

	schema, err := h.svc.EnableSchema(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrSchemaNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "detection_schema", id)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, schema)
}

// VersionHistoryHandler handles GET /schemas/{id}/versions
func (h *Handler) VersionHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/schemas")

	resp, err := h.svc.GetVersionHistory(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrSchemaNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "detection_schema", id)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, resp)
}

// SetParameterSetHandler handles PUT /schemas/{id}/parameters
func (h *Handler) SetParameterSetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/schemas")

	var req struct {
		ParameterSet string `json:"parameter_set"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONAPIValidationError(w, "Invalid request body")
		return
	}

	schema, err := h.svc.SetActiveParameterSet(r.Context(), id, req.ParameterSet)
	if err != nil {
		if errors.Is(err, repository.ErrSchemaNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "detection_schema", id)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, schema)
}

// =============================================================================
// Case Handlers
// =============================================================================

// CasesHandler handles /api/v1/cases routes
func (h *Handler) CasesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListCases(w, r)
	case http.MethodPost:
		h.CreateCase(w, r)
	default:
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
	}
}

// CaseHandler handles /api/v1/cases/{id} routes
func (h *Handler) CaseHandler(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path, "/api/v1/cases")
	if id == "" {
		httputil.WriteJSONAPIValidationError(w, "Case ID required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.GetCase(w, r, id)
	case http.MethodPut:
		h.UpdateCase(w, r, id)
	default:
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
	}
}

// CreateCase handles POST /api/v1/cases
func (h *Handler) CreateCase(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONAPIValidationError(w, "Invalid request body")
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		httputil.WriteJSONAPIError(w, http.StatusUnauthorized, "auth_required", "Authentication required", "User ID not found in context")
		return
	}

	c, err := h.svc.CreateCase(r.Context(), &req, userID)
	if err != nil {
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusCreated, c)
}

// GetCase handles GET /api/v1/cases/{id}
func (h *Handler) GetCase(w http.ResponseWriter, r *http.Request, id string) {
	c, err := h.svc.GetCase(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrCaseNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "case", id)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, c)
}

// ListCases handles GET /api/v1/cases
func (h *Handler) ListCases(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))   //nolint:errcheck // defaults to 0 on error, handled by service
	limit, _ := strconv.Atoi(q.Get("limit")) //nolint:errcheck // defaults to 0 on error, handled by service

	req := &models.ListCasesRequest{
		Page:       page,
		Limit:      limit,
		Status:     q.Get("status"),
		Priority:   q.Get("priority"),
		AssigneeID: q.Get("assignee_id"),
	}

	resp, err := h.svc.ListCases(r.Context(), req)
	if err != nil {
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, resp)
}

// UpdateCase handles PUT /api/v1/cases/{id}
func (h *Handler) UpdateCase(w http.ResponseWriter, r *http.Request, id string) {
	var req models.UpdateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONAPIValidationError(w, "Invalid request body")
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		httputil.WriteJSONAPIError(w, http.StatusUnauthorized, "auth_required", "Authentication required", "User ID not found in context")
		return
	}

	c, err := h.svc.UpdateCase(r.Context(), id, &req, userID)
	if err != nil {
		if errors.Is(err, repository.ErrCaseNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "case", id)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, c)
}

// CloseCaseHandler handles PUT /api/v1/cases/{id}/close
func (h *Handler) CloseCaseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/v1/cases")

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		httputil.WriteJSONAPIError(w, http.StatusUnauthorized, "auth_required", "Authentication required", "User ID not found in context")
		return
	}

	c, err := h.svc.CloseCase(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrCaseNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "case", id)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, c)
}

// ReopenCaseHandler handles PUT /api/v1/cases/{id}/reopen
func (h *Handler) ReopenCaseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/v1/cases")

	c, err := h.svc.ReopenCase(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrCaseNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "case", id)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, c)
}

// CaseAlertsHandler handles /api/v1/cases/{id}/alerts routes
func (h *Handler) CaseAlertsHandler(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path, "/api/v1/cases")

	switch r.Method {
	case http.MethodGet:
		h.GetCaseAlerts(w, r, id)
	case http.MethodPost:
		h.AddAlertsToCase(w, r, id)
	default:
		httputil.WriteJSONAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method Not Allowed", "")
	}
}

// GetCaseAlerts handles GET /api/v1/cases/{id}/alerts
func (h *Handler) GetCaseAlerts(w http.ResponseWriter, r *http.Request, caseID string) {
	alerts, err := h.svc.GetCaseAlerts(r.Context(), caseID)
	if err != nil {
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, map[string]interface{}{
		"case_id": caseID,
		"alerts":  alerts,
	})
}

// AddAlertsToCase handles POST /api/v1/cases/{id}/alerts
func (h *Handler) AddAlertsToCase(w http.ResponseWriter, r *http.Request, caseID string) {
	var req models.AddAlertsToCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONAPIValidationError(w, "Invalid request body")
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		httputil.WriteJSONAPIError(w, http.StatusUnauthorized, "auth_required", "Authentication required", "User ID not found in context")
		return
	}

	if err := h.svc.AddAlertsToCase(r.Context(), caseID, req.AlertIDs, userID); err != nil {
		if errors.Is(err, repository.ErrCaseNotFound) {
			httputil.WriteJSONAPINotFoundError(w, "case", caseID)
			return
		}
		httputil.WriteJSONAPIInternalError(w, err.Error())
		return
	}

	httputil.WriteJSONAPI(w, http.StatusOK, map[string]interface{}{
		"case_id":   caseID,
		"added":     len(req.AlertIDs),
		"alert_ids": req.AlertIDs,
	})
}
