package seeder

import (
	"math/rand"
	"time"
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

// applyFilterToEvent recursively applies filter conditions to an event
func (eg *eventGenerator) applyFilterToEvent(event map[string]interface{}, filter *QueryFilter) {
	if filter.Type != "" {
		// Compound filter - apply all conditions
		for _, cond := range filter.Conditions {
			eg.applyFilterToEvent(event, &cond)
		}
	} else {
		// Simple filter - set the field value
		setFieldValue(event, filter.Field, filter.Value)
	}
}

// applyGroupByValues applies group_by values to an event
func (eg *eventGenerator) applyGroupByValues(event map[string]interface{}, groupByValues map[string]interface{}) {
	for field, value := range groupByValues {
		setFieldValue(event, field, value)
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
