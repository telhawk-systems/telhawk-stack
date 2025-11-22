-- Respond service initial schema
-- This combines schemas from both rules and alerting services

-- Detection schemas table (from rules service)
CREATE TABLE IF NOT EXISTS detection_schemas (
    id UUID NOT NULL,
    version_id UUID PRIMARY KEY,
    model JSONB NOT NULL,
    view JSONB NOT NULL,
    controller JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    disabled_at TIMESTAMP WITH TIME ZONE,
    disabled_by UUID,
    hidden_at TIMESTAMP WITH TIME ZONE,
    hidden_by UUID,
    active_parameter_set VARCHAR(255),
    CONSTRAINT detection_schemas_model_not_null CHECK (model IS NOT NULL AND model != 'null'::jsonb),
    CONSTRAINT detection_schemas_view_not_null CHECK (view IS NOT NULL AND view != 'null'::jsonb),
    CONSTRAINT detection_schemas_controller_not_null CHECK (controller IS NOT NULL AND controller != 'null'::jsonb)
);

-- Index for looking up schemas by stable ID
CREATE INDEX IF NOT EXISTS idx_detection_schemas_id ON detection_schemas(id);

-- Index for finding latest versions efficiently
CREATE INDEX IF NOT EXISTS idx_detection_schemas_created_at ON detection_schemas(id, created_at DESC);

-- Cases table (from alerting service)
CREATE TABLE IF NOT EXISTS cases (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'open',
    priority VARCHAR(50) NOT NULL DEFAULT 'medium',
    assignee_id UUID,
    detection_schema_id UUID,
    detection_schema_version_id UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMP WITH TIME ZONE,
    closed_by UUID,
    CONSTRAINT cases_status_check CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
    CONSTRAINT cases_priority_check CHECK (priority IN ('low', 'medium', 'high', 'critical'))
);

-- Case alerts junction table (from alerting service)
CREATE TABLE IF NOT EXISTS case_alerts (
    id UUID PRIMARY KEY,
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    alert_id VARCHAR(255) NOT NULL,
    added_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    added_by UUID,
    UNIQUE(case_id, alert_id)
);

-- Indexes for case queries
CREATE INDEX IF NOT EXISTS idx_cases_status ON cases(status);
CREATE INDEX IF NOT EXISTS idx_cases_detection_schema_id ON cases(detection_schema_id);
CREATE INDEX IF NOT EXISTS idx_case_alerts_case_id ON case_alerts(case_id);
CREATE INDEX IF NOT EXISTS idx_case_alerts_alert_id ON case_alerts(alert_id);
