package seeder

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

// RuleBasedGenerator generates events that match detection rules
type RuleBasedGenerator struct {
	Rule       *DetectionRule
	Multiplier float64                // Exceed threshold by this factor (default: 1.5)
	Params     map[string]interface{} // Override parameters from YAML config
}

// NewRuleBasedGenerator creates a new rule-based event generator
func NewRuleBasedGenerator(rule *DetectionRule, multiplier float64, params map[string]interface{}) *RuleBasedGenerator {
	if multiplier <= 0 {
		multiplier = 1.5 // Default multiplier
	}

	return &RuleBasedGenerator{
		Rule:       rule,
		Multiplier: multiplier,
		Params:     params,
	}
}

// GenerateEvents creates events based on the rule type
func (g *RuleBasedGenerator) GenerateEvents() ([]HECEvent, error) {
	// Check if rule is supported
	if supported, reason := g.Rule.IsSupported(); !supported {
		return nil, fmt.Errorf("rule not supported: %s", reason)
	}

	switch g.Rule.Model.CorrelationType {
	case "event_count":
		return g.generateForEventCount()
	case "value_count":
		return g.generateForValueCount()
	default:
		return nil, fmt.Errorf("unsupported correlation type: %s", g.Rule.Model.CorrelationType)
	}
}

// generateForEventCount generates events for event_count correlation type
func (g *RuleBasedGenerator) generateForEventCount() ([]HECEvent, error) {
	// Extract rule parameters
	threshold, operator, err := g.Rule.GetThreshold()
	if err != nil {
		return nil, fmt.Errorf("failed to get threshold: %w", err)
	}

	timeWindow, err := g.Rule.GetTimeWindow()
	if err != nil {
		return nil, fmt.Errorf("failed to get time window: %w", err)
	}

	filter, err := g.Rule.GetQueryFilter()
	if err != nil {
		return nil, fmt.Errorf("failed to get query filter: %w", err)
	}

	groupByFields, err := g.Rule.GetGroupByFields()
	if err != nil {
		return nil, fmt.Errorf("failed to get group_by fields: %w", err)
	}

	// Calculate number of events to generate
	eventCount := g.calculateEventCount(threshold, operator)

	log.Printf("Generating events for rule '%s' (%s):", g.Rule.Name, g.Rule.Model.CorrelationType)
	log.Printf("  Rule threshold: %.0f events in %s", threshold, timeWindow)
	log.Printf("  Generating: %d events (%.1fx multiplier)", eventCount, g.Multiplier)

	// Generate events
	events := make([]HECEvent, eventCount)
	now := time.Now()

	// Generate consistent values for group_by fields
	groupByValues := g.generateGroupByValues(groupByFields)

	for i := 0; i < eventCount; i++ {
		// Calculate event time with jitter
		eventTime := g.calculateEventTime(now, timeWindow, i, eventCount)

		// Create base event matching the filter
		event := g.createEventMatchingFilter(filter)

		// Apply group_by values to ensure events are correlated
		g.applyGroupByValues(event, groupByValues)

		// Create HEC event
		events[i] = HECEvent{
			Time:       float64(eventTime.Unix()) + float64(eventTime.Nanosecond())/1e9,
			Event:      event,
			SourceType: g.determineSourceType(event),
		}
	}

	log.Printf("  ✓ Generated %d events", len(events))

	return events, nil
}

// generateForValueCount generates events for value_count correlation type
func (g *RuleBasedGenerator) generateForValueCount() ([]HECEvent, error) {
	// Extract rule parameters
	threshold, operator, err := g.Rule.GetThreshold()
	if err != nil {
		return nil, fmt.Errorf("failed to get threshold: %w", err)
	}

	timeWindow, err := g.Rule.GetTimeWindow()
	if err != nil {
		return nil, fmt.Errorf("failed to get time window: %w", err)
	}

	filter, err := g.Rule.GetQueryFilter()
	if err != nil {
		return nil, fmt.Errorf("failed to get query filter: %w", err)
	}

	countField, err := g.Rule.GetValueCountField()
	if err != nil {
		return nil, fmt.Errorf("failed to get value count field: %w", err)
	}

	groupByFields, err := g.Rule.GetGroupByFields()
	if err != nil {
		return nil, fmt.Errorf("failed to get group_by fields: %w", err)
	}

	// Calculate number of unique values to generate
	uniqueValueCount := g.calculateEventCount(threshold, operator)

	log.Printf("Generating events for rule '%s' (%s):", g.Rule.Name, g.Rule.Model.CorrelationType)
	log.Printf("  Rule threshold: %.0f unique values in %s", threshold, timeWindow)
	log.Printf("  Generating: %d unique values (%.1fx multiplier)", uniqueValueCount, g.Multiplier)

	// Generate events with varying values
	events := make([]HECEvent, uniqueValueCount)
	now := time.Now()

	// Generate consistent values for group_by fields
	groupByValues := g.generateGroupByValues(groupByFields)

	for i := 0; i < uniqueValueCount; i++ {
		// Calculate event time with jitter
		eventTime := g.calculateEventTime(now, timeWindow, i, uniqueValueCount)

		// Create base event matching the filter
		event := g.createEventMatchingFilter(filter)

		// Apply group_by values to ensure events are correlated
		g.applyGroupByValues(event, groupByValues)

		// Generate unique value for the count field
		uniqueValue := g.generateUniqueValueForField(countField, i)
		g.setFieldValue(event, countField, uniqueValue)

		// Debug: print first event
		if i == 0 {
			log.Printf("  DEBUG: First event structure:")
			log.Printf("    class_uid: %v", event["class_uid"])
			log.Printf("    src_endpoint: %v", event["src_endpoint"])
			log.Printf("    dst_endpoint: %v", event["dst_endpoint"])
			log.Printf("    Field '%s' = %v", countField, uniqueValue)
		}

		// Create HEC event
		events[i] = HECEvent{
			Time:       float64(eventTime.Unix()) + float64(eventTime.Nanosecond())/1e9,
			Event:      event,
			SourceType: g.determineSourceType(event),
		}
	}

	log.Printf("  ✓ Generated %d events with unique %s values", len(events), countField)

	return events, nil
}

// calculateEventCount determines how many events to generate based on threshold and operator
func (g *RuleBasedGenerator) calculateEventCount(threshold float64, operator string) int {
	var baseCount float64

	switch operator {
	case "gt":
		// Greater than: threshold + 1, then multiply
		baseCount = (threshold + 1) * g.Multiplier
	case "gte":
		// Greater than or equal: threshold, then multiply
		baseCount = threshold * g.Multiplier
	case "eq":
		// Equal: exactly threshold
		baseCount = threshold
	default:
		// For other operators, use threshold * multiplier
		baseCount = threshold * g.Multiplier
	}

	return int(math.Ceil(baseCount))
}

// calculateEventTime calculates the timestamp for an event with jitter
func (g *RuleBasedGenerator) calculateEventTime(now time.Time, timeWindow time.Duration, index, total int) time.Time {
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
func (g *RuleBasedGenerator) createEventMatchingFilter(filter *QueryFilter) map[string]interface{} {
	event := make(map[string]interface{})

	// Apply all filter conditions to the event
	g.applyFilterToEvent(event, filter)

	// Add required OCSF fields based on class_uid
	g.addRequiredOCSFFields(event)

	return event
}

// applyFilterToEvent recursively applies filter conditions to an event
func (g *RuleBasedGenerator) applyFilterToEvent(event map[string]interface{}, filter *QueryFilter) {
	if filter.Type != "" {
		// Compound filter - apply all conditions
		for _, cond := range filter.Conditions {
			g.applyFilterToEvent(event, &cond)
		}
	} else {
		// Simple filter - set the field value
		g.setFieldValue(event, filter.Field, filter.Value)
	}
}

// setFieldValue sets a nested field value using dot notation
func (g *RuleBasedGenerator) setFieldValue(event map[string]interface{}, fieldPath string, value interface{}) {
	// Remove leading dot if present
	fieldPath = fmt.Sprintf("%v", fieldPath)
	fieldPath = fieldPath[1:] // Remove leading dot

	parts := splitFieldPath(fieldPath)
	current := event

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - set the value
			current[part] = value
		} else {
			// Intermediate part - ensure nested map exists
			if _, exists := current[part]; !exists {
				current[part] = make(map[string]interface{})
			}
			current = current[part].(map[string]interface{})
		}
	}
}

// splitFieldPath splits a field path like ".actor.user.name" into parts
func splitFieldPath(path string) []string {
	// Remove leading dot
	path = fmt.Sprintf("%v", path)
	if len(path) > 0 && path[0] == '.' {
		path = path[1:]
	}

	// Use strings.Split for simplicity and correctness
	return strings.Split(path, ".")
}

// generateGroupByValues creates consistent values for group_by fields
func (g *RuleBasedGenerator) generateGroupByValues(groupByFields []string) map[string]interface{} {
	values := make(map[string]interface{})

	for _, field := range groupByFields {
		// Check if override exists in params
		paramKey := field[1:] // Remove leading dot
		if val, exists := g.Params[paramKey]; exists {
			values[field] = val
			continue
		}

		// Generate appropriate value based on field name
		values[field] = g.generateValueForField(field)
	}

	return values
}

// applyGroupByValues applies group_by values to an event
func (g *RuleBasedGenerator) applyGroupByValues(event map[string]interface{}, groupByValues map[string]interface{}) {
	for field, value := range groupByValues {
		g.setFieldValue(event, field, value)
	}
}

// generateValueForField generates an appropriate value for a given field
func (g *RuleBasedGenerator) generateValueForField(field string) interface{} {
	// Check for specific field patterns in order of specificity
	fieldLower := ""
	for _, c := range field {
		if c >= 'A' && c <= 'Z' {
			fieldLower += string(c + 32)
		} else {
			fieldLower += string(c)
		}
	}

	// IP addresses
	if containsIgnoreCase(field, ".ip") || containsIgnoreCase(field, "_ip") {
		return gofakeit.IPv4Address()
	}
	// User names
	if containsIgnoreCase(field, "user.name") || containsIgnoreCase(field, "username") {
		return gofakeit.Username()
	}
	// Ports
	if containsIgnoreCase(field, ".port") || containsIgnoreCase(field, "_port") {
		// OCSF defines port as string (can be numeric like "443" or named like "https")
		return fmt.Sprintf("%d", rand.Intn(65535-1024)+1024)
	}
	// Hostnames
	if containsIgnoreCase(field, "hostname") || containsIgnoreCase(field, ".host") {
		return gofakeit.DomainName()
	}
	// Email
	if containsIgnoreCase(field, "email") {
		return gofakeit.Email()
	}

	// Default: random string
	return gofakeit.Word()
}

// generateUniqueValueForField generates a unique value for a field (for value_count)
func (g *RuleBasedGenerator) generateUniqueValueForField(field string, index int) interface{} {
	// Common field patterns
	if containsIgnoreCase(field, "port") {
		// Generate sequential ports starting from 1024
		// OCSF defines port as string (can be numeric like "443" or named like "https")
		return fmt.Sprintf("%d", 1024+index)
	}
	if containsIgnoreCase(field, "ip") || containsIgnoreCase(field, "addr") {
		// Generate IPs in a range
		return fmt.Sprintf("10.0.%d.%d", index/256, index%256)
	}
	if containsIgnoreCase(field, "user.name") || containsIgnoreCase(field, "username") {
		return fmt.Sprintf("user%d", index)
	}
	if containsIgnoreCase(field, "hostname") || containsIgnoreCase(field, "host") {
		return fmt.Sprintf("host-%d.example.com", index)
	}

	// Default: indexed value
	return fmt.Sprintf("value-%d", index)
}

// determineSourceType determines the appropriate OCSF source type based on event content
func (g *RuleBasedGenerator) determineSourceType(event map[string]interface{}) string {
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

// addRequiredOCSFFields adds mandatory OCSF fields based on the event's class_uid
func (g *RuleBasedGenerator) addRequiredOCSFFields(event map[string]interface{}) {
	classUIDRaw, ok := event["class_uid"]
	if !ok {
		return
	}

	// Convert to int for comparison
	var classUID int
	switch v := classUIDRaw.(type) {
	case int:
		classUID = v
	case float64:
		classUID = int(v)
	default:
		return
	}

	// Add category_uid based on class_uid
	switch classUID {
	case 3002: // Authentication
		event["category_uid"] = 3
		event["category_name"] = "Identity & Access Management"
		event["class_name"] = "Authentication"
		if _, exists := event["activity_id"]; !exists {
			event["activity_id"] = 1 // Logon
		}
		if _, exists := event["activity_name"]; !exists {
			event["activity_name"] = "Logon"
		}
		if _, exists := event["severity_id"]; !exists {
			if statusID, ok := event["status_id"].(int); ok && statusID == 2 {
				event["severity_id"] = 3 // Medium for failures
			} else {
				event["severity_id"] = 1 // Informational
			}
		}
		if _, exists := event["status"]; !exists {
			if statusID, ok := event["status_id"].(int); ok && statusID == 2 {
				event["status"] = "Failure"
			} else {
				event["status"] = "Success"
			}
		}
	case 4001: // Network Activity
		event["category_uid"] = 4
		event["category_name"] = "Network Activity"
		event["class_name"] = "Network Activity"
		if _, exists := event["activity_id"]; !exists {
			event["activity_id"] = 5 // Traffic
		}
		if _, exists := event["activity_name"]; !exists {
			event["activity_name"] = "Traffic"
		}
		if _, exists := event["severity_id"]; !exists {
			event["severity_id"] = 1 // Informational
		}
		// Add connection_info if not exists (required for network events)
		if _, exists := event["connection_info"]; !exists {
			event["connection_info"] = map[string]interface{}{
				"protocol_name": "TCP",
				"direction":     "outbound",
			}
		}
	case 1007: // Process Activity
		event["category_uid"] = 1
		event["category_name"] = "System Activity"
		event["class_name"] = "Process Activity"
		if _, exists := event["activity_id"]; !exists {
			event["activity_id"] = 1 // Launch
		}
		if _, exists := event["activity_name"]; !exists {
			event["activity_name"] = "Launch"
		}
		if _, exists := event["severity_id"]; !exists {
			event["severity_id"] = 1
		}
	default:
		// Generic defaults
		event["category_uid"] = 0
		event["category_name"] = "Unknown"
		if _, exists := event["severity_id"]; !exists {
			event["severity_id"] = 1
		}
	}

	// Add metadata
	if _, exists := event["metadata"]; !exists {
		event["metadata"] = map[string]interface{}{
			"product": map[string]interface{}{
				"vendor_name": "TelHawk",
				"name":        "Event Seeder",
				"version":     "2.0.0-rule-based",
			},
		}
	}

	// Add time field if missing (use current time)
	if _, exists := event["time"]; !exists {
		event["time"] = time.Now().Unix()
	}
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
