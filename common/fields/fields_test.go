package fields

import (
	"sort"
	"testing"
)

func TestIsValidField(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected bool
	}{
		// Explicit fields
		{name: "class_uid with dot", field: ".class_uid", expected: true},
		{name: "class_uid without dot", field: "class_uid", expected: true},
		{name: "status_id", field: ".status_id", expected: true},
		{name: "severity_id", field: ".severity_id", expected: true},
		{name: "time", field: ".time", expected: true},

		// Nested explicit fields
		{name: "user.name", field: ".user.name", expected: true},
		{name: "user.uid", field: ".user.uid", expected: true},
		{name: "src_endpoint.ip", field: ".src_endpoint.ip", expected: true},
		{name: "src_endpoint.port", field: ".src_endpoint.port", expected: true},
		{name: "dst_endpoint.hostname", field: ".dst_endpoint.hostname", expected: true},
		{name: "process.name", field: ".process.name", expected: true},
		{name: "process.cmd_line", field: ".process.cmd_line", expected: true},
		{name: "file.path", field: ".file.path", expected: true},

		// Deeply nested explicit fields
		{name: "metadata.product.name", field: ".metadata.product.name", expected: true},
		{name: "metadata.product.vendor_name", field: ".metadata.product.vendor_name", expected: true},
		{name: "connection_info.protocol_name", field: ".connection_info.protocol_name", expected: true},

		// Wildcard prefix fields (device is under dynamic mapping)
		{name: "device.hostname", field: ".device.hostname", expected: true},
		{name: "device.ip", field: ".device.ip", expected: true},
		{name: "device.os.name", field: ".device.os.name", expected: true},
		{name: "actor.user.name (wildcard)", field: ".actor.user.name", expected: true},
		{name: "actor.user.uid (wildcard)", field: ".actor.user.uid", expected: true},
		{name: "actor.session.uid (wildcard)", field: ".actor.session.uid", expected: true},
		{name: "metadata.custom_field (wildcard)", field: ".metadata.custom_field", expected: true},
		{name: "properties.custom (wildcard)", field: ".properties.custom", expected: true},

		// Invalid fields
		{name: "invalid root field", field: ".invalid_field", expected: false},
		{name: "invalid nested field", field: ".src_endpoint.invalid", expected: false},
		{name: "deeply invalid", field: ".foo.bar.baz", expected: false},
		{name: "typo in field name", field: ".class_uiid", expected: false},
		{name: "non-existent object", field: ".network.source_ip", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidField(tt.field)
			if result != tt.expected {
				t.Errorf("IsValidField(%q) = %v, want %v", tt.field, result, tt.expected)
			}
		})
	}
}

func TestValidateFields(t *testing.T) {
	tests := []struct {
		name            string
		fields          []string
		expectedInvalid []string
	}{
		{
			name:            "all valid fields",
			fields:          []string{".class_uid", ".status_id", ".src_endpoint.ip"},
			expectedInvalid: nil,
		},
		{
			name:            "all invalid fields",
			fields:          []string{".invalid1", ".invalid2"},
			expectedInvalid: []string{".invalid1", ".invalid2"},
		},
		{
			name:            "mixed valid and invalid",
			fields:          []string{".class_uid", ".invalid_field", ".src_endpoint.ip", ".bad.field"},
			expectedInvalid: []string{".invalid_field", ".bad.field"},
		},
		{
			name:            "empty list",
			fields:          []string{},
			expectedInvalid: nil,
		},
		{
			name:            "wildcard allowed fields",
			fields:          []string{".actor.user.name", ".actor.user.custom_attr", ".device.os.version"},
			expectedInvalid: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateFields(tt.fields)
			if len(result) != len(tt.expectedInvalid) {
				t.Errorf("ValidateFields() returned %d invalid fields, want %d", len(result), len(tt.expectedInvalid))
				t.Errorf("Got: %v", result)
				t.Errorf("Want: %v", tt.expectedInvalid)
				return
			}

			// Sort both slices for comparison
			sort.Strings(result)
			sort.Strings(tt.expectedInvalid)

			for i, field := range result {
				if field != tt.expectedInvalid[i] {
					t.Errorf("ValidateFields() invalid field at index %d = %q, want %q", i, field, tt.expectedInvalid[i])
				}
			}
		})
	}
}

func TestGetFieldInfo(t *testing.T) {
	tests := []struct {
		name         string
		field        string
		expectNil    bool
		expectedType string
	}{
		{name: "class_uid", field: ".class_uid", expectNil: false, expectedType: "integer"},
		{name: "class_name", field: ".class_name", expectNil: false, expectedType: "keyword"},
		{name: "time", field: ".time", expectNil: false, expectedType: "date"},
		{name: "src_endpoint.ip", field: ".src_endpoint.ip", expectNil: false, expectedType: "ip"},
		{name: "message", field: ".message", expectNil: false, expectedType: "text"},
		{name: "invalid field", field: ".invalid", expectNil: true, expectedType: ""},
		{name: "wildcard field", field: ".actor.user.name", expectNil: true, expectedType: ""}, // Not explicitly defined
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := GetFieldInfo(tt.field)
			if tt.expectNil {
				if info != nil {
					t.Errorf("GetFieldInfo(%q) = %+v, want nil", tt.field, info)
				}
			} else {
				if info == nil {
					t.Errorf("GetFieldInfo(%q) = nil, want non-nil", tt.field)
				} else if info.Type != tt.expectedType {
					t.Errorf("GetFieldInfo(%q).Type = %q, want %q", tt.field, info.Type, tt.expectedType)
				}
			}
		})
	}
}

func TestListFields(t *testing.T) {
	fields := ListFields()

	// Should have at least the core fields
	if len(fields) < 30 {
		t.Errorf("ListFields() returned %d fields, expected at least 30", len(fields))
	}

	// Check that some known fields are in the list
	expectedFields := []string{".class_uid", ".status_id", ".time", ".src_endpoint.ip"}
	fieldSet := make(map[string]bool)
	for _, f := range fields {
		fieldSet[f] = true
	}

	for _, expected := range expectedFields {
		if !fieldSet[expected] {
			t.Errorf("ListFields() missing expected field %q", expected)
		}
	}
}
