package normalizer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
)

// OCSFPassthroughNormalizer handles events that are already in OCSF format.
// It extracts the OCSF event from the payload without transformation.
type OCSFPassthroughNormalizer struct{}

// Supports checks if the sourcetype indicates an OCSF-formatted event
func (OCSFPassthroughNormalizer) Supports(format, sourceType string) bool {
	return format == "json" && strings.HasPrefix(strings.ToLower(sourceType), "ocsf:")
}

// Normalize extracts the OCSF event from the payload
func (OCSFPassthroughNormalizer) Normalize(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode ocsf payload: %w", err)
	}

	// The OCSF event might be wrapped in an "event" field (HEC format)
	var ocsfData map[string]interface{}
	if eventData, ok := payload["event"].(map[string]interface{}); ok {
		ocsfData = eventData
	} else {
		ocsfData = payload
	}

	// Handle time field - convert Unix timestamp to RFC3339 string if needed
	if timeVal, ok := ocsfData["time"].(float64); ok {
		// Convert Unix timestamp to time.Time
		eventTime := time.Unix(int64(timeVal), int64((timeVal-float64(int64(timeVal)))*1e9))
		ocsfData["time"] = eventTime.Format(time.RFC3339Nano)
	}

	// The payload should already be a complete OCSF event
	// Marshal it back and unmarshal into ocsf.Event to preserve all fields
	eventBytes, err := json.Marshal(ocsfData)
	if err != nil {
		return nil, fmt.Errorf("marshal ocsf event: %w", err)
	}

	var event ocsf.Event
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		return nil, fmt.Errorf("unmarshal ocsf event: %w", err)
	}

	// Set/override envelope metadata
	event.ObservedTime = envelope.ReceivedAt
	if event.Raw.Data == nil {
		event.Raw = ocsf.RawDescriptor{
			Format: envelope.Format,
			Data:   payload,
		}
	}

	// Ensure properties are set
	if event.Properties == nil {
		event.Properties = make(map[string]string)
	}
	event.Properties["source"] = envelope.Source
	event.Properties["source_type"] = envelope.SourceType

	return &event, nil
}
