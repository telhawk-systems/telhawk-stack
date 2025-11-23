package server

import (
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/common/middleware"

	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/handlers"
	authmw "github.com/telhawk-systems/telhawk-stack/authenticate/internal/middleware"
)

// NewRouter constructs a ServeMux with auth API routes registered.
func NewRouter(h *handlers.AuthHandler, authMW *authmw.AuthMiddleware) http.Handler {
	mux := http.NewServeMux()

	// Authentication endpoints (public - no auth required)
	mux.HandleFunc("/api/v1/auth/login", h.Login)
	mux.HandleFunc("/api/v1/auth/refresh", h.RefreshToken)

	// Service-to-service validation endpoints (internal, no user auth)
	mux.HandleFunc("/api/v1/auth/validate", h.ValidateToken)
	mux.HandleFunc("/api/v1/auth/validate-hec", h.ValidateHECToken)

	// Token revocation (requires auth)
	mux.HandleFunc("/api/v1/auth/revoke", authMW.RequireAuth(h.RevokeToken))

	// User scope endpoint (requires auth - user needs to see their own scope)
	mux.HandleFunc("GET /api/v1/auth/scope", authMW.RequireAuth(h.GetUserScope))

	// User management endpoints (protected with specific permissions)
	mux.HandleFunc("POST /api/v1/users/create", authMW.RequirePermission("users:create")(h.CreateUser))
	mux.HandleFunc("GET /api/v1/users/get", authMW.RequirePermission("users:read")(h.GetUser))
	mux.HandleFunc("PUT /api/v1/users/update", authMW.RequirePermission("users:update")(h.UpdateUser))
	mux.HandleFunc("PATCH /api/v1/users/update", authMW.RequirePermission("users:update")(h.UpdateUser))
	mux.HandleFunc("DELETE /api/v1/users/delete", authMW.RequirePermission("users:delete")(h.DeleteUser))
	mux.HandleFunc("POST /api/v1/users/reset-password", authMW.RequirePermission("users:reset_password")(h.ResetPassword))
	mux.HandleFunc("GET /api/v1/users", authMW.RequirePermission("users:read")(h.ListUsers))

	// HEC token management endpoints (protected with specific permissions)
	mux.HandleFunc("/api/v1/hec/tokens", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			authMW.RequirePermission("tokens:create")(h.CreateHECToken)(w, r)
		case http.MethodGet:
			authMW.RequirePermission("tokens:read")(h.ListHECTokens)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/hec/tokens/revoke", authMW.RequirePermission("tokens:revoke")(h.RevokeHECTokenHandler))

	// RESTful endpoint for revoking specific token by ID: /api/v1/hec/tokens/{id}/revoke
	mux.HandleFunc("/api/v1/hec/tokens/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Check if path matches /api/v1/hec/tokens/{id}/revoke
		if strings.HasPrefix(path, "/api/v1/hec/tokens/") && strings.HasSuffix(path, "/revoke") {
			if r.Method == http.MethodDelete || r.Method == http.MethodPost {
				authMW.RequirePermission("tokens:revoke")(h.RevokeHECTokenByIDHandler)(w, r)
				return
			}
		}
		http.Error(w, "Not found", http.StatusNotFound)
	})

	// Health check (public)
	mux.HandleFunc("/healthz", h.HealthCheck)

	return middleware.RequestID(mux)
}
