-- Respond Service Database Schema
-- PostgreSQL initialization script
--
-- Immutable Versioning Pattern:
-- - id: Stable entity identifier (UUIDv7 - timestamp encodes created_at)
-- - version_id: Row version identifier (UUIDv7 - timestamp encodes updated_at)
-- - Content changes create new rows with same id but new version_id
-- - Lifecycle changes (disable/delete/close) UPDATE existing rows (no new version)
-- - NO created_at or updated_at columns - UUIDv7 timestamps encode this information

-- Detection schemas table (from rules service)
-- Immutability pattern: version_id is UUIDv7 (timestamp encodes created_at)
CREATE TABLE IF NOT EXISTS detection_schemas (
    -- Identity (UUIDv7: timestamp = created_at)
    id UUID NOT NULL,
    -- Version (UUIDv7: timestamp = updated_at, PRIMARY KEY)
    version_id UUID PRIMARY KEY,

    -- Detection schema data
    model JSONB NOT NULL,
    view JSONB NOT NULL,
    controller JSONB NOT NULL,
    active_parameter_set VARCHAR(255),

    -- Lifecycle (immutable pattern)
    disabled_at TIMESTAMP WITH TIME ZONE,
    disabled_by UUID,
    hidden_at TIMESTAMP WITH TIME ZONE,
    hidden_by UUID,

    -- Constraints
    CONSTRAINT detection_schemas_model_not_null CHECK (model IS NOT NULL AND model != 'null'::jsonb),
    CONSTRAINT detection_schemas_view_not_null CHECK (view IS NOT NULL AND view != 'null'::jsonb),
    CONSTRAINT detection_schemas_controller_not_null CHECK (controller IS NOT NULL AND controller != 'null'::jsonb)
);

-- Index for looking up schemas by stable ID
CREATE INDEX IF NOT EXISTS idx_detection_schemas_id ON detection_schemas(id);

-- Cases table (from alerting service)
-- Immutability pattern: id is UUIDv7 (timestamp encodes created_at)
CREATE TABLE IF NOT EXISTS cases (
    -- Identity (UUIDv7: timestamp = created_at, PRIMARY KEY)
    id UUID PRIMARY KEY,

    -- Case data
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'open',
    priority VARCHAR(50) NOT NULL DEFAULT 'medium',
    assignee_id UUID,
    detection_schema_id UUID,
    detection_schema_version_id UUID,

    -- Lifecycle (immutable pattern)
    closed_at TIMESTAMP WITH TIME ZONE,
    closed_by UUID,

    -- Constraints
    CONSTRAINT cases_status_check CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
    CONSTRAINT cases_priority_check CHECK (priority IN ('low', 'medium', 'high', 'critical'))
);

-- Case alerts junction table (from alerting service)
-- Immutability pattern: id is UUIDv7 (timestamp encodes when alert was added to case)
CREATE TABLE IF NOT EXISTS case_alerts (
    -- Identity (UUIDv7: timestamp = added_at, PRIMARY KEY)
    id UUID PRIMARY KEY,

    -- Junction data
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    alert_id VARCHAR(255) NOT NULL,
    added_by UUID,

    UNIQUE(case_id, alert_id)
);

-- Indexes for case queries
CREATE INDEX IF NOT EXISTS idx_cases_status ON cases(status);
CREATE INDEX IF NOT EXISTS idx_cases_detection_schema_id ON cases(detection_schema_id);
CREATE INDEX IF NOT EXISTS idx_case_alerts_case_id ON case_alerts(case_id);
CREATE INDEX IF NOT EXISTS idx_case_alerts_alert_id ON case_alerts(alert_id);

-- ============================================================================
-- COMMENTS (Documentation)
-- ============================================================================

COMMENT ON TABLE detection_schemas IS 'Versioned detection rule schemas with immutable pattern';
COMMENT ON COLUMN detection_schemas.id IS 'Stable identifier grouping all versions of the same logical rule (UUIDv7 timestamp = created_at)';
COMMENT ON COLUMN detection_schemas.version_id IS 'Version-specific UUID (UUIDv7 timestamp = updated_at, time-ordered)';
COMMENT ON COLUMN detection_schemas.disabled_at IS 'When rule was disabled (NULL = active, still evaluated)';
COMMENT ON COLUMN detection_schemas.hidden_at IS 'When rule was hidden (NULL = visible in UI, soft delete)';

COMMENT ON TABLE cases IS 'Security cases for investigation and incident response';
COMMENT ON COLUMN cases.id IS 'Case ID (UUIDv7 timestamp = created_at)';
COMMENT ON COLUMN cases.closed_at IS 'When case was closed (NULL = open/in_progress/resolved)';

COMMENT ON TABLE case_alerts IS 'Junction table linking cases to alerts in OpenSearch';
COMMENT ON COLUMN case_alerts.id IS 'Assignment ID (UUIDv7 timestamp = when alert was added to case)';
COMMENT ON COLUMN case_alerts.alert_id IS 'OpenSearch document ID from telhawk-alerts-* index';
