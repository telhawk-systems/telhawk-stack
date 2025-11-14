package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders_AllHeadersSet(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expectedHeaders := map[string]string{
		"X-Frame-Options":           "DENY",
		"X-Content-Type-Options":    "nosniff",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "geolocation=(), microphone=(), camera=()",
		"X-XSS-Protection":          "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := rr.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected %s header to be '%s', got '%s'", header, expectedValue, actualValue)
		}
	}
}

func TestSecurityHeaders_CSP_StrictPolicy(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	csp := rr.Header().Get("Content-Security-Policy")

	// Verify all CSP directives are present
	expectedDirectives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self'",
		"img-src 'self' data:",
		"font-src 'self'",
		"connect-src 'self'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}

	for _, directive := range expectedDirectives {
		if !strings.Contains(csp, directive) {
			t.Errorf("Expected CSP to contain '%s', got '%s'", directive, csp)
		}
	}

	// Verify no unsafe directives
	unsafeDirectives := []string{
		"unsafe-inline",
		"unsafe-eval",
		"*",
	}

	for _, unsafe := range unsafeDirectives {
		if strings.Contains(csp, unsafe) {
			t.Errorf("CSP should not contain unsafe directive '%s', got '%s'", unsafe, csp)
		}
	}
}

func TestSecurityHeaders_HSTS_SecureMode(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	hsts := rr.Header().Get("Strict-Transport-Security")
	expectedHSTS := "max-age=31536000; includeSubDomains; preload"

	if hsts != expectedHSTS {
		t.Errorf("Expected HSTS header '%s', got '%s'", expectedHSTS, hsts)
	}
}

func TestSecurityHeaders_NoHSTS_InsecureMode(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: false}
	middleware := SecurityHeaders(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	hsts := rr.Header().Get("Strict-Transport-Security")

	if hsts != "" {
		t.Errorf("Expected no HSTS header in insecure mode, got '%s'", hsts)
	}

	// Other security headers should still be set
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("Expected X-Frame-Options to still be set in insecure mode")
	}
}

func TestSecurityHeaders_XFrameOptions_DENY(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	xfo := rr.Header().Get("X-Frame-Options")
	if xfo != "DENY" {
		t.Errorf("Expected X-Frame-Options 'DENY', got '%s'", xfo)
	}
}

func TestSecurityHeaders_XContentTypeOptions_Nosniff(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	xcto := rr.Header().Get("X-Content-Type-Options")
	if xcto != "nosniff" {
		t.Errorf("Expected X-Content-Type-Options 'nosniff', got '%s'", xcto)
	}
}

func TestSecurityHeaders_ReferrerPolicy(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	rp := rr.Header().Get("Referrer-Policy")
	if rp != "strict-origin-when-cross-origin" {
		t.Errorf("Expected Referrer-Policy 'strict-origin-when-cross-origin', got '%s'", rp)
	}
}

func TestSecurityHeaders_PermissionsPolicy(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	pp := rr.Header().Get("Permissions-Policy")
	expectedPP := "geolocation=(), microphone=(), camera=()"

	if pp != expectedPP {
		t.Errorf("Expected Permissions-Policy '%s', got '%s'", expectedPP, pp)
	}

	// Verify specific permissions are disabled
	requiredPerms := []string{"geolocation=()", "microphone=()", "camera=()"}
	for _, perm := range requiredPerms {
		if !strings.Contains(pp, perm) {
			t.Errorf("Expected Permissions-Policy to contain '%s'", perm)
		}
	}
}

func TestSecurityHeaders_XXSSProtection(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	xss := rr.Header().Get("X-XSS-Protection")
	if xss != "1; mode=block" {
		t.Errorf("Expected X-XSS-Protection '1; mode=block', got '%s'", xss)
	}
}

func TestSecurityHeaders_DoesNotOverrideResponseStatus(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusNoContent,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
			})

			handler := middleware(next)

			req := httptest.NewRequest("GET", "/", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != statusCode {
				t.Errorf("Expected status %d, got %d", statusCode, rr.Code)
			}

			// Headers should still be set
			if rr.Header().Get("X-Frame-Options") != "DENY" {
				t.Error("Expected security headers to be set even with non-200 status")
			}
		})
	}
}

func TestSecurityHeaders_DoesNotOverrideResponseBody(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	expectedBody := `{"message": "test response", "data": [1, 2, 3]}`

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	})

	handler := middleware(next)

	req := httptest.NewRequest("GET", "/api/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Body.String() != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, rr.Body.String())
	}

	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type to be preserved")
	}

	// Security headers should be added
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("Expected security headers to be added")
	}
}

func TestSecurityHeaders_PreservesExistingHeaders(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "custom-value")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Existing headers should be preserved
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type to be preserved")
	}
	if rr.Header().Get("X-Custom-Header") != "custom-value" {
		t.Error("Expected X-Custom-Header to be preserved")
	}
	if rr.Header().Get("Cache-Control") != "no-cache" {
		t.Error("Expected Cache-Control to be preserved")
	}

	// Security headers should be added
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("Expected security headers to be added")
	}
}

func TestSecurityHeaders_WorksWithAllHTTPMethods(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware(next)

			req := httptest.NewRequest(method, "/", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Header().Get("X-Frame-Options") != "DENY" {
				t.Errorf("Expected security headers for %s method", method)
			}
		})
	}
}

func TestSecurityHeaders_MiddlewareChaining(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	securityMiddleware := SecurityHeaders(cfg)

	// Create a chain with multiple middlewares
	finalCalled := false
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		finalCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("final"))
	})

	otherMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Other-Middleware", "present")
			next.ServeHTTP(w, r)
		})
	}

	// Chain: security -> other -> final
	handler := securityMiddleware(otherMiddleware(finalHandler))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !finalCalled {
		t.Error("Expected final handler to be called")
	}

	// Both middlewares should have set headers
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("Expected security headers from security middleware")
	}

	if rr.Header().Get("X-Other-Middleware") != "present" {
		t.Error("Expected header from other middleware")
	}

	if rr.Body.String() != "final" {
		t.Errorf("Expected body 'final', got '%s'", rr.Body.String())
	}
}

func TestSecurityHeaders_ConfigVariations(t *testing.T) {
	tests := []struct {
		name         string
		cookieSecure bool
		expectHSTS   bool
	}{
		{
			name:         "Secure cookies enabled",
			cookieSecure: true,
			expectHSTS:   true,
		},
		{
			name:         "Secure cookies disabled",
			cookieSecure: false,
			expectHSTS:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SecurityConfig{CookieSecure: tt.cookieSecure}
			middleware := SecurityHeaders(cfg)

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware(next)

			req := httptest.NewRequest("GET", "/", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			hsts := rr.Header().Get("Strict-Transport-Security")
			if tt.expectHSTS {
				if hsts == "" {
					t.Error("Expected HSTS header when CookieSecure is true")
				}
			} else {
				if hsts != "" {
					t.Error("Expected no HSTS header when CookieSecure is false")
				}
			}

			// All other headers should always be set
			if rr.Header().Get("X-Frame-Options") != "DENY" {
				t.Error("Expected X-Frame-Options regardless of CookieSecure setting")
			}
			if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Error("Expected X-Content-Type-Options regardless of CookieSecure setting")
			}
		})
	}
}

func TestSecurityHeaders_AppliedToAllPaths(t *testing.T) {
	cfg := SecurityConfig{CookieSecure: true}
	middleware := SecurityHeaders(cfg)

	paths := []string{
		"/",
		"/api/auth/login",
		"/api/dashboard",
		"/static/css/main.css",
		"/api/v1/users/123",
		"/health",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware(next)

			req := httptest.NewRequest("GET", path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Header().Get("X-Frame-Options") != "DENY" {
				t.Errorf("Expected security headers for path: %s", path)
			}
		})
	}
}
