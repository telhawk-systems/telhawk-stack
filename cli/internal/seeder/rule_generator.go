package seeder

import (
	"fmt"
	"log"
	"math"
	"time"
)

// RuleBasedGenerator generates events that match detection rules
type RuleBasedGenerator struct {
	Rule       *DetectionRule
	Multiplier float64                // Exceed threshold by this factor (default: 1.5)
	Params     map[string]interface{} // Override parameters from YAML config

	// Internal components
	eventGen   *eventGenerator
	fieldGen   *fieldGenerator
	ocsfEnrich *ocsfEnricher
}

// NewRuleBasedGenerator creates a new rule-based event generator
func NewRuleBasedGenerator(rule *DetectionRule, multiplier float64, params map[string]interface{}) *RuleBasedGenerator {
	if multiplier <= 0 {
		multiplier = 1.5 // Default multiplier
	}

	fieldGen := newFieldGenerator(params)

	return &RuleBasedGenerator{
		Rule:       rule,
		Multiplier: multiplier,
		Params:     params,
		eventGen:   &eventGenerator{},
		fieldGen:   fieldGen,
		ocsfEnrich: newOCSFEnricher(fieldGen),
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
	groupByValues := g.fieldGen.generateGroupByValues(groupByFields)

	for i := 0; i < eventCount; i++ {
		// Calculate event time with jitter
		eventTime := g.eventGen.calculateEventTime(now, timeWindow, i, eventCount)

		// Create base event matching the filter
		event := g.eventGen.createEventMatchingFilter(filter)

		// Apply group_by values to ensure events are correlated
		g.eventGen.applyGroupByValues(event, groupByValues)

		// Add required OCSF fields
		g.ocsfEnrich.enrichEvent(event)

		// Create HEC event
		events[i] = HECEvent{
			Time:       float64(eventTime.Unix()) + float64(eventTime.Nanosecond())/1e9,
			Event:      event,
			SourceType: g.eventGen.determineSourceType(event),
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
	groupByValues := g.fieldGen.generateGroupByValues(groupByFields)

	for i := 0; i < uniqueValueCount; i++ {
		// Calculate event time with jitter
		eventTime := g.eventGen.calculateEventTime(now, timeWindow, i, uniqueValueCount)

		// Create base event matching the filter
		event := g.eventGen.createEventMatchingFilter(filter)

		// Apply group_by values to ensure events are correlated
		g.eventGen.applyGroupByValues(event, groupByValues)

		// Generate unique value for the count field
		uniqueValue := g.fieldGen.generateUniqueValueForField(countField, i)
		setFieldValue(event, countField, uniqueValue)

		// Enrich network events with required OCSF fields
		g.ocsfEnrich.enrichNetworkEvent(event, groupByValues, i)

		// Add required OCSF fields
		g.ocsfEnrich.enrichEvent(event)

		// Debug: print first event
		if i == 0 {
			log.Printf("  DEBUG: First event structure:")
			log.Printf("    class_uid: %v (type: %T)", event["class_uid"], event["class_uid"])
			log.Printf("    src_endpoint: %v", event["src_endpoint"])
			log.Printf("    dst_endpoint: %v", event["dst_endpoint"])
			log.Printf("    Field '%s' = %v", countField, uniqueValue)
		}

		// Create HEC event
		events[i] = HECEvent{
			Time:       float64(eventTime.Unix()) + float64(eventTime.Nanosecond())/1e9,
			Event:      event,
			SourceType: g.eventGen.determineSourceType(event),
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
