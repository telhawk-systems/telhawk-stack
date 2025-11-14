package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	// Handler that returns 200 OK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	tests := []struct {
		name                 string
		config               CORSConfig
		origin               string
		method               string
		expectOriginHeader   bool
		expectedOrigin       string
		expectCredentials    bool
		expectedMethods      string
		expectedHeaders      string
		expectedMaxAge       string
		expectedStatus       int
		expectedResponseBody string
	}{
		{
			name: "exact origin match",
			config: CORSConfig{
				AllowedOrigins:   []string{"https://example.com"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
				MaxAge:           600,
			},
			origin:               "https://example.com",
			method:               "GET",
			expectOriginHeader:   true,
			expectedOrigin:       "https://example.com",
			expectCredentials:    true,
			expectedMethods:      "GET, POST",
			expectedHeaders:      "Content-Type",
			expectedMaxAge:       "600",
			expectedStatus:       http.StatusOK,
			expectedResponseBody: "OK",
		},
		{
			name: "wildcard subdomain match",
			config: CORSConfig{
				AllowedOrigins: []string{"*.example.com"},
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"Authorization"},
				MaxAge:         300,
			},
			origin:               "https://app.example.com",
			method:               "GET",
			expectOriginHeader:   true,
			expectedOrigin:       "https://app.example.com",
			expectCredentials:    false,
			expectedMethods:      "GET",
			expectedHeaders:      "Authorization",
			expectedMaxAge:       "300",
			expectedStatus:       http.StatusOK,
			expectedResponseBody: "OK",
		},
		{
			name: "wildcard subdomain no match",
			config: CORSConfig{
				AllowedOrigins: []string{"*.example.com"},
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"Authorization"},
			},
			origin:               "https://example.com",
			method:               "GET",
			expectOriginHeader:   false,
			expectedMethods:      "GET",
			expectedHeaders:      "Authorization",
			expectedMaxAge:       "300",
			expectedStatus:       http.StatusOK,
			expectedResponseBody: "OK",
		},
		{
			name: "origin not in allowed list",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"Content-Type"},
			},
			origin:               "https://evil.com",
			method:               "GET",
			expectOriginHeader:   false,
			expectedMethods:      "GET",
			expectedHeaders:      "Content-Type",
			expectedMaxAge:       "300",
			expectedStatus:       http.StatusOK,
			expectedResponseBody: "OK",
		},
		{
			name: "preflight OPTIONS request",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET", "POST", "PUT"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
			},
			origin:             "https://example.com",
			method:             "OPTIONS",
			expectOriginHeader: true,
			expectedOrigin:     "https://example.com",
			expectedMethods:    "GET, POST, PUT",
			expectedHeaders:    "Content-Type, Authorization",
			expectedMaxAge:     "300",
			expectedStatus:     http.StatusNoContent,
			// OPTIONS request should not call next handler
			expectedResponseBody: "",
		},
		{
			name: "no origin header",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"Content-Type"},
			},
			origin:               "",
			method:               "GET",
			expectOriginHeader:   false,
			expectedMethods:      "GET",
			expectedHeaders:      "Content-Type",
			expectedMaxAge:       "300",
			expectedStatus:       http.StatusOK,
			expectedResponseBody: "OK",
		},
		{
			name: "default max age",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET"},
				AllowedHeaders: []string{"Content-Type"},
				MaxAge:         0, // Should default to 300
			},
			origin:               "https://example.com",
			method:               "GET",
			expectOriginHeader:   true,
			expectedOrigin:       "https://example.com",
			expectedMethods:      "GET",
			expectedHeaders:      "Content-Type",
			expectedMaxAge:       "300",
			expectedStatus:       http.StatusOK,
			expectedResponseBody: "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(tt.method, "http://example.com/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Apply CORS middleware
			corsMiddleware := CORS(tt.config)
			corsHandler := corsMiddleware(handler)

			// Execute request
			corsHandler.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check Access-Control-Allow-Origin header
			originHeader := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectOriginHeader {
				if originHeader != tt.expectedOrigin {
					t.Errorf("expected Access-Control-Allow-Origin %q, got %q", tt.expectedOrigin, originHeader)
				}
			} else {
				if originHeader != "" {
					t.Errorf("expected no Access-Control-Allow-Origin header, got %q", originHeader)
				}
			}

			// Check Access-Control-Allow-Methods
			methodsHeader := w.Header().Get("Access-Control-Allow-Methods")
			if methodsHeader != tt.expectedMethods {
				t.Errorf("expected Access-Control-Allow-Methods %q, got %q", tt.expectedMethods, methodsHeader)
			}

			// Check Access-Control-Allow-Headers
			headersHeader := w.Header().Get("Access-Control-Allow-Headers")
			if headersHeader != tt.expectedHeaders {
				t.Errorf("expected Access-Control-Allow-Headers %q, got %q", tt.expectedHeaders, headersHeader)
			}

			// Check Access-Control-Allow-Credentials
			credentialsHeader := w.Header().Get("Access-Control-Allow-Credentials")
			if tt.expectCredentials {
				if credentialsHeader != "true" {
					t.Errorf("expected Access-Control-Allow-Credentials %q, got %q", "true", credentialsHeader)
				}
			} else {
				if credentialsHeader == "true" {
					t.Errorf("expected no Access-Control-Allow-Credentials or false, got %q", credentialsHeader)
				}
			}

			// Check Access-Control-Max-Age
			maxAgeHeader := w.Header().Get("Access-Control-Max-Age")
			if maxAgeHeader != tt.expectedMaxAge {
				t.Errorf("expected Access-Control-Max-Age %q, got %q", tt.expectedMaxAge, maxAgeHeader)
			}

			// Check response body
			if w.Body.String() != tt.expectedResponseBody {
				t.Errorf("expected response body %q, got %q", tt.expectedResponseBody, w.Body.String())
			}
		})
	}
}

func TestCORS_MultipleOrigins(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com", "https://app.example.com", "*.subdomain.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	tests := []struct {
		origin         string
		expectAllowed  bool
		expectedOrigin string
	}{
		{
			origin:         "https://example.com",
			expectAllowed:  true,
			expectedOrigin: "https://example.com",
		},
		{
			origin:         "https://app.example.com",
			expectAllowed:  true,
			expectedOrigin: "https://app.example.com",
		},
		{
			origin:         "https://test.subdomain.com",
			expectAllowed:  true,
			expectedOrigin: "https://test.subdomain.com",
		},
		{
			origin:        "https://evil.com",
			expectAllowed: false,
		},
	}

	corsMiddleware := CORS(config)
	corsHandler := corsMiddleware(handler)

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/test", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			corsHandler.ServeHTTP(w, req)

			originHeader := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectAllowed {
				if originHeader != tt.expectedOrigin {
					t.Errorf("expected origin %q to be allowed, got %q", tt.expectedOrigin, originHeader)
				}
			} else {
				if originHeader != "" {
					t.Errorf("expected origin %q to be blocked, but got %q", tt.origin, originHeader)
				}
			}
		})
	}
}
