package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/telhawk-systems/telhawk-stack/alerting/internal/models"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/service"
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
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
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
	json.NewEncoder(w).Encode(c)
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
	json.NewEncoder(w).Encode(response)
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
	json.NewEncoder(w).Encode(c)
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "closed"})
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "reopened"})
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "alerts added"})
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"case_id": caseID,
		"alerts":  alerts,
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
