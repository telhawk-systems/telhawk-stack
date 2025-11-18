package correlation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

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

	// Merge with parameter set if active
	baseParams = mergeParameterSets(baseParams, schema)

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
