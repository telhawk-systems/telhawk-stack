package correlation

import (
	"context"
	"fmt"
	"time"
)

// ValueCountEvaluator evaluates value_count correlation rules
type ValueCountEvaluator struct {
	queryExecutor *QueryExecutor
	stateManager  *StateManager
}

// NewValueCountEvaluator creates a new value count evaluator
func NewValueCountEvaluator(queryExecutor *QueryExecutor, stateManager *StateManager) *ValueCountEvaluator {
	return &ValueCountEvaluator{
		queryExecutor: queryExecutor,
		stateManager:  stateManager,
	}
}

// Evaluate executes value_count correlation logic
func (e *ValueCountEvaluator) Evaluate(ctx context.Context, schema *DetectionSchema) ([]*Alert, error) {
	// Extract parameters
	paramsMap, ok := schema.Model["parameters"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parameters must be an object")
	}

	params := &ValueCountParams{}

	// Extract time_window
	if tw, ok := paramsMap["time_window"].(string); ok {
		params.TimeWindow = tw
	}

	// Extract field
	if f, ok := paramsMap["field"].(string); ok {
		params.Field = f
	}

	// Extract group_by
	if gb, ok := paramsMap["group_by"].([]interface{}); ok {
		params.GroupBy = make([]string, len(gb))
		for i, v := range gb {
			if s, ok := v.(string); ok {
				params.GroupBy[i] = s
			}
		}
	}

	// Extract threshold (handle both int and object formats)
	if threshold, ok := paramsMap["threshold"]; ok {
		switch t := threshold.(type) {
		case float64:
			params.Threshold = int(t)
		case int:
			params.Threshold = t
		case map[string]interface{}:
			if val, ok := t["value"].(float64); ok {
				params.Threshold = int(val)
			}
			if op, ok := t["operator"].(string); ok {
				params.Operator = op
			}
		}
	}

	// Extract operator (if not already set from threshold object)
	if params.Operator == "" {
		if op, ok := paramsMap["operator"].(string); ok {
			params.Operator = op
		} else {
			params.Operator = "gte" // Default operator
		}
	}

	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Parse time window
	timeWindow, err := time.ParseDuration(params.TimeWindow)
	if err != nil {
		return nil, fmt.Errorf("invalid time window: %w", err)
	}

	// Extract query from model.parameters (paramsMap already declared above)
	queryInterface, ok := paramsMap["query"]
	if !ok {
		return nil, fmt.Errorf("query not found in model.parameters")
	}

	query, err := parseQueryObject(queryInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	// Execute cardinality query
	counts, err := e.queryExecutor.ExecuteCardinalityQuery(ctx, query, timeWindow, params.Field, params.GroupBy)
	if err != nil {
		return nil, fmt.Errorf("failed to execute cardinality query: %w", err)
	}

	// Get threshold (already extracted in params)
	threshold := int64(params.Threshold)

	// Generate alerts
	alerts := make([]*Alert, 0)
	for groupKey, distinctCount := range counts {
		if meetsThreshold(distinctCount, threshold, params.Operator) {
			alert := &Alert{
				RuleID:          schema.ID,
				RuleVersionID:   schema.VersionID,
				Title:           schema.View["title"].(string),
				Severity:        schema.View["severity"].(string),
				Description:     schema.View["description"].(string),
				Time:            time.Now(),
				CorrelationType: string(TypeValueCount),
				Metadata: map[string]interface{}{
					"distinct_count": distinctCount,
					"field":          params.Field,
					"time_window":    params.TimeWindow,
					"threshold":      threshold,
					"group_key":      groupKey,
				},
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}
