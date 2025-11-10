-- Alerting Service Schema
-- Cases and Case-Alert associations

-- Cases table
CREATE TABLE IF NOT EXISTS cases (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
    status TEXT NOT NULL CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')) DEFAULT 'open',
    assignee UUID,  -- Reference to user in auth service
    created_by UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP,  -- Status change timestamp
    closed_at TIMESTAMP,
    closed_by UUID
);

-- Indexes for common queries
CREATE INDEX idx_cases_status ON cases(status);
CREATE INDEX idx_cases_severity ON cases(severity);
CREATE INDEX idx_cases_assignee ON cases(assignee);
CREATE INDEX idx_cases_created_at ON cases(created_at DESC);

-- Case-Alert junction table
-- Links cases to alerts (stored in OpenSearch)
CREATE TABLE IF NOT EXISTS case_alerts (
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    alert_id TEXT NOT NULL,  -- OpenSearch document ID
    detection_schema_id UUID NOT NULL,  -- Stable detection schema ID
    detection_schema_version_id UUID NOT NULL,  -- Version-specific ID
    added_at TIMESTAMP NOT NULL DEFAULT NOW(),
    added_by UUID NOT NULL,
    PRIMARY KEY (case_id, alert_id)
);

-- Indexes for lookups
CREATE INDEX idx_case_alerts_case_id ON case_alerts(case_id);
CREATE INDEX idx_case_alerts_alert_id ON case_alerts(alert_id);
CREATE INDEX idx_case_alerts_schema_id ON case_alerts(detection_schema_id);

-- Comments for documentation
COMMENT ON TABLE cases IS 'Security cases for investigation and incident response';
COMMENT ON TABLE case_alerts IS 'Junction table linking cases to alerts in OpenSearch';
COMMENT ON COLUMN case_alerts.alert_id IS 'OpenSearch document ID from telhawk-alerts-* index';
COMMENT ON COLUMN case_alerts.detection_schema_id IS 'Stable detection schema ID (groups versions)';
COMMENT ON COLUMN case_alerts.detection_schema_version_id IS 'Version-specific detection schema ID';
