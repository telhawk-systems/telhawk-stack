package models

type CreateUserRequest struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Roles    []string `json:"roles,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ValidateTokenRequest struct {
	Token string `json:"token"`
}

type RevokeTokenRequest struct {
	Token string `json:"token"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type ValidateTokenResponse struct {
	Valid  bool     `json:"valid"`
	UserID string   `json:"user_id,omitempty"`
	Roles  []string `json:"roles,omitempty"`
}

type ValidateHECTokenRequest struct {
	Token string `json:"token"`
}

type ValidateHECTokenResponse struct {
	Valid     bool   `json:"valid"`
	TokenID   string `json:"token_id,omitempty"`
	TokenName string `json:"token_name,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"` // Client tenant for data isolation
}

type UpdateUserRequest struct {
	Email   string   `json:"email,omitempty"`
	Roles   []string `json:"roles,omitempty"`
	Enabled *bool    `json:"enabled,omitempty"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

type CreateHECTokenRequest struct {
	Name      string `json:"name"`
	ExpiresIn string `json:"expires_in,omitempty"`
}

type RevokeHECTokenRequest struct {
	Token string `json:"token"`
}
