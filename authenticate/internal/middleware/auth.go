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
	RolesKey              contextKey = "roles"
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

		roles := resp.Roles

		// If permissions are stale (roles changed since token was issued),
		// reload fresh user data from the database
		if resp.PermissionsStale {
			user, err := m.authService.GetUserByID(r.Context(), resp.UserID)
			if err == nil && user != nil {
				roles = user.Roles
			}
			// If we can't load fresh data, fall back to JWT roles (better than failing)
		}

		ctx := context.WithValue(r.Context(), UserIDKey, resp.UserID)
		ctx = context.WithValue(ctx, RolesKey, roles)
		ctx = context.WithValue(ctx, PermissionsVersionKey, resp.PermissionsVersion)
		ctx = context.WithValue(ctx, PermissionsStaleKey, resp.PermissionsStale)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (m *AuthMiddleware) RequireRole(role string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			roles, ok := r.Context().Value(RolesKey).([]string)
			if !ok {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			hasRole := false
			for _, r := range roles {
				if r == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
