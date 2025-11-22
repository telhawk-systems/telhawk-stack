-- Rollback initial schema
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS hec_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
