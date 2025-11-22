package models

import (
	"strings"
	"time"
)

// TenantType represents the tier of a tenant in the hierarchy
type TenantType string

const (
	TenantTypePlatform     TenantType = "platform"
	TenantTypeOrganization TenantType = "organization"
	TenantTypeClient       TenantType = "client"
)

// Tenant represents a multi-tenant entity (platform, organization, or client)
// Uses immutable versioning: ID (UUIDv7) = created_at, VersionID (UUIDv7) = updated_at
type Tenant struct {
	ID        string     `json:"id"`         // Stable entity ID (UUIDv7 timestamp = created_at)
	VersionID string     `json:"version_id"` // Row version ID (UUIDv7 timestamp = updated_at)
	Type      TenantType `json:"type"`
	ParentID  *string    `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Settings  string     `json:"settings"` // JSON string

	// Audit (version_id timestamp = when, these fields = who)
	CreatedBy *string `json:"created_by,omitempty"` // NULL for platform (bootstrap)
	UpdatedBy *string `json:"updated_by,omitempty"` // Who created this version

	// Lifecycle (immutable pattern)
	DisabledAt *time.Time `json:"disabled_at,omitempty"`
	DisabledBy *string    `json:"disabled_by,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
	DeletedBy  *string    `json:"deleted_by,omitempty"`
}

// IsActive returns true if tenant is not disabled or deleted
func (t *Tenant) IsActive() bool {
	return t.DisabledAt == nil && t.DeletedAt == nil
}

// TenantResponse is the API response format for tenants
type TenantResponse struct {
	ID        string     `json:"id"`
	VersionID string     `json:"version_id"`
	Type      TenantType `json:"type"`
	ParentID  *string    `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Enabled   bool       `json:"enabled"`
}

// ToResponse converts a Tenant to an API response format
func (t *Tenant) ToResponse() *TenantResponse {
	return &TenantResponse{
		ID:        t.ID,
		VersionID: t.VersionID,
		Type:      t.Type,
		ParentID:  t.ParentID,
		Name:      t.Name,
		Slug:      t.Slug,
		Enabled:   t.IsActive(),
	}
}

// Role represents a role definition with ordinal-based power hierarchy
// Uses immutable versioning: ID (UUIDv7) = created_at, VersionID (UUIDv7) = updated_at
// Tier is derived from OrganizationID/ClientID:
//   - NULL/NULL = platform
//   - org/NULL = organization
//   - org/client = client
type Role struct {
	ID             string  `json:"id"`                        // Stable entity ID (UUIDv7 timestamp = created_at)
	VersionID      string  `json:"version_id"`                // Row version ID (UUIDv7 timestamp = updated_at)
	OrganizationID *string `json:"organization_id,omitempty"` // Owning org (NULL = platform/template)
	ClientID       *string `json:"client_id,omitempty"`       // Specific client (NULL = org-level or platform)
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	Ordinal        int     `json:"ordinal"`
	Description    *string `json:"description,omitempty"`
	IsSystem       bool    `json:"is_system"`
	IsProtected    bool    `json:"is_protected"`
	IsTemplate     bool    `json:"is_template"` // Template roles copied on tenant creation

	// Audit (version_id timestamp = when, these fields = who)
	CreatedBy *string `json:"created_by,omitempty"` // NULL for seed data
	UpdatedBy *string `json:"updated_by,omitempty"` // Who created this version

	// Lifecycle
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	DeletedBy *string    `json:"deleted_by,omitempty"`

	// Loaded via join
	Permissions []Permission `json:"permissions,omitempty"`
}

// IsActive returns true if role is not deleted
func (r *Role) IsActive() bool {
	return r.DeletedAt == nil
}

// Tier returns the tenant tier this role belongs to, derived from org/client IDs
func (r *Role) Tier() TenantType {
	if r.ClientID != nil {
		return TenantTypeClient
	}
	if r.OrganizationID != nil {
		return TenantTypeOrganization
	}
	return TenantTypePlatform
}

// IsPlatformRole returns true if this is a platform-level role
func (r *Role) IsPlatformRole() bool {
	return r.OrganizationID == nil && r.ClientID == nil && !r.IsTemplate
}

// IsOrgRole returns true if this is an organization-level role
func (r *Role) IsOrgRole() bool {
	return r.OrganizationID != nil && r.ClientID == nil
}

// IsClientRole returns true if this is a client-level role
func (r *Role) IsClientRole() bool {
	return r.ClientID != nil
}

// HasPermission checks if this role has a specific permission
func (r *Role) HasPermission(resource, action string) bool {
	for _, p := range r.Permissions {
		if p.Resource == resource && p.Action == action {
			return true
		}
	}
	return false
}

// HasPermissionString checks if this role has a permission in "resource:action" format
func (r *Role) HasPermissionString(permission string) bool {
	parts := strings.SplitN(permission, ":", 2)
	if len(parts) != 2 {
		return false
	}
	return r.HasPermission(parts[0], parts[1])
}

// RoleResponse is the API response format for roles
type RoleResponse struct {
	ID             string     `json:"id"`
	VersionID      string     `json:"version_id"`
	Tier           TenantType `json:"tier"` // Derived from organization_id/client_id
	OrganizationID *string    `json:"organization_id,omitempty"`
	ClientID       *string    `json:"client_id,omitempty"`
	Name           string     `json:"name"`
	Slug           string     `json:"slug"`
	Ordinal        int        `json:"ordinal"`
	Description    *string    `json:"description,omitempty"`
	IsSystem       bool       `json:"is_system"`
	IsProtected    bool       `json:"is_protected"`
	IsTemplate     bool       `json:"is_template"`
}

// ToResponse converts a Role to an API response format
func (r *Role) ToResponse() *RoleResponse {
	return &RoleResponse{
		ID:             r.ID,
		VersionID:      r.VersionID,
		Tier:           r.Tier(),
		OrganizationID: r.OrganizationID,
		ClientID:       r.ClientID,
		Name:           r.Name,
		Slug:           r.Slug,
		Ordinal:        r.Ordinal,
		Description:    r.Description,
		IsSystem:       r.IsSystem,
		IsProtected:    r.IsProtected,
		IsTemplate:     r.IsTemplate,
	}
}

// Permission represents a resource:action permission
// Uses ID (UUIDv7) for created_at timestamp (static seed data, no versioning)
type Permission struct {
	ID          string  `json:"id"` // UUIDv7 timestamp = created_at
	Resource    string  `json:"resource"`
	Action      string  `json:"action"`
	Description *string `json:"description,omitempty"`
}

// String returns the permission in "resource:action" format
func (p *Permission) String() string {
	return p.Resource + ":" + p.Action
}

// PermissionResponse is the API response format for permissions
type PermissionResponse struct {
	ID          string  `json:"id"`
	Resource    string  `json:"resource"`
	Action      string  `json:"action"`
	Description *string `json:"description,omitempty"`
}

// ToResponse converts a Permission to an API response format
func (p *Permission) ToResponse() *PermissionResponse {
	return &PermissionResponse{
		ID:          p.ID,
		Resource:    p.Resource,
		Action:      p.Action,
		Description: p.Description,
	}
}

// UserRole represents a user's role assignment within a tenant
// Uses ID (UUIDv7) for created_at timestamp (append-only with revocation)
type UserRole struct {
	ID                   string   `json:"id"` // UUIDv7 timestamp = created_at (granted_at)
	UserID               string   `json:"user_id"`
	RoleID               string   `json:"role_id"`
	TenantID             string   `json:"tenant_id"`
	ScopeOrganizationIDs []string `json:"scope_organization_ids,omitempty"`
	ScopeClientIDs       []string `json:"scope_client_ids,omitempty"`
	GrantedBy            *string  `json:"granted_by,omitempty"`

	// Lifecycle (revocation only)
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	RevokedBy *string    `json:"revoked_by,omitempty"`

	// Loaded via join
	Role   *Role   `json:"role,omitempty"`
	Tenant *Tenant `json:"tenant,omitempty"`
}

// IsActive returns true if user role assignment is not revoked
func (ur *UserRole) IsActive() bool {
	return ur.RevokedAt == nil
}

// UserRoleResponse is the API response format for user role assignments
type UserRoleResponse struct {
	ID       string        `json:"id"` // UUIDv7 timestamp = granted_at
	UserID   string        `json:"user_id"`
	RoleID   string        `json:"role_id"`
	TenantID string        `json:"tenant_id"`
	Role     *RoleResponse `json:"role,omitempty"`
}

// ToResponse converts a UserRole to an API response format
func (ur *UserRole) ToResponse() *UserRoleResponse {
	resp := &UserRoleResponse{
		ID:       ur.ID,
		UserID:   ur.UserID,
		RoleID:   ur.RoleID,
		TenantID: ur.TenantID,
	}
	if ur.Role != nil {
		resp.Role = ur.Role.ToResponse()
	}
	return resp
}

// Well-known role IDs (from seed data)
const (
	// Platform roles (actual roles, not templates)
	RoleIDPlatformRoot    = "00000000-0000-0000-0001-000000000001"
	RoleIDPlatformOwner   = "00000000-0000-0000-0001-000000000010"
	RoleIDPlatformAdmin   = "00000000-0000-0000-0001-000000000020"
	RoleIDPlatformAnalyst = "00000000-0000-0000-0001-000000000030"

	// Organization role templates (is_template=true, copied on org creation)
	RoleTemplateOrgOwner   = "00000000-0000-0000-0002-000000000010"
	RoleTemplateOrgAdmin   = "00000000-0000-0000-0002-000000000020"
	RoleTemplateOrgAnalyst = "00000000-0000-0000-0002-000000000030"

	// Client role templates (is_template=true, copied on client creation)
	RoleTemplateClientOwner   = "00000000-0000-0000-0003-000000000010"
	RoleTemplateClientAdmin   = "00000000-0000-0000-0003-000000000020"
	RoleTemplateClientAnalyst = "00000000-0000-0000-0003-000000000030"

	// Default Organization roles (actual roles for default org)
	RoleIDDefaultOrgOwner   = "00000000-0000-0000-0010-000000000010"
	RoleIDDefaultOrgAdmin   = "00000000-0000-0000-0010-000000000020"
	RoleIDDefaultOrgAnalyst = "00000000-0000-0000-0010-000000000030"

	// Default Client roles (actual roles for default client)
	RoleIDDefaultClientOwner   = "00000000-0000-0000-0011-000000000010"
	RoleIDDefaultClientAdmin   = "00000000-0000-0000-0011-000000000020"
	RoleIDDefaultClientAnalyst = "00000000-0000-0000-0011-000000000030"
)

// Well-known tenant IDs
const (
	TenantIDPlatform      = "00000000-0000-0000-0000-000000000001"
	TenantIDDefaultOrg    = "00000000-0000-0000-0000-000000000010"
	TenantIDDefaultClient = "00000000-0000-0000-0000-000000000011"
)

// Well-known user IDs
const (
	UserIDRoot = "00000000-0000-0000-0000-000000000002"
)

// Protected role slugs that cannot be created or modified
var ProtectedRoleSlugs = []string{"root", "admin"}

// IsProtectedSlug checks if a role slug is protected
func IsProtectedSlug(slug string) bool {
	for _, protected := range ProtectedRoleSlugs {
		if slug == protected {
			return true
		}
	}
	return false
}
