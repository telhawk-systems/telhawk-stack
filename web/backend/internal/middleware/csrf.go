package middleware

import (
	"crypto/rand"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/csrf"
)

func CSRF(cookieSecure bool) func(http.Handler) http.Handler {
	csrfKeyStr := getEnv("CSRF_KEY", "")

	var csrfKey []byte
	if csrfKeyStr == "" {
		// Generate a random key if not provided
		csrfKey = make([]byte, 32)
		if _, err := rand.Read(csrfKey); err != nil {
			panic("Failed to generate CSRF key: " + err.Error())
		}
		log.Println("WARNING: CSRF_KEY not set, generated random key (will change on restart)")
	} else {
		csrfKey = []byte(csrfKeyStr)
		if len(csrfKey) != 32 {
			panic("CSRF_KEY must be exactly 32 bytes")
		}
	}

	// Paths that should be exempt from CSRF protection
	// Authenticated endpoints are exempt because they're already protected by JWT auth
	exemptPaths := map[string]bool{
		"/api/auth/login":  true,  // Login is the first POST, needs to work without CSRF
		"/api/health":      true,  // Health check
	}

	// Prefixes for authenticated endpoints that don't need CSRF (already have JWT auth)
	exemptPrefixes := []string{
		"/api/query/",   // Query service - JWT protected
		"/api/core/",    // Core service - JWT protected
		"/api/auth/me",  // Current user endpoint - JWT protected
		"/api/auth/api", // Auth service API endpoints - JWT protected
	}

	csrfProtection := csrf.Protect(
		csrfKey,
		csrf.Secure(cookieSecure),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.Path("/"),
		csrf.RequestHeader("X-CSRF-Token"),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("CSRF validation failed for %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			log.Printf("CSRF Headers - X-CSRF-Token: '%s...', Cookie: '%s...'",
				r.Header.Get("X-CSRF-Token")[:min(20, len(r.Header.Get("X-CSRF-Token")))],
				r.Header.Get("Cookie")[:min(50, len(r.Header.Get("Cookie")))])
			if cookie, err := r.Cookie("_gorilla_csrf"); err == nil {
				log.Printf("CSRF Cookie value: %s...", cookie.Value[:min(30, len(cookie.Value))])
			} else {
				log.Printf("CSRF Cookie error: %v", err)
			}
			http.Error(w, "CSRF token validation failed", http.StatusForbidden)
		})),
	)

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
			csrfProtection(next).ServeHTTP(w, r)
		})
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
