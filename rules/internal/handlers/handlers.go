package handlers

import (
	"encoding/json"
	"github.com/telhawk-systems/telhawk-stack/common/httputil"
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
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// GetCorrelationTypes handles GET /correlation/types
func (h *Handler) GetCorrelationTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	types := []map[string]interface{}{
		{
			"type":        "event_count",
			"name":        "Event Count",
			"description": "Alert when number of matching events exceeds threshold within time window",
			"category":    "aggregation",
			"tier":        1,
			"parameters": []map[string]interface{}{
				{
					"name":        "time_window",
					"type":        "duration",
					"required":    true,
					"description": "Lookback window (e.g., \"5m\", \"1h\")",
					"example":     "5m",
				},
				{
					"name":        "group_by",
					"type":        "array[string]",
					"required":    false,
					"description": "Fields to group by (per-entity counting)",
					"example":     []string{".actor.user.name", ".src_endpoint.ip"},
				},
			},
			"controller_parameters": []map[string]interface{}{
				{
					"name":        "threshold",
					"type":        "integer",
					"required":    true,
					"description": "Minimum event count to trigger",
					"example":     10,
				},
				{
					"name":        "operator",
					"type":        "string",
					"required":    false,
					"description": "Comparison operator (gt, gte, lt, lte, eq, ne)",
					"default":     "gt",
					"example":     "gt",
				},
			},
			"use_cases": []string{
				"Brute force detection (10+ failed logins in 5 minutes)",
				"DDoS detection (1000+ requests in 1 minute)",
				"Excessive file access (100+ files accessed in 10 minutes)",
			},
		},
		{
			"type":        "value_count",
			"name":        "Value Count (Cardinality)",
			"description": "Alert when number of distinct values exceeds threshold",
			"category":    "aggregation",
			"tier":        1,
			"parameters": []map[string]interface{}{
				{
					"name":        "time_window",
					"type":        "duration",
					"required":    true,
					"description": "Lookback window",
					"example":     "10m",
				},
				{
					"name":        "field",
					"type":        "string",
					"required":    true,
					"description": "Field to count distinct values",
					"example":     ".dst_endpoint.port",
				},
				{
					"name":        "group_by",
					"type":        "array[string]",
					"required":    false,
					"description": "Grouping fields",
					"example":     []string{".src_endpoint.ip"},
				},
			},
			"controller_parameters": []map[string]interface{}{
				{
					"name":        "threshold",
					"type":        "integer",
					"required":    true,
					"description": "Minimum distinct value count",
					"example":     100,
				},
				{
					"name":        "operator",
					"type":        "string",
					"required":    false,
					"description": "Comparison operator (gt, gte, lt, lte, eq, ne)",
					"default":     "gt",
					"example":     "gt",
				},
			},
			"use_cases": []string{
				"Password spray (1 user tries to login as 50+ different users)",
				"Port scanning (1 source hits 100+ destination ports)",
				"Data exfiltration (1 session accesses 1000+ unique files)",
			},
		},
		{
			"type":        "temporal",
			"name":        "Temporal Correlation (Unordered)",
			"description": "Alert when multiple events occur within time proximity (any order)",
			"category":    "multi-event",
			"tier":        1,
			"status":      "implemented",
			"parameters": []map[string]interface{}{
				{
					"name":        "time_window",
					"type":        "duration",
					"required":    true,
					"description": "Maximum time span for events",
					"example":     "5m",
				},
				{
					"name":        "queries",
					"type":        "array[query]",
					"required":    true,
					"description": "List of event queries to match (minimum 2)",
					"example":     "See documentation",
				},
				{
					"name":        "min_matches",
					"type":        "integer",
					"required":    false,
					"description": "Minimum queries that must match (default: all)",
					"example":     2,
				},
				{
					"name":        "group_by",
					"type":        "array[string]",
					"required":    false,
					"description": "Correlation key fields",
					"example":     []string{".actor.user.name"},
				},
			},
			"use_cases": []string{
				"Suspicious activity cluster (failed login AND file delete AND network connection within 5 min)",
				"Co-occurrence detection (A and B both happen, any order)",
			},
		},
		{
			"type":        "temporal_ordered",
			"name":        "Temporal Ordered (Sequence)",
			"description": "Alert when events occur in specific sequence within time window",
			"category":    "multi-event",
			"tier":        1,
			"status":      "implemented",
			"parameters": []map[string]interface{}{
				{
					"name":        "time_window",
					"type":        "duration",
					"required":    true,
					"description": "Maximum time between first and last",
					"example":     "30m",
				},
				{
					"name":        "sequence",
					"type":        "array[step]",
					"required":    true,
					"description": "Ordered list of event queries (minimum 2)",
					"example":     "See documentation",
				},
				{
					"name":        "max_gap",
					"type":        "duration",
					"required":    false,
					"description": "Maximum time between consecutive events",
					"example":     "10m",
				},
				{
					"name":        "group_by",
					"type":        "array[string]",
					"required":    false,
					"description": "Correlation key fields",
					"example":     []string{".actor.user.name"},
				},
			},
			"use_cases": []string{
				"Attack chain detection (recon → exploit → persistence)",
				"Multi-stage attack (privilege escalation → lateral movement → exfiltration)",
			},
		},
		{
			"type":        "join",
			"name":        "Join Correlation",
			"description": "Correlate events from different types/sources by matching field values",
			"category":    "multi-event",
			"tier":        1,
			"status":      "implemented",
			"use_cases": []string{
				"Failed auth followed by successful privileged action (compromised service account)",
				"DNS query matched with network connection (C2 detection)",
			},
		},
		{
			"type":        "suppression",
			"name":        "Alert Suppression",
			"description": "Alert deduplication and throttling (applies to any correlation type)",
			"category":    "meta",
			"tier":        1,
			"note":        "Configured in controller.suppression, not model.correlation_type",
			"parameters": []map[string]interface{}{
				{
					"name":        "enabled",
					"type":        "boolean",
					"required":    true,
					"description": "Enable alert suppression",
					"example":     true,
				},
				{
					"name":        "window",
					"type":        "duration",
					"required":    true,
					"description": "Suppression window duration",
					"example":     "1h",
				},
				{
					"name":        "key",
					"type":        "array[string]",
					"required":    true,
					"description": "Fields to group alerts by (suppression key)",
					"example":     []string{".actor.user.name", ".src_endpoint.ip"},
				},
				{
					"name":        "max_alerts",
					"type":        "integer",
					"required":    false,
					"description": "Max alerts per window per key (default: 1)",
					"default":     1,
					"example":     1,
				},
			},
		},
		{
			"type":        "baseline_deviation",
			"name":        "Baseline Deviation (Anomaly)",
			"description": "Alert when current behavior deviates from learned historical baseline",
			"category":    "ml",
			"tier":        1,
			"status":      "planned",
			"use_cases": []string{
				"User logs in at unusual time (normally 9-5, now at 3 AM)",
				"Process memory usage 5x higher than baseline",
				"File access volume 10x normal",
			},
		},
		{
			"type":        "missing_event",
			"name":        "Missing Event (Absence Detection)",
			"description": "Alert when expected event does NOT occur within expected interval",
			"category":    "absence",
			"tier":        1,
			"status":      "planned",
			"use_cases": []string{
				"Heartbeat monitoring (endpoint agent silent for 10 minutes)",
				"Scheduled job didn't run (backup expected every 24h)",
				"Log source went silent (no logs from firewall in 5 minutes)",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": types,
		"meta": map[string]interface{}{
			"total":       len(types),
			"implemented": 5,
			"planned":     len(types) - 5,
		},
	})
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
	httputil.WriteJSON(w, http.StatusCreated, map[string]interface{}{
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
		if err == service.ErrBuiltinRuleProtected {
			http.Error(w, "Builtin rules cannot be modified", http.StatusForbidden)
			return
		}
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
		if err == service.ErrBuiltinRuleProtected {
			http.Error(w, "Builtin rules cannot be disabled", http.StatusForbidden)
			return
		}
		log.Printf("Error disabling schema: %v", err)
		http.Error(w, "Failed to disable schema", http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
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

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
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
		if err == service.ErrBuiltinRuleProtected {
			http.Error(w, "Builtin rules cannot be deleted", http.StatusForbidden)
			return
		}
		log.Printf("Error hiding schema: %v", err)
		http.Error(w, "Failed to hide schema", http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "hidden"})
}

// SetActiveParameterSet handles PUT /schemas/:id/parameters
func (h *Handler) SetActiveParameterSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract version ID from path
	path := r.URL.Path
	versionID := path[len("/schemas/"):]
	if len(versionID) > len("/parameters") {
		versionID = versionID[:len(versionID)-len("/parameters")]
	}

	// Parse request body
	var req struct {
		ActiveParameterSet string `json:"active_parameter_set"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ActiveParameterSet == "" {
		http.Error(w, "active_parameter_set is required", http.StatusBadRequest)
		return
	}

	// Update active parameter set
	if err := h.service.SetActiveParameterSet(r.Context(), versionID, req.ActiveParameterSet); err != nil {
		log.Printf("Error setting active parameter set: %v", err)
		http.Error(w, "Failed to set active parameter set", http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":               "updated",
		"active_parameter_set": req.ActiveParameterSet,
	})
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
