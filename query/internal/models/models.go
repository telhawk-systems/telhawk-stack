package models

import "time"

// TimeRange bounds a query using RFC3339 timestamps.
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// SortOptions defines sort field and direction for search results.
type SortOptions struct {
	Field string `json:"field"`
	Order string `json:"order"`
}

// SearchRequest captures the SPL query and optional constraints.
type SearchRequest struct {
	Query         string                   `json:"query"`
	TimeRange     *TimeRange               `json:"time_range,omitempty"`
	Limit         int                      `json:"limit,omitempty"`
	Sort          *SortOptions             `json:"sort,omitempty"`
	IncludeFields []string                 `json:"include_fields,omitempty"`
	SearchAfter   []interface{}            `json:"search_after,omitempty"`
	Aggregations  map[string]AggregationRequest `json:"aggregations,omitempty"`
}

// AggregationRequest defines an aggregation to compute.
type AggregationRequest struct {
	Type  string                 `json:"type"`
	Field string                 `json:"field,omitempty"`
	Size  int                    `json:"size,omitempty"`
	Opts  map[string]interface{} `json:"opts,omitempty"`
}

// SearchResponse is returned after executing a search.
type SearchResponse struct {
	RequestID    string                   `json:"request_id"`
	LatencyMS    int                      `json:"latency_ms"`
	ResultCount  int                      `json:"result_count"`
	TotalMatches int                      `json:"total_matches,omitempty"`
	Results      []map[string]interface{} `json:"results"`
	SearchAfter  []interface{}            `json:"search_after,omitempty"`
	Aggregations map[string]interface{}   `json:"aggregations,omitempty"`
}

// AlertSchedule controls when an alert runs and its lookback window.
type AlertSchedule struct {
	IntervalMinutes int `json:"interval_minutes"`
	LookbackMinutes int `json:"lookback_minutes"`
}

// Alert represents a saved query that emits notifications when triggered.
type Alert struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description,omitempty"`
	Query           string        `json:"query"`
	Severity        string        `json:"severity"`
	Schedule        AlertSchedule `json:"schedule"`
	Status          string        `json:"status"`
	LastTriggeredAt *time.Time    `json:"last_triggered_at,omitempty"`
	Owner           string        `json:"owner,omitempty"`
}

// AlertRequest is used to create or update an alert definition.
type AlertRequest struct {
	ID          *string       `json:"id,omitempty"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Query       string        `json:"query"`
	Severity    string        `json:"severity"`
	Schedule    AlertSchedule `json:"schedule"`
	Status      string        `json:"status,omitempty"`
	Owner       string        `json:"owner,omitempty"`
}

// AlertPatchRequest allows partial updates of alert metadata.
type AlertPatchRequest struct {
	Status string `json:"status,omitempty"`
	Owner  string `json:"owner,omitempty"`
}

// AlertListResponse wraps a slice of alerts with pagination metadata.
type AlertListResponse struct {
	Alerts     []Alert `json:"alerts"`
	NextCursor *string `json:"next_cursor,omitempty"`
}

// Dashboard aggregates visual widgets for the SOC UI.
type Dashboard struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Widgets     []map[string]interface{} `json:"widgets"`
}

// DashboardListResponse contains dashboards available to the caller.
type DashboardListResponse struct {
	Dashboards []Dashboard `json:"dashboards"`
}

// ExportRequest represents a bulk export job definition.
type ExportRequest struct {
	Query               string     `json:"query"`
	TimeRange           *TimeRange `json:"time_range,omitempty"`
	Format              string     `json:"format"`
	Compression         string     `json:"compression,omitempty"`
	NotificationChannel string     `json:"notification_channel,omitempty"`
}

// ExportResponse is returned when an export job is created.
type ExportResponse struct {
	ExportID  string    `json:"export_id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ErrorResponse formalizes error messages returned to clients.
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// HealthResponse is emitted for liveness probes.
type HealthResponse struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}
