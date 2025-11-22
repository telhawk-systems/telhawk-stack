// Package nats provides NATS message broker integration for the respond service.
package nats

import "time"

// CorrelationJobRequest is published to search.jobs.correlate to request
// the search service to evaluate a detection rule against the event data.
type CorrelationJobRequest struct {
	JobID           string                 `json:"job_id"`
	SchemaID        string                 `json:"schema_id"`
	SchemaVersionID string                 `json:"schema_version_id"`
	Query           string                 `json:"query"`
	TimeRange       TimeRange              `json:"time_range"`
	AggregationKey  string                 `json:"aggregation_key,omitempty"`
	Threshold       int                    `json:"threshold"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
}

// TimeRange represents a time window for correlation queries.
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// CorrelationJobResponse is received from search.results.correlate
// containing the results of a correlation evaluation.
type CorrelationJobResponse struct {
	JobID           string             `json:"job_id"`
	SchemaID        string             `json:"schema_id"`
	SchemaVersionID string             `json:"schema_version_id"`
	Success         bool               `json:"success"`
	Error           string             `json:"error,omitempty"`
	Triggered       bool               `json:"triggered"`
	MatchCount      int                `json:"match_count"`
	Matches         []CorrelationMatch `json:"matches,omitempty"`
	TookMs          int64              `json:"took_ms"`
}

// CorrelationMatch represents a single match from a correlation evaluation.
type CorrelationMatch struct {
	AggregationKey string                   `json:"aggregation_key"`
	EventCount     int                      `json:"event_count"`
	Events         []map[string]interface{} `json:"events,omitempty"`
	FirstSeen      time.Time                `json:"first_seen"`
	LastSeen       time.Time                `json:"last_seen"`
}

// AlertCreatedEvent is published to respond.alerts.created when a new alert
// is created from a triggered detection rule.
type AlertCreatedEvent struct {
	AlertID         string                 `json:"alert_id"`
	SchemaID        string                 `json:"schema_id"`
	SchemaVersionID string                 `json:"schema_version_id"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Severity        string                 `json:"severity"`
	TriggeredAt     time.Time              `json:"triggered_at"`
	EventCount      int                    `json:"event_count"`
	AggregationKey  string                 `json:"aggregation_key,omitempty"`
	Fields          map[string]interface{} `json:"fields,omitempty"`
}

// AlertUpdatedEvent is published to respond.alerts.updated when an alert
// status is changed.
type AlertUpdatedEvent struct {
	AlertID   string    `json:"alert_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

// CaseCreatedEvent is published to respond.cases.created when a new security
// case is opened.
type CaseCreatedEvent struct {
	CaseID      string    `json:"case_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Priority    string    `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	AlertCount  int       `json:"alert_count"`
}

// CaseUpdatedEvent is published to respond.cases.updated when a case status
// or other attributes are changed.
type CaseUpdatedEvent struct {
	CaseID    string    `json:"case_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

// CaseAssignedEvent is published to respond.cases.assigned when a case is
// assigned to an analyst.
type CaseAssignedEvent struct {
	CaseID     string    `json:"case_id"`
	AssigneeID string    `json:"assignee_id"`
	AssignedAt time.Time `json:"assigned_at"`
	AssignedBy string    `json:"assigned_by"`
}
