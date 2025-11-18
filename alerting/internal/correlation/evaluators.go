package correlation

import (
	"context"
	"time"
)

// CorrelationEvaluator interface for type-specific evaluators
type CorrelationEvaluator interface {
	Evaluate(ctx context.Context, schema *DetectionSchema) ([]*Alert, error)
}

// DetectionSchema represents a detection rule schema (simplified)
type DetectionSchema struct {
	ID         string
	VersionID  string
	Model      map[string]interface{}
	View       map[string]interface{}
	Controller map[string]interface{}
}

// Event represents an OCSF event matched by correlation
type Event struct {
	Time      time.Time              `json:"time"`
	RawSource map[string]interface{} `json:"raw_source"`
	Fields    map[string]interface{} `json:"fields"`
}

// Alert represents a generated alert
type Alert struct {
	RuleID          string                 `json:"rule_id"`
	RuleVersionID   string                 `json:"rule_version_id"`
	Title           string                 `json:"title"`
	Severity        string                 `json:"severity"`
	Description     string                 `json:"description"`
	Time            time.Time              `json:"time"`
	CorrelationType string                 `json:"correlation_type,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
	Events          []*Event               `json:"events,omitempty"`
}
