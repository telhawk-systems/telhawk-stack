package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestRequestID(t *testing.T) {
	tests := []struct {
		name              string
		existingRequestID string
		expectNewID       bool
	}{
		{
			name:              "generates new request ID when not present",
			existingRequestID: "",
			expectNewID:       true,
		},
		{
			name:              "propagates existing request ID",
			existingRequestID: "existing-req-123",
			expectNewID:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRequestID string
			var contextHasRequestID bool

			// Handler that captures the request ID from context
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequestID = GetRequestID(r.Context())
				contextHasRequestID = capturedRequestID != ""
				w.WriteHeader(http.StatusOK)
			})

			// Create request
			req := httptest.NewRequest("GET", "http://example.com/test", nil)
			if tt.existingRequestID != "" {
				req.Header.Set("X-Request-ID", tt.existingRequestID)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Apply middleware
			middleware := RequestID(handler)
			middleware.ServeHTTP(w, req)

			// Check response header
			responseRequestID := w.Header().Get("X-Request-ID")
			if responseRequestID == "" {
				t.Error("expected X-Request-ID header in response")
			}

			// Check context
			if !contextHasRequestID {
				t.Error("expected request ID in context")
			}

			// Verify ID handling
			if tt.expectNewID {
				// Should be a valid UUID
				if _, err := uuid.Parse(capturedRequestID); err != nil {
					t.Errorf("expected valid UUID, got %q: %v", capturedRequestID, err)
				}

				// Response header should match context
				if responseRequestID != capturedRequestID {
					t.Errorf("response header %q doesn't match context %q", responseRequestID, capturedRequestID)
				}
			} else {
				// Should use existing request ID
				if capturedRequestID != tt.existingRequestID {
					t.Errorf("expected request ID %q, got %q", tt.existingRequestID, capturedRequestID)
				}

				if responseRequestID != tt.existingRequestID {
					t.Errorf("expected response header %q, got %q", tt.existingRequestID, responseRequestID)
				}
			}
		})
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    bool
		requestID   string
		expectEmpty bool
	}{
		{
			name:        "returns request ID from context",
			setupCtx:    true,
			requestID:   "test-req-456",
			expectEmpty: false,
		},
		{
			name:        "returns empty string when not in context",
			setupCtx:    false,
			requestID:   "",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				result := GetRequestID(r.Context())

				if tt.expectEmpty {
					if result != "" {
						t.Errorf("expected empty string, got %q", result)
					}
				} else {
					if result != tt.requestID {
						t.Errorf("expected %q, got %q", tt.requestID, result)
					}
				}
			})

			req := httptest.NewRequest("GET", "http://example.com/test", nil)
			if tt.setupCtx {
				req.Header.Set("X-Request-ID", tt.requestID)
				middleware := RequestID(handler)
				middleware.ServeHTTP(httptest.NewRecorder(), req)
			} else {
				handler.ServeHTTP(httptest.NewRecorder(), req)
			}
		})
	}
}

func TestRequestID_UniqueIDs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestID(handler)

	// Make multiple requests and collect request IDs
	requestIDs := make(map[string]bool)
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		if requestID == "" {
			t.Fatal("expected request ID in response")
		}

		// Check for duplicates
		if requestIDs[requestID] {
			t.Errorf("duplicate request ID generated: %s", requestID)
		}
		requestIDs[requestID] = true

		// Verify it's a valid UUID
		if _, err := uuid.Parse(requestID); err != nil {
			t.Errorf("invalid UUID generated: %s", requestID)
		}
	}

	// Verify we generated 100 unique IDs
	if len(requestIDs) != 100 {
		t.Errorf("expected 100 unique IDs, got %d", len(requestIDs))
	}
}

func TestRequestID_Propagation(t *testing.T) {
	// Test that request ID is properly propagated through the context chain
	var level1ID, level2ID, level3ID string

	level3Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		level3ID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	level2Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		level2ID = GetRequestID(r.Context())
		level3Handler.ServeHTTP(w, r)
	})

	level1Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		level1ID = GetRequestID(r.Context())
		level2Handler.ServeHTTP(w, r)
	})

	middleware := RequestID(level1Handler)

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// All levels should see the same request ID
	if level1ID == "" || level2ID == "" || level3ID == "" {
		t.Fatal("expected request ID at all levels")
	}

	if level1ID != level2ID {
		t.Errorf("level1 ID %q != level2 ID %q", level1ID, level2ID)
	}

	if level2ID != level3ID {
		t.Errorf("level2 ID %q != level3 ID %q", level2ID, level3ID)
	}
}

func TestRequestIDKey_Type(t *testing.T) {
	// Verify RequestIDKey is of the correct type
	var key interface{} = RequestIDKey
	if _, ok := key.(contextKey); !ok {
		t.Errorf("RequestIDKey should be of type contextKey")
	}
}
