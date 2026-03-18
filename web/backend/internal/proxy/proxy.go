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

		// Copy only an explicit allowlist of headers to avoid forwarding hop-by-hop,
		// Host, or attacker-controlled trust headers to backend services.
		allowedHeaders := map[string]bool{
			"Content-Type":     true,
			"Accept":           true,
			"Accept-Language":  true,
			"Accept-Encoding":  true,
			"X-Request-Id":     true,
			"X-Correlation-Id": true,
		}
		for key, values := range r.Header {
			if allowedHeaders[http.CanonicalHeaderKey(key)] {
				proxyReq.Header[http.CanonicalHeaderKey(key)] = values
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

		// Scope headers (X-Scope-Type, X-Organization-ID, X-Client-ID) must not be
		// forwarded from the client request; they would allow any authenticated user
		// to impersonate another tenant. They are intentionally omitted here and must
		// be set by backend services from their own JWT validation if needed.

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
		if _, err := io.Copy(w, resp.Body); err != nil {
			log.Printf("Error copying proxy response body: %v", err)
		}
	})
}
