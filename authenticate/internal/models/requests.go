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
	Valid              bool     `json:"valid"`
	UserID             string   `json:"user_id,omitempty"`
	Roles              []string `json:"roles,omitempty"`
	PermissionsVersion int      `json:"permissions_version,omitempty"`
	PermissionsStale   bool     `json:"permissions_stale,omitempty"` // True if JWT version != DB version
	OrganizationID     string   `json:"organization_id,omitempty"`   // Primary organization for data isolation
	ClientID           string   `json:"client_id,omitempty"`         // Primary client for data isolation
}

type ValidateHECTokenRequest struct {
	Token string `json:"token"`
}

type ValidateHECTokenResponse struct {
	Valid     bool   `json:"valid"`
	TokenID   string `json:"token_id,omitempty"`
	TokenName string `json:"token_name,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	ClientID  string `json:"client_id,omitempty"` // Client for data isolation
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
	ClientID  string `json:"client_id"`
	ExpiresIn string `json:"expires_in,omitempty"`
}

type RevokeHECTokenRequest struct {
	Token string `json:"token"`
}
