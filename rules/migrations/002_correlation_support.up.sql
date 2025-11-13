-- Migration: Add correlation support to detection_schemas
-- Adds constraints and indexes for correlation types

-- Add check constraint for valid correlation types
ALTER TABLE detection_schemas
ADD CONSTRAINT valid_correlation_type
CHECK (
  model->>'correlation_type' IN (
    'event_count', 'value_count', 'temporal', 'temporal_ordered',
    'join', 'suppression', 'baseline_deviation', 'missing_event'
  ) OR model->>'correlation_type' IS NULL
);

-- Add indexes for correlation queries

-- Index for filtering by correlation type
CREATE INDEX idx_detection_schemas_correlation_type
ON detection_schemas ((model->>'correlation_type'))
WHERE disabled_at IS NULL;

-- Index for active parameter set lookup
CREATE INDEX idx_detection_schemas_active_params
ON detection_schemas ((model->>'active_parameter_set'))
WHERE disabled_at IS NULL;

-- Index for correlation type + severity filtering (common query pattern)
CREATE INDEX idx_detection_schemas_corr_severity
ON detection_schemas ((model->>'correlation_type'), (view->>'severity'))
WHERE disabled_at IS NULL;

-- Update comment documenting correlation support
COMMENT ON COLUMN detection_schemas.model IS
'Detection model configuration including optional correlation_type, parameters, parameter_sets, and active_parameter_set for advanced multi-event correlation';
