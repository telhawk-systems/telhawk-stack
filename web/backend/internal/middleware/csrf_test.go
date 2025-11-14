package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRF_ExemptPaths(t *testing.T) {
	middleware := CSRF(false)

	exemptPaths := []string{
		"/api/auth/login",
		"/api/health",
	}

	for _, path := range exemptPaths {
		t.Run(path, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			})

			handler := middleware(next)

			// POST request to exempt path (would normally require CSRF)
			req := httptest.NewRequest("POST", path, strings.NewReader(`{"test":"data"}`))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if !nextCalled {
				t.Error("Expected next handler to be called for exempt path")
			}

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}

			if body := rr.Body.String(); body != "success" {
				t.Errorf("Expected body 'success', got '%s'", body)
			}
		})
	}
}

func TestCSRF_ExemptPrefixes(t *testing.T) {
	middleware := CSRF(false)

	tests := []struct {
		name string
		path string
	}{
		{
			name: "Query service prefix",
			path: "/api/query/v1/search",
		},
		{
			name: "Query service root",
			path: "/api/query/",
		},
		{
			name: "Core service prefix",
			path: "/api/core/v1/events",
		},
		{
			name: "Auth me endpoint",
			path: "/api/auth/me",
		},
		{
			name: "Auth API prefix",
			path: "/api/auth/api/v1/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware(next)

			req := httptest.NewRequest("POST", tt.path, strings.NewReader(`{"test":"data"}`))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if !nextCalled {
				t.Errorf("Expected next handler to be called for exempt prefix: %s", tt.path)
			}

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestCSRF_NonExemptPaths_SameOrigin(t *testing.T) {
	middleware := CSRF(false)

	tests := []struct {
		name   string
		path   string
		method string
	}{
		{
			name:   "Dashboard endpoint",
			path:   "/api/dashboard/metrics",
			method: "POST",
		},
		{
			name:   "Settings endpoint",
			path:   "/api/settings",
			method: "PUT",
		},
		{
			name:   "User profile",
			path:   "/api/users/123",
			method: "PATCH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware(next)

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{"test":"data"}`))
			req.Header.Set("Content-Type", "application/json")
			// Go 1.25 CrossOriginProtection checks Sec-Fetch-Site and Origin headers
			// Same-origin requests should pass
			req.Header.Set("Sec-Fetch-Site", "same-origin")
			req.Header.Set("Origin", "http://example.com")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if !nextCalled {
				t.Errorf("Expected next handler to be called for same-origin request to %s", tt.path)
			}

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestCSRF_NonExemptPaths_CrossOrigin(t *testing.T) {
	middleware := CSRF(false)

	tests := []struct {
		name   string
		path   string
		method string
	}{
		{
			name:   "Dashboard POST",
			path:   "/api/dashboard/update",
			method: "POST",
		},
		{
			name:   "Settings PUT",
			path:   "/api/settings",
			method: "PUT",
		},
		{
			name:   "Profile DELETE",
			path:   "/api/profile/avatar",
			method: "DELETE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware(next)

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{"test":"data"}`))
			req.Header.Set("Content-Type", "application/json")
			// Cross-origin request should be blocked by CSRF protection
			req.Header.Set("Sec-Fetch-Site", "cross-site")
			req.Header.Set("Origin", "http://evil.com")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if nextCalled {
				t.Errorf("Expected next handler NOT to be called for cross-origin request to %s", tt.path)
			}

			if rr.Code != http.StatusForbidden {
				t.Errorf("Expected status 403, got %d", rr.Code)
			}

			expectedBody := "CSRF token validation failed\n"
			if body := rr.Body.String(); body != expectedBody {
				t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
			}
		})
	}
}

func TestCSRF_GetRequestsAlwaysAllowed(t *testing.T) {
	middleware := CSRF(false)

	paths := []string{
		"/api/dashboard/metrics",
		"/api/users/123",
		"/api/settings",
		"/api/profile",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware(next)

			// GET requests should always be allowed (no CSRF protection needed)
			req := httptest.NewRequest("GET", path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if !nextCalled {
				t.Errorf("Expected next handler to be called for GET request to %s", path)
			}

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestCSRF_SecureAndInsecureModes(t *testing.T) {
	tests := []struct {
		name         string
		cookieSecure bool
	}{
		{
			name:         "Secure mode",
			cookieSecure: true,
		},
		{
			name:         "Insecure mode",
			cookieSecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := CSRF(tt.cookieSecure)

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware(next)

			// Test exempt path works in both modes
			req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{}`))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if !nextCalled {
				t.Error("Expected next handler to be called for exempt path")
			}

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestCSRF_PrefixMatchingEdgeCases(t *testing.T) {
	middleware := CSRF(false)

	tests := []struct {
		name        string
		path        string
		shouldBlock bool
	}{
		{
			name:        "Exact prefix match",
			path:        "/api/query/",
			shouldBlock: false,
		},
		{
			name:        "Prefix with path",
			path:        "/api/query/v1/search",
			shouldBlock: false,
		},
		{
			name:        "Similar but different prefix",
			path:        "/api/queries/",
			shouldBlock: true,
		},
		{
			name:        "Substring but not prefix",
			path:        "/other/api/query/",
			shouldBlock: true,
		},
		{
			name:        "Auth me exact",
			path:        "/api/auth/me",
			shouldBlock: false,
		},
		{
			name:        "Auth me with trailing",
			path:        "/api/auth/me/profile",
			shouldBlock: false,
		},
		{
			name:        "Auth but not me",
			path:        "/api/auth/other",
			shouldBlock: true,
		},
		{
			name:        "Auth api prefix",
			path:        "/api/auth/api/v1/users",
			shouldBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware(next)

			req := httptest.NewRequest("POST", tt.path, strings.NewReader(`{}`))
			// Set cross-origin to trigger CSRF check
			req.Header.Set("Sec-Fetch-Site", "cross-site")
			req.Header.Set("Origin", "http://evil.com")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if tt.shouldBlock {
				if nextCalled {
					t.Errorf("Expected request to be blocked for path: %s", tt.path)
				}
				if rr.Code != http.StatusForbidden {
					t.Errorf("Expected status 403, got %d for path: %s", rr.Code, tt.path)
				}
			} else {
				if !nextCalled {
					t.Errorf("Expected request to be allowed for path: %s", tt.path)
				}
				if rr.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d for path: %s", rr.Code, tt.path)
				}
			}
		})
	}
}

func TestCSRF_MiddlewareChaining(t *testing.T) {
	middleware := CSRF(false)

	// Create a chain of handlers to verify middleware properly chains
	finalCalled := false
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		finalCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("final handler"))
	})

	// Add another middleware in the chain
	middleCalled := false
	middleHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middleCalled = true
			w.Header().Set("X-Custom", "value")
			next.ServeHTTP(w, r)
		})
	}

	// Build chain: CSRF -> middle -> final
	handler := middleware(middleHandler(finalHandler))

	// Test with exempt path
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !middleCalled {
		t.Error("Expected middle handler to be called")
	}

	if !finalCalled {
		t.Error("Expected final handler to be called")
	}

	if rr.Header().Get("X-Custom") != "value" {
		t.Error("Expected custom header from middle handler")
	}

	if body := rr.Body.String(); body != "final handler" {
		t.Errorf("Expected body from final handler, got '%s'", body)
	}
}

func TestCSRF_DenyHandlerLogging(t *testing.T) {
	middleware := CSRF(false)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called")
	})

	handler := middleware(next)

	// Cross-origin POST to non-exempt path should trigger deny handler
	req := httptest.NewRequest("POST", "/api/settings", strings.NewReader(`{}`))
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Origin", "http://evil.com")
	req.RemoteAddr = "192.0.2.1:1234"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}

	expectedBody := "CSRF token validation failed\n"
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}
