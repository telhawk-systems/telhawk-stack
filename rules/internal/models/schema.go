package models

import "time"

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

// Pagination metadata for list responses
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
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
	ID               string         `json:"id"`
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
