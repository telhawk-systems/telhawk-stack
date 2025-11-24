package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/common/httputil"

	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/models"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Note: Method checking is handled by mux pattern "POST /api/v1/users/create"

	// Authentication check: require X-User-ID header (set by auth middleware)
	actorID := r.Header.Get("X-User-ID")
	if actorID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Authorization check: require admin role
	roles := r.Header.Get("X-User-Roles")
	if !strings.Contains(roles, "admin") {
		http.Error(w, "Forbidden: admin role required", http.StatusForbidden)
		return
	}

	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ipAddress := httputil.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	user, err := h.service.CreateUser(r.Context(), &req, actorID, ipAddress, userAgent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to JSON:API format
	resp := user.ToResponse()
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "user",
			"id":   resp.ID,
			"attributes": map[string]interface{}{
				"version_id": resp.VersionID,
				"username":   resp.Username,
				"email":      resp.Email,
				"roles":      resp.Roles,
				"enabled":    resp.Enabled,
			},
		},
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	httputil.WriteJSON(w, http.StatusCreated, response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ipAddress := httputil.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	resp, err := h.service.Login(r.Context(), &req, ipAddress, userAgent)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ValidateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.ValidateToken(r.Context(), req.Token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RevokeTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.RevokeToken(r.Context(), req.Token); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) ValidateHECToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ValidateHECTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ipAddress := httputil.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	hecToken, err := h.service.ValidateHECToken(r.Context(), req.Token, ipAddress, userAgent)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.ValidateHECTokenResponse{
			Valid: false,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.ValidateHECTokenResponse{
		Valid:     true,
		TokenID:   hecToken.ID,
		TokenName: hecToken.Name,
		UserID:    hecToken.UserID,
		ClientID:  hecToken.ClientID,
	})
}

func (h *AuthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get scope from headers
	scopeType := r.Header.Get("X-Scope-Type")
	orgIDStr := r.Header.Get("X-Organization-ID")
	clientIDStr := r.Header.Get("X-Client-ID")

	var users []*models.User
	var err error

	// If scope headers present, filter users by scope
	if scopeType != "" {
		var orgID, clientID *string
		if orgIDStr != "" {
			orgID = &orgIDStr
		}
		if clientIDStr != "" {
			clientID = &clientIDStr
		}
		users, err = h.service.ListUsersByScope(r.Context(), scopeType, orgID, clientID)
	} else {
		// No scope specified - return all users (admin fallback)
		users, err = h.service.ListUsers(r.Context())
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to JSON:API format
	data := make([]map[string]interface{}, len(users))
	for i, user := range users {
		resp := user.ToResponse()
		data[i] = map[string]interface{}{
			"type": "user",
			"id":   resp.ID,
			"attributes": map[string]interface{}{
				"version_id": resp.VersionID,
				"username":   resp.Username,
				"email":      resp.Email,
				"roles":      resp.Roles,
				"enabled":    resp.Enabled,
			},
		}
	}

	response := map[string]interface{}{
		"data": data,
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "user id required", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Convert to JSON:API format
	resp := user.ToResponse()
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "user",
			"id":   resp.ID,
			"attributes": map[string]interface{}{
				"version_id": resp.VersionID,
				"username":   resp.Username,
				"email":      resp.Email,
				"roles":      resp.Roles,
				"enabled":    resp.Enabled,
			},
		},
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "user id required", http.StatusBadRequest)
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	actorID := r.Header.Get("X-User-ID")
	ipAddress := httputil.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	user, err := h.service.UpdateUserDetails(r.Context(), userID, &req, actorID, ipAddress, userAgent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to JSON:API format
	resp := user.ToResponse()
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "user",
			"id":   resp.ID,
			"attributes": map[string]interface{}{
				"version_id": resp.VersionID,
				"username":   resp.Username,
				"email":      resp.Email,
				"roles":      resp.Roles,
				"enabled":    resp.Enabled,
			},
		},
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "user id required", http.StatusBadRequest)
		return
	}

	actorID := r.Header.Get("X-User-ID")
	ipAddress := httputil.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	if err := h.service.DeleteUser(r.Context(), userID, actorID, ipAddress, userAgent); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "user id required", http.StatusBadRequest)
		return
	}

	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	actorID := r.Header.Get("X-User-ID")
	ipAddress := httputil.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	if err := h.service.ResetPassword(r.Context(), userID, req.NewPassword, actorID, ipAddress, userAgent); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) CreateHECToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateHECTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ipAddress := httputil.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	token, err := h.service.CreateHECToken(r.Context(), userID, req.ClientID, req.Name, req.ExpiresIn, ipAddress, userAgent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return full token in JSON:API format (only shown once at creation)
	resp := token.ToResponse()
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "hec-token",
			"id":   resp.ID,
			"attributes": map[string]interface{}{
				"token":      resp.Token,
				"name":       resp.Name,
				"user_id":    resp.UserID,
				"client_id":  resp.ClientID,
				"enabled":    resp.Enabled,
				"expires_at": resp.ExpiresAt,
			},
		},
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	httputil.WriteJSON(w, http.StatusCreated, response)
}

func (h *AuthHandler) ListHECTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roles := r.Header.Get("X-User-Roles")
	isAdmin := strings.Contains(roles, "admin")

	var data []map[string]interface{}

	if isAdmin {
		// Admin users see all tokens with usernames
		usernames, tokens, err := h.service.ListAllHECTokensWithUsernames(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data = make([]map[string]interface{}, len(tokens))
		for i, token := range tokens {
			username := usernames[token.UserID]
			resp := token.ToMaskedResponseWithUsername(username)
			data[i] = map[string]interface{}{
				"type": "hec-token",
				"id":   resp.ID,
				"attributes": map[string]interface{}{
					"token":      resp.Token,
					"name":       resp.Name,
					"user_id":    resp.UserID,
					"username":   resp.Username,
					"enabled":    resp.Enabled,
					"expires_at": resp.ExpiresAt,
				},
			}
		}
	} else {
		// Regular users only see their own tokens without usernames
		tokens, err := h.service.ListHECTokensByUser(r.Context(), userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data = make([]map[string]interface{}, len(tokens))
		for i, token := range tokens {
			resp := token.ToMaskedResponse()
			data[i] = map[string]interface{}{
				"type": "hec-token",
				"id":   resp.ID,
				"attributes": map[string]interface{}{
					"token":      resp.Token,
					"name":       resp.Name,
					"user_id":    resp.UserID,
					"enabled":    resp.Enabled,
					"expires_at": resp.ExpiresAt,
				},
			}
		}
	}

	response := map[string]interface{}{
		"data": data,
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) RevokeHECTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.RevokeHECTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ipAddress := httputil.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	if err := h.service.RevokeHECTokenByUser(r.Context(), req.Token, userID, ipAddress, userAgent); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RevokeHECTokenByIDHandler handles RESTful revocation with token ID in URL path
// Endpoint: DELETE /api/v1/hec/tokens/{id}/revoke
func (h *AuthHandler) RevokeHECTokenByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract token ID from path: /api/v1/hec/tokens/{id}/revoke
	path := r.URL.Path
	// Remove prefix and suffix to get token ID
	tokenID := strings.TrimPrefix(path, "/api/v1/hec/tokens/")
	tokenID = strings.TrimSuffix(tokenID, "/revoke")

	if tokenID == "" {
		http.Error(w, "Token ID is required", http.StatusBadRequest)
		return
	}

	ipAddress := httputil.GetClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	if err := h.service.RevokeHECTokenByID(r.Context(), tokenID, userID, ipAddress, userAgent); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetUserScope returns the user's accessible organizations and clients for the scope picker
func (h *AuthHandler) GetUserScope(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rolesHeader := r.Header.Get("X-User-Roles")
	var roles []string
	if rolesHeader != "" {
		roles = strings.Split(rolesHeader, ",")
	}

	scope, err := h.service.GetUserScope(r.Context(), userID, roles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scope)
}
