package seeder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DetectionRule represents a complete detection rule from JSON
type DetectionRule struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Model       RuleModel      `json:"model"`
	View        RuleView       `json:"view"`
	Controller  RuleController `json:"controller"`
}

// RuleModel contains the detection logic
type RuleModel struct {
	CorrelationType string                 `json:"correlation_type"`
	Parameters      map[string]interface{} `json:"parameters"`
}

// RuleView contains display information
type RuleView struct {
	Title       string          `json:"title"`
	Severity    string          `json:"severity"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Tags        []string        `json:"tags"`
	MITREAttack MITREAttackInfo `json:"mitre_attack"`
}

// MITREAttackInfo contains MITRE ATT&CK framework information
type MITREAttackInfo struct {
	Tactics    []string `json:"tactics"`
	Techniques []string `json:"techniques"`
}

// RuleController contains evaluation configuration
type RuleController struct {
	Detection RuleDetectionConfig `json:"detection"`
	Response  RuleResponseConfig  `json:"response"`
}

// RuleDetectionConfig contains detection-specific settings
type RuleDetectionConfig struct {
	SuppressionWindow string `json:"suppression_window"`
}

// RuleResponseConfig contains response actions
type RuleResponseConfig struct {
	Actions           []interface{} `json:"actions"`
	SeverityThreshold string        `json:"severity_threshold"`
}

// QueryFilter represents a filter in the rule's query
type QueryFilter struct {
	Type       string        `json:"type,omitempty"` // "and", "or", or empty for simple filter
	Field      string        `json:"field,omitempty"`
	Operator   string        `json:"operator,omitempty"`
	Value      interface{}   `json:"value,omitempty"`
	Conditions []QueryFilter `json:"conditions,omitempty"` // For compound filters
}

// ParseRuleFile reads and parses a detection rule JSON file
func ParseRuleFile(path string) (*DetectionRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule file: %w", err)
	}

	var rule DetectionRule
	if err := json.Unmarshal(data, &rule); err != nil {
		return nil, fmt.Errorf("failed to parse rule JSON: %w", err)
	}

	return &rule, nil
}

// LoadRulesFromDirectory loads all JSON rule files from a directory
func LoadRulesFromDirectory(dirPath string) ([]*DetectionRule, error) {
	var rules []*DetectionRule

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Skip README and other non-rule files
		if strings.HasPrefix(entry.Name(), "README") {
			continue
		}

		rulePath := filepath.Join(dirPath, entry.Name())
		rule, err := ParseRuleFile(rulePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", entry.Name(), err)
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// IsSupported checks if the rule's correlation type is supported for event generation
func (r *DetectionRule) IsSupported() (bool, string) {
	supported := map[string]bool{
		"event_count":        true,
		"value_count":        true,
		"temporal":           false, // Phase 2
		"temporal_ordered":   false, // Phase 2
		"join":               false, // Phase 2
		"suppression":        false, // Skip
		"baseline_deviation": false, // Skip - needs historical data
		"missing_event":      false, // Skip - inverse logic
	}

	isSupported, exists := supported[r.Model.CorrelationType]
	if !exists {
		return false, fmt.Sprintf("unknown correlation type '%s'", r.Model.CorrelationType)
	}

	if !isSupported {
		return false, fmt.Sprintf("correlation type '%s' not yet supported", r.Model.CorrelationType)
	}

	return true, ""
}

// GetTimeWindow extracts the time window from rule parameters
func (r *DetectionRule) GetTimeWindow() (time.Duration, error) {
	timeWindowStr, ok := r.Model.Parameters["time_window"].(string)
	if !ok {
		return 0, fmt.Errorf("time_window not found or not a string")
	}

	return time.ParseDuration(timeWindowStr)
}

// GetThreshold extracts the threshold value and operator from rule parameters
func (r *DetectionRule) GetThreshold() (value float64, operator string, err error) {
	thresholdMap, ok := r.Model.Parameters["threshold"].(map[string]interface{})
	if !ok {
		return 0, "", fmt.Errorf("threshold not found or not a map")
	}

	// Value can be int or float
	switch v := thresholdMap["value"].(type) {
	case float64:
		value = v
	case int:
		value = float64(v)
	default:
		return 0, "", fmt.Errorf("threshold value is not a number")
	}

	operator, ok = thresholdMap["operator"].(string)
	if !ok {
		return 0, "", fmt.Errorf("threshold operator not found or not a string")
	}

	return value, operator, nil
}

// GetQueryFilter extracts the query filter from rule parameters
func (r *DetectionRule) GetQueryFilter() (*QueryFilter, error) {
	queryMap, ok := r.Model.Parameters["query"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("query not found or not a map")
	}

	filterMap, ok := queryMap["filter"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("filter not found in query")
	}

	return parseQueryFilter(filterMap)
}

// parseQueryFilter recursively parses a filter map into a QueryFilter struct
func parseQueryFilter(filterMap map[string]interface{}) (*QueryFilter, error) {
	filter := &QueryFilter{}

	// Check if this is a compound filter (has "type" and "conditions")
	if filterType, ok := filterMap["type"].(string); ok {
		filter.Type = filterType

		if conditions, ok := filterMap["conditions"].([]interface{}); ok {
			for _, condMap := range conditions {
				condMapTyped, ok := condMap.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("condition is not a map")
				}

				subFilter, err := parseQueryFilter(condMapTyped)
				if err != nil {
					return nil, fmt.Errorf("failed to parse sub-filter: %w", err)
				}

				filter.Conditions = append(filter.Conditions, *subFilter)
			}
		}
	} else {
		// Simple filter (has field, operator, value)
		if field, ok := filterMap["field"].(string); ok {
			filter.Field = field
		} else {
			return nil, fmt.Errorf("field not found in filter")
		}

		if operator, ok := filterMap["operator"].(string); ok {
			filter.Operator = operator
		} else {
			return nil, fmt.Errorf("operator not found in filter")
		}

		filter.Value = filterMap["value"]
	}

	return filter, nil
}

// GetGroupByFields extracts the group_by fields from rule parameters
func (r *DetectionRule) GetGroupByFields() ([]string, error) {
	groupByRaw, ok := r.Model.Parameters["group_by"]
	if !ok {
		return nil, nil // group_by is optional
	}

	groupBySlice, ok := groupByRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("group_by is not an array")
	}

	var fields []string
	for _, item := range groupBySlice {
		field, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("group_by item is not a string")
		}
		fields = append(fields, field)
	}

	return fields, nil
}

// GetValueCountField extracts the field to count distinct values for (value_count rules)
func (r *DetectionRule) GetValueCountField() (string, error) {
	if r.Model.CorrelationType != "value_count" {
		return "", fmt.Errorf("rule is not a value_count type")
	}

	field, ok := r.Model.Parameters["field"].(string)
	if !ok {
		return "", fmt.Errorf("field not found for value_count rule")
	}

	return field, nil
}

// MatchesFilter checks if an event matches the given filter
func MatchesFilter(event map[string]interface{}, filter *QueryFilter) bool {
	if filter.Type != "" {
		// Compound filter
		switch filter.Type {
		case "and":
			for _, cond := range filter.Conditions {
				if !MatchesFilter(event, &cond) {
					return false
				}
			}
			return true
		case "or":
			for _, cond := range filter.Conditions {
				if MatchesFilter(event, &cond) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}

	// Simple filter - get field value from event
	fieldValue := getFieldValue(event, filter.Field)
	if fieldValue == nil {
		return false
	}

	// Compare based on operator
	return compareValues(fieldValue, filter.Operator, filter.Value)
}

// getFieldValue extracts a nested field value using dot notation (e.g., ".actor.user.name")
func getFieldValue(event map[string]interface{}, fieldPath string) interface{} {
	// Remove leading dot if present
	fieldPath = strings.TrimPrefix(fieldPath, ".")

	parts := strings.Split(fieldPath, ".")
	var current interface{} = event

	for _, part := range parts {
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = currentMap[part]
		if current == nil {
			return nil
		}
	}

	return current
}

// compareValues compares a field value against a filter value using the specified operator
func compareValues(fieldValue interface{}, operator string, filterValue interface{}) bool {
	switch operator {
	case "eq":
		return fieldValue == filterValue
	case "ne":
		return fieldValue != filterValue
	case "gt":
		return compareNumeric(fieldValue, filterValue, func(a, b float64) bool { return a > b })
	case "gte":
		return compareNumeric(fieldValue, filterValue, func(a, b float64) bool { return a >= b })
	case "lt":
		return compareNumeric(fieldValue, filterValue, func(a, b float64) bool { return a < b })
	case "lte":
		return compareNumeric(fieldValue, filterValue, func(a, b float64) bool { return a <= b })
	case "in":
		// filterValue should be an array
		filterSlice, ok := filterValue.([]interface{})
		if !ok {
			return false
		}
		for _, item := range filterSlice {
			if fieldValue == item {
				return true
			}
		}
		return false
	case "contains":
		fieldStr, ok1 := fieldValue.(string)
		filterStr, ok2 := filterValue.(string)
		if ok1 && ok2 {
			return strings.Contains(fieldStr, filterStr)
		}
		return false
	default:
		return false
	}
}

// compareNumeric compares two values numerically
func compareNumeric(a, b interface{}, compareFn func(float64, float64) bool) bool {
	var aFloat, bFloat float64

	switch v := a.(type) {
	case float64:
		aFloat = v
	case int:
		aFloat = float64(v)
	default:
		return false
	}

	switch v := b.(type) {
	case float64:
		bFloat = v
	case int:
		bFloat = float64(v)
	default:
		return false
	}

	return compareFn(aFloat, bFloat)
}
