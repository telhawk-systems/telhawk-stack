package correlation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

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

	// Merge with parameter set if active
	baseParams = mergeParameterSets(baseParams, schema)

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
