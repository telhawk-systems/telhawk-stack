package correlation

import (
	"encoding/json"
	"fmt"
)

// Validator validates correlation parameters
type Validator struct{}

// NewValidator creates a new correlation validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateParameters validates correlation parameters for a given type
func (v *Validator) ValidateParameters(correlationType CorrelationType, params map[string]interface{}) error {
	if !correlationType.IsValid() {
		return fmt.Errorf("invalid correlation type: %s", correlationType)
	}

	// Convert map to JSON and back to specific type for validation
	switch correlationType {
	case TypeEventCount:
		return v.validateEventCount(params)
	case TypeValueCount:
		return v.validateValueCount(params)
	case TypeTemporal:
		return v.validateTemporal(params)
	case TypeTemporalOrdered:
		return v.validateTemporalOrdered(params)
	case TypeJoin:
		return v.validateJoin(params)
	case TypeSuppression:
		return v.validateSuppression(params)
	case TypeBaselineDeviation:
		return v.validateBaselineDeviation(params)
	case TypeMissingEvent:
		return v.validateMissingEvent(params)
	default:
		return fmt.Errorf("unsupported correlation type: %s", correlationType)
	}
}

// validateEventCount validates event_count parameters
func (v *Validator) validateEventCount(params map[string]interface{}) error {
	var p EventCountParams
	if err := unmarshalParams(params, &p); err != nil {
		return fmt.Errorf("failed to parse event_count parameters: %w", err)
	}
	return p.Validate()
}

// validateValueCount validates value_count parameters
func (v *Validator) validateValueCount(params map[string]interface{}) error {
	var p ValueCountParams
	if err := unmarshalParams(params, &p); err != nil {
		return fmt.Errorf("failed to parse value_count parameters: %w", err)
	}
	return p.Validate()
}

// validateTemporal validates temporal parameters
func (v *Validator) validateTemporal(params map[string]interface{}) error {
	var p TemporalParams
	if err := unmarshalParams(params, &p); err != nil {
		return fmt.Errorf("failed to parse temporal parameters: %w", err)
	}
	return p.Validate()
}

// validateTemporalOrdered validates temporal_ordered parameters
func (v *Validator) validateTemporalOrdered(params map[string]interface{}) error {
	var p TemporalOrderedParams
	if err := unmarshalParams(params, &p); err != nil {
		return fmt.Errorf("failed to parse temporal_ordered parameters: %w", err)
	}
	return p.Validate()
}

// validateJoin validates join parameters
func (v *Validator) validateJoin(params map[string]interface{}) error {
	var p JoinParams
	if err := unmarshalParams(params, &p); err != nil {
		return fmt.Errorf("failed to parse join parameters: %w", err)
	}
	return p.Validate()
}

// validateSuppression validates suppression parameters
func (v *Validator) validateSuppression(params map[string]interface{}) error {
	var p SuppressionParams
	if err := unmarshalParams(params, &p); err != nil {
		return fmt.Errorf("failed to parse suppression parameters: %w", err)
	}
	return p.Validate()
}

// validateBaselineDeviation validates baseline_deviation parameters
func (v *Validator) validateBaselineDeviation(params map[string]interface{}) error {
	var p BaselineDeviationParams
	if err := unmarshalParams(params, &p); err != nil {
		return fmt.Errorf("failed to parse baseline_deviation parameters: %w", err)
	}
	return p.Validate()
}

// validateMissingEvent validates missing_event parameters
func (v *Validator) validateMissingEvent(params map[string]interface{}) error {
	var p MissingEventParams
	if err := unmarshalParams(params, &p); err != nil {
		return fmt.Errorf("failed to parse missing_event parameters: %w", err)
	}
	return p.Validate()
}

// unmarshalParams converts a generic map to a specific parameter type
func unmarshalParams(params map[string]interface{}, target interface{}) error {
	// Convert map to JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Unmarshal JSON to target type
	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	return nil
}

// GetParameterSchema returns the JSON schema for a correlation type
func (v *Validator) GetParameterSchema(correlationType CorrelationType) (map[string]interface{}, error) {
	switch correlationType {
	case TypeEventCount:
		return map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"time_window": map[string]interface{}{"type": "string", "description": "Lookback window (e.g., '5m', '1h')"},
				"threshold":   map[string]interface{}{"type": "integer", "description": "Minimum event count to trigger"},
				"operator":    map[string]interface{}{"type": "string", "enum": []string{"gt", "gte", "lt", "lte", "eq", "ne"}, "default": "gt"},
				"group_by":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			},
			"required": []string{"time_window", "threshold"},
		}, nil
	case TypeSuppression:
		return map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"enabled":         map[string]interface{}{"type": "boolean", "default": true},
				"window":          map[string]interface{}{"type": "string", "description": "Suppression window duration (e.g., '1h', '24h')"},
				"key":             map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Fields to group alerts by"},
				"max_alerts":      map[string]interface{}{"type": "integer", "default": 1, "description": "Max alerts per window per key"},
				"reset_on_change": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Fields that reset suppression if changed"},
			},
			"required": []string{"window", "key"},
		}, nil
	// Add schemas for other types as needed
	default:
		return nil, fmt.Errorf("schema not available for correlation type: %s", correlationType)
	}
}
