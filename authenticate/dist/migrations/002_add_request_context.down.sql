-- Migration 002 DOWN: Remove request context columns

-- Audit log
DROP INDEX IF EXISTS idx_audit_log_source_type;
ALTER TABLE audit_log DROP COLUMN IF EXISTS source_type;

-- Role permissions
ALTER TABLE role_permissions DROP COLUMN IF EXISTS granted_source_type;
ALTER TABLE role_permissions DROP COLUMN IF EXISTS granted_from_ip;

-- User roles
ALTER TABLE user_roles DROP COLUMN IF EXISTS revoked_source_type;
ALTER TABLE user_roles DROP COLUMN IF EXISTS revoked_from_ip;
ALTER TABLE user_roles DROP COLUMN IF EXISTS granted_source_type;
ALTER TABLE user_roles DROP COLUMN IF EXISTS granted_from_ip;

-- HEC token usage is in Redis, nothing to drop here

-- HEC tokens (immutable lifecycle columns only)
ALTER TABLE hec_tokens DROP COLUMN IF EXISTS revoked_source_type;
ALTER TABLE hec_tokens DROP COLUMN IF EXISTS revoked_from_ip;
ALTER TABLE hec_tokens DROP COLUMN IF EXISTS disabled_source_type;
ALTER TABLE hec_tokens DROP COLUMN IF EXISTS disabled_from_ip;
ALTER TABLE hec_tokens DROP COLUMN IF EXISTS created_source_type;
ALTER TABLE hec_tokens DROP COLUMN IF EXISTS created_from_ip;

-- Sessions
DROP INDEX IF EXISTS idx_sessions_source_type;
DROP INDEX IF EXISTS idx_sessions_ip_address;
ALTER TABLE sessions DROP COLUMN IF EXISTS revoked_source_type;
ALTER TABLE sessions DROP COLUMN IF EXISTS revoked_from_ip;
ALTER TABLE sessions DROP COLUMN IF EXISTS source_type;
ALTER TABLE sessions DROP COLUMN IF EXISTS user_agent;
ALTER TABLE sessions DROP COLUMN IF EXISTS ip_address;

-- Roles
ALTER TABLE roles DROP COLUMN IF EXISTS deleted_source_type;
ALTER TABLE roles DROP COLUMN IF EXISTS deleted_from_ip;
ALTER TABLE roles DROP COLUMN IF EXISTS created_source_type;
ALTER TABLE roles DROP COLUMN IF EXISTS created_from_ip;

-- Users
DROP INDEX IF EXISTS idx_users_created_from_ip;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_source_type;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_from_ip;
ALTER TABLE users DROP COLUMN IF EXISTS disabled_source_type;
ALTER TABLE users DROP COLUMN IF EXISTS disabled_from_ip;
ALTER TABLE users DROP COLUMN IF EXISTS created_source_type;
ALTER TABLE users DROP COLUMN IF EXISTS created_from_ip;

-- Clients
ALTER TABLE clients DROP COLUMN IF EXISTS deleted_source_type;
ALTER TABLE clients DROP COLUMN IF EXISTS deleted_from_ip;
ALTER TABLE clients DROP COLUMN IF EXISTS disabled_source_type;
ALTER TABLE clients DROP COLUMN IF EXISTS disabled_from_ip;
ALTER TABLE clients DROP COLUMN IF EXISTS created_source_type;
ALTER TABLE clients DROP COLUMN IF EXISTS created_from_ip;

-- Organizations
ALTER TABLE organizations DROP COLUMN IF EXISTS deleted_source_type;
ALTER TABLE organizations DROP COLUMN IF EXISTS deleted_from_ip;
ALTER TABLE organizations DROP COLUMN IF EXISTS disabled_source_type;
ALTER TABLE organizations DROP COLUMN IF EXISTS disabled_from_ip;
ALTER TABLE organizations DROP COLUMN IF EXISTS created_source_type;
ALTER TABLE organizations DROP COLUMN IF EXISTS created_from_ip;
