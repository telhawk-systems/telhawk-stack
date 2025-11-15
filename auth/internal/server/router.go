package server

import (
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/common/middleware"

	"github.com/telhawk-systems/telhawk-stack/auth/internal/handlers"
)

// NewRouter constructs a ServeMux with auth API routes registered.
func NewRouter(h *handlers.AuthHandler) http.Handler {
	mux := http.NewServeMux()

	// Authentication endpoints
	mux.HandleFunc("/api/v1/auth/login", h.Login)
	mux.HandleFunc("/api/v1/auth/refresh", h.RefreshToken)
	mux.HandleFunc("/api/v1/auth/validate", h.ValidateToken)
	mux.HandleFunc("/api/v1/auth/validate-hec", h.ValidateHECToken)
	mux.HandleFunc("/api/v1/auth/revoke", h.RevokeToken)

	// User management endpoints (admin-only, requires authentication)
	// Use Go 1.22+ method routing for explicit path matching
	mux.HandleFunc("POST /api/v1/users/create", h.CreateUser)
	mux.HandleFunc("GET /api/v1/users/get", h.GetUser)
	mux.HandleFunc("PUT /api/v1/users/update", h.UpdateUser)
	mux.HandleFunc("PATCH /api/v1/users/update", h.UpdateUser)
	mux.HandleFunc("DELETE /api/v1/users/delete", h.DeleteUser)
	mux.HandleFunc("POST /api/v1/users/reset-password", h.ResetPassword)
	mux.HandleFunc("GET /api/v1/users", h.ListUsers)

	// HEC token management endpoints
	mux.HandleFunc("/api/v1/hec/tokens", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreateHECToken(w, r)
		case http.MethodGet:
			h.ListHECTokens(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/hec/tokens/revoke", h.RevokeHECTokenHandler)

	// RESTful endpoint for revoking specific token by ID: /api/v1/hec/tokens/{id}/revoke
	mux.HandleFunc("/api/v1/hec/tokens/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Check if path matches /api/v1/hec/tokens/{id}/revoke
		if strings.HasPrefix(path, "/api/v1/hec/tokens/") && strings.HasSuffix(path, "/revoke") {
			if r.Method == http.MethodDelete || r.Method == http.MethodPost {
				h.RevokeHECTokenByIDHandler(w, r)
				return
			}
		}
		http.Error(w, "Not found", http.StatusNotFound)
	})

	// Health check
	mux.HandleFunc("/healthz", h.HealthCheck)

	return middleware.RequestID(mux)
}
