package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// CORSConfig holds CORS middleware configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// CORS returns a middleware that handles Cross-Origin Resource Sharing (CORS)
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	allowedMethods := strings.Join(config.AllowedMethods, ", ")
	allowedHeaders := strings.Join(config.AllowedHeaders, ", ")
	maxAge := "300"
	if config.MaxAge > 0 {
		maxAge = fmt.Sprintf("%d", config.MaxAge)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed (supports wildcards like "*.example.com")
			if origin != "" {
				for _, allowed := range config.AllowedOrigins {
					matched := false

					// Wildcard match: "*.example.com" matches "app.example.com"
					if strings.HasPrefix(allowed, "*.") {
						suffix := strings.TrimPrefix(allowed, "*")
						if strings.HasSuffix(origin, suffix) {
							matched = true
						}
					} else if origin == allowed {
						// Exact match
						matched = true
					}

					if matched {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Max-Age", maxAge)

			// Handle preflight OPTIONS request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
