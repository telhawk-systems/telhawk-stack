package models

import "time"

// HEC event format matching Splunk's HTTP Event Collector
type HECEvent struct {
	Time       *float64               `json:"time,omitempty"`
	Host       string                 `json:"host,omitempty"`
	Source     string                 `json:"source,omitempty"`
	SourceType string                 `json:"sourcetype,omitempty"`
	Index      string                 `json:"index,omitempty"`
	Event      interface{}            `json:"event"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
}

// HEC batch request
type HECBatchRequest struct {
	Events []HECEvent `json:"events"`
}

// HEC response matching Splunk format
type HECResponse struct {
	Text     string `json:"text"`
	Code     int    `json:"code"`
	AckID    int64  `json:"ackId,omitempty"`
	InvalidEventNumber int `json:"invalid-event-number,omitempty"`
}

// Internal event representation
type Event struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	Host       string                 `json:"host"`
	Source     string                 `json:"source"`
	SourceType string                 `json:"sourcetype"`
	SourceIP   string                 `json:"source_ip"`
	Index      string                 `json:"index"`
	Event      interface{}            `json:"event"`
	Fields     map[string]interface{} `json:"fields"`
	Raw        []byte                 `json:"raw"`
	HECTokenID string                 `json:"hec_token_id"`
	Signature  string                 `json:"signature"`
}

type IngestionStats struct {
	TotalEvents     int64     `json:"total_events"`
	TotalBytes      int64     `json:"total_bytes"`
	SuccessfulEvents int64    `json:"successful_events"`
	FailedEvents    int64     `json:"failed_events"`
	LastEvent       time.Time `json:"last_event"`
}
