package hec

import (
	"testing"
)

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		expected   string
	}{
		{
			name:       "Valid Splunk format",
			authHeader: "Splunk abc123def456",
			expected:   "abc123def456",
		},
		{
			name:       "Valid Telhawk format",
			authHeader: "Telhawk xyz789token",
			expected:   "xyz789token",
		},
		{
			name:       "Valid Bearer format",
			authHeader: "Bearer jwt-token-here",
			expected:   "jwt-token-here",
		},
		{
			name:       "Case insensitive - splunk lowercase",
			authHeader: "splunk token123",
			expected:   "token123",
		},
		{
			name:       "Case insensitive - SPLUNK uppercase",
			authHeader: "SPLUNK token456",
			expected:   "token456",
		},
		{
			name:       "Case insensitive - Bearer mixed case",
			authHeader: "bearer token789",
			expected:   "token789",
		},
		{
			name:       "Empty header",
			authHeader: "",
			expected:   "",
		},
		{
			name:       "Invalid format - no space",
			authHeader: "Splunkabc123",
			expected:   "",
		},
		{
			name:       "Invalid format - unsupported scheme",
			authHeader: "Basic abc123",
			expected:   "",
		},
		{
			name:       "Invalid format - only scheme",
			authHeader: "Splunk",
			expected:   "",
		},
		{
			name:       "Token with spaces",
			authHeader: "Splunk token with spaces",
			expected:   "token with spaces",
		},
		{
			name:       "Multiple spaces in header",
			authHeader: "Splunk   token123",
			expected:   "  token123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractToken(tt.authHeader)
			if result != tt.expected {
				t.Errorf("ExtractToken(%q) = %q, want %q", tt.authHeader, result, tt.expected)
			}
		})
	}
}

func TestValidateEventFormat(t *testing.T) {
	tests := []struct {
		name    string
		event   interface{}
		wantErr bool
	}{
		{
			name:    "Valid event - map",
			event:   map[string]interface{}{"event": "test"},
			wantErr: false,
		},
		{
			name:    "Valid event - string",
			event:   "simple event",
			wantErr: false,
		},
		{
			name:    "Valid event - struct",
			event:   struct{ Event string }{Event: "test"},
			wantErr: false,
		},
		{
			name:    "Invalid event - nil",
			event:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEventFormat(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEventFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != ErrInvalidEvent {
				t.Errorf("ValidateEventFormat() error = %v, want %v", err, ErrInvalidEvent)
			}
		})
	}
}

func TestHECError(t *testing.T) {
	tests := []struct {
		name     string
		hecError *HECError
		expected string
	}{
		{
			name:     "ErrInvalidEvent",
			hecError: ErrInvalidEvent,
			expected: "Invalid data format",
		},
		{
			name:     "ErrNoData",
			hecError: ErrNoData,
			expected: "No data",
		},
		{
			name:     "ErrUnauthorized",
			hecError: ErrUnauthorized,
			expected: "Invalid authorization",
		},
		{
			name:     "ErrServerBusy",
			hecError: ErrServerBusy,
			expected: "Server is busy",
		},
		{
			name:     "Custom error",
			hecError: &HECError{Code: 100, Text: "Custom error message"},
			expected: "Custom error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.hecError.Error()
			if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHECErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		hecError *HECError
		code     int
	}{
		{
			name:     "Invalid data format code",
			hecError: ErrInvalidEvent,
			code:     6,
		},
		{
			name:     "No data code",
			hecError: ErrNoData,
			code:     5,
		},
		{
			name:     "Unauthorized code",
			hecError: ErrUnauthorized,
			code:     4,
		},
		{
			name:     "Server busy code",
			hecError: ErrServerBusy,
			code:     9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hecError.Code != tt.code {
				t.Errorf("Code = %d, want %d", tt.hecError.Code, tt.code)
			}
		})
	}
}
