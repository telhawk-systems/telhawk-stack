package handlers

import (
	"encoding/json"
	"github.com/telhawk-systems/telhawk-stack/common/httputil"
	"log"
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/auth"
)

type AuthHandler struct {
	authClient   *auth.Client
	cookieDomain string
	cookieSecure bool
}

func NewAuthHandler(authClient *auth.Client, cookieDomain string, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		authClient:   authClient,
		cookieDomain: cookieDomain,
		cookieSecure: cookieSecure,
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) GetCSRFToken(w http.ResponseWriter, r *http.Request) {
	// Go 1.25's CrossOriginProtection uses header-based validation
	// (Sec-Fetch-Site and Origin) instead of tokens
	// Return empty token for backward compatibility with frontend
	log.Printf("CSRF token requested from %s (using header-based protection)", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"csrf_token": "",
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	csrfToken := r.Header.Get("X-CSRF-Token")
	if len(csrfToken) > 20 {
		csrfToken = csrfToken[:20] + "..."
	}
	log.Printf("Login attempt from %s with CSRF header: %s", r.RemoteAddr, csrfToken)

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	loginResp, err := h.authClient.Login(req.Username, req.Password)
	if err != nil {
		log.Printf("Login error: %v", err)
		http.Error(w, "Login failed", http.StatusUnauthorized)
		return
	}

	h.setAccessTokenCookie(w, loginResp.AccessToken, loginResp.ExpiresIn)
	h.setRefreshTokenCookie(w, loginResp.RefreshToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Login successful",
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshCookie, err := r.Cookie("refresh_token")
	if err == nil && refreshCookie.Value != "" {
		if err := h.authClient.RevokeToken(refreshCookie.Value); err != nil {
			log.Printf("Token revocation error: %v", err)
		}
	}

	h.clearAuthCookies(w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logout successful",
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	roles := auth.GetRoles(r.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id": userID,
		"roles":   roles,
	})
}

func (h *AuthHandler) setAccessTokenCookie(w http.ResponseWriter, token string, expiresIn int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		Domain:   h.cookieDomain,
		MaxAge:   expiresIn,
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		Domain:   h.cookieDomain,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *AuthHandler) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		Domain:   h.cookieDomain,
		MaxAge:   -1,
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		Domain:   h.cookieDomain,
		MaxAge:   -1,
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}
