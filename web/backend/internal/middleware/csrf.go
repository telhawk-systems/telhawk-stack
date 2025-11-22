package middleware

import (
	"log"
	"net/http"
)

func CSRF(cookieSecure bool) func(http.Handler) http.Handler {
	// Paths that should be exempt from CSRF protection
	// Authenticated endpoints are exempt because they're already protected by JWT auth
	exemptPaths := map[string]bool{
		"/api/auth/login": true, // Login is the first POST, needs to work without CSRF
		"/api/health":     true, // Health check
	}

	// Prefixes for authenticated endpoints that don't need CSRF (already have JWT auth)
	exemptPrefixes := []string{
		"/api/search/",  // Search service - JWT protected
		"/api/core/",    // Core service - JWT protected
		"/api/auth/me",  // Current user endpoint - JWT protected
		"/api/auth/api", // Auth service API endpoints - JWT protected
	}

	// Use Go 1.25's built-in CSRF protection
	csrfProtection := http.NewCrossOriginProtection()
	csrfProtection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("CSRF validation failed for %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		http.Error(w, "CSRF token validation failed", http.StatusForbidden)
	}))

	// Return a middleware that exempts certain paths
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this exact path should be exempt from CSRF
			if exemptPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Check if this path starts with an exempt prefix
			for _, prefix := range exemptPrefixes {
				if len(r.URL.Path) >= len(prefix) && r.URL.Path[:len(prefix)] == prefix {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Apply CSRF protection for all other paths
			csrfProtection.Handler(next).ServeHTTP(w, r)
		})
	}
}
