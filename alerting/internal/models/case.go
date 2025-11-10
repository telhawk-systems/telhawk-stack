package models

import "time"

// Case represents a security case for investigation
type Case struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Severity    string     `json:"severity"` // info, low, medium, high, critical
	Status      string     `json:"status"`   // open, in_progress, resolved, closed
	Assignee    *string    `json:"assignee,omitempty"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	ClosedBy    *string    `json:"closed_by,omitempty"`
	AlertCount  int        `json:"alert_count,omitempty"` // Calculated field
}

// CaseAlert represents the association between a case and an alert
type CaseAlert struct {
	CaseID                   string    `json:"case_id"`
	AlertID                  string    `json:"alert_id"` // OpenSearch document ID
	DetectionSchemaID        string    `json:"detection_schema_id"`
	DetectionSchemaVersionID string    `json:"detection_schema_version_id"`
	AddedAt                  time.Time `json:"added_at"`
	AddedBy                  string    `json:"added_by"`
}

// CreateCaseRequest represents the request to create a new case
type CreateCaseRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Severity    string  `json:"severity"` // info, low, medium, high, critical
	Assignee    *string `json:"assignee,omitempty"`
}

// UpdateCaseRequest represents the request to update a case
type UpdateCaseRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Severity    *string `json:"severity,omitempty"`
	Status      *string `json:"status,omitempty"`
	Assignee    *string `json:"assignee,omitempty"`
}

// AddAlertsToCaseRequest represents the request to add alerts to a case
type AddAlertsToCaseRequest struct {
	AlertIDs []string `json:"alert_ids"`
}

// ListCasesRequest represents query parameters for listing cases
type ListCasesRequest struct {
	Page     int
	Limit    int
	Status   string
	Severity string
	Assignee string
}

// ListCasesResponse represents the response for listing cases
type ListCasesResponse struct {
	Cases      []*Case    `json:"cases"`
	Pagination Pagination `json:"pagination"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
