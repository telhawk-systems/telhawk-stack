package models

import "time"

type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Roles        []string   `json:"roles"`
	CreatedAt    time.Time  `json:"created_at"`
	DisabledAt   *time.Time `json:"disabled_at,omitempty"`
	DisabledBy   *string    `json:"disabled_by,omitempty"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	DeletedBy    *string    `json:"deleted_by,omitempty"`
}

// IsActive returns true if user is not disabled or deleted
func (u *User) IsActive() bool {
	return u.DisabledAt == nil && u.DeletedAt == nil
}

type HECToken struct {
	ID         string     `json:"id"`
	Token      string     `json:"token"`
	Name       string     `json:"name"`
	UserID     string     `json:"user_id"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	DisabledAt *time.Time `json:"disabled_at,omitempty"`
	DisabledBy *string    `json:"disabled_by,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	RevokedBy  *string    `json:"revoked_by,omitempty"`
}

// IsActive returns true if token is not disabled, revoked, or expired
func (t *HECToken) IsActive() bool {
	if t.DisabledAt != nil || t.RevokedAt != nil {
		return false
	}
	if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

// HECTokenResponse is the API response format that includes the computed enabled field
type HECTokenResponse struct {
	ID        string     `json:"id"`
	Token     string     `json:"token"`
	Name      string     `json:"name"`
	UserID    string     `json:"user_id"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// ToResponse converts a HECToken to an API response format with full token (only use at creation)
func (t *HECToken) ToResponse() *HECTokenResponse {
	return &HECTokenResponse{
		ID:        t.ID,
		Token:     t.Token,
		Name:      t.Name,
		UserID:    t.UserID,
		Enabled:   t.IsActive(),
		CreatedAt: t.CreatedAt,
		ExpiresAt: t.ExpiresAt,
	}
}

// MaskToken masks a token showing only first and last 8 characters
func MaskToken(token string) string {
	if len(token) <= 16 {
		return token // Don't mask very short tokens
	}
	return token[:8] + "..." + token[len(token)-8:]
}

// ToMaskedResponse converts a HECToken to an API response format with masked token
func (t *HECToken) ToMaskedResponse() *HECTokenResponse {
	return &HECTokenResponse{
		ID:        t.ID,
		Token:     MaskToken(t.Token),
		Name:      t.Name,
		UserID:    t.UserID,
		Enabled:   t.IsActive(),
		CreatedAt: t.CreatedAt,
		ExpiresAt: t.ExpiresAt,
	}
}

type Session struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	RevokedBy    *string    `json:"revoked_by,omitempty"`
}

// IsActive returns true if session is not revoked or expired
func (s *Session) IsActive() bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(time.Now())
}

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleAnalyst  Role = "analyst"
	RoleViewer   Role = "viewer"
	RoleIngester Role = "ingester"
)
