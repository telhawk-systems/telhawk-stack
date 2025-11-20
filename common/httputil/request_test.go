package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func() *http.Request
		expectedIP   string
		description  string
	}{
		{
			name: "X-Forwarded-For with single IP",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.195")
				return req
			},
			expectedIP:  "203.0.113.195",
			description: "Should return the single IP from X-Forwarded-For",
		},
		{
			name: "X-Forwarded-For with multiple IPs",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
				return req
			},
			expectedIP:  "203.0.113.195",
			description: "Should return first (client) IP from comma-separated list",
		},
		{
			name: "X-Forwarded-For with spaces",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "  203.0.113.195  , 70.41.3.18")
				return req
			},
			expectedIP:  "203.0.113.195",
			description: "Should trim whitespace from first IP",
		},
		{
			name: "X-Real-IP when no X-Forwarded-For",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Real-IP", "198.51.100.42")
				return req
			},
			expectedIP:  "198.51.100.42",
			description: "Should return X-Real-IP when X-Forwarded-For is absent",
		},
		{
			name: "RemoteAddr when no proxy headers",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "192.0.2.1:54321"
				return req
			},
			expectedIP:  "192.0.2.1:54321",
			description: "Should return RemoteAddr when no proxy headers present",
		},
		{
			name: "X-Forwarded-For takes precedence over X-Real-IP",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.195")
				req.Header.Set("X-Real-IP", "198.51.100.42")
				req.RemoteAddr = "192.0.2.1:54321"
				return req
			},
			expectedIP:  "203.0.113.195",
			description: "X-Forwarded-For should take precedence",
		},
		{
			name: "Empty X-Forwarded-For falls back to X-Real-IP",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "")
				req.Header.Set("X-Real-IP", "198.51.100.42")
				return req
			},
			expectedIP:  "198.51.100.42",
			description: "Should fall back to X-Real-IP if X-Forwarded-For is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			got := GetClientIP(req)
			if got != tt.expectedIP {
				t.Errorf("GetClientIP() = %v, want %v\nDescription: %s", got, tt.expectedIP, tt.description)
			}
		})
	}
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		defaultVal  int
		expected    int
		description string
	}{
		{
			name:        "Valid positive integer",
			input:       "42",
			defaultVal:  10,
			expected:    42,
			description: "Should parse valid positive integer",
		},
		{
			name:        "Valid zero",
			input:       "0",
			defaultVal:  10,
			expected:    0,
			description: "Should parse zero correctly",
		},
		{
			name:        "Valid negative integer",
			input:       "-5",
			defaultVal:  10,
			expected:    -5,
			description: "Should parse negative integer",
		},
		{
			name:        "Empty string uses default",
			input:       "",
			defaultVal:  25,
			expected:    25,
			description: "Should return default for empty string",
		},
		{
			name:        "Invalid string uses default",
			input:       "abc",
			defaultVal:  30,
			expected:    30,
			description: "Should return default for non-numeric string",
		},
		{
			name:        "Invalid mixed string uses default",
			input:       "12abc",
			defaultVal:  35,
			expected:    35,
			description: "Should return default for mixed string",
		},
		{
			name:        "Whitespace uses default",
			input:       "  ",
			defaultVal:  40,
			expected:    40,
			description: "Should return default for whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseIntParam(tt.input, tt.defaultVal)
			if got != tt.expected {
				t.Errorf("ParseIntParam(%q, %d) = %d, want %d\nDescription: %s",
					tt.input, tt.defaultVal, got, tt.expected, tt.description)
			}
		})
	}
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func() *http.Request
		defaultLimit int
		maxLimit     int
		expected     Pagination
		description  string
	}{
		{
			name: "Default values when no parameters",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/items", nil)
			},
			defaultLimit: 50,
			maxLimit:     1000,
			expected:     Pagination{Page: 1, Limit: 50},
			description:  "Should use defaults when no query parameters",
		},
		{
			name: "Valid page and limit",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/items?page=3&limit=25", nil)
			},
			defaultLimit: 50,
			maxLimit:     1000,
			expected:     Pagination{Page: 3, Limit: 25},
			description:  "Should parse valid page and limit",
		},
		{
			name: "Limit exceeds max - should cap",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/items?page=1&limit=5000", nil)
			},
			defaultLimit: 50,
			maxLimit:     1000,
			expected:     Pagination{Page: 1, Limit: 1000},
			description:  "Should cap limit at maxLimit",
		},
		{
			name: "Page less than 1 - should default to 1",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/items?page=0&limit=25", nil)
			},
			defaultLimit: 50,
			maxLimit:     1000,
			expected:     Pagination{Page: 1, Limit: 25},
			description:  "Should set page to 1 if less than 1",
		},
		{
			name: "Negative page - should default to 1",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/items?page=-5&limit=25", nil)
			},
			defaultLimit: 50,
			maxLimit:     1000,
			expected:     Pagination{Page: 1, Limit: 25},
			description:  "Should set page to 1 if negative",
		},
		{
			name: "Invalid page uses default",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/items?page=abc&limit=25", nil)
			},
			defaultLimit: 50,
			maxLimit:     1000,
			expected:     Pagination{Page: 1, Limit: 25},
			description:  "Should use default page for invalid value",
		},
		{
			name: "Invalid limit uses default",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/items?page=2&limit=xyz", nil)
			},
			defaultLimit: 50,
			maxLimit:     1000,
			expected:     Pagination{Page: 2, Limit: 50},
			description:  "Should use default limit for invalid value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			got := ParsePagination(req, tt.defaultLimit, tt.maxLimit)
			if got.Page != tt.expected.Page || got.Limit != tt.expected.Limit {
				t.Errorf("ParsePagination() = %+v, want %+v\nDescription: %s",
					got, tt.expected, tt.description)
			}
		})
	}
}

func TestPaginationOffset(t *testing.T) {
	tests := []struct {
		name        string
		pagination  Pagination
		expected    int
		description string
	}{
		{
			name:        "First page offset is 0",
			pagination:  Pagination{Page: 1, Limit: 50},
			expected:    0,
			description: "Page 1 should have offset 0",
		},
		{
			name:        "Second page offset equals limit",
			pagination:  Pagination{Page: 2, Limit: 50},
			expected:    50,
			description: "Page 2 should have offset equal to limit",
		},
		{
			name:        "Third page offset",
			pagination:  Pagination{Page: 3, Limit: 25},
			expected:    50,
			description: "Page 3 with limit 25 should have offset 50",
		},
		{
			name:        "Large page number",
			pagination:  Pagination{Page: 100, Limit: 20},
			expected:    1980,
			description: "Should correctly calculate large offsets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pagination.Offset()
			if got != tt.expected {
				t.Errorf("Pagination.Offset() = %d, want %d\nDescription: %s",
					got, tt.expected, tt.description)
			}
		})
	}
}

// Benchmark tests
func BenchmarkGetClientIP_NoProxyHeaders(b *testing.B) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:54321"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetClientIP(req)
	}
}

func BenchmarkGetClientIP_XForwardedFor(b *testing.B) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetClientIP(req)
	}
}

func BenchmarkParseIntParam(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseIntParam("42", 10)
	}
}

func BenchmarkParsePagination(b *testing.B) {
	req := httptest.NewRequest("GET", "/items?page=3&limit=25", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParsePagination(req, 50, 1000)
	}
}
