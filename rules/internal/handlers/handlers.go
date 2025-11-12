package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/telhawk-systems/telhawk-stack/rules/internal/models"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/service"
)

type Handler struct {
	service *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{service: service}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// CreateSchema handles POST /api/v1/schemas
func (h *Handler) CreateSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := "00000000-0000-0000-0000-000000000001" // Placeholder

	schema, err := h.service.CreateSchema(r.Context(), &req, userID)
	if err != nil {
		log.Printf("Error creating schema: %v", err)
		http.Error(w, "Failed to create schema", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": toJSONAPIResource(schema),
	})
}

// UpdateSchema handles PUT /schemas/:id
func (h *Handler) UpdateSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	id := r.URL.Path[len("/schemas/"):]
	if id == "" {
		http.Error(w, "Schema ID required", http.StatusBadRequest)
		return
	}

	var req models.UpdateSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := "00000000-0000-0000-0000-000000000001" // Placeholder

	schema, err := h.service.UpdateSchema(r.Context(), id, &req, userID)
	if err != nil {
		log.Printf("Error updating schema: %v", err)
		http.Error(w, "Failed to update schema", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": toJSONAPIResource(schema),
	})
}

// ListSchemas handles GET /api/v1/schemas
func (h *Handler) ListSchemas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	req := &models.ListSchemasRequest{
		Page:            parseInt(r.URL.Query().Get("page"), 1),
		Limit:           parseInt(r.URL.Query().Get("limit"), 50),
		Severity:        r.URL.Query().Get("severity"),
		Title:           r.URL.Query().Get("title"),
		ID:              r.URL.Query().Get("id"),
		IncludeDisabled: r.URL.Query().Get("include_disabled") == "true",
		IncludeHidden:   r.URL.Query().Get("include_hidden") == "true",
	}

	response, err := h.service.ListSchemas(r.Context(), req)
	if err != nil {
		log.Printf("Error listing schemas: %v", err)
		http.Error(w, "Failed to list schemas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(toJSONAPICollection(response.Schemas, &response.Pagination))
}

// GetSchema handles GET /schemas/:id
func (h *Handler) GetSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path (simple implementation)
	id := r.URL.Path[len("/schemas/"):]
	if id == "" {
		http.Error(w, "Schema ID required", http.StatusBadRequest)
		return
	}

	// Check for version query parameter
	var version *int
	if v := r.URL.Query().Get("version"); v != "" {
		if vInt, err := strconv.Atoi(v); err == nil {
			version = &vInt
		}
	}

	schema, err := h.service.GetSchema(r.Context(), id, version)
	if err != nil {
		log.Printf("Error getting schema: %v", err)
		http.Error(w, "Schema not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": toJSONAPIResource(schema),
	})
}

// GetVersionHistory handles GET /schemas/:id/versions
func (h *Handler) GetVersionHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	path := r.URL.Path
	id := path[len("/schemas/"):]
	if len(id) > len("/versions") {
		id = id[:len(id)-len("/versions")]
	}

	response, err := h.service.GetVersionHistory(r.Context(), id)
	if err != nil {
		log.Printf("Error getting version history: %v", err)
		http.Error(w, "Schema not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(toJSONAPIVersionCollection(response.Versions))
}

// DisableSchema handles PUT /schemas/:id/disable
func (h *Handler) DisableSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract version ID from path
	path := r.URL.Path
	versionID := path[len("/schemas/"):]
	if len(versionID) > len("/disable") {
		versionID = versionID[:len(versionID)-len("/disable")]
	}

	// TODO: Extract user ID from JWT token
	userID := "00000000-0000-0000-0000-000000000001"

	if err := h.service.DisableSchema(r.Context(), versionID, userID); err != nil {
		log.Printf("Error disabling schema: %v", err)
		http.Error(w, "Failed to disable schema", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "disabled"})
}

// EnableSchema handles PUT /schemas/:id/enable
func (h *Handler) EnableSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract version ID from path
	path := r.URL.Path
	versionID := path[len("/schemas/"):]
	if len(versionID) > len("/enable") {
		versionID = versionID[:len(versionID)-len("/enable")]
	}

	if err := h.service.EnableSchema(r.Context(), versionID); err != nil {
		log.Printf("Error enabling schema: %v", err)
		http.Error(w, "Failed to enable schema", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "enabled"})
}

// HideSchema handles DELETE /schemas/:id
func (h *Handler) HideSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract version ID from path
	versionID := r.URL.Path[len("/schemas/"):]

	// TODO: Extract user ID from JWT token
	userID := "00000000-0000-0000-0000-000000000001"

	if err := h.service.HideSchema(r.Context(), versionID, userID); err != nil {
		log.Printf("Error hiding schema: %v", err)
		http.Error(w, "Failed to hide schema", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "hidden"})
}

// Helper function to parse integer query parameters
func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultVal
}

// JSON:API response helpers
func toJSONAPIResource(schema *models.DetectionSchema) map[string]interface{} {
	return map[string]interface{}{
		"type": "detection-schema",
		"id":   schema.ID,
		"attributes": map[string]interface{}{
			"version_id":  schema.VersionID,
			"model":       schema.Model,
			"view":        schema.View,
			"controller":  schema.Controller,
			"created_by":  schema.CreatedBy,
			"created_at":  schema.CreatedAt,
			"disabled_at": schema.DisabledAt,
			"disabled_by": schema.DisabledBy,
			"hidden_at":   schema.HiddenAt,
			"hidden_by":   schema.HiddenBy,
			"version":     schema.Version,
		},
	}
}

func toJSONAPICollection(schemas []*models.DetectionSchema, pagination *models.Pagination) map[string]interface{} {
	data := make([]map[string]interface{}, len(schemas))
	for i, schema := range schemas {
		data[i] = toJSONAPIResource(schema)
	}

	response := map[string]interface{}{
		"data": data,
	}

	if pagination != nil {
		response["meta"] = map[string]interface{}{
			"pagination": pagination,
		}
	}

	return response
}

func toJSONAPIVersionCollection(versions []*models.DetectionSchemaVersion) map[string]interface{} {
	data := make([]map[string]interface{}, len(versions))
	for i, version := range versions {
		data[i] = map[string]interface{}{
			"type": "detection-schema-version",
			"id":   version.VersionID,
			"attributes": map[string]interface{}{
				"version":     version.Version,
				"title":       version.Title,
				"created_by":  version.CreatedBy,
				"created_at":  version.CreatedAt,
				"disabled_at": version.DisabledAt,
				"changes":     version.Changes,
			},
		}
	}

	return map[string]interface{}{
		"data": data,
	}
}
