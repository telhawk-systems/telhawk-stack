-- TelHawk Auth Database Schema - Rollback
-- Drops all tables in reverse dependency order

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

-- Tenants
DROP TABLE IF EXISTS tenants;
