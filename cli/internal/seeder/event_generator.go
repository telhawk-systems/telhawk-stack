package seeder

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

// eventGenerator handles event creation and timing logic
type eventGenerator struct{}

// calculateEventTime calculates the timestamp for an event with jitter
func (eg *eventGenerator) calculateEventTime(now time.Time, timeWindow time.Duration, index, total int) time.Time {
	// Use same jittered distribution as baseline events
	baseInterval := float64(timeWindow) / float64(total)
	baseOffset := time.Duration(float64(index) * baseInterval)

	// Add jitter: ±40% of base interval
	jitterRange := baseInterval * 0.4
	jitter := time.Duration((rand.Float64()*2.0 - 1.0) * jitterRange)

	totalOffset := baseOffset + jitter
	if totalOffset < 0 {
		totalOffset = 0
	}
	if totalOffset > timeWindow {
		totalOffset = timeWindow
	}

	// Events are placed going backwards from now
	return now.Add(-(timeWindow - totalOffset))
}

// createEventMatchingFilter creates an event that matches the given filter
func (eg *eventGenerator) createEventMatchingFilter(filter *QueryFilter) map[string]interface{} {
	event := make(map[string]interface{})

	// Apply all filter conditions to the event
	eg.applyFilterToEvent(event, filter)

	return event
}

// createEventMatchingFilterWithSeed creates an event matching the filter, using seeded values where provided
// This ensures consistency when generating multiple events that need the same group_by values
func (eg *eventGenerator) createEventMatchingFilterWithSeed(filter *QueryFilter, seedValues map[string]interface{}) map[string]interface{} {
	event := make(map[string]interface{})

	// Pre-populate with seed values
	for field, value := range seedValues {
		setFieldValue(event, field, value)
	}

	// Apply filter conditions - this will use existing values where they satisfy the filter
	eg.applyFilterToEventWithSeed(event, filter, seedValues)

	return event
}

// applyFilterToEventWithSeed applies filter conditions, using seed values when available
func (eg *eventGenerator) applyFilterToEventWithSeed(event map[string]interface{}, filter *QueryFilter, seedValues map[string]interface{}) {
	if filter.Type != "" {
		switch filter.Type {
		case "and":
			// AND filter - apply ALL conditions
			for _, cond := range filter.Conditions {
				eg.applyFilterToEventWithSeed(event, &cond, seedValues)
			}
		case "or":
			// OR filter - find a branch that matches the seed values, or pick one randomly
			if len(filter.Conditions) > 0 {
				// Try to find a branch that's compatible with seed values
				selectedBranch := eg.findCompatibleORBranch(filter.Conditions, seedValues)
				if selectedBranch >= 0 {
					eg.applyFilterToEventWithSeed(event, &filter.Conditions[selectedBranch], seedValues)
				} else {
					// No compatible branch found, pick randomly (shouldn't happen if seed was from same filter)
					selectedBranch = rand.Intn(len(filter.Conditions))
					eg.applyFilterToEventWithSeed(event, &filter.Conditions[selectedBranch], seedValues)
				}
			}
		case "not":
			// NOT filter - generate values that DON'T match the conditions
			eg.applyNonMatchingConditions(event, filter.Conditions)
		default:
			for _, cond := range filter.Conditions {
				eg.applyFilterToEventWithSeed(event, &cond, seedValues)
			}
		}
	} else {
		// Simple filter - check if we have a seed value for this field
		if seedValue, hasSeed := seedValues[filter.Field]; hasSeed {
			// Verify seed value satisfies the filter condition
			if eg.valueSatisfiesFilter(seedValue, filter) {
				setFieldValue(event, filter.Field, seedValue)
				return
			}
		}
		// No compatible seed, apply filter normally
		eg.applySimpleFilter(event, filter)
	}
}

// findCompatibleORBranch finds an OR branch index that's compatible with seed values
func (eg *eventGenerator) findCompatibleORBranch(branches []QueryFilter, seedValues map[string]interface{}) int {
	for i, branch := range branches {
		if eg.branchCompatibleWithSeed(&branch, seedValues) {
			return i
		}
	}
	return -1 // No compatible branch found
}

// branchCompatibleWithSeed checks if a filter branch is compatible with seed values
func (eg *eventGenerator) branchCompatibleWithSeed(filter *QueryFilter, seedValues map[string]interface{}) bool {
	if filter.Type != "" {
		switch filter.Type {
		case "and":
			// All conditions must be compatible
			for _, cond := range filter.Conditions {
				if !eg.branchCompatibleWithSeed(&cond, seedValues) {
					return false
				}
			}
			return true
		case "or":
			// At least one condition must be compatible
			for _, cond := range filter.Conditions {
				if eg.branchCompatibleWithSeed(&cond, seedValues) {
					return true
				}
			}
			return false
		case "not":
			// For NOT, seed values should NOT match the inner conditions
			for _, cond := range filter.Conditions {
				if seedValue, hasSeed := seedValues[cond.Field]; hasSeed {
					if eg.valueSatisfiesFilter(seedValue, &cond) {
						return false // Seed matches what we're trying to NOT match
					}
				}
			}
			return true
		}
	}

	// Simple filter - check if seed value satisfies it
	if seedValue, hasSeed := seedValues[filter.Field]; hasSeed {
		return eg.valueSatisfiesFilter(seedValue, filter)
	}
	// No seed for this field, compatible by default
	return true
}

// valueSatisfiesFilter checks if a value satisfies a simple filter condition
func (eg *eventGenerator) valueSatisfiesFilter(value interface{}, filter *QueryFilter) bool {
	switch filter.Operator {
	case "eq":
		return value == filter.Value
	case "in":
		if arr, ok := filter.Value.([]interface{}); ok {
			for _, item := range arr {
				if value == item {
					return true
				}
			}
		}
		return false
	case "contains":
		if strVal, ok := value.(string); ok {
			if filterStr, ok := filter.Value.(string); ok {
				return strings.Contains(strVal, filterStr)
			}
		}
		return false
	default:
		return true // Unknown operator, assume compatible
	}
}

// applyFilterToEvent recursively applies filter conditions to an event
// Handles AND (all conditions), OR (pick one branch), and NOT (generate non-matching values)
func (eg *eventGenerator) applyFilterToEvent(event map[string]interface{}, filter *QueryFilter) {
	if filter.Type != "" {
		switch filter.Type {
		case "and":
			// AND filter - apply ALL conditions
			for _, cond := range filter.Conditions {
				eg.applyFilterToEvent(event, &cond)
			}
		case "or":
			// OR filter - pick ONE branch randomly and fully apply it
			if len(filter.Conditions) > 0 {
				// Pick a random branch
				selectedBranch := rand.Intn(len(filter.Conditions))
				eg.applyFilterToEvent(event, &filter.Conditions[selectedBranch])
			}
		case "not":
			// NOT filter - generate values that DON'T match the conditions
			// Use applyNonMatchingConditions to handle multiple conditions on the same field
			eg.applyNonMatchingConditions(event, filter.Conditions)
		default:
			// Unknown type - apply all conditions as fallback
			for _, cond := range filter.Conditions {
				eg.applyFilterToEvent(event, &cond)
			}
		}
	} else {
		// Simple filter - set the field value
		eg.applySimpleFilter(event, filter)
	}
}

// applySimpleFilter applies a simple (non-compound) filter condition
func (eg *eventGenerator) applySimpleFilter(event map[string]interface{}, filter *QueryFilter) {
	value := filter.Value

	switch filter.Operator {
	case "in":
		// Pick a random element from the array
		if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
			value = arr[rand.Intn(len(arr))]
		}
	case "contains":
		// Generate a string that contains the required substring
		value = eg.generateContainingString(filter.Field, filter.Value)
	case "regex", "matches":
		// For regex/matches, generate a plausible matching value
		value = eg.generateRegexMatchingValue(filter.Field, filter.Value)
	case "eq":
		// Use exact value
	case "gt", "gte":
		// For numeric comparisons, ensure we exceed the threshold
		if numVal, ok := toFloat64(value); ok {
			value = numVal + 1
		}
	case "lt", "lte":
		// For numeric comparisons, ensure we're below the threshold
		if numVal, ok := toFloat64(value); ok {
			if numVal > 1 {
				value = numVal - 1
			} else {
				value = 0
			}
		}
	}

	setFieldValue(event, filter.Field, value)
}

// applyNonMatchingFilter generates values that DON'T match the condition
// Used for NOT filters
func (eg *eventGenerator) applyNonMatchingFilter(event map[string]interface{}, filter *QueryFilter) {
	if filter.Type != "" {
		// Compound filter within NOT - apply De Morgan's laws
		switch filter.Type {
		case "and":
			// NOT(A AND B) = NOT(A) OR NOT(B) - fail at least one condition
			// We'll fail ALL conditions to be safe
			for _, cond := range filter.Conditions {
				eg.applyNonMatchingFilter(event, &cond)
			}
		case "or":
			// NOT(A OR B) = NOT(A) AND NOT(B) - fail all conditions
			for _, cond := range filter.Conditions {
				eg.applyNonMatchingFilter(event, &cond)
			}
		default:
			// No type but has conditions - need to fail all conditions
			// Group conditions by field to handle multiple contains on same field
			eg.applyNonMatchingConditions(event, filter.Conditions)
		}
	} else {
		// Simple filter - generate non-matching value
		eg.applyNonMatchingSimpleFilter(event, filter)
	}
}

// applyNonMatchingConditions handles multiple conditions, grouping by field
// to ensure a single value that fails all conditions for each field
func (eg *eventGenerator) applyNonMatchingConditions(event map[string]interface{}, conditions []QueryFilter) {
	// Group conditions by field
	fieldConditions := make(map[string][]QueryFilter)
	for _, cond := range conditions {
		if cond.Field != "" {
			fieldConditions[cond.Field] = append(fieldConditions[cond.Field], cond)
		} else if cond.Type != "" {
			// Nested compound filter - recurse
			eg.applyNonMatchingFilter(event, &cond)
		}
	}

	// For each field, generate a value that fails all conditions for that field
	for field, conds := range fieldConditions {
		value := eg.generateValueFailingAllConditions(field, conds)
		setFieldValue(event, field, value)
	}
}

// generateValueFailingAllConditions generates a value that fails all given conditions for a field
func (eg *eventGenerator) generateValueFailingAllConditions(field string, conditions []QueryFilter) interface{} {
	// Collect all "contains" substrings to avoid
	var substringsToAvoid []string
	var valuesToAvoid []interface{}
	var arraysToAvoid [][]interface{}

	for _, cond := range conditions {
		switch cond.Operator {
		case "contains":
			if s, ok := cond.Value.(string); ok {
				substringsToAvoid = append(substringsToAvoid, s)
			}
		case "eq":
			valuesToAvoid = append(valuesToAvoid, cond.Value)
		case "in":
			if arr, ok := cond.Value.([]interface{}); ok {
				arraysToAvoid = append(arraysToAvoid, arr)
			}
		}
	}

	// Generate a value that avoids all substrings
	if len(substringsToAvoid) > 0 {
		return eg.generateStringAvoidingAll(field, substringsToAvoid)
	}

	// Generate a value not in any of the arrays and not equal to any values
	if len(arraysToAvoid) > 0 || len(valuesToAvoid) > 0 {
		return eg.generateValueAvoidingAll(field, valuesToAvoid, arraysToAvoid)
	}

	// Fallback
	return eg.generateDifferentValue(field, nil)
}

// generateStringAvoidingAll generates a string that doesn't contain any of the given substrings
func (eg *eventGenerator) generateStringAvoidingAll(field string, substrings []string) string {
	fieldLower := strings.ToLower(field)

	// Pre-defined safe values based on field type
	var candidates []string

	if strings.Contains(fieldLower, "path") {
		candidates = []string{
			"C:\\Users\\Public\\safe_file.txt",
			"C:\\Temp\\legitimate.exe",
			"/home/user/document.txt",
			"/tmp/safe_script.sh",
			"D:\\Data\\report.pdf",
		}
	} else if strings.Contains(fieldLower, "cmd_line") {
		candidates = []string{
			"notepad.exe readme.txt",
			"/usr/bin/ls -la",
			"python3 script.py",
			"cat /etc/hosts",
		}
	} else {
		candidates = []string{
			"safe_value_1",
			"legitimate_data",
			"normal_content",
			"standard_entry",
		}
	}

	// Find a candidate that doesn't contain any of the substrings
	for _, candidate := range candidates {
		containsAny := false
		for _, substr := range substrings {
			if strings.Contains(candidate, substr) {
				containsAny = true
				break
			}
		}
		if !containsAny {
			return candidate
		}
	}

	// Generate a random word and verify it
	for i := 0; i < 100; i++ {
		candidate := gofakeit.Word()
		containsAny := false
		for _, substr := range substrings {
			if strings.Contains(candidate, substr) {
				containsAny = true
				break
			}
		}
		if !containsAny {
			return candidate
		}
	}

	return "safe_fallback_value"
}

// generateValueAvoidingAll generates a value not equal to any specific values and not in any arrays
func (eg *eventGenerator) generateValueAvoidingAll(field string, valuesToAvoid []interface{}, arraysToAvoid [][]interface{}) interface{} {
	// Create a set of all values to avoid
	avoid := make(map[interface{}]bool)
	for _, v := range valuesToAvoid {
		avoid[v] = true
	}
	for _, arr := range arraysToAvoid {
		for _, v := range arr {
			avoid[v] = true
		}
	}

	// Generate candidates until we find one not in the set
	for i := 0; i < 100; i++ {
		candidate := eg.generateDifferentValue(field, nil)
		if !avoid[candidate] {
			return candidate
		}
	}

	return "unique_safe_value"
}

// applyNonMatchingSimpleFilter generates a value that doesn't match a simple filter
func (eg *eventGenerator) applyNonMatchingSimpleFilter(event map[string]interface{}, filter *QueryFilter) {
	var nonMatchingValue interface{}

	switch filter.Operator {
	case "eq":
		// Generate any value except the specified one
		nonMatchingValue = eg.generateDifferentValue(filter.Field, filter.Value)
	case "in":
		// Generate a value NOT in the array
		nonMatchingValue = eg.generateValueNotInArray(filter.Field, filter.Value)
	case "contains":
		// Generate a string that doesn't contain the substring
		nonMatchingValue = eg.generateNonContainingString(filter.Field, filter.Value)
	default:
		// For other operators, generate a generic different value
		nonMatchingValue = eg.generateDifferentValue(filter.Field, filter.Value)
	}

	setFieldValue(event, filter.Field, nonMatchingValue)
}

// generateContainingString generates a string that contains the required substring
func (eg *eventGenerator) generateContainingString(field string, value interface{}) string {
	substr, ok := value.(string)
	if !ok {
		return fmt.Sprintf("%v", value)
	}

	// Generate context around the substring based on field type
	fieldLower := strings.ToLower(field)

	if strings.Contains(fieldLower, "cmd_line") || strings.Contains(fieldLower, "cmdline") {
		// Command line - wrap in realistic command context
		return eg.generateCmdLineWithSubstring(substr)
	}
	if strings.Contains(fieldLower, "path") {
		// File path - wrap in path context
		return eg.generatePathWithSubstring(substr)
	}
	if strings.Contains(fieldLower, "process.name") {
		// Process name - just return the value (might be exact)
		return substr
	}

	// Default: prefix with some context
	return fmt.Sprintf("prefix_%s_suffix", substr)
}

// generateCmdLineWithSubstring generates a realistic command line containing the substring
func (eg *eventGenerator) generateCmdLineWithSubstring(substr string) string {
	// Common command line patterns based on the substring
	substrLower := strings.ToLower(substr)

	// PowerShell cmdlets
	if strings.HasPrefix(substrLower, "invoke-") ||
		strings.HasPrefix(substrLower, "get-") ||
		strings.HasPrefix(substrLower, "set-") ||
		strings.HasPrefix(substrLower, "new-") ||
		strings.HasPrefix(substrLower, "remove-") ||
		strings.HasPrefix(substrLower, "clear-") ||
		strings.HasPrefix(substrLower, "register-") ||
		strings.HasPrefix(substrLower, "compress-") {
		return fmt.Sprintf("powershell.exe -ExecutionPolicy Bypass -Command \"%s\"", substr)
	}

	// Windows commands
	if strings.Contains(substrLower, "wevtutil") ||
		strings.Contains(substrLower, "schtasks") ||
		strings.Contains(substrLower, "auditpol") {
		return fmt.Sprintf("%s /q", substr)
	}

	// Path-like substrings
	if strings.HasPrefix(substr, "/") || strings.HasPrefix(substr, "C:\\") {
		return fmt.Sprintf("cat %s", substr)
	}

	// API calls / injection patterns
	if strings.Contains(substrLower, "alloc") ||
		strings.Contains(substrLower, "memory") ||
		strings.Contains(substrLower, "thread") ||
		strings.Contains(substrLower, "inject") {
		return fmt.Sprintf("powershell.exe -Command \"[Reflection.Assembly]::LoadWithPartialName('Microsoft.CSharp'); %s\"", substr)
	}

	// Flags (e.g., "-p", "/create", "-e")
	if strings.HasPrefix(substr, "-") || strings.HasPrefix(substr, "/") {
		return fmt.Sprintf("tool.exe %s argument", substr)
	}

	// Default: wrap in a generic command
	return fmt.Sprintf("cmd.exe /c %s", substr)
}

// generatePathWithSubstring generates a path containing the substring
func (eg *eventGenerator) generatePathWithSubstring(substr string) string {
	// If it's already a full path, return as-is
	if strings.HasPrefix(substr, "/") || strings.HasPrefix(substr, "C:\\") {
		return substr
	}

	// If it looks like an extension
	if strings.HasPrefix(substr, ".") {
		return fmt.Sprintf("C:\\Users\\admin\\Documents\\file%s", substr)
	}

	// Otherwise wrap it
	return fmt.Sprintf("/var/tmp/%s", substr)
}

// generateRegexMatchingValue generates a value that would match a regex pattern
func (eg *eventGenerator) generateRegexMatchingValue(field string, value interface{}) string {
	pattern, ok := value.(string)
	if !ok {
		return fmt.Sprintf("%v", value)
	}

	// Simple heuristic: if pattern contains literal text, use that
	// Remove common regex metacharacters to extract literals
	literal := strings.NewReplacer(
		"^", "", "$", "", ".*", "", ".+", "", "\\d+", "123", "\\w+", "word",
		"[", "", "]", "", "(", "", ")", "", "|", "", "?", "", "*", "", "+", "",
	).Replace(pattern)

	if literal != "" {
		return literal
	}

	// Fallback to field-based generation
	return gofakeit.Word()
}

// generateDifferentValue generates a value different from the specified one
func (eg *eventGenerator) generateDifferentValue(field string, excludeValue interface{}) interface{} {
	fieldLower := strings.ToLower(field)

	// Generate based on field type
	if strings.Contains(fieldLower, "path") {
		// Generate a different path
		paths := []string{
			"C:\\Program Files\\legitimate.exe",
			"/usr/local/bin/safe_program",
			"C:\\Windows\\System32\\notepad.exe",
			"/opt/app/service",
		}
		return paths[rand.Intn(len(paths))]
	}

	if strings.Contains(fieldLower, "process.name") {
		// Generate a different process name
		return "legitimate_process.exe"
	}

	// Default: return a random word
	return gofakeit.Word()
}

// generateValueNotInArray generates a value not in the specified array
func (eg *eventGenerator) generateValueNotInArray(field string, value interface{}) interface{} {
	arr, ok := value.([]interface{})
	if !ok {
		return eg.generateDifferentValue(field, value)
	}

	// Create a set of excluded values
	excluded := make(map[interface{}]bool)
	for _, v := range arr {
		excluded[v] = true
	}

	// Generate candidates until we find one not in the array
	for i := 0; i < 100; i++ {
		candidate := eg.generateDifferentValue(field, nil)
		if !excluded[candidate] {
			return candidate
		}
	}

	// Fallback
	return "not_in_list_value"
}

// generateNonContainingString generates a string that doesn't contain the substring
func (eg *eventGenerator) generateNonContainingString(field string, value interface{}) string {
	substr, ok := value.(string)
	if !ok {
		return gofakeit.Word()
	}

	// Generate something that clearly doesn't contain the substring
	fieldLower := strings.ToLower(field)

	if strings.Contains(fieldLower, "path") {
		result := "C:\\SafePath\\legitimate_file.txt"
		if !strings.Contains(result, substr) {
			return result
		}
	}

	if strings.Contains(fieldLower, "cmd_line") {
		result := "notepad.exe document.txt"
		if !strings.Contains(result, substr) {
			return result
		}
	}

	// Generate random word that doesn't contain the substring
	for i := 0; i < 100; i++ {
		candidate := gofakeit.Word()
		if !strings.Contains(candidate, substr) {
			return candidate
		}
	}

	return "safe_value"
}

// toFloat64 converts a value to float64 if possible
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}

// applyGroupByValues applies group_by values to an event
// Only sets values if the field is not already set (preserves filter-set values)
func (eg *eventGenerator) applyGroupByValues(event map[string]interface{}, groupByValues map[string]interface{}) {
	for field, value := range groupByValues {
		// Check if the field already has a value (set by filter)
		existingValue := getFieldValue(event, field)
		if existingValue == nil {
			// Field not set, use the group_by value
			setFieldValue(event, field, value)
		}
		// If field already has a value, keep it (was set by filter to match rule criteria)
	}
}

// determineSourceType determines the appropriate OCSF source type based on event content
func (eg *eventGenerator) determineSourceType(event map[string]interface{}) string {
	classUID, ok := event["class_uid"]
	if !ok {
		return "ocsf:generic"
	}

	switch classUID {
	case 3002:
		return "ocsf:authentication"
	case 4001:
		return "ocsf:network_activity"
	case 1007:
		return "ocsf:process_activity"
	case 4006:
		return "ocsf:file_activity"
	case 4003:
		return "ocsf:dns_activity"
	case 4002:
		return "ocsf:http_activity"
	case 2004:
		return "ocsf:detection_finding"
	default:
		return "ocsf:generic"
	}
}
