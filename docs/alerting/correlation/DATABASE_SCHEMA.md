## Database Schema

### No New Tables Required

All correlation configuration embedded in existing `detection_schemas` table JSONB columns.

### Schema Validation

Add constraint to ensure correlation_type is valid:

```sql
-- Add check constraint for correlation types
ALTER TABLE detection_schemas
ADD CONSTRAINT valid_correlation_type
CHECK (
  model->>'correlation_type' IN (
    'event_count', 'value_count', 'temporal', 'temporal_ordered',
    'join', 'suppression', 'baseline_deviation', 'missing_event'
  ) OR model->>'correlation_type' IS NULL
);
```

### Indexes for Performance

```sql
-- Index for filtering by correlation type
CREATE INDEX idx_detection_schemas_correlation_type
ON detection_schemas ((model->>'correlation_type'))
WHERE disabled_at IS NULL;

-- Index for active parameter set lookup
CREATE INDEX idx_detection_schemas_active_params
ON detection_schemas ((model->>'active_parameter_set'))
WHERE disabled_at IS NULL;

-- Index for correlation type + severity filtering
CREATE INDEX idx_detection_schemas_corr_severity
ON detection_schemas ((model->>'correlation_type'), (view->>'severity'))
WHERE disabled_at IS NULL;
```

### Migration Script

```sql
-- Migration: Add correlation support to detection_schemas
-- File: rules/migrations/002_correlation_support.up.sql

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
CREATE INDEX idx_detection_schemas_correlation_type
ON detection_schemas ((model->>'correlation_type'))
WHERE disabled_at IS NULL;

CREATE INDEX idx_detection_schemas_active_params
ON detection_schemas ((model->>'active_parameter_set'))
WHERE disabled_at IS NULL;

-- Add comment documenting correlation support
COMMENT ON COLUMN detection_schemas.model IS
'Detection model configuration including correlation_type, parameters, and parameter_sets';
```

```sql
-- Migration: Remove correlation support
-- File: rules/migrations/002_correlation_support.down.sql

DROP INDEX IF EXISTS idx_detection_schemas_active_params;
DROP INDEX IF EXISTS idx_detection_schemas_correlation_type;

ALTER TABLE detection_schemas DROP CONSTRAINT IF EXISTS valid_correlation_type;
```

---

