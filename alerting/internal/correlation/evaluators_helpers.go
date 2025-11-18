package correlation

import (
	"encoding/json"
	"fmt"

	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

// meetsThreshold checks if count meets threshold with given operator
func meetsThreshold(count, threshold int64, operator string) bool {
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

// mergeParameterSets merges parameter sets with base parameters
// Active parameter set values override base parameter values
func mergeParameterSets(baseParams map[string]interface{}, schema *DetectionSchema) map[string]interface{} {
	// Check for active parameter set
	activeSet, ok := schema.Model["active_parameter_set"].(string)
	if !ok || activeSet == "" {
		return baseParams
	}

	// Get parameter sets
	sets, ok := schema.Model["parameter_sets"].([]interface{})
	if !ok {
		return baseParams
	}

	// Find matching parameter set
	for _, set := range sets {
		setMap, ok := set.(map[string]interface{})
		if !ok {
			continue
		}

		if setMap["name"] == activeSet {
			setParams, ok := setMap["parameters"].(map[string]interface{})
			if !ok {
				return baseParams
			}

			// Merge: base parameters + set overrides
			merged := make(map[string]interface{})
			for k, v := range baseParams {
				merged[k] = v
			}
			for k, v := range setParams {
				merged[k] = v // Override with set values
			}
			return merged
		}
	}

	return baseParams
}

// parseQueryObject converts interface{} query to *model.Query
func parseQueryObject(queryInterface interface{}) (*model.Query, error) {
	var query model.Query
	queryBytes, err := json.Marshal(queryInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}
	if err := json.Unmarshal(queryBytes, &query); err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}
	return &query, nil
}

// extractGroupKey creates a group key from event fields
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

// getFieldValue retrieves a value from nested map using dot-notation path
// Supports jq-style paths like ".actor.user.name"
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

// splitFieldPath splits a dot-notation field path into parts
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
