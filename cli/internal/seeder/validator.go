package seeder

import (
	"fmt"
	"log"
)

// ValidateEventsMatchRule validates that generated events match the rule criteria
// This is a simplified validator - for full validation, would need to integrate
// with the alerting service's correlation engine
func ValidateEventsMatchRule(rule *DetectionRule, events []HECEvent) error {
	if len(events) == 0 {
		return fmt.Errorf("no events generated")
	}

	// Get filter to validate against
	filter, err := rule.GetQueryFilter()
	if err != nil {
		return fmt.Errorf("failed to get query filter: %w", err)
	}

	// Validate all events match the filter
	matchCount := 0
	for i, event := range events {
		if MatchesFilter(event.Event, filter) {
			matchCount++
		} else {
			log.Printf("WARN: Event %d does not match filter", i)
		}
	}

	if matchCount == 0 {
		return fmt.Errorf("no events match the rule filter")
	}

	log.Printf("  ✓ Validation: %d/%d events match rule criteria", matchCount, len(events))

	// Additional validation based on correlation type
	switch rule.Model.CorrelationType {
	case "event_count":
		return validateEventCount(rule, events)
	case "value_count":
		return validateValueCount(rule, events)
	default:
		return nil // No additional validation for other types
	}
}

// validateEventCount validates event_count rule requirements
func validateEventCount(rule *DetectionRule, events []HECEvent) error {
	threshold, operator, err := rule.GetThreshold()
	if err != nil {
		return fmt.Errorf("failed to get threshold: %w", err)
	}

	eventCount := float64(len(events))

	// Check if event count meets threshold
	switch operator {
	case "gt":
		if eventCount <= threshold {
			return fmt.Errorf("event count %.0f does not exceed threshold %.0f", eventCount, threshold)
		}
	case "gte":
		if eventCount < threshold {
			return fmt.Errorf("event count %.0f is less than threshold %.0f", eventCount, threshold)
		}
	case "eq":
		if eventCount != threshold {
			return fmt.Errorf("event count %.0f does not equal threshold %.0f", eventCount, threshold)
		}
	case "lt":
		if eventCount >= threshold {
			return fmt.Errorf("event count %.0f is not less than threshold %.0f", eventCount, threshold)
		}
	case "lte":
		if eventCount > threshold {
			return fmt.Errorf("event count %.0f exceeds threshold %.0f", eventCount, threshold)
		}
	}

	// Validate group_by consistency
	groupByFields, err := rule.GetGroupByFields()
	if err != nil {
		return fmt.Errorf("failed to get group_by fields: %w", err)
	}

	if len(groupByFields) > 0 {
		if err := validateGroupByConsistency(events, groupByFields); err != nil {
			return fmt.Errorf("group_by validation failed: %w", err)
		}
		log.Printf("  ✓ Group by fields are consistent across all events")
	}

	return nil
}

// validateValueCount validates value_count rule requirements
func validateValueCount(rule *DetectionRule, events []HECEvent) error {
	threshold, operator, err := rule.GetThreshold()
	if err != nil {
		return fmt.Errorf("failed to get threshold: %w", err)
	}

	countField, err := rule.GetValueCountField()
	if err != nil {
		return fmt.Errorf("failed to get value count field: %w", err)
	}

	// Count unique values for the specified field
	uniqueValues := make(map[interface{}]bool)
	for _, event := range events {
		value := getFieldValue(event.Event, countField)
		if value != nil {
			uniqueValues[value] = true
		}
	}

	uniqueCount := float64(len(uniqueValues))

	// Check if unique count meets threshold
	switch operator {
	case "gt":
		if uniqueCount <= threshold {
			return fmt.Errorf("unique value count %.0f does not exceed threshold %.0f", uniqueCount, threshold)
		}
	case "gte":
		if uniqueCount < threshold {
			return fmt.Errorf("unique value count %.0f is less than threshold %.0f", uniqueCount, threshold)
		}
	case "eq":
		if uniqueCount != threshold {
			return fmt.Errorf("unique value count %.0f does not equal threshold %.0f", uniqueCount, threshold)
		}
	}

	log.Printf("  ✓ Generated %d unique values for field %s", len(uniqueValues), countField)

	// Validate group_by consistency
	groupByFields, err := rule.GetGroupByFields()
	if err != nil {
		return fmt.Errorf("failed to get group_by fields: %w", err)
	}

	if len(groupByFields) > 0 {
		if err := validateGroupByConsistency(events, groupByFields); err != nil {
			return fmt.Errorf("group_by validation failed: %w", err)
		}
		log.Printf("  ✓ Group by fields are consistent across all events")
	}

	return nil
}

// validateGroupByConsistency checks that all events have the same values for group_by fields
func validateGroupByConsistency(events []HECEvent, groupByFields []string) error {
	if len(events) == 0 {
		return nil
	}

	// Get expected values from first event
	expectedValues := make(map[string]interface{})
	for _, field := range groupByFields {
		expectedValues[field] = getFieldValue(events[0].Event, field)
	}

	// Verify all other events have the same values
	for i, event := range events {
		for _, field := range groupByFields {
			actualValue := getFieldValue(event.Event, field)
			expectedValue := expectedValues[field]

			if actualValue != expectedValue {
				return fmt.Errorf("event %d has inconsistent value for %s: got %v, expected %v",
					i, field, actualValue, expectedValue)
			}
		}
	}

	return nil
}

// debugFilterMismatch logs details about why an event doesn't match a filter
func debugFilterMismatch(event map[string]interface{}, filter *QueryFilter, indent string) {
	if filter.Type != "" {
		log.Printf("%sCompound filter type=%s", indent, filter.Type)
		for i, cond := range filter.Conditions {
			matches := MatchesFilter(event, &cond)
			log.Printf("%s  Condition %d: matches=%v", indent, i, matches)
			if !matches {
				debugFilterMismatch(event, &cond, indent+"    ")
			}
		}
	} else {
		value := getFieldValue(event, filter.Field)
		matches := MatchesFilter(event, filter)
		log.Printf("%sSimple filter: field=%s op=%s filterValue=%v actualValue=%v matches=%v",
			indent, filter.Field, filter.Operator, filter.Value, value, matches)
	}
}

// ValidateRuleCanBeGenerated checks if a rule can be used for event generation
func ValidateRuleCanBeGenerated(rule *DetectionRule) error {
	// Check if rule is supported
	if supported, reason := rule.IsSupported(); !supported {
		return fmt.Errorf("rule not supported: %s", reason)
	}

	// Validate required parameters exist
	if _, err := rule.GetTimeWindow(); err != nil {
		return fmt.Errorf("invalid time_window: %w", err)
	}

	if _, _, err := rule.GetThreshold(); err != nil {
		return fmt.Errorf("invalid threshold: %w", err)
	}

	filter, err := rule.GetQueryFilter()
	if err != nil {
		return fmt.Errorf("invalid query filter: %w", err)
	}

	// Validate OCSF field paths in the filter
	ocsfVal := newOCSFValidator()
	if fieldErrors := ocsfVal.ValidateFilterFields(filter); len(fieldErrors) > 0 {
		errMsg := "OCSF schema validation failed:\n"
		for _, e := range fieldErrors {
			errMsg += fmt.Sprintf("  - %s\n", e.Error())
		}
		return fmt.Errorf("%s", errMsg)
	}

	// Type-specific validation
	switch rule.Model.CorrelationType {
	case "value_count":
		countField, err := rule.GetValueCountField()
		if err != nil {
			return fmt.Errorf("invalid value count field: %w", err)
		}
		// Validate the value count field path
		if err := ocsfVal.ValidateFieldPath(countField); err != nil {
			return fmt.Errorf("invalid value count field path '%s': %w", countField, err)
		}
	}

	// Validate group_by field paths
	groupByFields, err := rule.GetGroupByFields()
	if err == nil && len(groupByFields) > 0 {
		for _, field := range groupByFields {
			if err := ocsfVal.ValidateFieldPath(field); err != nil {
				return fmt.Errorf("invalid group_by field path '%s': %w", field, err)
			}
		}
	}

	return nil
}
