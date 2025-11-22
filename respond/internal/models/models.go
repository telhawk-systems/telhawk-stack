// Package models provides data models for the respond service.
package models

import "time"

// =============================================================================
// Detection Schema Models (from rules service)
// =============================================================================

// DetectionSchema represents a versioned detection rule
type DetectionSchema struct {
	ID         string                 `json:"id"`         // Stable rule identifier
	VersionID  string                 `json:"version_id"` // Version-specific UUID (UUID v7)
	Model      map[string]interface{} `json:"model"`      // Data model and aggregation config
	View       map[string]interface{} `json:"view"`       // Presentation and display config
	Controller map[string]interface{} `json:"controller"` // Detection logic and evaluation
	CreatedBy  string                 `json:"created_by"`
	CreatedAt  time.Time              `json:"created_at"`
	DisabledAt *time.Time             `json:"disabled_at,omitempty"`
	DisabledBy *string                `json:"disabled_by,omitempty"`
	HiddenAt   *time.Time             `json:"hidden_at,omitempty"`
	HiddenBy   *string                `json:"hidden_by,omitempty"`
	Version    int                    `json:"version,omitempty"` // Calculated by ROW_NUMBER()
}

// IsActive returns true if schema is not disabled or hidden
func (s *DetectionSchema) IsActive() bool {
	return s.DisabledAt == nil && s.HiddenAt == nil
}

// CreateSchemaRequest is the API request for creating a new detection schema
type CreateSchemaRequest struct {
	ID         string                 `json:"id,omitempty"` // Optional: for builtin rules with deterministic IDs
	Model      map[string]interface{} `json:"model"`
	View       map[string]interface{} `json:"view"`
	Controller map[string]interface{} `json:"controller"`
}

// UpdateSchemaRequest is the API request for updating a schema (creates new version)
type UpdateSchemaRequest struct {
	Model      map[string]interface{} `json:"model"`
	View       map[string]interface{} `json:"view"`
	Controller map[string]interface{} `json:"controller"`
}

// ListSchemasRequest contains filters for listing schemas
type ListSchemasRequest struct {
	Page            int    `json:"page"`
	Limit           int    `json:"limit"`
	Severity        string `json:"severity,omitempty"`
	Title           string `json:"title,omitempty"`
	ID              string `json:"id,omitempty"` // Filter by stable rule ID
	IncludeDisabled bool   `json:"include_disabled"`
	IncludeHidden   bool   `json:"include_hidden"`
}

// ListSchemasResponse contains paginated schema results
type ListSchemasResponse struct {
	Schemas    []*DetectionSchema `json:"schemas"`
	Pagination Pagination         `json:"pagination"`
}

// VersionHistoryResponse contains all versions of a detection schema
type VersionHistoryResponse struct {
	ID       string                    `json:"id"`
	Title    string                    `json:"title"`
	Versions []*DetectionSchemaVersion `json:"versions"`
}

// DetectionSchemaVersion represents a single version in history
type DetectionSchemaVersion struct {
	VersionID  string     `json:"version_id"`
	Version    int        `json:"version"`
	Title      string     `json:"title"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	DisabledAt *time.Time `json:"disabled_at,omitempty"`
	Changes    string     `json:"changes,omitempty"`
}

// TestSchemaRequest is the API request for testing a detection schema
type TestSchemaRequest struct {
	TimeRange TimeRange `json:"time_range"`
	DryRun    bool      `json:"dry_run"`
}

// TimeRange represents a time window for queries
type TimeRange struct {
	From string `json:"from"` // ISO8601 timestamp
	To   string `json:"to"`   // ISO8601 timestamp
}

// TestSchemaResponse contains results from testing a detection schema
type TestSchemaResponse struct {
	SchemaID         string         `json:"schema_id"`
	VersionID        string         `json:"version_id"`
	SchemaTitle      string         `json:"schema_title"`
	TimeRange        TimeRange      `json:"time_range"`
	WouldTrigger     bool           `json:"would_trigger"`
	TriggerCount     int            `json:"trigger_count"`
	Triggers         []AlertTrigger `json:"triggers,omitempty"`
	TotalEventsMatch int            `json:"total_events_matched"`
	EvaluationMs     int            `json:"evaluation_duration_ms"`
}

// AlertTrigger represents a single alert that would have been triggered
type AlertTrigger struct {
	TriggeredAt    string                 `json:"triggered_at"`
	AggregationKey string                 `json:"aggregation_key"`
	EventCount     int                    `json:"event_count"`
	Fields         map[string]interface{} `json:"fields"`
}

// =============================================================================
// Case Models (from alerting service)
// =============================================================================

// Case represents a security case for investigation
type Case struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Severity    string     `json:"severity"` // info, low, medium, high, critical
	Status      string     `json:"status"`   // open, in_progress, resolved, closed
	Priority    string     `json:"priority"` // low, medium, high, critical
	AssigneeID  *string    `json:"assignee_id,omitempty"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	ClosedBy    *string    `json:"closed_by,omitempty"`
	AlertCount  int        `json:"alert_count,omitempty"` // Calculated field

	// Links to detection schema that triggered the case
	DetectionSchemaID        *string `json:"detection_schema_id,omitempty"`
	DetectionSchemaVersionID *string `json:"detection_schema_version_id,omitempty"`
}

// CaseAlert represents the association between a case and an alert
type CaseAlert struct {
	ID                       string    `json:"id"`
	CaseID                   string    `json:"case_id"`
	AlertID                  string    `json:"alert_id"` // OpenSearch document ID
	DetectionSchemaID        string    `json:"detection_schema_id,omitempty"`
	DetectionSchemaVersionID string    `json:"detection_schema_version_id,omitempty"`
	AddedAt                  time.Time `json:"added_at"`
	AddedBy                  string    `json:"added_by"`
}

// CreateCaseRequest represents the request to create a new case
type CreateCaseRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Severity    string   `json:"severity"` // info, low, medium, high, critical
	Priority    string   `json:"priority"` // low, medium, high, critical
	AssigneeID  *string  `json:"assignee_id,omitempty"`
	AlertIDs    []string `json:"alert_ids,omitempty"` // Optional alerts to add immediately
}

// UpdateCaseRequest represents the request to update a case
type UpdateCaseRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Severity    *string `json:"severity,omitempty"`
	Status      *string `json:"status,omitempty"`
	Priority    *string `json:"priority,omitempty"`
	AssigneeID  *string `json:"assignee_id,omitempty"`
}

// AddAlertsToCaseRequest represents the request to add alerts to a case
type AddAlertsToCaseRequest struct {
	AlertIDs []string `json:"alert_ids"`
}

// ListCasesRequest represents query parameters for listing cases
type ListCasesRequest struct {
	Page       int
	Limit      int
	Status     string
	Severity   string
	Priority   string
	AssigneeID string
}

// ListCasesResponse represents the response for listing cases
type ListCasesResponse struct {
	Cases      []*Case    `json:"cases"`
	Pagination Pagination `json:"pagination"`
}

// =============================================================================
// Alert Models (stored in OpenSearch but queried via this service)
// =============================================================================

// Alert represents a security alert generated by a detection schema
type Alert struct {
	AlertID                  string                 `json:"alert_id"`
	DetectionSchemaID        string                 `json:"detection_schema_id"`
	DetectionSchemaVersionID string                 `json:"detection_schema_version_id"`
	DetectionSchemaTitle     string                 `json:"detection_schema_title"`
	CaseID                   *string                `json:"case_id,omitempty"`
	Title                    string                 `json:"title"`
	Description              string                 `json:"description"`
	Severity                 string                 `json:"severity"`
	Priority                 string                 `json:"priority"`
	Status                   string                 `json:"status"` // open, investigating, resolved, false_positive
	TriggeredAt              time.Time              `json:"triggered_at"`
	EventCount               int                    `json:"event_count"`
	MatchedEvents            []string               `json:"matched_events,omitempty"`
	Fields                   map[string]interface{} `json:"fields,omitempty"`
	MitreAttack              *MitreAttack           `json:"mitre_attack,omitempty"`
}

// MitreAttack contains MITRE ATT&CK metadata
type MitreAttack struct {
	Tactics    []string `json:"tactics,omitempty"`
	Techniques []string `json:"techniques,omitempty"`
}

// ListAlertsRequest represents query parameters for listing alerts
type ListAlertsRequest struct {
	Page                     int
	Limit                    int
	Severity                 string
	Status                   string
	Priority                 string
	From                     *time.Time
	To                       *time.Time
	DetectionSchemaID        string
	DetectionSchemaVersionID string
	CaseID                   string
}

// ListAlertsResponse represents the response for listing alerts
type ListAlertsResponse struct {
	Alerts     []*Alert   `json:"alerts"`
	Pagination Pagination `json:"pagination"`
}

// UpdateAlertRequest represents the request to update an alert
type UpdateAlertRequest struct {
	Status     *string `json:"status,omitempty"`
	AssignedTo *string `json:"assigned_to,omitempty"`
	Notes      *string `json:"notes,omitempty"`
}

// =============================================================================
// Shared Types
// =============================================================================

// Pagination metadata for list responses
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error      string                 `json:"error"`
	Message    string                 `json:"message"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// Case status constants
const (
	CaseStatusOpen       = "open"
	CaseStatusInProgress = "in_progress"
	CaseStatusResolved   = "resolved"
	CaseStatusClosed     = "closed"
)

// Case priority constants
const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

// Severity constants
const (
	SeverityInfo          = "info"
	SeverityLow           = "low"
	SeverityMedium        = "medium"
	SeverityHigh          = "high"
	SeverityCritical      = "critical"
	SeverityInformational = "informational"
)

// Alert status constants
const (
	AlertStatusOpen          = "open"
	AlertStatusInvestigating = "investigating"
	AlertStatusResolved      = "resolved"
	AlertStatusFalsePositive = "false_positive"
)
