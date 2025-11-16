package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/telhawk-systems/telhawk-stack/common/httputil"

	"github.com/telhawk-systems/telhawk-stack/alerting/internal/models"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/service"
)

// StorageClient defines the interface for querying OpenSearch
type StorageClient interface {
	Query(method, path string, body []byte) ([]byte, error)
}

type Handler struct {
	service       *service.Service
	storageClient StorageClient
}

func NewHandler(service *service.Service, storageClient StorageClient) *Handler {
	return &Handler{
		service:       service,
		storageClient: storageClient,
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// CreateCase handles POST /api/v1/cases
func (h *Handler) CreateCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := "00000000-0000-0000-0000-000000000001" // Placeholder

	c, err := h.service.CreateCase(r.Context(), &req, userID)
	if err != nil {
		log.Printf("Error creating case: %v", err)
		http.Error(w, "Failed to create case", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	httputil.WriteJSON(w, http.StatusCreated, c)
}

// GetCase handles GET /api/v1/cases/:id
func (h *Handler) GetCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	id := r.URL.Path[len("/api/v1/cases/"):]
	if id == "" {
		http.Error(w, "Case ID required", http.StatusBadRequest)
		return
	}

	c, err := h.service.GetCase(r.Context(), id)
	if err != nil {
		log.Printf("Error getting case: %v", err)
		http.Error(w, "Case not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(c); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// ListCases handles GET /api/v1/cases
func (h *Handler) ListCases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	req := &models.ListCasesRequest{
		Page:     parseInt(r.URL.Query().Get("page"), 1),
		Limit:    parseInt(r.URL.Query().Get("limit"), 50),
		Status:   r.URL.Query().Get("status"),
		Severity: r.URL.Query().Get("severity"),
		Assignee: r.URL.Query().Get("assignee"),
	}

	response, err := h.service.ListCases(r.Context(), req)
	if err != nil {
		log.Printf("Error listing cases: %v", err)
		http.Error(w, "Failed to list cases", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// UpdateCase handles PUT /api/v1/cases/:id
func (h *Handler) UpdateCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	id := r.URL.Path[len("/api/v1/cases/"):]
	if id == "" {
		http.Error(w, "Case ID required", http.StatusBadRequest)
		return
	}

	var req models.UpdateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := "00000000-0000-0000-0000-000000000001" // Placeholder

	c, err := h.service.UpdateCase(r.Context(), id, &req, userID)
	if err != nil {
		log.Printf("Error updating case: %v", err)
		http.Error(w, "Failed to update case", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(c); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// CloseCase handles PUT /api/v1/cases/:id/close
func (h *Handler) CloseCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	path := r.URL.Path
	id := path[len("/api/v1/cases/"):]
	if len(id) > len("/close") {
		id = id[:len(id)-len("/close")]
	}

	// TODO: Extract user ID from JWT token
	userID := "00000000-0000-0000-0000-000000000001"

	if err := h.service.CloseCase(r.Context(), id, userID); err != nil {
		log.Printf("Error closing case: %v", err)
		http.Error(w, "Failed to close case", http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "closed"})
}

// ReopenCase handles PUT /api/v1/cases/:id/reopen
func (h *Handler) ReopenCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	path := r.URL.Path
	id := path[len("/api/v1/cases/"):]
	if len(id) > len("/reopen") {
		id = id[:len(id)-len("/reopen")]
	}

	if err := h.service.ReopenCase(r.Context(), id); err != nil {
		log.Printf("Error reopening case: %v", err)
		http.Error(w, "Failed to reopen case", http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "reopened"})
}

// AddAlertsToCase handles POST /api/v1/cases/:id/alerts
func (h *Handler) AddAlertsToCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract case ID from path
	path := r.URL.Path
	caseID := path[len("/api/v1/cases/"):]
	if len(caseID) > len("/alerts") {
		caseID = caseID[:len(caseID)-len("/alerts")]
	}

	var req models.AddAlertsToCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := "00000000-0000-0000-0000-000000000001"

	if err := h.service.AddAlertsToCase(r.Context(), caseID, &req, userID); err != nil {
		log.Printf("Error adding alerts to case: %v", err)
		http.Error(w, "Failed to add alerts to case", http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "alerts added"})
}

// GetCaseAlerts handles GET /api/v1/cases/:id/alerts
func (h *Handler) GetCaseAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract case ID from path
	path := r.URL.Path
	caseID := path[len("/api/v1/cases/"):]
	if len(caseID) > len("/alerts") {
		caseID = caseID[:len(caseID)-len("/alerts")]
	}

	alerts, err := h.service.GetCaseAlerts(r.Context(), caseID)
	if err != nil {
		log.Printf("Error getting case alerts: %v", err)
		http.Error(w, "Failed to get case alerts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"case_id": caseID,
		"alerts":  alerts,
	}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// ListAlerts handles GET /api/v1/alerts
func (h *Handler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	page := parseInt(r.URL.Query().Get("page"), 1)
	limit := parseInt(r.URL.Query().Get("limit"), 20)
	severity := r.URL.Query().Get("severity")
	detectionSchemaID := r.URL.Query().Get("detection_schema_id")
	// Note: status, case_id, and priority filtering can be added in the future

	// Build OpenSearch query
	mustClauses := []map[string]interface{}{}

	if severity != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{"severity.keyword": severity},
		})
	}

	if detectionSchemaID != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{"detection_schema_id.keyword": detectionSchemaID},
		})
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": mustClauses,
			},
		},
		"sort": []map[string]interface{}{
			{"time": map[string]string{"order": "desc"}},
		},
		"from": (page - 1) * limit,
		"size": limit,
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		log.Printf("Error building query: %v", err)
		http.Error(w, "Failed to build query", http.StatusInternalServerError)
		return
	}

	// Query OpenSearch
	respBody, err := h.storageClient.Query("POST", "/telhawk-alerts-*/_search", queryJSON)
	if err != nil {
		log.Printf("Error querying alerts: %v", err)
		http.Error(w, "Failed to query alerts", http.StatusInternalServerError)
		return
	}

	// Parse response
	var searchResp struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string                 `json:"_id"`
				Index  string                 `json:"_index"`
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.Unmarshal(respBody, &searchResp); err != nil {
		log.Printf("Error parsing response: %v", err)
		http.Error(w, "Failed to parse response", http.StatusInternalServerError)
		return
	}

	// Build response
	alerts := make([]map[string]interface{}, 0, len(searchResp.Hits.Hits))
	for _, hit := range searchResp.Hits.Hits {
		alert := hit.Source
		alert["_id"] = hit.ID
		alert["_index"] = hit.Index
		alerts = append(alerts, alert)
	}

	response := map[string]interface{}{
		"alerts": alerts,
		"total":  searchResp.Hits.Total.Value,
		"page":   page,
		"limit":  limit,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// GetAlert handles GET /api/v1/alerts/:id
func (h *Handler) GetAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	id := r.URL.Path[len("/api/v1/alerts/"):]
	if id == "" {
		http.Error(w, "Alert ID required", http.StatusBadRequest)
		return
	}

	// Query OpenSearch for the specific alert
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"ids": map[string]interface{}{
				"values": []string{id},
			},
		},
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		log.Printf("Error building query: %v", err)
		http.Error(w, "Failed to build query", http.StatusInternalServerError)
		return
	}

	respBody, err := h.storageClient.Query("POST", "/telhawk-alerts-*/_search", queryJSON)
	if err != nil {
		log.Printf("Error querying alert: %v", err)
		http.Error(w, "Failed to query alert", http.StatusInternalServerError)
		return
	}

	// Parse response
	var searchResp struct {
		Hits struct {
			Hits []struct {
				ID     string                 `json:"_id"`
				Index  string                 `json:"_index"`
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.Unmarshal(respBody, &searchResp); err != nil {
		log.Printf("Error parsing response: %v", err)
		http.Error(w, "Failed to parse response", http.StatusInternalServerError)
		return
	}

	if len(searchResp.Hits.Hits) == 0 {
		http.Error(w, "Alert not found", http.StatusNotFound)
		return
	}

	alert := searchResp.Hits.Hits[0].Source
	alert["_id"] = searchResp.Hits.Hits[0].ID
	alert["_index"] = searchResp.Hits.Hits[0].Index

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(alert); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
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
