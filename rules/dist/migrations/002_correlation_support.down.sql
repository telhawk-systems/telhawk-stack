-- Migration: Remove correlation support
-- Rollback correlation constraints and indexes

-- Drop indexes
DROP INDEX IF EXISTS idx_detection_schemas_corr_severity;
DROP INDEX IF EXISTS idx_detection_schemas_active_params;
DROP INDEX IF EXISTS idx_detection_schemas_correlation_type;

-- Drop constraint
ALTER TABLE detection_schemas DROP CONSTRAINT IF EXISTS valid_correlation_type;

-- Restore original comment
COMMENT ON COLUMN detection_schemas.model IS
'Data model: fields, aggregation, time windows, thresholds';
