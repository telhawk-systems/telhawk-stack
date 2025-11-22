package fields

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateRule_SimpleFilter(t *testing.T) {
	// Simple rule with valid fields
	model := map[string]interface{}{
		"correlation_type": "event_count",
		"parameters": map[string]interface{}{
			"time_window": "5m",
			"query": map[string]interface{}{
				"filter": map[string]interface{}{
					"field":    ".class_uid",
					"operator": "eq",
					"value":    3002,
				},
			},
			"group_by": []interface{}{".actor.user.name", ".src_endpoint.ip"},
		},
	}

	err := ValidateRule(model, nil, nil)
	if err != nil {
		t.Errorf("ValidateRule() error = %v, want nil", err)
	}
}

func TestValidateRule_CompoundFilter(t *testing.T) {
	// Rule with compound AND filter
	model := map[string]interface{}{
		"correlation_type": "event_count",
		"parameters": map[string]interface{}{
			"query": map[string]interface{}{
				"filter": map[string]interface{}{
					"type": "and",
					"conditions": []interface{}{
						map[string]interface{}{
							"field":    ".class_uid",
							"operator": "eq",
							"value":    3002,
						},
						map[string]interface{}{
							"field":    ".status_id",
							"operator": "eq",
							"value":    2,
						},
					},
				},
			},
		},
	}

	err := ValidateRule(model, nil, nil)
	if err != nil {
		t.Errorf("ValidateRule() error = %v, want nil", err)
	}
}

func TestValidateRule_InvalidFieldInFilter(t *testing.T) {
	model := map[string]interface{}{
		"correlation_type": "event_count",
		"parameters": map[string]interface{}{
			"query": map[string]interface{}{
				"filter": map[string]interface{}{
					"field":    ".invalid_field",
					"operator": "eq",
					"value":    123,
				},
			},
		},
	}

	err := ValidateRule(model, nil, nil)
	if err == nil {
		t.Error("ValidateRule() error = nil, want validation error")
		return
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("ValidateRule() error type = %T, want *ValidationError", err)
		return
	}

	if len(validationErr.InvalidFields) != 1 {
		t.Errorf("ValidateRule() invalid fields count = %d, want 1", len(validationErr.InvalidFields))
	}

	if validationErr.InvalidFields[0].Path != ".invalid_field" {
		t.Errorf("ValidateRule() invalid field path = %q, want %q", validationErr.InvalidFields[0].Path, ".invalid_field")
	}
}

func TestValidateRule_InvalidFieldInGroupBy(t *testing.T) {
	model := map[string]interface{}{
		"correlation_type": "event_count",
		"parameters": map[string]interface{}{
			"group_by": []interface{}{".invalid_group_field"},
		},
	}

	err := ValidateRule(model, nil, nil)
	if err == nil {
		t.Error("ValidateRule() error = nil, want validation error")
		return
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("ValidateRule() error type = %T, want *ValidationError", err)
		return
	}

	if len(validationErr.InvalidFields) != 1 {
		t.Errorf("ValidateRule() invalid fields count = %d, want 1", len(validationErr.InvalidFields))
	}

	if !strings.Contains(validationErr.InvalidFields[0].Location, "group_by") {
		t.Errorf("ValidateRule() invalid field location = %q, want contains 'group_by'", validationErr.InvalidFields[0].Location)
	}
}

func TestValidateRule_InvalidFieldInValueCount(t *testing.T) {
	model := map[string]interface{}{
		"correlation_type": "value_count",
		"parameters": map[string]interface{}{
			"field": ".invalid_value_field",
		},
	}

	err := ValidateRule(model, nil, nil)
	if err == nil {
		t.Error("ValidateRule() error = nil, want validation error")
		return
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("ValidateRule() error type = %T, want *ValidationError", err)
		return
	}

	if validationErr.InvalidFields[0].Location != "model.parameters.field" {
		t.Errorf("ValidateRule() invalid field location = %q, want %q", validationErr.InvalidFields[0].Location, "model.parameters.field")
	}
}

func TestValidateRule_TemporalOrderedSequence(t *testing.T) {
	// Rule with temporal_ordered sequence
	model := map[string]interface{}{
		"correlation_type": "temporal_ordered",
		"parameters": map[string]interface{}{
			"time_window": "15m",
			"group_by":    []interface{}{".actor.user.name"},
			"sequence": []interface{}{
				map[string]interface{}{
					"step": 1,
					"name": "failed_login",
					"query": map[string]interface{}{
						"filter": map[string]interface{}{
							"type": "and",
							"conditions": []interface{}{
								map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    3002,
								},
							},
						},
					},
				},
				map[string]interface{}{
					"step": 2,
					"name": "escalation",
					"query": map[string]interface{}{
						"filter": map[string]interface{}{
							"field":    ".class_uid",
							"operator": "eq",
							"value":    3001,
						},
					},
				},
			},
		},
	}

	err := ValidateRule(model, nil, nil)
	if err != nil {
		t.Errorf("ValidateRule() error = %v, want nil", err)
	}
}

func TestValidateRule_InvalidFieldInSequence(t *testing.T) {
	model := map[string]interface{}{
		"correlation_type": "temporal_ordered",
		"parameters": map[string]interface{}{
			"sequence": []interface{}{
				map[string]interface{}{
					"step": 1,
					"query": map[string]interface{}{
						"filter": map[string]interface{}{
							"field":    ".invalid_sequence_field",
							"operator": "eq",
							"value":    123,
						},
					},
				},
			},
		},
	}

	err := ValidateRule(model, nil, nil)
	if err == nil {
		t.Error("ValidateRule() error = nil, want validation error")
		return
	}

	validationErr := err.(*ValidationError)
	if !strings.Contains(validationErr.InvalidFields[0].Location, "sequence") {
		t.Errorf("ValidateRule() invalid field location = %q, want contains 'sequence'", validationErr.InvalidFields[0].Location)
	}
}

func TestValidateRule_FieldsOrder(t *testing.T) {
	view := map[string]interface{}{
		"title":    "Test Rule",
		"severity": "high",
		"fields_order": []interface{}{
			"device.hostname",
			"process.name",
			"actor.user.name",
			"event_count", // Aggregation field - should be allowed
		},
	}

	err := ValidateRule(nil, view, nil)
	if err != nil {
		t.Errorf("ValidateRule() error = %v, want nil", err)
	}
}

func TestValidateRule_InvalidFieldsOrder(t *testing.T) {
	view := map[string]interface{}{
		"title":    "Test Rule",
		"severity": "high",
		"fields_order": []interface{}{
			"device.hostname",
			"invalid.display.field",
		},
	}

	err := ValidateRule(nil, view, nil)
	if err == nil {
		t.Error("ValidateRule() error = nil, want validation error")
		return
	}

	validationErr := err.(*ValidationError)
	if validationErr.InvalidFields[0].Location != "view.fields_order" {
		t.Errorf("ValidateRule() invalid field location = %q, want %q", validationErr.InvalidFields[0].Location, "view.fields_order")
	}
}

func TestValidateRule_MultipleErrors(t *testing.T) {
	model := map[string]interface{}{
		"correlation_type": "event_count",
		"parameters": map[string]interface{}{
			"field": ".invalid_field_1",
			"query": map[string]interface{}{
				"filter": map[string]interface{}{
					"field": ".invalid_field_2",
				},
			},
			"group_by": []interface{}{".invalid_field_3"},
		},
	}

	view := map[string]interface{}{
		"fields_order": []interface{}{"invalid_field_4"},
	}

	err := ValidateRule(model, view, nil)
	if err == nil {
		t.Error("ValidateRule() error = nil, want validation error")
		return
	}

	validationErr := err.(*ValidationError)
	if len(validationErr.InvalidFields) != 4 {
		t.Errorf("ValidateRule() invalid fields count = %d, want 4", len(validationErr.InvalidFields))
		for _, f := range validationErr.InvalidFields {
			t.Logf("Invalid field: %q at %q", f.Path, f.Location)
		}
	}
}

func TestValidateRule_RealWorldFailedLogins(t *testing.T) {
	// Real detection rule from alerting/rules/failed_logins.json
	ruleJSON := `{
		"name": "failed_logins",
		"model": {
			"correlation_type": "event_count",
			"parameters": {
				"time_window": "5m",
				"query": {
					"filter": {
						"type": "and",
						"conditions": [
							{"field": ".class_uid", "operator": "eq", "value": 3002},
							{"field": ".status_id", "operator": "eq", "value": 2}
						]
					}
				},
				"threshold": {"value": 5, "operator": "gte"},
				"group_by": [".actor.user.name", ".src_endpoint.ip"]
			}
		},
		"view": {
			"title": "Multiple Failed Login Attempts",
			"severity": "medium"
		},
		"controller": {
			"detection": {"suppression_window": "10m"}
		}
	}`

	var rule map[string]interface{}
	if err := json.Unmarshal([]byte(ruleJSON), &rule); err != nil {
		t.Fatalf("Failed to parse rule JSON: %v", err)
	}

	model, _ := rule["model"].(map[string]interface{})
	view, _ := rule["view"].(map[string]interface{})
	controller, _ := rule["controller"].(map[string]interface{})

	err := ValidateRule(model, view, controller)
	if err != nil {
		t.Errorf("ValidateRule() error = %v, want nil for real-world rule", err)
	}
}

func TestValidateRule_RealWorldPortScanning(t *testing.T) {
	// Real detection rule from alerting/rules/port_scanning.json
	ruleJSON := `{
		"name": "port_scanning",
		"model": {
			"correlation_type": "value_count",
			"parameters": {
				"time_window": "10m",
				"query": {
					"filter": {
						"field": ".class_uid",
						"operator": "eq",
						"value": 4001
					}
				},
				"field": ".dst_endpoint.port",
				"threshold": {"value": 20, "operator": "gt"},
				"group_by": [".src_endpoint.ip"]
			}
		},
		"view": {
			"title": "Port Scanning Detected",
			"severity": "high"
		},
		"controller": {}
	}`

	var rule map[string]interface{}
	if err := json.Unmarshal([]byte(ruleJSON), &rule); err != nil {
		t.Fatalf("Failed to parse rule JSON: %v", err)
	}

	model, _ := rule["model"].(map[string]interface{})
	view, _ := rule["view"].(map[string]interface{})
	controller, _ := rule["controller"].(map[string]interface{})

	err := ValidateRule(model, view, controller)
	if err != nil {
		t.Errorf("ValidateRule() error = %v, want nil for real-world rule", err)
	}
}

func TestExtractFields(t *testing.T) {
	model := map[string]interface{}{
		"parameters": map[string]interface{}{
			"field": ".dst_endpoint.port",
			"query": map[string]interface{}{
				"filter": map[string]interface{}{
					"type": "and",
					"conditions": []interface{}{
						map[string]interface{}{"field": ".class_uid"},
						map[string]interface{}{"field": ".status_id"},
					},
				},
			},
			"group_by": []interface{}{".actor.user.name"},
		},
	}

	view := map[string]interface{}{
		"fields_order": []interface{}{"device.hostname", "event_count"},
	}

	fields := ExtractFields(model, view, nil)

	// Should extract: .dst_endpoint.port, .class_uid, .status_id, .actor.user.name, .device.hostname
	// (event_count is an aggregation field and should be excluded from view extraction)
	if len(fields) != 5 {
		t.Errorf("ExtractFields() returned %d fields, want 5", len(fields))
		t.Logf("Fields: %v", fields)
	}

	expectedFields := map[string]bool{
		".dst_endpoint.port": true,
		".class_uid":         true,
		".status_id":         true,
		".actor.user.name":   true,
		".device.hostname":   true,
	}

	for _, f := range fields {
		if !expectedFields[f] {
			t.Errorf("ExtractFields() unexpected field %q", f)
		}
	}
}

func TestValidationError_Error(t *testing.T) {
	ve := &ValidationError{
		InvalidFields: []InvalidField{
			{Path: ".invalid1", Location: "model.query.filter"},
			{Path: ".invalid2", Location: "model.group_by"},
		},
	}

	errStr := ve.Error()
	if !strings.Contains(errStr, ".invalid1") {
		t.Errorf("ValidationError.Error() = %q, want to contain '.invalid1'", errStr)
	}
	if !strings.Contains(errStr, ".invalid2") {
		t.Errorf("ValidationError.Error() = %q, want to contain '.invalid2'", errStr)
	}
	if !strings.Contains(errStr, "model.query.filter") {
		t.Errorf("ValidationError.Error() = %q, want to contain 'model.query.filter'", errStr)
	}
}

func TestValidationError_HasErrors(t *testing.T) {
	ve := &ValidationError{}
	if ve.HasErrors() {
		t.Error("ValidationError.HasErrors() = true, want false for empty")
	}

	ve.InvalidFields = []InvalidField{{Path: ".foo", Location: "bar"}}
	if !ve.HasErrors() {
		t.Error("ValidationError.HasErrors() = false, want true for non-empty")
	}
}
