package auth

import (
	"context"
	"log"
	"net/http"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	RolesKey  contextKey = "roles"
)

type Middleware struct {
	authClient   *Client
	cookieDomain string
	cookieSecure bool
}

func NewMiddleware(authClient *Client, cookieDomain string, cookieSecure bool) *Middleware {
	return &Middleware{
		authClient:   authClient,
		cookieDomain: cookieDomain,
		cookieSecure: cookieSecure,
	}
}

func (m *Middleware) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessCookie, err := r.Cookie("access_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		validateResp, err := m.authClient.ValidateToken(accessCookie.Value)
		if err != nil {
			log.Printf("Token validation error: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !validateResp.Valid {
			refreshCookie, err := r.Cookie("refresh_token")
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			loginResp, err := m.authClient.RefreshToken(refreshCookie.Value)
			if err != nil {
				log.Printf("Token refresh error: %v", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			m.setAccessTokenCookie(w, loginResp.AccessToken, loginResp.ExpiresIn)

			newValidateResp, err := m.authClient.ValidateToken(loginResp.AccessToken)
			if err != nil || !newValidateResp.Valid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			validateResp = newValidateResp
		}

		ctx := context.WithValue(r.Context(), UserIDKey, validateResp.UserID)
		ctx = context.WithValue(ctx, RolesKey, validateResp.Roles)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) setAccessTokenCookie(w http.ResponseWriter, token string, expiresIn int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		Domain:   m.cookieDomain,
		MaxAge:   expiresIn,
		Secure:   m.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (m *Middleware) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		Domain:   m.cookieDomain,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		Secure:   m.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (m *Middleware) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		Domain:   m.cookieDomain,
		MaxAge:   -1,
		Secure:   m.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		Domain:   m.cookieDomain,
		MaxAge:   -1,
		Secure:   m.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

func GetRoles(ctx context.Context) []string {
	if roles, ok := ctx.Value(RolesKey).([]string); ok {
		return roles
	}
	return []string{}
}
