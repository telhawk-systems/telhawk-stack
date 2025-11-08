package middleware

import (
	"net/http"
)

type SecurityConfig struct {
	CookieSecure bool
}

func SecurityHeaders(cfg SecurityConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			
			// CSP without unsafe-inline - strict security
			csp := "default-src 'self'; " +
				"script-src 'self'; " +
				"style-src 'self'; " +
				"img-src 'self' data:; " +
				"font-src 'self'; " +
				"connect-src 'self'; " +
				"frame-ancestors 'none'; " +
				"base-uri 'self'; " +
				"form-action 'self'"
			
			w.Header().Set("Content-Security-Policy", csp)
			
			if cfg.CookieSecure {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}
			
			next.ServeHTTP(w, r)
		})
	}
}
