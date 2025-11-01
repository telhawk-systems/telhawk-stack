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

	event := &ocsf.Event{
		Class:        "generic_event",
		Category:     "activity",
		Activity:     fmt.Sprintf("ingest:%s", envelope.SourceType),
		Severity:     "unknown",
		ObservedTime: envelope.ReceivedAt, // placeholder until payload mapping defined
		Schema: ocsf.SchemaMetadata{
			Namespace: "ocsf",
			Version:   "1.1.0",
		},
		Raw: ocsf.RawDescriptor{
			Format: envelope.Format,
			Data:   payload,
		},
		Properties: map[string]interface{}{
			"source":      envelope.Source,
			"source_type": envelope.SourceType,
		},
	}

	if t, ok := payload["time"].(float64); ok {
		event.ObservedTime = time.Unix(int64(t), int64((t-float64(int64(t)))*1e9))
	}

	return event, nil
}
