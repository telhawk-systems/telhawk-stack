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
	
	return csrf.Protect(
		csrfKey,
		csrf.Secure(cookieSecure),
		csrf.SameSite(csrf.SameSiteLaxMode),  // Changed from Strict to Lax for testing
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
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
