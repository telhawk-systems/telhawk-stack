package correlation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

// JoinEvaluator evaluates join correlation rules (multi-query with field matching)
type JoinEvaluator struct {
	queryExecutor *QueryExecutor
	stateManager  *StateManager
}

// NewJoinEvaluator creates a new join evaluator
func NewJoinEvaluator(queryExecutor *QueryExecutor, stateManager *StateManager) *JoinEvaluator {
	return &JoinEvaluator{
		queryExecutor: queryExecutor,
		stateManager:  stateManager,
	}
}

// Evaluate evaluates a join correlation rule
func (e *JoinEvaluator) Evaluate(ctx context.Context, schema *DetectionSchema) ([]*Alert, error) {
	// Extract parameters
	params, err := e.extractParameters(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to extract parameters: %w", err)
	}

	// Parse time window
	timeWindow, err := time.ParseDuration(params.TimeWindow)
	if err != nil {
		return nil, fmt.Errorf("invalid time_window: %w", err)
	}

	// Execute left query
	leftResult, err := e.queryExecutor.ExecuteQuery(ctx, params.LeftQuery.Query, timeWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to execute left query: %w", err)
	}

	// Execute right query
	rightResult, err := e.queryExecutor.ExecuteQuery(ctx, params.RightQuery.Query, timeWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to execute right query: %w", err)
	}

	// Perform join
	joinedPairs := e.performJoin(leftResult.Events, rightResult.Events, params)

	// Generate alerts for joined events
	alerts := []*Alert{}
	for _, pair := range joinedPairs {
		alert := e.createAlert(schema, params, pair, timeWindow)
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// extractParameters extracts join parameters from schema
func (e *JoinEvaluator) extractParameters(schema *DetectionSchema) (*JoinParams, error) {
	schemaModel := schema.Model
	params := &JoinParams{}

	// Get base parameters
	baseParams, ok := schemaModel["parameters"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing parameters")
	}

	// Merge with parameter set if active
	baseParams = mergeParameterSets(baseParams, schema)

	// Extract time_window
	if tw, ok := baseParams["time_window"].(string); ok {
		params.TimeWindow = tw
	} else {
		return nil, fmt.Errorf("missing time_window")
	}

	// Extract left_query
	if lq, ok := baseParams["left_query"].(map[string]interface{}); ok {
		var query model.Query
		queryBytes, err := json.Marshal(lq["query"])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal left query: %w", err)
		}
		if err := json.Unmarshal(queryBytes, &query); err != nil {
			return nil, fmt.Errorf("failed to parse left query: %w", err)
		}
		params.LeftQuery = QueryConfig{
			Name:  lq["name"].(string),
			Query: &query,
		}
	} else {
		return nil, fmt.Errorf("missing left_query")
	}

	// Extract right_query
	if rq, ok := baseParams["right_query"].(map[string]interface{}); ok {
		var query model.Query
		queryBytes, err := json.Marshal(rq["query"])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal right query: %w", err)
		}
		if err := json.Unmarshal(queryBytes, &query); err != nil {
			return nil, fmt.Errorf("failed to parse right query: %w", err)
		}
		params.RightQuery = QueryConfig{
			Name:  rq["name"].(string),
			Query: &query,
		}
	} else {
		return nil, fmt.Errorf("missing right_query")
	}

	// Extract join_conditions
	if jc, ok := baseParams["join_conditions"].([]interface{}); ok {
		params.JoinConditions = make([]JoinCondition, len(jc))
		for i, c := range jc {
			condMap := c.(map[string]interface{})
			params.JoinConditions[i] = JoinCondition{
				LeftField:  condMap["left_field"].(string),
				RightField: condMap["right_field"].(string),
				Operator:   condMap["operator"].(string),
			}
		}
	} else {
		return nil, fmt.Errorf("missing join_conditions")
	}

	// Extract join_type (optional)
	if jt, ok := baseParams["join_type"].(string); ok {
		params.JoinType = jt
	} else {
		params.JoinType = "inner" // Default
	}

	return params, nil
}

// performJoin performs the join operation between left and right events
func (e *JoinEvaluator) performJoin(leftEvents, rightEvents []*Event, params *JoinParams) [][]*Event {
	var joined [][]*Event

	for _, leftEvent := range leftEvents {
		for _, rightEvent := range rightEvents {
			// Check all join conditions
			allMatch := true
			for _, condition := range params.JoinConditions {
				leftValue := getFieldValue(leftEvent.Fields, condition.LeftField)
				rightValue := getFieldValue(rightEvent.Fields, condition.RightField)

				if !e.compareValues(leftValue, rightValue, condition.Operator) {
					allMatch = false
					break
				}
			}

			if allMatch {
				joined = append(joined, []*Event{leftEvent, rightEvent})
			}
		}
	}

	return joined
}

// compareValues compares two values using the specified operator
func (e *JoinEvaluator) compareValues(left, right interface{}, operator string) bool {
	// Handle nil cases
	if left == nil || right == nil {
		return operator == "ne" && left != right
	}

	// Convert to strings for comparison
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	switch operator {
	case "eq":
		return leftStr == rightStr
	case "ne":
		return leftStr != rightStr
	default:
		return leftStr == rightStr
	}
}

// createAlert creates an alert from join correlation results
func (e *JoinEvaluator) createAlert(schema *DetectionSchema, params *JoinParams, eventPair []*Event, timeWindow time.Duration) *Alert {
	view := schema.View
	title := view["title"].(string)
	severity := view["severity"].(string)
	description := view["description"].(string)

	// Calculate time gap between events
	var timeGap time.Duration
	if len(eventPair) == 2 {
		timeGap = eventPair[1].Time.Sub(eventPair[0].Time)
		if timeGap < 0 {
			timeGap = -timeGap
		}
	}

	metadata := map[string]interface{}{
		"time_window": params.TimeWindow,
		"left_query":  params.LeftQuery.Name,
		"right_query": params.RightQuery.Name,
		"join_type":   params.JoinType,
		"time_gap":    timeGap.String(),
		"event_count": len(eventPair),
	}

	return &Alert{
		RuleID:          schema.ID,
		RuleVersionID:   schema.VersionID,
		Title:           title,
		Severity:        severity,
		Description:     description,
		Time:            time.Now(),
		CorrelationType: string(TypeJoin),
		Metadata:        metadata,
		Events:          eventPair,
	}
}
