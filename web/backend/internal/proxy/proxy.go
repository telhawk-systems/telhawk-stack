package proxy

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/auth"
)

type Proxy struct {
	targetURL  string
	authClient *auth.Client
	httpClient *http.Client
}

func NewProxy(targetURL string, authClient *auth.Client) *Proxy {
	return &Proxy{
		targetURL:  targetURL,
		authClient: authClient,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetURL := p.targetURL + r.URL.Path
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
		if err != nil {
			log.Printf("Proxy request creation error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		// Inject Authorization header from access_token cookie if not present
		if proxyReq.Header.Get("Authorization") == "" {
			if c, err := r.Cookie("access_token"); err == nil && c.Value != "" {
				proxyReq.Header.Set("Authorization", "Bearer "+c.Value)
			}
		}

		userID := auth.GetUserID(r.Context())
		if userID != "" {
			proxyReq.Header.Set("X-User-ID", userID)
		}

		roles := auth.GetRoles(r.Context())
		if len(roles) > 0 {
			// Join roles with comma for X-User-Roles header
			rolesStr := ""
			for i, role := range roles {
				if i > 0 {
					rolesStr += ","
				}
				rolesStr += role
			}
			proxyReq.Header.Set("X-User-Roles", rolesStr)
		}

		// Forward scope headers for multi-organization data isolation
		// These headers are set by the frontend ScopeProvider
		if scopeType := r.Header.Get("X-Scope-Type"); scopeType != "" {
			proxyReq.Header.Set("X-Scope-Type", scopeType)
		}
		if orgID := r.Header.Get("X-Organization-ID"); orgID != "" {
			proxyReq.Header.Set("X-Organization-ID", orgID)
		}
		if clientID := r.Header.Get("X-Client-ID"); clientID != "" {
			proxyReq.Header.Set("X-Client-ID", clientID)
		}

		resp, err := p.httpClient.Do(proxyReq)
		if err != nil {
			log.Printf("Proxy request error: %v", err)
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
