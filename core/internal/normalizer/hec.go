package normalizer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
)

// HECNormalizer converts Splunk HEC style payloads into generic OCSF events.
type HECNormalizer struct{}

// Supports indicates HEC normalizer should handle JSON payloads with sourceType hec.
func (HECNormalizer) Supports(format, sourceType string) bool {
	return format == "json" && sourceType == "hec"
}

// Normalize performs a minimal mapping from HEC event envelope to OCSF Event.
func (HECNormalizer) Normalize(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error) {
	_ = ctx
	var payload map[string]interface{}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode hec payload: %w", err)
	}

	// Base event (class_uid 0) for generic events
	categoryUID := ocsf.CategoryOther
	classUID := 0
	activityID := ocsf.StatusUnknown

	eventTime := envelope.ReceivedAt
	if t, ok := payload["time"].(float64); ok {
		eventTime = time.Unix(int64(t), int64((t-float64(int64(t)))*1e9))
	}

	event := &ocsf.Event{
		// Required OCSF fields
		CategoryUID: categoryUID,
		ClassUID:    classUID,
		ActivityID:  activityID,
		TypeUID:     ocsf.ComputeTypeUID(categoryUID, classUID, activityID),
		Time:        eventTime,
		SeverityID:  ocsf.SeverityUnknown,

		// Human-readable fields
		Class:    "base_event",
		Category: "other",
		Activity: fmt.Sprintf("ingest:%s", envelope.SourceType),
		Severity: "Unknown",

		// Status
		StatusID: ocsf.StatusUnknown,
		Status:   "Unknown",

		// Timing
		ObservedTime: envelope.ReceivedAt,

		// Metadata
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Name:   "TelHawk Stack",
				Vendor: "TelHawk Systems",
			},
			Version:  "1.1.0",
			Profiles: []string{},
		},

		// Raw data preservation
		Raw: ocsf.RawDescriptor{
			Format: envelope.Format,
			Data:   payload,
		},

		// Additional properties
		Properties: map[string]string{
			"source":      envelope.Source,
			"source_type": envelope.SourceType,
		},
	}

	return event, nil
}
