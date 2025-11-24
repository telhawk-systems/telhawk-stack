package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/auth"
	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/proxy"
)

// createMockAuthServer creates a test auth service
func createMockAuthServer() (*httptest.Server, *auth.Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"valid": true, "user_id": "test-user", "roles": ["admin"]}`))
	}))
	return server, auth.NewClient(server.URL)
}

// createMockBackendServer creates a mock backend service
func createMockBackendServer(response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
}

// setupTestStaticDir creates a temporary directory with test files
func setupTestStaticDir(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "router-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	indexHTML := filepath.Join(tmpDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<html>Test Index</html>"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create index.html: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestNewRouter_HealthCheck(t *testing.T) {
	authServer, authClient := createMockAuthServer()
	defer authServer.Close()

	staticDir, cleanup := setupTestStaticDir(t)
	defer cleanup()

	queryServer := createMockBackendServer("query response")
	defer queryServer.Close()

	cfg := RouterConfig{
		AuthHandler:       handlers.NewAuthHandler(authClient, "localhost", false),
		DashboardHandler:  handlers.NewDashboardHandler(queryServer.URL, ""),
		AuthMiddleware:    auth.NewMiddleware(authClient, "localhost", false),
		AuthenticateProxy: proxy.NewProxy(authServer.URL, authClient),
		SearchProxy:       proxy.NewProxy(queryServer.URL, authClient),
		CoreProxy:         proxy.NewProxy(queryServer.URL, authClient),
		RespondProxy:      proxy.NewProxy(queryServer.URL, authClient),
		StaticDir:         staticDir,
	}

	router := NewRouter(cfg)

	req := httptest.NewRequest("GET", "/api/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	expectedBody := `{"status":"ok","service":"web"}`
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestNewRouter_AuthEndpoints_GetCSRFToken(t *testing.T) {
	authServer, authClient := createMockAuthServer()
	defer authServer.Close()

	staticDir, cleanup := setupTestStaticDir(t)
	defer cleanup()

	queryServer := createMockBackendServer("query response")
	defer queryServer.Close()

	cfg := RouterConfig{
		AuthHandler:       handlers.NewAuthHandler(authClient, "localhost", false),
		DashboardHandler:  handlers.NewDashboardHandler(queryServer.URL, ""),
		AuthMiddleware:    auth.NewMiddleware(authClient, "localhost", false),
		AuthenticateProxy: proxy.NewProxy(authServer.URL, authClient),
		SearchProxy:       proxy.NewProxy(queryServer.URL, authClient),
		CoreProxy:         proxy.NewProxy(queryServer.URL, authClient),
		RespondProxy:      proxy.NewProxy(queryServer.URL, authClient),
		StaticDir:         staticDir,
	}

	router := NewRouter(cfg)

	req := httptest.NewRequest("GET", "/api/auth/csrf-token", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "csrf_token") {
		t.Error("Expected response to contain csrf_token")
	}
}

func TestNewRouter_ProtectedRoute_RequiresAuth(t *testing.T) {
	authServer, authClient := createMockAuthServer()
	defer authServer.Close()

	staticDir, cleanup := setupTestStaticDir(t)
	defer cleanup()

	queryServer := createMockBackendServer("query response")
	defer queryServer.Close()

	cfg := RouterConfig{
		AuthHandler:       handlers.NewAuthHandler(authClient, "localhost", false),
		DashboardHandler:  handlers.NewDashboardHandler(queryServer.URL, ""),
		AuthMiddleware:    auth.NewMiddleware(authClient, "localhost", false),
		AuthenticateProxy: proxy.NewProxy(authServer.URL, authClient),
		SearchProxy:       proxy.NewProxy(queryServer.URL, authClient),
		CoreProxy:         proxy.NewProxy(queryServer.URL, authClient),
		RespondProxy:      proxy.NewProxy(queryServer.URL, authClient),
		StaticDir:         staticDir,
	}

	router := NewRouter(cfg)

	protectedRoutes := []string{
		"/api/auth/me",
		"/api/dashboard/metrics",
		"/api/search/v1/search",
		"/api/core/v1/events",
		"/api/rules/v1/schemas",
		"/api/alerting/v1/cases",
	}

	for _, route := range protectedRoutes {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest("GET", route, nil)
			// No auth cookie - should be unauthorized
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401 for protected route %s without auth, got %d", route, rr.Code)
			}
		})
	}
}

func TestNewRouter_StaticFileServing(t *testing.T) {
	authServer, authClient := createMockAuthServer()
	defer authServer.Close()

	staticDir, cleanup := setupTestStaticDir(t)
	defer cleanup()

	queryServer := createMockBackendServer("query response")
	defer queryServer.Close()

	cfg := RouterConfig{
		AuthHandler:       handlers.NewAuthHandler(authClient, "localhost", false),
		DashboardHandler:  handlers.NewDashboardHandler(queryServer.URL, ""),
		AuthMiddleware:    auth.NewMiddleware(authClient, "localhost", false),
		AuthenticateProxy: proxy.NewProxy(authServer.URL, authClient),
		SearchProxy:       proxy.NewProxy(queryServer.URL, authClient),
		CoreProxy:         proxy.NewProxy(queryServer.URL, authClient),
		RespondProxy:      proxy.NewProxy(queryServer.URL, authClient),
		StaticDir:         staticDir,
	}

	router := NewRouter(cfg)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	expectedBody := "<html>Test Index</html>"
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestNewRouter_SPAFallback(t *testing.T) {
	authServer, authClient := createMockAuthServer()
	defer authServer.Close()

	staticDir, cleanup := setupTestStaticDir(t)
	defer cleanup()

	queryServer := createMockBackendServer("query response")
	defer queryServer.Close()

	cfg := RouterConfig{
		AuthHandler:       handlers.NewAuthHandler(authClient, "localhost", false),
		DashboardHandler:  handlers.NewDashboardHandler(queryServer.URL, ""),
		AuthMiddleware:    auth.NewMiddleware(authClient, "localhost", false),
		AuthenticateProxy: proxy.NewProxy(authServer.URL, authClient),
		SearchProxy:       proxy.NewProxy(queryServer.URL, authClient),
		CoreProxy:         proxy.NewProxy(queryServer.URL, authClient),
		RespondProxy:      proxy.NewProxy(queryServer.URL, authClient),
		StaticDir:         staticDir,
	}

	router := NewRouter(cfg)

	spaRoutes := []string{
		"/dashboard",
		"/users/123",
		"/settings",
	}

	for _, route := range spaRoutes {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest("GET", route, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200 for SPA route %s, got %d", route, rr.Code)
			}

			// Should serve index.html for SPA routes
			expectedBody := "<html>Test Index</html>"
			if body := rr.Body.String(); body != expectedBody {
				t.Errorf("Expected index.html for SPA route, got '%s'", body)
			}
		})
	}
}

func TestNewRouter_RequestIDMiddleware(t *testing.T) {
	authServer, authClient := createMockAuthServer()
	defer authServer.Close()

	staticDir, cleanup := setupTestStaticDir(t)
	defer cleanup()

	queryServer := createMockBackendServer("query response")
	defer queryServer.Close()

	cfg := RouterConfig{
		AuthHandler:       handlers.NewAuthHandler(authClient, "localhost", false),
		DashboardHandler:  handlers.NewDashboardHandler(queryServer.URL, ""),
		AuthMiddleware:    auth.NewMiddleware(authClient, "localhost", false),
		AuthenticateProxy: proxy.NewProxy(authServer.URL, authClient),
		SearchProxy:       proxy.NewProxy(queryServer.URL, authClient),
		CoreProxy:         proxy.NewProxy(queryServer.URL, authClient),
		RespondProxy:      proxy.NewProxy(queryServer.URL, authClient),
		StaticDir:         staticDir,
	}

	router := NewRouter(cfg)

	req := httptest.NewRequest("GET", "/api/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Check that X-Request-ID header is set by the RequestID middleware
	requestID := rr.Header().Get("X-Request-Id")
	if requestID == "" {
		t.Error("Expected X-Request-Id header to be set by RequestID middleware")
	}
}

func TestNewRouter_StripPrefixForProxies(t *testing.T) {
	authServer, authClient := createMockAuthServer()
	defer authServer.Close()

	staticDir, cleanup := setupTestStaticDir(t)
	defer cleanup()

	receivedPaths := make(map[string]string)
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPaths[r.Header.Get("X-Test-Key")] = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backendServer.Close()

	cfg := RouterConfig{
		AuthHandler:       handlers.NewAuthHandler(authClient, "localhost", false),
		DashboardHandler:  handlers.NewDashboardHandler(backendServer.URL, ""),
		AuthMiddleware:    auth.NewMiddleware(authClient, "localhost", false),
		AuthenticateProxy: proxy.NewProxy(backendServer.URL, authClient),
		SearchProxy:       proxy.NewProxy(backendServer.URL, authClient),
		CoreProxy:         proxy.NewProxy(backendServer.URL, authClient),
		RespondProxy:      proxy.NewProxy(backendServer.URL, authClient),
		StaticDir:         staticDir,
	}

	router := NewRouter(cfg)

	tests := []struct {
		name         string
		requestPath  string
		expectedPath string
		testKey      string
	}{
		{
			name:         "Query proxy strips /api/query",
			requestPath:  "/api/search/v1/search",
			expectedPath: "/v1/search",
			testKey:      "query-test",
		},
		{
			name:         "Core proxy strips /api/core",
			requestPath:  "/api/core/v1/events",
			expectedPath: "/v1/events",
			testKey:      "core-test",
		},
		{
			name:         "Rules proxy strips /api/rules",
			requestPath:  "/api/rules/v1/schemas",
			expectedPath: "/v1/schemas",
			testKey:      "rules-test",
		},
		{
			name:         "Alerting proxy strips /api/alerting",
			requestPath:  "/api/alerting/v1/cases",
			expectedPath: "/v1/cases",
			testKey:      "alerting-test",
		},
		{
			name:         "Auth proxy strips /api/auth",
			requestPath:  "/api/auth/api/v1/users",
			expectedPath: "/api/v1/users",
			testKey:      "auth-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			req.Header.Set("X-Test-Key", tt.testKey)
			req.AddCookie(&http.Cookie{Name: "access_token", Value: "test-token"})
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if receivedPath, ok := receivedPaths[tt.testKey]; ok {
				if receivedPath != tt.expectedPath {
					t.Errorf("Expected backend to receive path '%s', got '%s'", tt.expectedPath, receivedPath)
				}
			} else {
				t.Errorf("Backend did not receive request with test key '%s'", tt.testKey)
			}
		})
	}
}

func TestNewRouter_RouteOrdering(t *testing.T) {
	authServer, authClient := createMockAuthServer()
	defer authServer.Close()

	staticDir, cleanup := setupTestStaticDir(t)
	defer cleanup()

	queryServer := createMockBackendServer("query response")
	defer queryServer.Close()

	cfg := RouterConfig{
		AuthHandler:       handlers.NewAuthHandler(authClient, "localhost", false),
		DashboardHandler:  handlers.NewDashboardHandler(queryServer.URL, ""),
		AuthMiddleware:    auth.NewMiddleware(authClient, "localhost", false),
		AuthenticateProxy: proxy.NewProxy(authServer.URL, authClient),
		SearchProxy:       proxy.NewProxy(queryServer.URL, authClient),
		CoreProxy:         proxy.NewProxy(queryServer.URL, authClient),
		RespondProxy:      proxy.NewProxy(queryServer.URL, authClient),
		StaticDir:         staticDir,
	}

	router := NewRouter(cfg)

	// API routes should take precedence over static file handler
	req := httptest.NewRequest("GET", "/api/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Should get health check response, not index.html
	if !strings.Contains(rr.Body.String(), "status") {
		t.Error("API routes should take precedence over static file handler")
	}

	expectedBody := `{"status":"ok","service":"web"}`
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("Expected health check response, got '%s'", body)
	}
}

func TestNewRouter_MethodRouting(t *testing.T) {
	authServer, authClient := createMockAuthServer()
	defer authServer.Close()

	staticDir, cleanup := setupTestStaticDir(t)
	defer cleanup()

	queryServer := createMockBackendServer("query response")
	defer queryServer.Close()

	cfg := RouterConfig{
		AuthHandler:       handlers.NewAuthHandler(authClient, "localhost", false),
		DashboardHandler:  handlers.NewDashboardHandler(queryServer.URL, ""),
		AuthMiddleware:    auth.NewMiddleware(authClient, "localhost", false),
		AuthenticateProxy: proxy.NewProxy(authServer.URL, authClient),
		SearchProxy:       proxy.NewProxy(queryServer.URL, authClient),
		CoreProxy:         proxy.NewProxy(queryServer.URL, authClient),
		RespondProxy:      proxy.NewProxy(queryServer.URL, authClient),
		StaticDir:         staticDir,
	}

	router := NewRouter(cfg)

	// Health check is registered as GET only
	t.Run("GET /api/health", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		expectedBody := `{"status":"ok","service":"web"}`
		if body := rr.Body.String(); body != expectedBody {
			t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
		}
	})

	// Non-GET methods to /api/health fall through to SPA handler
	t.Run("POST /api/health falls through to SPA", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/health", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		// Falls through to SPA handler which serves index.html
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		// Should serve index.html, not health check response
		if strings.Contains(rr.Body.String(), "status") {
			t.Error("POST should not match health endpoint, should fall through to SPA")
		}
	})
}

func TestNewRouter_ConfigComplete(t *testing.T) {
	authServer, authClient := createMockAuthServer()
	defer authServer.Close()

	staticDir, cleanup := setupTestStaticDir(t)
	defer cleanup()

	queryServer := createMockBackendServer("query response")
	defer queryServer.Close()

	// Test with complete config
	cfg := RouterConfig{
		AuthHandler:       handlers.NewAuthHandler(authClient, "localhost", false),
		DashboardHandler:  handlers.NewDashboardHandler(queryServer.URL, ""),
		AuthMiddleware:    auth.NewMiddleware(authClient, "localhost", false),
		AuthenticateProxy: proxy.NewProxy(authServer.URL, authClient),
		SearchProxy:       proxy.NewProxy(queryServer.URL, authClient),
		CoreProxy:         proxy.NewProxy(queryServer.URL, authClient),
		RespondProxy:      proxy.NewProxy(queryServer.URL, authClient),
		StaticDir:         staticDir,
	}

	router := NewRouter(cfg)

	if router == nil {
		t.Fatal("Expected router to be non-nil")
	}

	// Verify all major routes are accessible
	routes := []struct {
		path           string
		method         string
		requiresAuth   bool
		expectedStatus int
	}{
		{"/api/health", "GET", false, http.StatusOK},
		{"/api/auth/csrf-token", "GET", false, http.StatusOK},
		{"/api/auth/me", "GET", true, http.StatusUnauthorized}, // No auth
		{"/", "GET", false, http.StatusOK},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != route.expectedStatus {
				t.Errorf("Expected status %d, got %d", route.expectedStatus, rr.Code)
			}
		})
	}
}
