package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/service"
)

type contextKey string

const (
	UserIDKey             contextKey = "user_id"
	PermissionsVersionKey contextKey = "permissions_version"
	PermissionsStaleKey   contextKey = "permissions_stale"
)

type AuthMiddleware struct {
	authService *service.AuthService
}

func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		token := parts[1]
		resp, err := m.authService.ValidateToken(r.Context(), token)
		if err != nil || !resp.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Load the full user with roles and permissions for RBAC checks
		// GetUserWithRoles loads UserRoles -> Role -> Permissions for permission checking
		user, err := m.authService.GetUserWithRoles(r.Context(), resp.UserID)
		if err != nil {
			http.Error(w, "Failed to load user data", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, resp.UserID)
		ctx = context.WithValue(ctx, PermissionsVersionKey, resp.PermissionsVersion)
		ctx = context.WithValue(ctx, PermissionsStaleKey, resp.PermissionsStale)
		ctx = context.WithValue(ctx, UserKey, user) // Store full user for RBAC checks

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
