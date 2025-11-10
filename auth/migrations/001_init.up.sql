-- TelHawk Auth Database Schema
-- PostgreSQL initialization script

-- Users table (immutable pattern with lifecycle timestamps)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    roles TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Lifecycle timestamps (immutable pattern)
    disabled_at TIMESTAMP,           -- User can't login (NULL = active)
    disabled_by UUID REFERENCES users(id),
    deleted_at TIMESTAMP,            -- Soft delete, hidden from UI (NULL = visible)
    deleted_by UUID REFERENCES users(id),

    CONSTRAINT users_not_self_disable CHECK (id != disabled_by),
    CONSTRAINT users_not_self_delete CHECK (id != deleted_by)
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_active ON users(username)
    WHERE disabled_at IS NULL AND deleted_at IS NULL;

-- Sessions table (immutable pattern)
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_token TEXT NOT NULL,
    refresh_token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Lifecycle timestamp instead of boolean
    revoked_at TIMESTAMP,            -- Session invalidated (NULL = active)
    revoked_by UUID REFERENCES users(id)
);

CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_sessions_active ON sessions(id)
    WHERE revoked_at IS NULL AND expires_at > NOW();

-- HEC tokens table (immutable pattern)
CREATE TABLE IF NOT EXISTS hec_tokens (
    id UUID PRIMARY KEY,
    token VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,

    -- Lifecycle timestamps
    disabled_at TIMESTAMP,           -- Token can't be used (NULL = active)
    disabled_by UUID REFERENCES users(id),
    revoked_at TIMESTAMP,            -- Token permanently invalidated (NULL = valid)
    revoked_by UUID REFERENCES users(id)
);

CREATE INDEX idx_hec_tokens_token ON hec_tokens(token);
CREATE INDEX idx_hec_tokens_user_id ON hec_tokens(user_id);
CREATE INDEX idx_hec_tokens_active ON hec_tokens(token)
    WHERE disabled_at IS NULL AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW());

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
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

-- No update triggers needed (immutable pattern - lifecycle managed via timestamps)

-- Insert default admin user (password: admin123 - change in production!)
-- Password hash for "admin123" using bcrypt cost 10
-- Note: Using UUID v4 for seed data (v7 used for runtime-generated IDs)
INSERT INTO users (id, username, email, password_hash, roles)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin',
    'admin@telhawk.local',
    '$2a$10$UslOjaejNBYO2PTBjrahduCEA/RM4x3nj0HXEtIGnsTDcPHnEWOva',
    ARRAY['admin']
) ON CONFLICT (username) DO NOTHING;

COMMENT ON TABLE users IS 'User accounts for TelHawk system';
COMMENT ON TABLE sessions IS 'Active authentication sessions with JWT and refresh tokens';
COMMENT ON TABLE hec_tokens IS 'HEC (HTTP Event Collector) tokens for data ingestion';
COMMENT ON TABLE audit_log IS 'Audit trail of all authentication and authorization events';
