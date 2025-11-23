package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/models"
)

// Context keys for RBAC data
const (
	UserKey contextKey = "user" // Full user object with roles and permissions
)

// UserFromContext retrieves the authenticated user from the request context
func UserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(UserKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// RequirePermission middleware checks if the user has a specific permission
// Permission format: "resource:action" (e.g., "users:create", "alerts:read")
func (m *AuthMiddleware) RequirePermission(permission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				writePermissionError(w, http.StatusUnauthorized, "authentication required", "")
				return
			}

			if !user.Can(permission) {
				writePermissionError(w, http.StatusForbidden,
					fmt.Sprintf("%s permission required", permission),
					permission,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission middleware checks if the user has at least one of the specified permissions
func (m *AuthMiddleware) RequireAnyPermission(permissions ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				writePermissionError(w, http.StatusUnauthorized, "authentication required", "")
				return
			}

			for _, permission := range permissions {
				if user.Can(permission) {
					next.ServeHTTP(w, r)
					return
				}
			}

			writePermissionError(w, http.StatusForbidden,
				fmt.Sprintf("one of %v permissions required", permissions),
				"",
			)
		})
	}
}

// RequireAllPermissions middleware checks if the user has all of the specified permissions
func (m *AuthMiddleware) RequireAllPermissions(permissions ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				writePermissionError(w, http.StatusUnauthorized, "authentication required", "")
				return
			}

			for _, permission := range permissions {
				if !user.Can(permission) {
					writePermissionError(w, http.StatusForbidden,
						fmt.Sprintf("%s permission required", permission),
						permission,
					)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOrdinal middleware ensures user has a role with ordinal <= maxOrdinal
// Lower ordinal = more powerful (0 is most powerful)
func (m *AuthMiddleware) RequireOrdinal(maxOrdinal int) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				writePermissionError(w, http.StatusUnauthorized, "authentication required", "")
				return
			}

			if user.LowestOrdinal() > maxOrdinal {
				writePermissionError(w, http.StatusForbidden,
					fmt.Sprintf("insufficient privilege level (requires ordinal <= %d)", maxOrdinal),
					"",
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireScopeTier middleware ensures user belongs to one of the specified scope tiers
// Scope tier is determined by: client_id NOT NULL → client, organization_id NOT NULL → org, both NULL → platform
func (m *AuthMiddleware) RequireScopeTier(allowedTiers ...models.ScopeTier) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				writePermissionError(w, http.StatusUnauthorized, "authentication required", "")
				return
			}

			userScopeTier := user.GetScopeTier()
			for _, allowed := range allowedTiers {
				if userScopeTier == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}

			writePermissionError(w, http.StatusForbidden,
				fmt.Sprintf("scope tier %s not authorized for this operation", userScopeTier),
				"",
			)
		})
	}
}

// PermissionError represents a permission-related error response
type PermissionError struct {
	Status     int    `json:"status"`
	Code       string `json:"code"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Permission string `json:"permission,omitempty"`
}

// writePermissionError writes a JSON:API style error response
func writePermissionError(w http.ResponseWriter, status int, detail string, permission string) {
	code := "forbidden"
	title := "Forbidden"
	if status == http.StatusUnauthorized {
		code = "unauthorized"
		title = "Unauthorized"
	}

	errResp := struct {
		Errors []PermissionError `json:"errors"`
	}{
		Errors: []PermissionError{
			{
				Status:     status,
				Code:       code,
				Title:      title,
				Detail:     detail,
				Permission: permission,
			},
		},
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errResp)
}
