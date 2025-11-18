package correlation

import (
	"context"
	"fmt"
	"time"
)

// EventCountEvaluator evaluates event_count correlation rules
type EventCountEvaluator struct {
	queryExecutor *QueryExecutor
	stateManager  *StateManager
}

// NewEventCountEvaluator creates a new event count evaluator
func NewEventCountEvaluator(queryExecutor *QueryExecutor, stateManager *StateManager) *EventCountEvaluator {
	return &EventCountEvaluator{
		queryExecutor: queryExecutor,
		stateManager:  stateManager,
	}
}

// Evaluate executes event_count correlation logic
func (e *EventCountEvaluator) Evaluate(ctx context.Context, schema *DetectionSchema) ([]*Alert, error) {
	// Extract parameters
	params, err := e.extractParameters(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to extract parameters: %w", err)
	}

	// Validate parameters
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Parse time window
	timeWindow, err := time.ParseDuration(params.TimeWindow)
	if err != nil {
		return nil, fmt.Errorf("invalid time window: %w", err)
	}

	// Extract query from model.parameters (where it actually is in the rule schema)
	paramsMap, ok := schema.Model["parameters"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parameters not found in model")
	}
	queryInterface, ok := paramsMap["query"]
	if !ok {
		return nil, fmt.Errorf("query not found in model.parameters")
	}

	// Parse query object
	query, err := parseQueryObject(queryInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	// Execute count query
	counts, err := e.queryExecutor.ExecuteCountQuery(ctx, query, timeWindow, params.GroupBy)
	if err != nil {
		return nil, fmt.Errorf("failed to execute count query: %w", err)
	}

	// Get threshold and operator from params (already extracted)
	threshold := int64(params.Threshold)
	operator := params.Operator

	// Generate alerts for groups that exceed threshold
	alerts := make([]*Alert, 0)
	for groupKey, count := range counts {
		if meetsThreshold(count, threshold, operator) {
			alert := e.createAlert(schema, params, groupKey, count, timeWindow)
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

// extractParameters extracts EventCountParams from schema
func (e *EventCountEvaluator) extractParameters(schema *DetectionSchema) (*EventCountParams, error) {
	// Get parameters from model
	paramsInterface, ok := schema.Model["parameters"]
	if !ok {
		return nil, fmt.Errorf("parameters not found in model")
	}

	// Merge with parameter set if active
	paramsMap, ok := paramsInterface.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parameters must be an object")
	}
	paramsMap = mergeParameterSets(paramsMap, schema)

	params := &EventCountParams{}

	// Extract time_window
	if tw, ok := paramsMap["time_window"].(string); ok {
		params.TimeWindow = tw
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
			// Threshold is an object like {"value": 5, "operator": "gte"}
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

	return params, nil
}

// createAlert creates an alert from evaluation results
func (e *EventCountEvaluator) createAlert(schema *DetectionSchema, params *EventCountParams, groupKey string, count int64, timeWindow time.Duration) *Alert {
	// Extract view fields
	view := schema.View
	title, _ := view["title"].(string)
	severity, _ := view["severity"].(string)
	description, _ := view["description"].(string)

	// Template variable replacement (simplified)
	metadata := map[string]interface{}{
		"event_count": count,
		"time_window": params.TimeWindow,
		"threshold":   params.Threshold,
		"group_key":   groupKey,
		"group_by":    params.GroupBy,
	}

	// Simple template replacement
	// In production, use proper template engine
	descWithValues := description
	// descWithValues would be processed with template engine here

	return &Alert{
		RuleID:          schema.ID,
		RuleVersionID:   schema.VersionID,
		Title:           title,
		Severity:        severity,
		Description:     descWithValues,
		Time:            time.Now(),
		CorrelationType: string(TypeEventCount),
		Metadata:        metadata,
	}
}
