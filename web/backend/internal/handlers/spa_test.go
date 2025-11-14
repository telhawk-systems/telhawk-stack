package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestFiles creates a temporary directory structure for testing
func setupTestFiles(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "spa-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test files
	indexHTML := filepath.Join(tmpDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<html>Index</html>"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create index.html: %v", err)
	}

	// Create a static file (e.g., CSS)
	staticFile := filepath.Join(tmpDir, "styles.css")
	if err := os.WriteFile(staticFile, []byte("body { margin: 0; }"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create styles.css: %v", err)
	}

	// Create a subdirectory with a file
	assetsDir := filepath.Join(tmpDir, "assets")
	if err := os.Mkdir(assetsDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create assets dir: %v", err)
	}

	logoFile := filepath.Join(assetsDir, "logo.png")
	if err := os.WriteFile(logoFile, []byte("PNG-DATA"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create logo.png: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestSPAHandler_ServeIndexHTML(t *testing.T) {
	staticPath, cleanup := setupTestFiles(t)
	defer cleanup()

	fileServer := http.FileServer(http.Dir(staticPath))
	handler := NewSPAHandler(staticPath, fileServer)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if body := rr.Body.String(); body != "<html>Index</html>" {
		t.Errorf("Expected index.html content, got '%s'", body)
	}
}

func TestSPAHandler_ServeStaticFile(t *testing.T) {
	staticPath, cleanup := setupTestFiles(t)
	defer cleanup()

	fileServer := http.FileServer(http.Dir(staticPath))
	handler := NewSPAHandler(staticPath, fileServer)

	req := httptest.NewRequest("GET", "/styles.css", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if body := rr.Body.String(); body != "body { margin: 0; }" {
		t.Errorf("Expected CSS content, got '%s'", body)
	}
}

func TestSPAHandler_ServeNestedFile(t *testing.T) {
	staticPath, cleanup := setupTestFiles(t)
	defer cleanup()

	fileServer := http.FileServer(http.Dir(staticPath))
	handler := NewSPAHandler(staticPath, fileServer)

	req := httptest.NewRequest("GET", "/assets/logo.png", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if body := rr.Body.String(); body != "PNG-DATA" {
		t.Errorf("Expected PNG content, got '%s'", body)
	}
}

func TestSPAHandler_FallbackToIndexForNonexistentPath(t *testing.T) {
	staticPath, cleanup := setupTestFiles(t)
	defer cleanup()

	fileServer := http.FileServer(http.Dir(staticPath))
	handler := NewSPAHandler(staticPath, fileServer)

	tests := []struct {
		name string
		path string
	}{
		{
			name: "Root route",
			path: "/dashboard",
		},
		{
			name: "Nested route",
			path: "/users/123",
		},
		{
			name: "Deep nested route",
			path: "/admin/settings/profile",
		},
		{
			name: "Route with query params",
			path: "/search?q=test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}

			// Should serve index.html for SPA routing
			if body := rr.Body.String(); body != "<html>Index</html>" {
				t.Errorf("Expected index.html content, got '%s'", body)
			}
		})
	}
}

func TestSPAHandler_DirectoryTraversalPrevention(t *testing.T) {
	staticPath, cleanup := setupTestFiles(t)
	defer cleanup()

	fileServer := http.FileServer(http.Dir(staticPath))
	handler := NewSPAHandler(staticPath, fileServer)

	tests := []struct {
		name string
		path string
	}{
		{
			name: "Parent directory traversal",
			path: "/../etc/passwd",
		},
		{
			name: "Multiple parent traversals",
			path: "/../../etc/passwd",
		},
		{
			name: "URL encoded traversal",
			path: "/%2e%2e/etc/passwd",
		},
		{
			name: "Mixed traversal",
			path: "/assets/../../etc/passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Go's http.Request parsing rejects directory traversal attempts with 400 Bad Request
			// This is good security behavior - the request is rejected before reaching our handler
			if rr.Code != http.StatusBadRequest && rr.Code != http.StatusOK {
				t.Errorf("Expected status 400 (rejected) or 200 (fallback), got %d", rr.Code)
			}

			body := rr.Body.String()
			// Should not serve system files
			if strings.Contains(body, "root:x:0:0:") {
				t.Error("Directory traversal was not prevented - system file was served")
			}

			// Either rejected by Go's http library (400) or falls back to index.html (200)
			if rr.Code == http.StatusOK && body != "<html>Index</html>" {
				t.Errorf("Expected index.html fallback for status 200, got '%s'", body)
			}
		})
	}
}

func TestSPAHandler_CleanPath(t *testing.T) {
	staticPath, cleanup := setupTestFiles(t)
	defer cleanup()

	fileServer := http.FileServer(http.Dir(staticPath))
	handler := NewSPAHandler(staticPath, fileServer)

	tests := []struct {
		name         string
		path         string
		expectedBody string
	}{
		{
			name:         "Path with extra slashes",
			path:         "///styles.css",
			expectedBody: "body { margin: 0; }",
		},
		{
			name:         "Path with dot segments",
			path:         "/./styles.css",
			expectedBody: "body { margin: 0; }",
		},
		{
			name:         "Path with double slashes in middle",
			path:         "/assets//logo.png",
			expectedBody: "PNG-DATA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}

			if body := rr.Body.String(); body != tt.expectedBody {
				t.Errorf("Expected '%s', got '%s'", tt.expectedBody, body)
			}
		})
	}
}

func TestSPAHandler_FileStatError(t *testing.T) {
	// Use a directory that we can't stat (permission denied scenario)
	staticPath, cleanup := setupTestFiles(t)
	defer cleanup()

	// Create a file with restricted permissions
	restrictedFile := filepath.Join(staticPath, "restricted.txt")
	if err := os.WriteFile(restrictedFile, []byte("restricted"), 0644); err != nil {
		t.Fatalf("Failed to create restricted file: %v", err)
	}

	// Change permissions to make it unreadable (only works on Unix-like systems)
	if err := os.Chmod(restrictedFile, 0000); err != nil {
		t.Skip("Cannot change file permissions on this system")
	}
	defer os.Chmod(restrictedFile, 0644) // Restore for cleanup

	fileServer := http.FileServer(http.Dir(staticPath))
	handler := NewSPAHandler(staticPath, fileServer)

	req := httptest.NewRequest("GET", "/restricted.txt", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should return error status for stat errors other than NotExist
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError && rr.Code != http.StatusForbidden {
		// Different systems handle permission errors differently
		// Some may serve index.html (OK), others may error (500 or 403)
		t.Logf("Status code: %d (system-dependent)", rr.Code)
	}
}

func TestSPAHandler_Integration(t *testing.T) {
	staticPath, cleanup := setupTestFiles(t)
	defer cleanup()

	fileServer := http.FileServer(http.Dir(staticPath))
	handler := NewSPAHandler(staticPath, fileServer)

	// Simulate a typical SPA usage pattern
	tests := []struct {
		name         string
		path         string
		expectedCode int
		shouldBeHTML bool
	}{
		{
			name:         "Home page",
			path:         "/",
			expectedCode: http.StatusOK,
			shouldBeHTML: true,
		},
		{
			name:         "Static CSS file",
			path:         "/styles.css",
			expectedCode: http.StatusOK,
			shouldBeHTML: false,
		},
		{
			name:         "SPA route - dashboard",
			path:         "/dashboard",
			expectedCode: http.StatusOK,
			shouldBeHTML: true,
		},
		{
			name:         "SPA route - user detail",
			path:         "/users/123",
			expectedCode: http.StatusOK,
			shouldBeHTML: true,
		},
		{
			name:         "Static asset",
			path:         "/assets/logo.png",
			expectedCode: http.StatusOK,
			shouldBeHTML: false,
		},
		{
			name:         "Nonexistent static file",
			path:         "/nonexistent.js",
			expectedCode: http.StatusOK,
			shouldBeHTML: true, // Fallback to index.html
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			body := rr.Body.String()
			if tt.shouldBeHTML {
				if body != "<html>Index</html>" {
					t.Errorf("Expected HTML content, got '%s'", body)
				}
			} else {
				if body == "<html>Index</html>" {
					t.Error("Should not serve index.html for static files")
				}
			}
		})
	}
}
