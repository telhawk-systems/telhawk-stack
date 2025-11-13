package correlation

import (
	"fmt"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

// CorrelationType represents the type of correlation analysis
type CorrelationType string

const (
	// Tier 1 Essential Types
	TypeEventCount        CorrelationType = "event_count"
	TypeValueCount        CorrelationType = "value_count"
	TypeTemporal          CorrelationType = "temporal"
	TypeTemporalOrdered   CorrelationType = "temporal_ordered"
	TypeJoin              CorrelationType = "join"
	TypeSuppression       CorrelationType = "suppression"
	TypeBaselineDeviation CorrelationType = "baseline_deviation"
	TypeMissingEvent      CorrelationType = "missing_event"
)

// IsValid checks if the correlation type is valid
func (ct CorrelationType) IsValid() bool {
	switch ct {
	case TypeEventCount, TypeValueCount, TypeTemporal, TypeTemporalOrdered,
		TypeJoin, TypeSuppression, TypeBaselineDeviation, TypeMissingEvent:
		return true
	default:
		return false
	}
}

// Parameters is a generic interface for correlation parameters
type Parameters interface {
	Validate() error
}

// EventCountParams parameters for event_count correlation
type EventCountParams struct {
	TimeWindow string   `json:"time_window" validate:"required,duration"` // e.g., "5m", "1h"
	Threshold  int      `json:"threshold" validate:"required,gt=0"`
	Operator   string   `json:"operator" validate:"required,oneof=gt gte lt lte eq ne"`
	GroupBy    []string `json:"group_by" validate:"omitempty,dive,required"`
}

// Validate validates EventCountParams
func (p *EventCountParams) Validate() error {
	if p.TimeWindow == "" {
		return fmt.Errorf("time_window is required")
	}
	if _, err := time.ParseDuration(p.TimeWindow); err != nil {
		return fmt.Errorf("invalid time_window: %w", err)
	}
	if p.Threshold <= 0 {
		return fmt.Errorf("threshold must be greater than 0")
	}
	if p.Operator == "" {
		return fmt.Errorf("operator is required")
	}
	validOperators := map[string]bool{"gt": true, "gte": true, "lt": true, "lte": true, "eq": true, "ne": true}
	if !validOperators[p.Operator] {
		return fmt.Errorf("invalid operator: %s", p.Operator)
	}
	return nil
}

// SuppressionParams parameters for alert suppression
type SuppressionParams struct {
	Enabled       bool     `json:"enabled"`
	Window        string   `json:"window" validate:"required,duration"`                // e.g., "1h", "24h"
	Key           []string `json:"key" validate:"required,min=1,dive,required"`        // Fields to group by
	MaxAlerts     int      `json:"max_alerts" validate:"omitempty,gt=0"`               // Default: 1
	ResetOnChange []string `json:"reset_on_change" validate:"omitempty,dive,required"` // Fields that reset suppression
}

// Validate validates SuppressionParams
func (p *SuppressionParams) Validate() error {
	if !p.Enabled {
		return nil // Skip validation if suppression is disabled
	}
	if p.Window == "" {
		return fmt.Errorf("window is required")
	}
	if _, err := time.ParseDuration(p.Window); err != nil {
		return fmt.Errorf("invalid window: %w", err)
	}
	if len(p.Key) == 0 {
		return fmt.Errorf("key is required and must have at least one field")
	}
	if p.MaxAlerts < 0 {
		return fmt.Errorf("max_alerts must be non-negative")
	}
	return nil
}

// ValueCountParams parameters for value_count correlation
type ValueCountParams struct {
	TimeWindow string   `json:"time_window" validate:"required,duration"`
	Field      string   `json:"field" validate:"required"`
	Threshold  int      `json:"threshold" validate:"required,gt=0"`
	Operator   string   `json:"operator" validate:"required,oneof=gt gte lt lte eq ne"`
	GroupBy    []string `json:"group_by" validate:"omitempty,dive,required"`
}

// Validate validates ValueCountParams
func (p *ValueCountParams) Validate() error {
	if p.TimeWindow == "" {
		return fmt.Errorf("time_window is required")
	}
	if _, err := time.ParseDuration(p.TimeWindow); err != nil {
		return fmt.Errorf("invalid time_window: %w", err)
	}
	if p.Field == "" {
		return fmt.Errorf("field is required")
	}
	if p.Threshold <= 0 {
		return fmt.Errorf("threshold must be greater than 0")
	}
	if p.Operator == "" {
		return fmt.Errorf("operator is required")
	}
	validOperators := map[string]bool{"gt": true, "gte": true, "lt": true, "lte": true, "eq": true, "ne": true}
	if !validOperators[p.Operator] {
		return fmt.Errorf("invalid operator: %s", p.Operator)
	}
	return nil
}

// QueryConfig represents a query for multi-event correlation
type QueryConfig struct {
	Name  string       `json:"name" validate:"required"`
	Query *model.Query `json:"query" validate:"required"`
}

// TemporalParams parameters for temporal correlation
type TemporalParams struct {
	TimeWindow string        `json:"time_window" validate:"required,duration"`
	Queries    []QueryConfig `json:"queries" validate:"required,min=2,dive"`
	MinMatches int           `json:"min_matches" validate:"omitempty,gt=0"`
	GroupBy    []string      `json:"group_by" validate:"omitempty,dive,required"`
}

// Validate validates TemporalParams
func (p *TemporalParams) Validate() error {
	if p.TimeWindow == "" {
		return fmt.Errorf("time_window is required")
	}
	if _, err := time.ParseDuration(p.TimeWindow); err != nil {
		return fmt.Errorf("invalid time_window: %w", err)
	}
	if len(p.Queries) < 2 {
		return fmt.Errorf("queries must have at least 2 entries")
	}
	for i, q := range p.Queries {
		if q.Name == "" {
			return fmt.Errorf("query[%d].name is required", i)
		}
		if q.Query == nil {
			return fmt.Errorf("query[%d].query is required", i)
		}
	}
	if p.MinMatches > len(p.Queries) {
		return fmt.Errorf("min_matches (%d) cannot be greater than number of queries (%d)", p.MinMatches, len(p.Queries))
	}
	return nil
}

// SequenceStep represents a step in an ordered sequence
type SequenceStep struct {
	Step  int          `json:"step" validate:"required,gt=0"`
	Name  string       `json:"name" validate:"required"`
	Query *model.Query `json:"query" validate:"required"`
}

// TemporalOrderedParams parameters for temporal_ordered correlation
type TemporalOrderedParams struct {
	TimeWindow  string         `json:"time_window" validate:"required,duration"`
	Sequence    []SequenceStep `json:"sequence" validate:"required,min=2,dive"`
	MaxGap      string         `json:"max_gap" validate:"omitempty,duration"`
	GroupBy     []string       `json:"group_by" validate:"omitempty,dive,required"`
	StrictOrder bool           `json:"strict_order"`
}

// Validate validates TemporalOrderedParams
func (p *TemporalOrderedParams) Validate() error {
	if p.TimeWindow == "" {
		return fmt.Errorf("time_window is required")
	}
	if _, err := time.ParseDuration(p.TimeWindow); err != nil {
		return fmt.Errorf("invalid time_window: %w", err)
	}
	if len(p.Sequence) < 2 {
		return fmt.Errorf("sequence must have at least 2 steps")
	}
	steps := make(map[int]bool)
	for i, s := range p.Sequence {
		if s.Step <= 0 {
			return fmt.Errorf("sequence[%d].step must be greater than 0", i)
		}
		if steps[s.Step] {
			return fmt.Errorf("duplicate step number: %d", s.Step)
		}
		steps[s.Step] = true
		if s.Name == "" {
			return fmt.Errorf("sequence[%d].name is required", i)
		}
		if s.Query == nil {
			return fmt.Errorf("sequence[%d].query is required", i)
		}
	}
	if p.MaxGap != "" {
		if _, err := time.ParseDuration(p.MaxGap); err != nil {
			return fmt.Errorf("invalid max_gap: %w", err)
		}
	}
	return nil
}

// JoinCondition represents a field matching condition for join correlation
type JoinCondition struct {
	LeftField  string `json:"left_field" validate:"required"`
	RightField string `json:"right_field" validate:"required"`
	Operator   string `json:"operator" validate:"required,oneof=eq ne"`
}

// JoinParams parameters for join correlation
type JoinParams struct {
	TimeWindow     string          `json:"time_window" validate:"required,duration"`
	LeftQuery      QueryConfig     `json:"left_query" validate:"required"`
	RightQuery     QueryConfig     `json:"right_query" validate:"required"`
	JoinConditions []JoinCondition `json:"join_conditions" validate:"required,min=1,dive"`
	JoinType       string          `json:"join_type" validate:"omitempty,oneof=inner left any"`
}

// Validate validates JoinParams
func (p *JoinParams) Validate() error {
	if p.TimeWindow == "" {
		return fmt.Errorf("time_window is required")
	}
	if _, err := time.ParseDuration(p.TimeWindow); err != nil {
		return fmt.Errorf("invalid time_window: %w", err)
	}
	if p.LeftQuery.Name == "" || p.LeftQuery.Query == nil {
		return fmt.Errorf("left_query.name and left_query.query are required")
	}
	if p.RightQuery.Name == "" || p.RightQuery.Query == nil {
		return fmt.Errorf("right_query.name and right_query.query are required")
	}
	if len(p.JoinConditions) == 0 {
		return fmt.Errorf("join_conditions must have at least one condition")
	}
	for i, jc := range p.JoinConditions {
		if jc.LeftField == "" {
			return fmt.Errorf("join_conditions[%d].left_field is required", i)
		}
		if jc.RightField == "" {
			return fmt.Errorf("join_conditions[%d].right_field is required", i)
		}
		if jc.Operator == "" {
			jc.Operator = "eq"
		}
		if jc.Operator != "eq" && jc.Operator != "ne" {
			return fmt.Errorf("invalid join_conditions[%d].operator: %s", i, jc.Operator)
		}
	}
	if p.JoinType == "" {
		p.JoinType = "inner"
	}
	return nil
}

// BaselineDeviationParams parameters for baseline_deviation correlation
type BaselineDeviationParams struct {
	BaselineWindow     string   `json:"baseline_window" validate:"required,duration"`
	ComparisonWindow   string   `json:"comparison_window" validate:"required,duration"`
	Field              string   `json:"field" validate:"required"`
	DeviationThreshold float64  `json:"deviation_threshold" validate:"required,gt=0"`
	Sensitivity        string   `json:"sensitivity" validate:"omitempty,oneof=low medium high"`
	GroupBy            []string `json:"group_by" validate:"required,min=1,dive,required"`
	MinBaselineSamples int      `json:"min_baseline_samples" validate:"omitempty,gte=0"`
}

// Validate validates BaselineDeviationParams
func (p *BaselineDeviationParams) Validate() error {
	if p.BaselineWindow == "" {
		return fmt.Errorf("baseline_window is required")
	}
	if _, err := time.ParseDuration(p.BaselineWindow); err != nil {
		return fmt.Errorf("invalid baseline_window: %w", err)
	}
	if p.ComparisonWindow == "" {
		return fmt.Errorf("comparison_window is required")
	}
	if _, err := time.ParseDuration(p.ComparisonWindow); err != nil {
		return fmt.Errorf("invalid comparison_window: %w", err)
	}
	if p.Field == "" {
		return fmt.Errorf("field is required")
	}
	if p.DeviationThreshold <= 0 {
		return fmt.Errorf("deviation_threshold must be greater than 0")
	}
	if len(p.GroupBy) == 0 {
		return fmt.Errorf("group_by is required and must have at least one field")
	}
	if p.Sensitivity != "" && p.Sensitivity != "low" && p.Sensitivity != "medium" && p.Sensitivity != "high" {
		return fmt.Errorf("invalid sensitivity: %s", p.Sensitivity)
	}
	return nil
}

// MissingEventParams parameters for missing_event correlation
type MissingEventParams struct {
	ExpectedInterval  string `json:"expected_interval" validate:"required,duration"`
	GracePeriod       string `json:"grace_period" validate:"omitempty,duration"`
	EntityField       string `json:"entity_field" validate:"required"`
	AlertAfterMissing int    `json:"alert_after_missing" validate:"omitempty,gte=1"`
}

// Validate validates MissingEventParams
func (p *MissingEventParams) Validate() error {
	if p.ExpectedInterval == "" {
		return fmt.Errorf("expected_interval is required")
	}
	if _, err := time.ParseDuration(p.ExpectedInterval); err != nil {
		return fmt.Errorf("invalid expected_interval: %w", err)
	}
	if p.GracePeriod != "" {
		if _, err := time.ParseDuration(p.GracePeriod); err != nil {
			return fmt.Errorf("invalid grace_period: %w", err)
		}
	}
	if p.EntityField == "" {
		return fmt.Errorf("entity_field is required")
	}
	if p.AlertAfterMissing < 0 {
		return fmt.Errorf("alert_after_missing must be non-negative")
	}
	return nil
}

// ParameterSet represents a named set of parameters
type ParameterSet struct {
	Name        string                 `json:"name" validate:"required"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters" validate:"required"`
}
