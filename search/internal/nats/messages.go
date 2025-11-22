// Package nats provides NATS message handling for the search service.
package nats

import "time"

// SearchJobRequest is the message format for search.jobs.query subject.
// It represents a request to execute an ad-hoc search query.
type SearchJobRequest struct {
	JobID     string                 `json:"job_id"`
	Query     string                 `json:"query"`
	TimeRange TimeRange              `json:"time_range"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	From      int                    `json:"from,omitempty"`
}

// TimeRange defines the time boundaries for a search.
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// SearchJobResponse is the message format for search results.
// It is published to reply subjects after processing a SearchJobRequest.
type SearchJobResponse struct {
	JobID     string                   `json:"job_id"`
	Success   bool                     `json:"success"`
	Error     string                   `json:"error,omitempty"`
	TotalHits int64                    `json:"total_hits"`
	Events    []map[string]interface{} `json:"events"`
	TookMs    int64                    `json:"took_ms"`
}

// CorrelationJobRequest is the message format for search.jobs.correlate subject.
// It represents a request to evaluate a detection rule correlation query.
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

// CorrelationJobResponse is the message format for search.results.correlate subject.
// It contains the results of a correlation evaluation.
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

// CorrelationMatch represents a single match from a correlation query.
// It groups events by an aggregation key (e.g., source IP, user).
type CorrelationMatch struct {
	AggregationKey string                   `json:"aggregation_key"`
	EventCount     int                      `json:"event_count"`
	Events         []map[string]interface{} `json:"events,omitempty"`
	FirstSeen      time.Time                `json:"first_seen"`
	LastSeen       time.Time                `json:"last_seen"`
}
