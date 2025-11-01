package models

import "time"

type Event struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	SourceIP  string                 `json:"source_ip"`
	EventData map[string]interface{} `json:"event_data"`
	Raw       []byte                 `json:"raw"`
	Signature string                 `json:"signature"` // HMAC for tamper detection
	Ingested  time.Time              `json:"ingested"`
	Processed time.Time              `json:"processed,omitempty"`
}

type IngestionLog struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	HECTokenID    string    `json:"hec_token_id"`
	SourceIP      string    `json:"source_ip"`
	EventCount    int       `json:"event_count"`
	BytesReceived int64     `json:"bytes_received"`
	Success       bool      `json:"success"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	Signature     string    `json:"signature"`
}
