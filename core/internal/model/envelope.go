package model

import "time"

// RawEventEnvelope carries the original ingestion payload forwarded to the core service.
type RawEventEnvelope struct {
	ID         string            `json:"id"`
	Source     string            `json:"source"`
	SourceType string            `json:"source_type"`
	Format     string            `json:"format"`
	Payload    []byte            `json:"payload"`
	Attributes map[string]string `json:"attributes,omitempty"`
	ReceivedAt time.Time         `json:"received_at"`
}
