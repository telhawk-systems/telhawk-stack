-- TelHawk Auth Database Schema - Rollback
-- Drops all tables, functions, and triggers in reverse dependency order

-- Drop trigger
DROP TRIGGER IF EXISTS user_roles_permissions_changed ON user_roles;

-- Drop functions
DROP FUNCTION IF EXISTS trigger_user_permissions_changed();
DROP FUNCTION IF EXISTS increment_user_permissions_version(UUID);

-- User role assignments
DROP TABLE IF EXISTS user_roles;

-- Role-permission mappings
DROP TABLE IF EXISTS role_permissions;

-- Permissions
DROP TABLE IF EXISTS permissions;

-- Roles
DROP TABLE IF EXISTS roles;

-- Audit log
DROP TABLE IF EXISTS audit_log;

-- HEC tokens
DROP TABLE IF EXISTS hec_tokens;

-- Sessions
DROP TABLE IF EXISTS sessions;

-- Users
DROP TABLE IF EXISTS users;

-- Clients (must drop before organizations due to FK)
DROP TABLE IF EXISTS clients;

-- Organizations
DROP TABLE IF EXISTS organizations;
