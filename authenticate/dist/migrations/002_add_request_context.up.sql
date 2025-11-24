-- Migration 002: Add request context (IP address, source type) for audit trails
--
-- Every mutation in the system should be traceable:
-- - WHO did it (already have: created_by, updated_by, etc.)
-- - WHEN they did it (already have: UUIDv7 timestamps, *_at columns)
-- - WHERE they did it from (NEW: IP address)
-- - HOW they did it (NEW: source_type - web/cli/api)
--
-- Source types: 0=unknown, 1=web, 2=cli, 3=api, 4=system/internal

-- ============================================================================
-- ORGANIZATIONS: Add audit context to versioned rows
-- ============================================================================

-- IP and source for row creation (new versions)
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS created_from_ip INET;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS created_source_type SMALLINT NOT NULL DEFAULT 0;

-- IP and source for lifecycle changes (disable/delete happen in-place)
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS disabled_from_ip INET;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS disabled_source_type SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS deleted_from_ip INET;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS deleted_source_type SMALLINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN organizations.created_from_ip IS 'IP address when this version was created';
COMMENT ON COLUMN organizations.created_source_type IS 'Source: 0=unknown, 1=web, 2=cli, 3=api, 4=system';
COMMENT ON COLUMN organizations.disabled_from_ip IS 'IP address when disabled';
COMMENT ON COLUMN organizations.disabled_source_type IS 'Source when disabled';
COMMENT ON COLUMN organizations.deleted_from_ip IS 'IP address when deleted';
COMMENT ON COLUMN organizations.deleted_source_type IS 'Source when deleted';

-- ============================================================================
-- CLIENTS: Add audit context to versioned rows
-- ============================================================================

ALTER TABLE clients ADD COLUMN IF NOT EXISTS created_from_ip INET;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS created_source_type SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS disabled_from_ip INET;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS disabled_source_type SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS deleted_from_ip INET;
ALTER TABLE clients ADD COLUMN IF NOT EXISTS deleted_source_type SMALLINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN clients.created_from_ip IS 'IP address when this version was created';
COMMENT ON COLUMN clients.created_source_type IS 'Source: 0=unknown, 1=web, 2=cli, 3=api, 4=system';
COMMENT ON COLUMN clients.disabled_from_ip IS 'IP address when disabled';
COMMENT ON COLUMN clients.disabled_source_type IS 'Source when disabled';
COMMENT ON COLUMN clients.deleted_from_ip IS 'IP address when deleted';
COMMENT ON COLUMN clients.deleted_source_type IS 'Source when deleted';

-- ============================================================================
-- USERS: Add audit context to versioned rows
-- ============================================================================

ALTER TABLE users ADD COLUMN IF NOT EXISTS created_from_ip INET;
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_source_type SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS disabled_from_ip INET;
ALTER TABLE users ADD COLUMN IF NOT EXISTS disabled_source_type SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_from_ip INET;
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_source_type SMALLINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN users.created_from_ip IS 'IP address when this version was created';
COMMENT ON COLUMN users.created_source_type IS 'Source: 0=unknown, 1=web, 2=cli, 3=api, 4=system';
COMMENT ON COLUMN users.disabled_from_ip IS 'IP address when disabled';
COMMENT ON COLUMN users.disabled_source_type IS 'Source when disabled';
COMMENT ON COLUMN users.deleted_from_ip IS 'IP address when deleted';
COMMENT ON COLUMN users.deleted_source_type IS 'Source when deleted';

-- Index for "which users were created from this IP" queries (security investigation)
CREATE INDEX IF NOT EXISTS idx_users_created_from_ip ON users(created_from_ip)
    WHERE created_from_ip IS NOT NULL;

-- ============================================================================
-- ROLES: Add audit context to versioned rows
-- ============================================================================

ALTER TABLE roles ADD COLUMN IF NOT EXISTS created_from_ip INET;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS created_source_type SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS deleted_from_ip INET;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS deleted_source_type SMALLINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN roles.created_from_ip IS 'IP address when this version was created';
COMMENT ON COLUMN roles.created_source_type IS 'Source: 0=unknown, 1=web, 2=cli, 3=api, 4=system';
COMMENT ON COLUMN roles.deleted_from_ip IS 'IP address when deleted';
COMMENT ON COLUMN roles.deleted_source_type IS 'Source when deleted';

-- ============================================================================
-- SESSIONS: Add IP and source tracking
-- ============================================================================

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS ip_address INET;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS user_agent TEXT;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS source_type SMALLINT NOT NULL DEFAULT 0;

-- Revocation context
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS revoked_from_ip INET;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS revoked_source_type SMALLINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_sessions_ip_address ON sessions(ip_address)
    WHERE ip_address IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_source_type ON sessions(source_type);

COMMENT ON COLUMN sessions.ip_address IS 'Client IP address at login time';
COMMENT ON COLUMN sessions.user_agent IS 'User-Agent header from login request';
COMMENT ON COLUMN sessions.source_type IS 'Login source: 0=unknown, 1=web, 2=cli, 3=api';
COMMENT ON COLUMN sessions.revoked_from_ip IS 'IP address when session was revoked';
COMMENT ON COLUMN sessions.revoked_source_type IS 'Source when revoked';

-- ============================================================================
-- HEC_TOKENS: Add creation context (append-only, no usage tracking here)
-- ============================================================================

-- Creation context
ALTER TABLE hec_tokens ADD COLUMN IF NOT EXISTS created_from_ip INET;
ALTER TABLE hec_tokens ADD COLUMN IF NOT EXISTS created_source_type SMALLINT NOT NULL DEFAULT 0;

-- Lifecycle context (these are one-time events, not repeated updates)
ALTER TABLE hec_tokens ADD COLUMN IF NOT EXISTS disabled_from_ip INET;
ALTER TABLE hec_tokens ADD COLUMN IF NOT EXISTS disabled_source_type SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE hec_tokens ADD COLUMN IF NOT EXISTS revoked_from_ip INET;
ALTER TABLE hec_tokens ADD COLUMN IF NOT EXISTS revoked_source_type SMALLINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN hec_tokens.created_from_ip IS 'IP address when token was created';
COMMENT ON COLUMN hec_tokens.created_source_type IS 'Creation source: 0=unknown, 1=web, 2=cli, 3=api';
COMMENT ON COLUMN hec_tokens.disabled_from_ip IS 'IP address when disabled';
COMMENT ON COLUMN hec_tokens.disabled_source_type IS 'Source when disabled';
COMMENT ON COLUMN hec_tokens.revoked_from_ip IS 'IP address when revoked';
COMMENT ON COLUMN hec_tokens.revoked_source_type IS 'Source when revoked';

-- ============================================================================
-- HEC TOKEN USAGE: Stored in Redis, not PostgreSQL
-- ============================================================================
-- Usage statistics (last_used_at, event_count, etc.) are stored in Redis
-- for real-time updates across multiple ingest instances.
--
-- Redis key structure (see common/hecstats package):
--   hec:stats:{token_id}                - Hash: last_used_at, last_used_ip, total_events
--   hec:hourly:{token_id}:{YYYYMMDDHH}  - Counter: events this hour (expires 48h)
--   hec:daily:{token_id}:{YYYYMMDD}     - Counter: events this day (expires 7d)
--   hec:ips:{token_id}:{YYYYMMDD}       - Set: unique IPs today (expires 7d)
--   hec:instances:{token_id}            - Hash: ingest_instance -> last_seen
--
-- This avoids write amplification on the immutable hec_tokens table.

-- ============================================================================
-- USER_ROLES: Add IP and source tracking for grants/revocations
-- ============================================================================

ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS granted_from_ip INET;
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS granted_source_type SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS revoked_from_ip INET;
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS revoked_source_type SMALLINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN user_roles.granted_from_ip IS 'IP address of admin who granted this role';
COMMENT ON COLUMN user_roles.granted_source_type IS 'Grant source: 0=unknown, 1=web, 2=cli, 3=api';
COMMENT ON COLUMN user_roles.revoked_from_ip IS 'IP address of admin who revoked this role';
COMMENT ON COLUMN user_roles.revoked_source_type IS 'Revocation source: 0=unknown, 1=web, 2=cli, 3=api';

-- ============================================================================
-- ROLE_PERMISSIONS: Add IP and source tracking
-- ============================================================================

ALTER TABLE role_permissions ADD COLUMN IF NOT EXISTS granted_from_ip INET;
ALTER TABLE role_permissions ADD COLUMN IF NOT EXISTS granted_source_type SMALLINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN role_permissions.granted_from_ip IS 'IP address when permission was granted to role';
COMMENT ON COLUMN role_permissions.granted_source_type IS 'Source: 0=unknown, 1=web, 2=cli, 3=api';

-- ============================================================================
-- AUDIT_LOG: Add source type (already has ip_address)
-- ============================================================================

ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS source_type SMALLINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_audit_log_source_type ON audit_log(source_type);

COMMENT ON COLUMN audit_log.source_type IS 'Request source: 0=unknown, 1=web, 2=cli, 3=api, 4=system';
