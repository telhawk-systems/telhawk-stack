package correlation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
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
	var query model.Query
	queryBytes, err := json.Marshal(queryInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}
	if err := json.Unmarshal(queryBytes, &query); err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	// Execute count query
	counts, err := e.queryExecutor.ExecuteCountQuery(ctx, &query, timeWindow, params.GroupBy)
	if err != nil {
		return nil, fmt.Errorf("failed to execute count query: %w", err)
	}

	// Get threshold and operator from params (already extracted)
	threshold := int64(params.Threshold)
	operator := params.Operator

	// Generate alerts for groups that exceed threshold
	alerts := make([]*Alert, 0)
	for groupKey, count := range counts {
		if e.meetsThreshold(count, threshold, operator) {
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

	// Check for active parameter set
	if activeSet, ok := schema.Model["active_parameter_set"].(string); ok && activeSet != "" {
		// Merge with parameter set
		if sets, ok := schema.Model["parameter_sets"].([]interface{}); ok {
			for _, set := range sets {
				setMap, ok := set.(map[string]interface{})
				if !ok {
					continue // Skip invalid parameter sets
				}
				if setMap["name"] == activeSet {
					// Merge parameters
					baseParams, ok := paramsInterface.(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid base parameters format")
					}
					setParams, ok := setMap["parameters"].(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid set parameters format")
					}
					merged := make(map[string]interface{})
					for k, v := range baseParams {
						merged[k] = v
					}
					for k, v := range setParams {
						merged[k] = v // Override with set values
					}
					paramsInterface = merged
					break
				}
			}
		}
	}

	// Convert to EventCountParams
	paramsMap, ok := paramsInterface.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parameters must be an object")
	}

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

// meetsThreshold checks if count meets threshold with given operator
func (e *EventCountEvaluator) meetsThreshold(count, threshold int64, operator string) bool {
	switch operator {
	case "gt":
		return count > threshold
	case "gte":
		return count >= threshold
	case "lt":
		return count < threshold
	case "lte":
		return count <= threshold
	case "eq":
		return count == threshold
	case "ne":
		return count != threshold
	default:
		return count > threshold // Default to gt
	}
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
	var query model.Query
	queryBytes, err := json.Marshal(queryInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}
	if err := json.Unmarshal(queryBytes, &query); err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	// Execute cardinality query
	counts, err := e.queryExecutor.ExecuteCardinalityQuery(ctx, &query, timeWindow, params.Field, params.GroupBy)
	if err != nil {
		return nil, fmt.Errorf("failed to execute cardinality query: %w", err)
	}

	// Get threshold (already extracted in params)
	threshold := int64(params.Threshold)

	// Generate alerts
	alerts := make([]*Alert, 0)
	for groupKey, distinctCount := range counts {
		if e.meetsThreshold(distinctCount, threshold, params.Operator) {
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

// meetsThreshold checks if count meets threshold with given operator
func (e *ValueCountEvaluator) meetsThreshold(count, threshold int64, operator string) bool {
	switch operator {
	case "gt":
		return count > threshold
	case "gte":
		return count >= threshold
	case "lt":
		return count < threshold
	case "lte":
		return count <= threshold
	case "eq":
		return count == threshold
	case "ne":
		return count != threshold
	default:
		return count > threshold
	}
}

// TemporalEvaluator evaluates temporal correlation rules (unordered events)
type TemporalEvaluator struct {
	queryExecutor *QueryExecutor
	stateManager  *StateManager
}

// NewTemporalEvaluator creates a new temporal evaluator
func NewTemporalEvaluator(queryExecutor *QueryExecutor, stateManager *StateManager) *TemporalEvaluator {
	return &TemporalEvaluator{
		queryExecutor: queryExecutor,
		stateManager:  stateManager,
	}
}

// Evaluate evaluates a temporal correlation rule
func (e *TemporalEvaluator) Evaluate(ctx context.Context, schema *DetectionSchema) ([]*Alert, error) {
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

	// Execute all queries and collect matching events
	matchedEventsByQuery := make(map[string][]*Event)
	for _, queryConfig := range params.Queries {
		result, err := e.queryExecutor.ExecuteQuery(ctx, queryConfig.Query, timeWindow)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query %s: %w", queryConfig.Name, err)
		}
		matchedEventsByQuery[queryConfig.Name] = result.Events
	}

	// Group events by group_by fields
	groupedEvents := e.groupEventsByFields(matchedEventsByQuery, params.GroupBy)

	// Generate alerts for groups that meet min_matches threshold
	alerts := []*Alert{}
	for groupKey, eventsByQuery := range groupedEvents {
		matchCount := len(eventsByQuery)
		if matchCount >= params.MinMatches {
			alert := e.createAlert(schema, params, groupKey, eventsByQuery, timeWindow)
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

// extractParameters extracts temporal parameters from schema
func (e *TemporalEvaluator) extractParameters(schema *DetectionSchema) (*TemporalParams, error) {
	schemaModel := schema.Model
	params := &TemporalParams{}

	// Get base parameters
	baseParams, ok := schemaModel["parameters"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing parameters")
	}

	// Check for active parameter set override
	if activeSet, ok := schemaModel["active_parameter_set"].(string); ok && activeSet != "" {
		if paramSets, ok := schemaModel["parameter_sets"].([]interface{}); ok {
			for _, ps := range paramSets {
				paramSet := ps.(map[string]interface{})
				if paramSet["name"].(string) == activeSet {
					if setParams, ok := paramSet["parameters"].(map[string]interface{}); ok {
						// Merge parameter set into base params
						for k, v := range setParams {
							baseParams[k] = v
						}
					}
					break
				}
			}
		}
	}

	// Extract time_window
	if tw, ok := baseParams["time_window"].(string); ok {
		params.TimeWindow = tw
	} else {
		return nil, fmt.Errorf("missing time_window")
	}

	// Extract queries
	if queries, ok := baseParams["queries"].([]interface{}); ok {
		params.Queries = make([]QueryConfig, len(queries))
		for i, q := range queries {
			queryMap := q.(map[string]interface{})

			// Parse query object
			var query model.Query
			queryBytes, err := json.Marshal(queryMap["query"])
			if err != nil {
				return nil, fmt.Errorf("failed to marshal query: %w", err)
			}
			if err := json.Unmarshal(queryBytes, &query); err != nil {
				return nil, fmt.Errorf("failed to parse query: %w", err)
			}

			params.Queries[i] = QueryConfig{
				Name:  queryMap["name"].(string),
				Query: &query,
			}
		}
	} else {
		return nil, fmt.Errorf("missing queries")
	}

	// Extract min_matches from controller
	if controller, ok := schema.Controller["detection"].(map[string]interface{}); ok {
		if mm, ok := controller["min_matches"].(int); ok {
			params.MinMatches = mm
		} else if mm, ok := controller["min_matches"].(float64); ok {
			params.MinMatches = int(mm)
		}
	}
	if params.MinMatches == 0 {
		params.MinMatches = len(params.Queries) // Default: all queries must match
	}

	// Extract group_by (optional)
	if gb, ok := baseParams["group_by"].([]interface{}); ok {
		params.GroupBy = make([]string, len(gb))
		for i, field := range gb {
			params.GroupBy[i] = field.(string)
		}
	}

	return params, nil
}

// groupEventsByFields groups events by group_by fields
func (e *TemporalEvaluator) groupEventsByFields(eventsByQuery map[string][]*Event, groupByFields []string) map[string]map[string][]*Event {
	grouped := make(map[string]map[string][]*Event)

	for queryName, events := range eventsByQuery {
		for _, event := range events {
			groupKey := extractGroupKey(event.Fields, groupByFields)

			if _, exists := grouped[groupKey]; !exists {
				grouped[groupKey] = make(map[string][]*Event)
			}
			grouped[groupKey][queryName] = append(grouped[groupKey][queryName], event)
		}
	}

	return grouped
}

// createAlert creates an alert from temporal correlation results
func (e *TemporalEvaluator) createAlert(schema *DetectionSchema, params *TemporalParams, groupKey string, eventsByQuery map[string][]*Event, timeWindow time.Duration) *Alert {
	view := schema.View
	title := view["title"].(string)
	severity := view["severity"].(string)
	description := view["description"].(string)

	// Collect all matched events
	allEvents := []*Event{}
	matchedQueries := []string{}
	for queryName, events := range eventsByQuery {
		matchedQueries = append(matchedQueries, queryName)
		allEvents = append(allEvents, events...)
	}

	metadata := map[string]interface{}{
		"time_window":     params.TimeWindow,
		"min_matches":     params.MinMatches,
		"matched_queries": matchedQueries,
		"match_count":     len(matchedQueries),
		"event_count":     len(allEvents),
		"group_key":       groupKey,
		"group_by":        params.GroupBy,
	}

	return &Alert{
		RuleID:          schema.ID,
		RuleVersionID:   schema.VersionID,
		Title:           title,
		Severity:        severity,
		Description:     description,
		Time:            time.Now(),
		CorrelationType: string(TypeTemporal),
		Metadata:        metadata,
		Events:          allEvents,
	}
}

// TemporalOrderedEvaluator evaluates temporal_ordered correlation rules (sequence detection)
type TemporalOrderedEvaluator struct {
	queryExecutor *QueryExecutor
	stateManager  *StateManager
}

// NewTemporalOrderedEvaluator creates a new temporal_ordered evaluator
func NewTemporalOrderedEvaluator(queryExecutor *QueryExecutor, stateManager *StateManager) *TemporalOrderedEvaluator {
	return &TemporalOrderedEvaluator{
		queryExecutor: queryExecutor,
		stateManager:  stateManager,
	}
}

// Evaluate evaluates a temporal_ordered correlation rule
func (e *TemporalOrderedEvaluator) Evaluate(ctx context.Context, schema *DetectionSchema) ([]*Alert, error) {
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

	// Parse max gap (optional)
	var maxGap time.Duration
	if params.MaxGap != "" {
		maxGap, err = time.ParseDuration(params.MaxGap)
		if err != nil {
			return nil, fmt.Errorf("invalid max_gap: %w", err)
		}
	} else {
		maxGap = timeWindow // Default: same as time window
	}

	// Execute all sequence queries and collect events
	sequenceEvents := make([][]*Event, len(params.Sequence))
	for i, seqStep := range params.Sequence {
		result, err := e.queryExecutor.ExecuteQuery(ctx, seqStep.Query, timeWindow)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query for step %d: %w", seqStep.Step, err)
		}
		sequenceEvents[i] = result.Events
	}

	// Find matching sequences grouped by group_by fields
	matchedSequences := e.findMatchingSequences(sequenceEvents, params, maxGap)

	// Generate alerts for matched sequences
	alerts := []*Alert{}
	for groupKey, sequences := range matchedSequences {
		for _, seq := range sequences {
			alert := e.createAlert(schema, params, groupKey, seq, timeWindow)
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

// extractParameters extracts temporal_ordered parameters from schema
func (e *TemporalOrderedEvaluator) extractParameters(schema *DetectionSchema) (*TemporalOrderedParams, error) {
	schemaModel := schema.Model
	params := &TemporalOrderedParams{}

	// Get base parameters
	baseParams, ok := schemaModel["parameters"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing parameters")
	}

	// Check for active parameter set override
	if activeSet, ok := schemaModel["active_parameter_set"].(string); ok && activeSet != "" {
		if paramSets, ok := schemaModel["parameter_sets"].([]interface{}); ok {
			for _, ps := range paramSets {
				paramSet := ps.(map[string]interface{})
				if paramSet["name"].(string) == activeSet {
					if setParams, ok := paramSet["parameters"].(map[string]interface{}); ok {
						for k, v := range setParams {
							baseParams[k] = v
						}
					}
					break
				}
			}
		}
	}

	// Extract time_window
	if tw, ok := baseParams["time_window"].(string); ok {
		params.TimeWindow = tw
	} else {
		return nil, fmt.Errorf("missing time_window")
	}

	// Extract max_gap (optional)
	if mg, ok := baseParams["max_gap"].(string); ok {
		params.MaxGap = mg
	}

	// Extract sequence
	if seq, ok := baseParams["sequence"].([]interface{}); ok {
		params.Sequence = make([]SequenceStep, len(seq))
		for i, s := range seq {
			stepMap := s.(map[string]interface{})

			// Parse query object
			var query model.Query
			queryBytes, err := json.Marshal(stepMap["query"])
			if err != nil {
				return nil, fmt.Errorf("failed to marshal query: %w", err)
			}
			if err := json.Unmarshal(queryBytes, &query); err != nil {
				return nil, fmt.Errorf("failed to parse query: %w", err)
			}

			step := SequenceStep{
				Name:  stepMap["name"].(string),
				Query: &query,
			}
			if stepNum, ok := stepMap["step"].(int); ok {
				step.Step = stepNum
			} else if stepNum, ok := stepMap["step"].(float64); ok {
				step.Step = int(stepNum)
			} else {
				step.Step = i + 1 // Default to position
			}
			params.Sequence[i] = step
		}
	} else {
		return nil, fmt.Errorf("missing sequence")
	}

	// Extract strict_order from controller (optional)
	if controller, ok := schema.Controller["detection"].(map[string]interface{}); ok {
		if so, ok := controller["strict_order"].(bool); ok {
			params.StrictOrder = so
		}
	}

	// Extract group_by (optional)
	if gb, ok := baseParams["group_by"].([]interface{}); ok {
		params.GroupBy = make([]string, len(gb))
		for i, field := range gb {
			params.GroupBy[i] = field.(string)
		}
	}

	return params, nil
}

// findMatchingSequences finds event sequences that match the pattern
func (e *TemporalOrderedEvaluator) findMatchingSequences(sequenceEvents [][]*Event, params *TemporalOrderedParams, maxGap time.Duration) map[string][][]*Event {
	matched := make(map[string][][]*Event)

	// Group first step events by group_by fields
	if len(sequenceEvents) == 0 || len(sequenceEvents[0]) == 0 {
		return matched
	}

	for _, firstEvent := range sequenceEvents[0] {
		groupKey := extractGroupKey(firstEvent.Fields, params.GroupBy)

		// Try to build complete sequence starting from this event
		sequence := e.buildSequence([]*Event{firstEvent}, 1, sequenceEvents, params, maxGap, groupKey)
		if len(sequence) > 0 {
			matched[groupKey] = append(matched[groupKey], sequence)
		}
	}

	return matched
}

// buildSequence recursively builds a matching sequence and returns it
func (e *TemporalOrderedEvaluator) buildSequence(current []*Event, nextStep int, allEvents [][]*Event, params *TemporalOrderedParams, maxGap time.Duration, groupKey string) []*Event {
	// Base case: completed sequence
	if nextStep >= len(allEvents) {
		return current
	}

	lastEvent := current[len(current)-1]

	// Find matching event for next step
	for _, candidate := range allEvents[nextStep] {
		// Check group key matches
		candGroupKey := extractGroupKey(candidate.Fields, params.GroupBy)
		if candGroupKey != groupKey {
			continue
		}

		// Check temporal ordering
		if candidate.Time.Before(lastEvent.Time) {
			continue // Must be after previous event
		}

		// Check max gap
		gap := candidate.Time.Sub(lastEvent.Time)
		if gap > maxGap {
			continue // Gap too large
		}

		// Found matching event, continue building sequence
		newSequence := append(current, candidate)
		result := e.buildSequence(newSequence, nextStep+1, allEvents, params, maxGap, groupKey)
		if len(result) > 0 {
			return result
		}
	}

	return []*Event{} // No match found
}

// createAlert creates an alert from sequence detection
func (e *TemporalOrderedEvaluator) createAlert(schema *DetectionSchema, params *TemporalOrderedParams, groupKey string, sequence []*Event, timeWindow time.Duration) *Alert {
	view := schema.View
	title := view["title"].(string)
	severity := view["severity"].(string)
	description := view["description"].(string)

	// Calculate sequence duration
	var duration time.Duration
	if len(sequence) > 1 {
		duration = sequence[len(sequence)-1].Time.Sub(sequence[0].Time)
	}

	stepNames := make([]string, len(params.Sequence))
	for i, step := range params.Sequence {
		stepNames[i] = step.Name
	}

	metadata := map[string]interface{}{
		"time_window":       params.TimeWindow,
		"max_gap":           params.MaxGap,
		"sequence_steps":    stepNames,
		"sequence_length":   len(sequence),
		"sequence_duration": duration.String(),
		"group_key":         groupKey,
		"group_by":          params.GroupBy,
	}

	return &Alert{
		RuleID:          schema.ID,
		RuleVersionID:   schema.VersionID,
		Title:           title,
		Severity:        severity,
		Description:     description,
		Time:            time.Now(),
		CorrelationType: string(TypeTemporalOrdered),
		Metadata:        metadata,
		Events:          sequence,
	}
}

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

	// Check for active parameter set override
	if activeSet, ok := schemaModel["active_parameter_set"].(string); ok && activeSet != "" {
		if paramSets, ok := schemaModel["parameter_sets"].([]interface{}); ok {
			for _, ps := range paramSets {
				paramSet := ps.(map[string]interface{})
				if paramSet["name"].(string) == activeSet {
					if setParams, ok := paramSet["parameters"].(map[string]interface{}); ok {
						for k, v := range setParams {
							baseParams[k] = v
						}
					}
					break
				}
			}
		}
	}

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

// Helper function to extract group key from event fields
func extractGroupKey(fields map[string]interface{}, groupByFields []string) string {
	if len(groupByFields) == 0 {
		return "default"
	}

	key := ""
	for _, field := range groupByFields {
		value := getFieldValue(fields, field)
		key += fmt.Sprintf("%v|", value)
	}
	return key
}

// Helper function to get field value from nested map
func getFieldValue(fields map[string]interface{}, fieldPath string) interface{} {
	// Handle jq-style paths like ".actor.user.name"
	if len(fieldPath) > 0 && fieldPath[0] == '.' {
		fieldPath = fieldPath[1:]
	}

	// Split path and traverse
	parts := splitFieldPath(fieldPath)
	current := fields

	for i, part := range parts {
		if i == len(parts)-1 {
			return current[part]
		}
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}

	return nil
}

// Helper function to split field path
func splitFieldPath(path string) []string {
	// Simple split by dot (could be enhanced for array indices)
	result := []string{}
	current := ""
	for _, ch := range path {
		if ch == '.' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
