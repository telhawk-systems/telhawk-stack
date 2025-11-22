-- TelHawk Auth Database Schema
-- PostgreSQL initialization script
--
-- Immutable Versioning Pattern:
-- - id: Stable entity identifier (UUIDv7 - timestamp encodes created_at)
-- - version_id: Row version identifier (UUIDv7 - timestamp encodes updated_at)
-- - Content changes create new rows with same id but new version_id
-- - Lifecycle changes (disable/delete) UPDATE existing rows (no new version)

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";  -- For gen_random_uuid() fallback

-- ============================================================================
-- TENANTS (Multi-tenant hierarchy: Platform -> Organization -> Client)
-- ============================================================================

CREATE TABLE IF NOT EXISTS tenants (
    -- Identity (UUIDv7: timestamp = created_at)
    id UUID NOT NULL,
    -- Version (UUIDv7: timestamp = updated_at, PRIMARY KEY)
    version_id UUID PRIMARY KEY,

    -- Tenant hierarchy
    type VARCHAR(20) NOT NULL,
    parent_id UUID,  -- References tenants(id), NULL only for platform
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    settings JSONB NOT NULL DEFAULT '{}',

    -- Audit (version_id UUIDv7 timestamp = when, these fields = who)
    created_by UUID,  -- References users(id), NULL only for platform (bootstrap)
    updated_by UUID,  -- References users(id), who created this version

    -- Lifecycle (immutable pattern)
    disabled_at TIMESTAMPTZ,
    disabled_by UUID,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID,

    -- Constraints
    CONSTRAINT valid_tenant_type CHECK (type IN ('platform', 'organization', 'client')),
    CONSTRAINT platform_no_parent CHECK (
        (type = 'platform' AND parent_id IS NULL) OR
        (type != 'platform' AND parent_id IS NOT NULL)
    ),
    CONSTRAINT platform_bootstrap CHECK (
        (type = 'platform' AND created_by IS NULL) OR
        (type != 'platform' AND created_by IS NOT NULL)
    )
);

-- Get latest version of each tenant
CREATE INDEX idx_tenants_id ON tenants(id);
CREATE INDEX idx_tenants_latest ON tenants(id, version_id DESC);

-- Ensure only one platform tenant exists (among non-deleted)
CREATE UNIQUE INDEX one_platform_tenant ON tenants ((true))
    WHERE type = 'platform' AND deleted_at IS NULL;

-- Unique slug per tenant type (active only)
CREATE UNIQUE INDEX idx_tenants_type_slug ON tenants(type, slug)
    WHERE deleted_at IS NULL;

-- Parent lookups
CREATE INDEX idx_tenants_parent ON tenants(parent_id)
    WHERE deleted_at IS NULL;

-- Active tenants by type
CREATE INDEX idx_tenants_type_active ON tenants(type)
    WHERE disabled_at IS NULL AND deleted_at IS NULL;

COMMENT ON TABLE tenants IS 'Multi-tenant hierarchy: Platform (singleton) -> Organization -> Client';
COMMENT ON COLUMN tenants.id IS 'Stable entity ID (UUIDv7 timestamp = created_at)';
COMMENT ON COLUMN tenants.version_id IS 'Row version ID (UUIDv7 timestamp = updated_at)';
COMMENT ON COLUMN tenants.created_by IS 'User who created this tenant (NULL for platform bootstrap)';
COMMENT ON COLUMN tenants.updated_by IS 'User who created this version row';

-- ============================================================================
-- USERS
-- ============================================================================

CREATE TABLE IF NOT EXISTS users (
    -- Identity (UUIDv7: timestamp = created_at)
    id UUID NOT NULL,
    -- Version (UUIDv7: timestamp = updated_at, PRIMARY KEY)
    version_id UUID PRIMARY KEY,

    -- User data
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    roles TEXT[] NOT NULL DEFAULT '{}',  -- Legacy roles (kept for backward compat)
    primary_tenant_id UUID,  -- References tenants(id)

    -- Audit (version_id UUIDv7 timestamp = when, these fields = who)
    created_by UUID,  -- References users(id), NULL for root user (bootstrap)
    updated_by UUID,  -- References users(id), who created this version

    -- Lifecycle (immutable pattern)
    disabled_at TIMESTAMPTZ,
    disabled_by UUID,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID,

    -- Self-action constraints (compare against id, not version_id)
    CONSTRAINT users_not_self_disable CHECK (id != disabled_by),
    CONSTRAINT users_not_self_delete CHECK (id != deleted_by)
);

-- Get latest version of each user
CREATE INDEX idx_users_id ON users(id);
CREATE INDEX idx_users_latest ON users(id, version_id DESC);

-- Unique constraints (on latest active version only - enforced via application)
CREATE UNIQUE INDEX idx_users_username ON users(username)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_users_email ON users(email)
    WHERE deleted_at IS NULL;

-- Active users
CREATE INDEX idx_users_active ON users(username)
    WHERE disabled_at IS NULL AND deleted_at IS NULL;

-- Users by tenant
CREATE INDEX idx_users_tenant ON users(primary_tenant_id)
    WHERE deleted_at IS NULL;

COMMENT ON TABLE users IS 'User accounts for TelHawk system';
COMMENT ON COLUMN users.id IS 'Stable entity ID (UUIDv7 timestamp = created_at)';
COMMENT ON COLUMN users.version_id IS 'Row version ID (UUIDv7 timestamp = updated_at)';
COMMENT ON COLUMN users.roles IS 'Legacy role array (use user_roles table for RBAC)';
COMMENT ON COLUMN users.primary_tenant_id IS 'User home tenant - determines base tier';
COMMENT ON COLUMN users.created_by IS 'User who created this account (NULL for root bootstrap)';
COMMENT ON COLUMN users.updated_by IS 'User who created this version row';

-- ============================================================================
-- SESSIONS (Append-only, no versioning needed)
-- ============================================================================

CREATE TABLE IF NOT EXISTS sessions (
    -- Identity (UUIDv7: timestamp = created_at)
    id UUID PRIMARY KEY,

    -- Session data
    user_id UUID NOT NULL,  -- References users(id)
    access_token TEXT NOT NULL,
    refresh_token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,

    -- Lifecycle (revocation only - no edits)
    revoked_at TIMESTAMPTZ,
    revoked_by UUID  -- References users(id)
);

CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_sessions_active ON sessions(id)
    WHERE revoked_at IS NULL;

COMMENT ON TABLE sessions IS 'Active authentication sessions (append-only)';
COMMENT ON COLUMN sessions.id IS 'Session ID (UUIDv7 timestamp = created_at)';

-- ============================================================================
-- HEC TOKENS (Append-only, no versioning needed)
-- ============================================================================

CREATE TABLE IF NOT EXISTS hec_tokens (
    -- Identity (UUIDv7: timestamp = created_at)
    id UUID PRIMARY KEY,

    -- Token data
    token VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,  -- References users(id) - token owner
    client_id UUID NOT NULL,  -- References tenants(id) - client for data isolation
    expires_at TIMESTAMPTZ,

    -- Audit
    created_by UUID NOT NULL,  -- References users(id) - who created (may differ from owner)

    -- Lifecycle
    disabled_at TIMESTAMPTZ,
    disabled_by UUID,
    revoked_at TIMESTAMPTZ,
    revoked_by UUID
);

CREATE INDEX idx_hec_tokens_token ON hec_tokens(token);
CREATE INDEX idx_hec_tokens_user_id ON hec_tokens(user_id);
CREATE INDEX idx_hec_tokens_client_id ON hec_tokens(client_id);
CREATE INDEX idx_hec_tokens_active ON hec_tokens(token)
    WHERE disabled_at IS NULL AND revoked_at IS NULL;

COMMENT ON TABLE hec_tokens IS 'HEC (HTTP Event Collector) tokens for data ingestion';
COMMENT ON COLUMN hec_tokens.id IS 'Token ID (UUIDv7 timestamp = created_at)';
COMMENT ON COLUMN hec_tokens.user_id IS 'Token owner (user who can use this token)';
COMMENT ON COLUMN hec_tokens.client_id IS 'Client - events ingested with this token belong to this client';
COMMENT ON COLUMN hec_tokens.created_by IS 'User who created this token (may differ from owner)';

-- ============================================================================
-- ROLES (Versioned)
-- ============================================================================

CREATE TABLE IF NOT EXISTS roles (
    -- Identity (UUIDv7: timestamp = created_at)
    id UUID NOT NULL,
    -- Version (UUIDv7: timestamp = updated_at, PRIMARY KEY)
    version_id UUID PRIMARY KEY,

    -- Role scope (tier derived: NULL/NULL=platform, org/NULL=org, org/client=client)
    organization_id UUID,  -- Which org owns this role (NULL for platform/templates)
    client_id UUID,        -- Which client this role is for (NULL for org-level roles)

    -- Role data
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL,
    ordinal SMALLINT NOT NULL DEFAULT 50,  -- 0-99, lower = more powerful
    description TEXT,

    -- System flags
    is_system BOOLEAN NOT NULL DEFAULT FALSE,    -- Cannot delete
    is_protected BOOLEAN NOT NULL DEFAULT FALSE, -- Ordinal 0, immutable
    is_template BOOLEAN NOT NULL DEFAULT FALSE,  -- Global template role

    -- Audit (version_id UUIDv7 timestamp = when, these fields = who)
    created_by UUID,  -- References users(id), NULL for seed data
    updated_by UUID,  -- References users(id), who created this version

    -- Lifecycle
    deleted_at TIMESTAMPTZ,
    deleted_by UUID,

    -- Constraints
    CONSTRAINT valid_ordinal CHECK (ordinal >= 0 AND ordinal <= 99),
    CONSTRAINT protected_requires_ordinal_zero CHECK (
        (is_protected = TRUE AND ordinal = 0) OR (is_protected = FALSE)
    ),
    CONSTRAINT protected_slug_restriction CHECK (
        (is_protected = TRUE AND slug IN ('root', 'admin')) OR
        (is_protected = FALSE AND slug NOT IN ('root', 'admin'))
    ),
    -- Templates must have NULL for both org and client
    CONSTRAINT templates_are_global CHECK (
        (is_template = TRUE AND organization_id IS NULL AND client_id IS NULL) OR
        (is_template = FALSE)
    ),
    -- Client requires org (can't have client without org)
    CONSTRAINT client_requires_org CHECK (
        (client_id IS NOT NULL AND organization_id IS NOT NULL) OR
        (client_id IS NULL)
    )
);

-- Get latest version of each role
CREATE INDEX idx_roles_id ON roles(id);
CREATE INDEX idx_roles_latest ON roles(id, version_id DESC);

-- Unique slug within scope (active only)
-- Platform/template roles: globally unique slug
CREATE UNIQUE INDEX idx_roles_platform_slug ON roles(slug)
    WHERE deleted_at IS NULL AND organization_id IS NULL AND client_id IS NULL;
-- Org roles: unique slug per organization
CREATE UNIQUE INDEX idx_roles_org_slug ON roles(organization_id, slug)
    WHERE deleted_at IS NULL AND organization_id IS NOT NULL AND client_id IS NULL;
-- Client roles: unique slug per client
CREATE UNIQUE INDEX idx_roles_client_slug ON roles(organization_id, client_id, slug)
    WHERE deleted_at IS NULL AND client_id IS NOT NULL;

-- Roles by organization
CREATE INDEX idx_roles_org ON roles(organization_id)
    WHERE deleted_at IS NULL AND organization_id IS NOT NULL;

-- Roles by client
CREATE INDEX idx_roles_client ON roles(client_id)
    WHERE deleted_at IS NULL AND client_id IS NOT NULL;

-- Template roles
CREATE INDEX idx_roles_templates ON roles(is_template)
    WHERE deleted_at IS NULL AND is_template = TRUE;

-- Roles by ordinal (for hierarchy queries)
CREATE INDEX idx_roles_ordinal ON roles(ordinal)
    WHERE deleted_at IS NULL;

COMMENT ON TABLE roles IS 'Role definitions with ordinal-based power hierarchy';
COMMENT ON COLUMN roles.id IS 'Stable entity ID (UUIDv7 timestamp = created_at)';
COMMENT ON COLUMN roles.version_id IS 'Row version ID (UUIDv7 timestamp = updated_at)';
COMMENT ON COLUMN roles.organization_id IS 'Owning org (NULL = platform/template)';
COMMENT ON COLUMN roles.client_id IS 'Specific client (NULL = org-level or platform)';
COMMENT ON COLUMN roles.ordinal IS 'Power level 0-99 (lower = more powerful)';
COMMENT ON COLUMN roles.is_template IS 'TRUE for template roles copied on tenant creation';
COMMENT ON COLUMN roles.created_by IS 'User who created this role (NULL for seed data)';
COMMENT ON COLUMN roles.updated_by IS 'User who created this version row';

-- ============================================================================
-- PERMISSIONS (Static, no versioning - seed data only)
-- ============================================================================

CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY,  -- UUIDv7: timestamp = created_at
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,

    CONSTRAINT unique_permission UNIQUE (resource, action)
);

COMMENT ON TABLE permissions IS 'Permission definitions (resource:action format)';

-- ============================================================================
-- ROLE_PERMISSIONS (Junction table)
-- ============================================================================

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL,  -- References roles(id)
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,  -- Locked for protected roles

    -- Audit
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID,  -- References users(id)

    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission ON role_permissions(permission_id);

COMMENT ON TABLE role_permissions IS 'Maps roles to their granted permissions';
COMMENT ON COLUMN role_permissions.is_locked IS 'When true, cannot be removed (protected roles)';

-- ============================================================================
-- USER_ROLES (Append-only with revocation)
-- ============================================================================

CREATE TABLE IF NOT EXISTS user_roles (
    -- Identity (UUIDv7: timestamp = created_at)
    id UUID PRIMARY KEY,

    -- Assignment
    user_id UUID NOT NULL,    -- References users(id)
    role_id UUID NOT NULL,    -- References roles(id)
    tenant_id UUID NOT NULL,  -- References tenants(id)

    -- Scope restrictions (NULL = unrestricted within tenant)
    scope_organization_ids UUID[],  -- For platform users
    scope_client_ids UUID[],        -- For org users

    -- Audit
    granted_by UUID,  -- References users(id)
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Lifecycle (revocation only)
    revoked_at TIMESTAMPTZ,
    revoked_by UUID  -- References users(id)
);

-- Unique active assignment per user/role/tenant
CREATE UNIQUE INDEX idx_user_roles_unique_active ON user_roles(user_id, role_id, tenant_id)
    WHERE revoked_at IS NULL;

-- User's roles
CREATE INDEX idx_user_roles_user ON user_roles(user_id)
    WHERE revoked_at IS NULL;

-- Users with a role
CREATE INDEX idx_user_roles_role ON user_roles(role_id)
    WHERE revoked_at IS NULL;

-- Users in a tenant
CREATE INDEX idx_user_roles_tenant ON user_roles(tenant_id)
    WHERE revoked_at IS NULL;

-- User-tenant combinations
CREATE INDEX idx_user_roles_user_tenant ON user_roles(user_id, tenant_id)
    WHERE revoked_at IS NULL;

COMMENT ON TABLE user_roles IS 'User role assignments within tenants';
COMMENT ON COLUMN user_roles.id IS 'Assignment ID (UUIDv7 timestamp = created_at)';
COMMENT ON COLUMN user_roles.scope_organization_ids IS 'Limit platform user to these orgs';
COMMENT ON COLUMN user_roles.scope_client_ids IS 'Limit org user to these clients';

-- ============================================================================
-- AUDIT LOG (Append-only)
-- ============================================================================

CREATE TABLE IF NOT EXISTS audit_log (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_type VARCHAR(50) NOT NULL,
    actor_id VARCHAR(255),
    actor_name VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,
    result VARCHAR(20) NOT NULL,
    error_message TEXT,
    metadata JSONB
);

CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp DESC);
CREATE INDEX idx_audit_log_actor_id ON audit_log(actor_id);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_result ON audit_log(result);
CREATE INDEX idx_audit_log_ip_address ON audit_log(ip_address);

COMMENT ON TABLE audit_log IS 'Audit trail of all authentication and authorization events';

-- ============================================================================
-- SEED DATA
-- ============================================================================

-- Platform tenant (singleton, bootstrap - created_by is NULL)
INSERT INTO tenants (id, version_id, type, parent_id, name, slug, settings, created_by, updated_by)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'platform',
    NULL,
    'TelHawk Platform',
    'platform',
    '{"description": "Root platform tenant"}'::jsonb,
    NULL,  -- bootstrap: platform creates itself
    NULL
) ON CONFLICT DO NOTHING;

-- Root/Admin user (password: admin123, bootstrap - created_by is NULL)
INSERT INTO users (id, version_id, username, email, password_hash, roles, primary_tenant_id, created_by, updated_by)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000002',
    'admin',
    'admin@telhawk.local',
    '$2a$10$UslOjaejNBYO2PTBjrahduCEA/RM4x3nj0HXEtIGnsTDcPHnEWOva',
    ARRAY['admin'],
    '00000000-0000-0000-0000-000000000001',
    NULL,  -- bootstrap: root creates itself
    NULL
) ON CONFLICT DO NOTHING;

-- Default Organization (created by root)
INSERT INTO tenants (id, version_id, type, parent_id, name, slug, settings, created_by, updated_by)
VALUES (
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000010',
    'organization',
    '00000000-0000-0000-0000-000000000001',  -- parent = platform
    'Default Organization',
    'default-org',
    '{"description": "Default organization for initial setup"}'::jsonb,
    '00000000-0000-0000-0000-000000000002',  -- created by root
    '00000000-0000-0000-0000-000000000002'
) ON CONFLICT DO NOTHING;

-- Default Client (created by root, under Default Organization)
INSERT INTO tenants (id, version_id, type, parent_id, name, slug, settings, created_by, updated_by)
VALUES (
    '00000000-0000-0000-0000-000000000011',
    '00000000-0000-0000-0000-000000000011',
    'client',
    '00000000-0000-0000-0000-000000000010',  -- parent = default org
    'Default Client',
    'default-client',
    '{"description": "Default client for initial setup"}'::jsonb,
    '00000000-0000-0000-0000-000000000002',  -- created by root
    '00000000-0000-0000-0000-000000000002'
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- PERMISSIONS (resource:action)
-- ============================================================================

-- Users & Authentication
INSERT INTO permissions (id, resource, action, description) VALUES
    ('00000000-0000-0001-0001-000000000001', 'users', 'create', 'Create new users'),
    ('00000000-0000-0001-0001-000000000002', 'users', 'read', 'View user profiles'),
    ('00000000-0000-0001-0001-000000000003', 'users', 'update', 'Modify user details'),
    ('00000000-0000-0001-0001-000000000004', 'users', 'delete', 'Delete/disable users'),
    ('00000000-0000-0001-0001-000000000005', 'users', 'assign_roles', 'Assign roles to users'),
    ('00000000-0000-0001-0001-000000000006', 'users', 'reset_password', 'Reset user passwords'),
    ('00000000-0000-0001-0002-000000000001', 'tokens', 'create', 'Create HEC/API tokens'),
    ('00000000-0000-0001-0002-000000000002', 'tokens', 'read', 'View token list'),
    ('00000000-0000-0001-0002-000000000003', 'tokens', 'revoke', 'Revoke tokens'),
    ('00000000-0000-0001-0002-000000000004', 'tokens', 'manage_all', 'Manage any user tokens')
ON CONFLICT (resource, action) DO NOTHING;

-- Detection & Response
INSERT INTO permissions (id, resource, action, description) VALUES
    ('00000000-0000-0001-0003-000000000001', 'rules', 'create', 'Create detection rules'),
    ('00000000-0000-0001-0003-000000000002', 'rules', 'read', 'View detection rules'),
    ('00000000-0000-0001-0003-000000000003', 'rules', 'update', 'Modify detection rules'),
    ('00000000-0000-0001-0003-000000000004', 'rules', 'delete', 'Delete detection rules'),
    ('00000000-0000-0001-0003-000000000005', 'rules', 'enable', 'Enable detection rules'),
    ('00000000-0000-0001-0003-000000000006', 'rules', 'disable', 'Disable detection rules'),
    ('00000000-0000-0001-0004-000000000001', 'alerts', 'read', 'View alerts'),
    ('00000000-0000-0001-0004-000000000002', 'alerts', 'acknowledge', 'Acknowledge alerts'),
    ('00000000-0000-0001-0004-000000000003', 'alerts', 'close', 'Close alerts'),
    ('00000000-0000-0001-0004-000000000004', 'alerts', 'assign', 'Assign alerts to users'),
    ('00000000-0000-0001-0004-000000000005', 'alerts', 'delete', 'Delete alerts'),
    ('00000000-0000-0001-0005-000000000001', 'cases', 'create', 'Create cases'),
    ('00000000-0000-0001-0005-000000000002', 'cases', 'read', 'View cases'),
    ('00000000-0000-0001-0005-000000000003', 'cases', 'update', 'Modify cases'),
    ('00000000-0000-0001-0005-000000000004', 'cases', 'close', 'Close cases'),
    ('00000000-0000-0001-0005-000000000005', 'cases', 'delete', 'Delete cases'),
    ('00000000-0000-0001-0005-000000000006', 'cases', 'assign', 'Assign cases to users')
ON CONFLICT (resource, action) DO NOTHING;

-- Search & Data
INSERT INTO permissions (id, resource, action, description) VALUES
    ('00000000-0000-0001-0006-000000000001', 'search', 'execute', 'Run searches'),
    ('00000000-0000-0001-0006-000000000002', 'search', 'export', 'Export search results'),
    ('00000000-0000-0001-0006-000000000003', 'search', 'save_queries', 'Save search queries'),
    ('00000000-0000-0001-0007-000000000001', 'events', 'read', 'View events')
ON CONFLICT (resource, action) DO NOTHING;

-- System Administration
INSERT INTO permissions (id, resource, action, description) VALUES
    ('00000000-0000-0001-0008-000000000001', 'system', 'configure', 'Modify system settings'),
    ('00000000-0000-0001-0008-000000000002', 'system', 'view_audit', 'View audit logs'),
    ('00000000-0000-0001-0008-000000000003', 'system', 'manage_integrations', 'Configure integrations'),
    ('00000000-0000-0001-0009-000000000001', 'tenants', 'create', 'Create organizations/clients'),
    ('00000000-0000-0001-0009-000000000002', 'tenants', 'read', 'View tenant information'),
    ('00000000-0000-0001-0009-000000000003', 'tenants', 'update', 'Modify tenant settings'),
    ('00000000-0000-0001-0009-000000000004', 'tenants', 'delete', 'Delete tenants')
ON CONFLICT (resource, action) DO NOTHING;

-- ============================================================================
-- ROLES
-- ============================================================================

-- Platform Roles (organization_id=NULL, client_id=NULL, is_template=FALSE)
INSERT INTO roles (id, version_id, organization_id, client_id, name, slug, ordinal, description, is_system, is_protected, is_template) VALUES
    ('00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0001-000000000001', NULL, NULL, 'Root', 'root', 0, 'Protected system superadmin - immutable', true, true, false),
    ('00000000-0000-0000-0001-000000000010', '00000000-0000-0000-0001-000000000010', NULL, NULL, 'Platform Owner', 'platform-owner', 10, 'Full platform control', true, false, false),
    ('00000000-0000-0000-0001-000000000020', '00000000-0000-0000-0001-000000000020', NULL, NULL, 'Platform Admin', 'platform-admin', 20, 'Platform administration', true, false, false),
    ('00000000-0000-0000-0001-000000000030', '00000000-0000-0000-0001-000000000030', NULL, NULL, 'Platform Analyst', 'platform-analyst', 30, 'Platform-wide data visibility', true, false, false)
ON CONFLICT DO NOTHING;

-- Organization Role Templates (organization_id=NULL, client_id=NULL, is_template=TRUE)
-- These are copied when a new organization is created
INSERT INTO roles (id, version_id, organization_id, client_id, name, slug, ordinal, description, is_system, is_protected, is_template) VALUES
    ('00000000-0000-0000-0002-000000000010', '00000000-0000-0000-0002-000000000010', NULL, NULL, 'Organization Owner', 'org-owner', 10, 'Full organization + client control', true, false, true),
    ('00000000-0000-0000-0002-000000000020', '00000000-0000-0000-0002-000000000020', NULL, NULL, 'Organization Admin', 'org-admin', 20, 'Organization administration', true, false, true),
    ('00000000-0000-0000-0002-000000000030', '00000000-0000-0000-0002-000000000030', NULL, NULL, 'Organization Analyst', 'org-analyst', 30, 'Organization data visibility', true, false, true)
ON CONFLICT DO NOTHING;

-- Client Role Templates (organization_id=NULL, client_id=NULL, is_template=TRUE)
-- These are copied when a new client is created
INSERT INTO roles (id, version_id, organization_id, client_id, name, slug, ordinal, description, is_system, is_protected, is_template) VALUES
    ('00000000-0000-0000-0003-000000000010', '00000000-0000-0000-0003-000000000010', NULL, NULL, 'Client Owner', 'client-owner', 10, 'Full client control', true, false, true),
    ('00000000-0000-0000-0003-000000000020', '00000000-0000-0000-0003-000000000020', NULL, NULL, 'Client Admin', 'client-admin', 20, 'Client administration', true, false, true),
    ('00000000-0000-0000-0003-000000000030', '00000000-0000-0000-0003-000000000030', NULL, NULL, 'Client Analyst', 'client-analyst', 30, 'Client data visibility', true, false, true)
ON CONFLICT DO NOTHING;

-- Default Organization Roles (copied from org templates)
-- organization_id = default org, client_id = NULL
INSERT INTO roles (id, version_id, organization_id, client_id, name, slug, ordinal, description, is_system, is_protected, is_template, created_by, updated_by) VALUES
    ('00000000-0000-0000-0010-000000000010', '00000000-0000-0000-0010-000000000010', '00000000-0000-0000-0000-000000000010', NULL, 'Organization Owner', 'org-owner', 10, 'Full organization + client control', false, false, false, '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002'),
    ('00000000-0000-0000-0010-000000000020', '00000000-0000-0000-0010-000000000020', '00000000-0000-0000-0000-000000000010', NULL, 'Organization Admin', 'org-admin', 20, 'Organization administration', false, false, false, '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002'),
    ('00000000-0000-0000-0010-000000000030', '00000000-0000-0000-0010-000000000030', '00000000-0000-0000-0000-000000000010', NULL, 'Organization Analyst', 'org-analyst', 30, 'Organization data visibility', false, false, false, '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002')
ON CONFLICT DO NOTHING;

-- Default Client Roles (copied from client templates)
-- organization_id = default org, client_id = default client
INSERT INTO roles (id, version_id, organization_id, client_id, name, slug, ordinal, description, is_system, is_protected, is_template, created_by, updated_by) VALUES
    ('00000000-0000-0000-0011-000000000010', '00000000-0000-0000-0011-000000000010', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000011', 'Client Owner', 'client-owner', 10, 'Full client control', false, false, false, '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002'),
    ('00000000-0000-0000-0011-000000000020', '00000000-0000-0000-0011-000000000020', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000011', 'Client Admin', 'client-admin', 20, 'Client administration', false, false, false, '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002'),
    ('00000000-0000-0000-0011-000000000030', '00000000-0000-0000-0011-000000000030', '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000011', 'Client Analyst', 'client-analyst', 30, 'Client data visibility', false, false, false, '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- ROLE PERMISSIONS
-- ============================================================================

-- Root role (ordinal 0) gets ALL permissions, locked
INSERT INTO role_permissions (role_id, permission_id, is_locked)
SELECT '00000000-0000-0000-0001-000000000001'::uuid, p.id, true
FROM permissions p
ON CONFLICT DO NOTHING;

-- Platform Owner (ordinal 10) gets all permissions except system:configure
INSERT INTO role_permissions (role_id, permission_id, is_locked)
SELECT '00000000-0000-0000-0001-000000000010'::uuid, p.id, false
FROM permissions p
WHERE NOT (p.resource = 'system' AND p.action = 'configure')
ON CONFLICT DO NOTHING;

-- Platform Admin (ordinal 20) gets management permissions
INSERT INTO role_permissions (role_id, permission_id, is_locked)
SELECT '00000000-0000-0000-0001-000000000020'::uuid, p.id, false
FROM permissions p
WHERE p.resource IN ('users', 'tokens', 'rules', 'alerts', 'cases', 'search', 'events', 'tenants')
   OR (p.resource = 'system' AND p.action IN ('view_audit', 'manage_integrations'))
ON CONFLICT DO NOTHING;

-- Platform Analyst (ordinal 30) gets read/view permissions
INSERT INTO role_permissions (role_id, permission_id, is_locked)
SELECT '00000000-0000-0000-0001-000000000030'::uuid, p.id, false
FROM permissions p
WHERE p.action IN ('read', 'execute', 'export', 'save_queries', 'acknowledge', 'view_audit')
ON CONFLICT DO NOTHING;

-- Organization Owner gets all org-relevant permissions
INSERT INTO role_permissions (role_id, permission_id, is_locked)
SELECT '00000000-0000-0000-0002-000000000010'::uuid, p.id, false
FROM permissions p
WHERE p.resource IN ('users', 'tokens', 'rules', 'alerts', 'cases', 'search', 'events')
   OR (p.resource = 'tenants' AND p.action IN ('create', 'read', 'update'))
ON CONFLICT DO NOTHING;

-- Organization Admin gets management permissions (no tenant create)
INSERT INTO role_permissions (role_id, permission_id, is_locked)
SELECT '00000000-0000-0000-0002-000000000020'::uuid, p.id, false
FROM permissions p
WHERE p.resource IN ('users', 'tokens', 'rules', 'alerts', 'cases', 'search', 'events')
   OR (p.resource = 'tenants' AND p.action = 'read')
ON CONFLICT DO NOTHING;

-- Organization Analyst gets read permissions
INSERT INTO role_permissions (role_id, permission_id, is_locked)
SELECT '00000000-0000-0000-0002-000000000030'::uuid, p.id, false
FROM permissions p
WHERE p.action IN ('read', 'execute', 'export', 'save_queries', 'acknowledge')
ON CONFLICT DO NOTHING;

-- Client Owner gets client-scoped permissions
INSERT INTO role_permissions (role_id, permission_id, is_locked)
SELECT '00000000-0000-0000-0003-000000000010'::uuid, p.id, false
FROM permissions p
WHERE (p.resource IN ('users', 'tokens', 'alerts', 'cases', 'search', 'events'))
   OR (p.resource = 'rules' AND p.action = 'read')
ON CONFLICT DO NOTHING;

-- Client Admin gets limited management permissions
INSERT INTO role_permissions (role_id, permission_id, is_locked)
SELECT '00000000-0000-0000-0003-000000000020'::uuid, p.id, false
FROM permissions p
WHERE (p.resource IN ('users', 'tokens') AND p.action IN ('read', 'create'))
   OR (p.resource IN ('alerts', 'cases', 'search', 'events') AND p.action IN ('read', 'execute', 'acknowledge', 'assign', 'close'))
   OR (p.resource = 'rules' AND p.action = 'read')
ON CONFLICT DO NOTHING;

-- Client Analyst gets read-only permissions
INSERT INTO role_permissions (role_id, permission_id, is_locked)
SELECT '00000000-0000-0000-0003-000000000030'::uuid, p.id, false
FROM permissions p
WHERE p.action IN ('read', 'execute', 'acknowledge')
  AND p.resource IN ('alerts', 'cases', 'search', 'events', 'rules')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- DEFAULT ORG/CLIENT ROLE PERMISSIONS (copy from templates)
-- ============================================================================

-- Default Org Owner (copy from org-owner template)
INSERT INTO role_permissions (role_id, permission_id, is_locked, granted_by)
SELECT '00000000-0000-0000-0010-000000000010'::uuid, rp.permission_id, false, '00000000-0000-0000-0000-000000000002'::uuid
FROM role_permissions rp WHERE rp.role_id = '00000000-0000-0000-0002-000000000010'::uuid
ON CONFLICT DO NOTHING;

-- Default Org Admin (copy from org-admin template)
INSERT INTO role_permissions (role_id, permission_id, is_locked, granted_by)
SELECT '00000000-0000-0000-0010-000000000020'::uuid, rp.permission_id, false, '00000000-0000-0000-0000-000000000002'::uuid
FROM role_permissions rp WHERE rp.role_id = '00000000-0000-0000-0002-000000000020'::uuid
ON CONFLICT DO NOTHING;

-- Default Org Analyst (copy from org-analyst template)
INSERT INTO role_permissions (role_id, permission_id, is_locked, granted_by)
SELECT '00000000-0000-0000-0010-000000000030'::uuid, rp.permission_id, false, '00000000-0000-0000-0000-000000000002'::uuid
FROM role_permissions rp WHERE rp.role_id = '00000000-0000-0000-0002-000000000030'::uuid
ON CONFLICT DO NOTHING;

-- Default Client Owner (copy from client-owner template)
INSERT INTO role_permissions (role_id, permission_id, is_locked, granted_by)
SELECT '00000000-0000-0000-0011-000000000010'::uuid, rp.permission_id, false, '00000000-0000-0000-0000-000000000002'::uuid
FROM role_permissions rp WHERE rp.role_id = '00000000-0000-0000-0003-000000000010'::uuid
ON CONFLICT DO NOTHING;

-- Default Client Admin (copy from client-admin template)
INSERT INTO role_permissions (role_id, permission_id, is_locked, granted_by)
SELECT '00000000-0000-0000-0011-000000000020'::uuid, rp.permission_id, false, '00000000-0000-0000-0000-000000000002'::uuid
FROM role_permissions rp WHERE rp.role_id = '00000000-0000-0000-0003-000000000020'::uuid
ON CONFLICT DO NOTHING;

-- Default Client Analyst (copy from client-analyst template)
INSERT INTO role_permissions (role_id, permission_id, is_locked, granted_by)
SELECT '00000000-0000-0000-0011-000000000030'::uuid, rp.permission_id, false, '00000000-0000-0000-0000-000000000002'::uuid
FROM role_permissions rp WHERE rp.role_id = '00000000-0000-0000-0003-000000000030'::uuid
ON CONFLICT DO NOTHING;

-- ============================================================================
-- ADMIN USER ROLE ASSIGNMENT
-- ============================================================================

-- Grant admin user the root role on platform tenant
INSERT INTO user_roles (id, user_id, role_id, tenant_id, granted_by)
VALUES (
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000002',  -- admin user
    '00000000-0000-0000-0001-000000000001',  -- root role
    '00000000-0000-0000-0000-000000000001',  -- platform tenant
    '00000000-0000-0000-0000-000000000002'   -- self-granted (bootstrap)
) ON CONFLICT DO NOTHING;
