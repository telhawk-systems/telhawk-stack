-- TelHawk Rules Database Schema
-- PostgreSQL initialization script

-- Detection schemas table (immutable versioning pattern)
CREATE TABLE IF NOT EXISTS detection_schemas (
    id UUID NOT NULL,                 -- Stable rule identifier (same for all versions)
    version_id UUID PRIMARY KEY,      -- Version-specific UUID (UUID v7, unique per version)
    model JSONB NOT NULL,             -- Data model and aggregation config
    view JSONB NOT NULL,              -- Presentation and display config
    controller JSONB NOT NULL,        -- Detection logic and evaluation config
    created_by UUID NOT NULL,         -- References users(id) in auth DB
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Lifecycle timestamps (immutable pattern)
    disabled_at TIMESTAMP,            -- Rule won't be evaluated (NULL = active)
    disabled_by UUID,                 -- References users(id) in auth DB
    hidden_at TIMESTAMP,              -- Soft delete, hidden from UI (NULL = visible)
    hidden_by UUID,                   -- References users(id) in auth DB

    -- Constraints to prevent JSON null values
    CONSTRAINT model_not_json_null CHECK (model::text != 'null'),
    CONSTRAINT view_not_json_null CHECK (view::text != 'null'),
    CONSTRAINT controller_not_json_null CHECK (controller::text != 'null'),

    -- Correlation type constraint
    CONSTRAINT valid_correlation_type CHECK (
      model->>'correlation_type' IN (
        'event_count', 'value_count', 'temporal', 'temporal_ordered',
        'join', 'suppression', 'baseline_deviation', 'missing_event'
      ) OR model->>'correlation_type' IS NULL
    )
);

CREATE INDEX idx_schemas_id_created ON detection_schemas(id, created_at DESC);
CREATE INDEX idx_schemas_id ON detection_schemas(id);
CREATE INDEX idx_schemas_version_id ON detection_schemas(version_id);
CREATE INDEX idx_schemas_created_at ON detection_schemas(created_at DESC);
CREATE INDEX idx_schemas_active ON detection_schemas(id, created_at DESC)
    WHERE disabled_at IS NULL AND hidden_at IS NULL;

-- GIN indexes for JSONB queries
CREATE INDEX idx_schemas_model ON detection_schemas USING GIN (model);
CREATE INDEX idx_schemas_view ON detection_schemas USING GIN (view);
CREATE INDEX idx_schemas_controller ON detection_schemas USING GIN (controller);

-- Correlation indexes
CREATE INDEX idx_detection_schemas_correlation_type
ON detection_schemas ((model->>'correlation_type'))
WHERE disabled_at IS NULL;

CREATE INDEX idx_detection_schemas_active_params
ON detection_schemas ((model->>'active_parameter_set'))
WHERE disabled_at IS NULL;

CREATE INDEX idx_detection_schemas_corr_severity
ON detection_schemas ((model->>'correlation_type'), (view->>'severity'))
WHERE disabled_at IS NULL;

COMMENT ON TABLE detection_schemas IS 'Versioned detection rule schemas with immutable pattern';
COMMENT ON COLUMN detection_schemas.id IS 'Stable identifier grouping all versions of the same logical rule';
COMMENT ON COLUMN detection_schemas.version_id IS 'Version-specific UUID (UUID v7, time-ordered)';
COMMENT ON COLUMN detection_schemas.model IS 'Detection model configuration including optional correlation_type, parameters, parameter_sets, and active_parameter_set for advanced multi-event correlation';
COMMENT ON COLUMN detection_schemas.view IS 'Display config: title, severity, description templates, MITRE ATT&CK';
COMMENT ON COLUMN detection_schemas.controller IS 'Detection logic: query, conditions, evaluation intervals';
COMMENT ON COLUMN detection_schemas.disabled_at IS 'When rule was disabled (NULL = active, still evaluated)';
COMMENT ON COLUMN detection_schemas.hidden_at IS 'When rule was hidden (NULL = visible in UI, soft delete)';
