package models

import "time"

// User represents a user account
// Uses immutable versioning: ID (UUIDv7) = created_at, VersionID (UUIDv7) = updated_at
type User struct {
	ID           string   `json:"id"`         // Stable entity ID (UUIDv7 timestamp = created_at)
	VersionID    string   `json:"version_id"` // Row version ID (UUIDv7 timestamp = updated_at)
	Username     string   `json:"username"`
	Email        string   `json:"email"`
	PasswordHash string   `json:"-"`
	Roles        []string `json:"roles"` // Legacy: simple role strings

	// Primary scope (determines default data visibility)
	// Scope: client_id NOT NULL → client, organization_id NOT NULL → org, both NULL → platform
	PrimaryOrganizationID *string `json:"primary_organization_id,omitempty"`
	PrimaryClientID       *string `json:"primary_client_id,omitempty"`

	// Audit (version_id timestamp = when, these fields = who)
	CreatedBy *string `json:"created_by,omitempty"` // NULL for root user (bootstrap)
	UpdatedBy *string `json:"updated_by,omitempty"` // Who created this version

	// Lifecycle (immutable pattern - no explicit timestamps, use UUIDv7)
	DisabledAt *time.Time `json:"disabled_at,omitempty"`
	DisabledBy *string    `json:"disabled_by,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
	DeletedBy  *string    `json:"deleted_by,omitempty"`

	// Permission version - incremented when roles/permissions change
	// Used for JWT validation: if JWT version != DB version, reload permissions
	PermissionsVersion int `json:"permissions_version"`

	// Loaded via join (not stored directly in users table)
	UserRoles []*UserRole `json:"user_roles,omitempty"`
}

// IsActive returns true if user is not disabled or deleted
func (u *User) IsActive() bool {
	return u.DisabledAt == nil && u.DeletedAt == nil
}

// GetScopeTier returns the scope tier of user's primary scope
// Determined by: client_id NOT NULL → client, organization_id NOT NULL → org, both NULL → platform
func (u *User) GetScopeTier() ScopeTier {
	if u.PrimaryClientID != nil {
		return ScopeTierClient
	}
	if u.PrimaryOrganizationID != nil {
		return ScopeTierOrganization
	}
	return ScopeTierPlatform
}

// UserResponse is the API response format that includes the computed enabled field
type UserResponse struct {
	ID        string   `json:"id"`
	VersionID string   `json:"version_id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	Enabled   bool     `json:"enabled"`
}

// ToResponse converts a User to an API response format
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		VersionID: u.VersionID,
		Username:  u.Username,
		Email:     u.Email,
		Roles:     u.Roles,
		Enabled:   u.IsActive(),
	}
}

// HECToken represents an HEC ingestion token
// Uses ID (UUIDv7) for created_at timestamp
type HECToken struct {
	ID        string `json:"id"` // UUIDv7 timestamp = created_at
	Token     string `json:"token"`
	Name      string `json:"name"`
	UserID    string `json:"user_id"`    // Token owner (who can use it)
	ClientID  string `json:"client_id"`  // Client for data isolation
	CreatedBy string `json:"created_by"` // Who created this token (audit)

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
	ID        string     `json:"id"` // UUIDv7 timestamp = created_at
	Token     string     `json:"token"`
	Name      string     `json:"name"`
	UserID    string     `json:"user_id"`
	ClientID  string     `json:"client_id"`
	Username  string     `json:"username,omitempty"` // Only included for admin users
	Enabled   bool       `json:"enabled"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// ToResponse converts a HECToken to an API response format with full token (only use at creation)
func (t *HECToken) ToResponse() *HECTokenResponse {
	return &HECTokenResponse{
		ID:        t.ID,
		Token:     t.Token,
		Name:      t.Name,
		UserID:    t.UserID,
		ClientID:  t.ClientID,
		Enabled:   t.IsActive(),
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
		ExpiresAt: t.ExpiresAt,
	}
}

// ToMaskedResponseWithUsername converts a HECToken to an API response format with masked token and username
func (t *HECToken) ToMaskedResponseWithUsername(username string) *HECTokenResponse {
	return &HECTokenResponse{
		ID:        t.ID,
		Token:     MaskToken(t.Token),
		Name:      t.Name,
		UserID:    t.UserID,
		Username:  username,
		Enabled:   t.IsActive(),
		ExpiresAt: t.ExpiresAt,
	}
}

// Session represents an authentication session
// Uses ID (UUIDv7) for created_at timestamp
type Session struct {
	ID           string     `json:"id"` // UUIDv7 timestamp = created_at
	UserID       string     `json:"user_id"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresAt    time.Time  `json:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	RevokedBy    *string    `json:"revoked_by,omitempty"`
}

// IsActive returns true if session is not revoked or expired
func (s *Session) IsActive() bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(time.Now())
}

// LegacyRole is the old simple role type (kept for backward compatibility)
type LegacyRole string

const (
	LegacyRoleAdmin    LegacyRole = "admin"
	LegacyRoleAnalyst  LegacyRole = "analyst"
	LegacyRoleViewer   LegacyRole = "viewer"
	LegacyRoleIngester LegacyRole = "ingester"
)

// =============================================================================
// RBAC Permission Checking Methods
// =============================================================================

// Can checks if user has a specific permission in "resource:action" format
func (u *User) Can(permission string) bool {
	// Protected role (ordinal 0) has all permissions
	if u.HasProtectedRole() {
		return true
	}

	// Check if any of user's roles grant this permission
	for _, ur := range u.UserRoles {
		if ur.Role != nil && ur.IsActive() && ur.Role.HasPermissionString(permission) {
			return true
		}
	}

	return false
}

// HasProtectedRole returns true if user has any protected role (ordinal 0)
func (u *User) HasProtectedRole() bool {
	for _, ur := range u.UserRoles {
		if ur.Role != nil && ur.IsActive() && ur.Role.IsProtected && ur.Role.Ordinal == 0 {
			return true
		}
	}
	return false
}

// LowestOrdinal returns the lowest (most powerful) ordinal among user's active roles
// Returns 100 if user has no roles (less powerful than any valid role)
func (u *User) LowestOrdinal() int {
	lowest := 100
	for _, ur := range u.UserRoles {
		if ur.Role != nil && ur.IsActive() && ur.Role.Ordinal < lowest {
			lowest = ur.Role.Ordinal
		}
	}
	return lowest
}

// HighestOrdinal returns the highest (least powerful) ordinal among user's active roles
// Returns -1 if user has no roles
func (u *User) HighestOrdinal() int {
	highest := -1
	for _, ur := range u.UserRoles {
		if ur.Role != nil && ur.IsActive() && ur.Role.Ordinal > highest {
			highest = ur.Role.Ordinal
		}
	}
	return highest
}

// CanManageUser checks if this user can manage (edit/delete) the target user
// Rules:
// 1. Cannot manage protected users (ordinal 0)
// 2. Must have users:update permission
// 3. Must be in same tier tree (platform can manage all, org can manage org+clients, etc.)
// 4. Can only manage users with ordinal >= own ordinal (same or higher number = same or less powerful)
func (u *User) CanManageUser(target *User) bool {
	// Cannot manage protected users
	if target.HasProtectedRole() {
		return false
	}

	// Must have users:update permission
	if !u.Can("users:update") {
		return false
	}

	// Check ordinal (can manage same or higher ordinal only)
	// Using lowest ordinal for both: if actor's most powerful role can manage target's most powerful role
	if u.LowestOrdinal() > target.LowestOrdinal() {
		return false
	}

	// Tier tree check would need hierarchy info - for now, allow if permissions pass
	// Full implementation requires loading org/client tree and checking scope

	return true
}

// CanResetPassword checks if this user can reset the target user's password
// Same rules as CanManageUser
func (u *User) CanResetPassword(target *User) bool {
	// Cannot reset password for protected users
	if target.HasProtectedRole() {
		return false
	}

	// Must have users:reset_password permission
	if !u.Can("users:reset_password") {
		return false
	}

	// Check ordinal
	if u.LowestOrdinal() > target.LowestOrdinal() {
		return false
	}

	return true
}

// CanAssignRole checks if this user can assign a specific role to another user
// Rules:
// 1. Cannot assign protected roles
// 2. Cannot assign roles with lower ordinal than own
// 3. Must have users:assign_roles permission
func (u *User) CanAssignRole(role *Role) bool {
	// Cannot assign protected roles
	if role.IsProtected {
		return false
	}

	// Must have users:assign_roles permission
	if !u.Can("users:assign_roles") {
		return false
	}

	// Cannot assign roles more powerful than own
	if role.Ordinal < u.LowestOrdinal() {
		return false
	}

	return true
}

// GetActiveRoles returns all active (non-revoked) user roles
func (u *User) GetActiveRoles() []*UserRole {
	var active []*UserRole
	for _, ur := range u.UserRoles {
		if ur.IsActive() {
			active = append(active, ur)
		}
	}
	return active
}

// GetPermissions returns all unique permissions from user's active roles
func (u *User) GetPermissions() []string {
	seen := make(map[string]bool)
	var permissions []string

	for _, ur := range u.UserRoles {
		if ur.Role != nil && ur.IsActive() {
			for _, p := range ur.Role.Permissions {
				key := p.String()
				if !seen[key] {
					seen[key] = true
					permissions = append(permissions, key)
				}
			}
		}
	}

	return permissions
}
