# Detection Rules System - Complete Overview

## Database Schema

### Location
- **Schema Migrations**: `/home/shorton/projects/telhawk-stack/rules/migrations/`
  - `001_init.up.sql` - Core detection_schemas table
  - `002_correlation_support.up.sql` - Correlation type constraints

### detection_schemas Table Structure

```sql
CREATE TABLE detection_schemas (
    id UUID NOT NULL,                 -- Stable rule identifier (same for all versions)
    version_id UUID PRIMARY KEY,      -- Version-specific UUID (UUID v7, unique per version)
    model JSONB NOT NULL,             -- Data model and aggregation config
    view JSONB NOT NULL,              -- Presentation and display config
    controller JSONB NOT NULL,        -- Detection logic and evaluation config
    created_by UUID NOT NULL,         -- References users(id) in auth DB
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    disabled_at TIMESTAMP,            -- Rule won't be evaluated (NULL = active)
    disabled_by UUID,                 -- References users(id) in auth DB
    hidden_at TIMESTAMP,              -- Soft delete, hidden from UI (NULL = visible)
    hidden_by UUID                    -- References users(id) in auth DB
);
```

### Key Design Patterns
- **Immutable Versioning**: Same `id` groups versions, new `version_id` for each update
- **UUID v7**: Time-ordered UUIDs for better B-tree performance
- **Lifecycle Timestamps**: Uses `disabled_at`, `hidden_at` instead of boolean flags
- **GIN Indexes**: JSONB columns indexed for efficient nested queries

### Indexes
- `idx_schemas_id_created` - For retrieving latest version
- `idx_schemas_active` - For active rules (WHERE disabled_at IS NULL AND hidden_at IS NULL)
- `idx_schemas_model`, `idx_schemas_view`, `idx_schemas_controller` - JSONB GIN indexes
- `idx_detection_schemas_correlation_type` - For filtering by correlation type
- `idx_detection_schemas_corr_severity` - For common query pattern (type + severity)

## Rules Service Code Structure

### Key Locations
- **Rules Service**: `/home/shorton/projects/telhawk-stack/rules/`
- **Models**: `rules/internal/models/schema.go`
- **Handlers**: `rules/internal/handlers/handlers.go`
- **Service Layer**: `rules/internal/service/service.go`
- **Repository**: `rules/internal/repository/postgres.go`

### Data Models (from schema.go)
```go
type DetectionSchema struct {
    ID         string                 // Stable rule identifier
    VersionID  string                 // Version-specific UUID (UUID v7)
    Model      map[string]interface{} // Data model and aggregation config
    View       map[string]interface{} // Presentation and display config
    Controller map[string]interface{} // Detection logic and evaluation
    CreatedBy  string                 // User ID
    CreatedAt  time.Time
    DisabledAt *time.Time            // nil = active
    DisabledBy *string
    HiddenAt   *time.Time            // nil = visible
    HiddenBy   *string
    Version    int                    // Calculated by ROW_NUMBER()
}
```

## Rule Structure - Three-Part Design

### 1. MODEL (Data Model & Aggregation)
Defines what data to collect and how to aggregate it.

**Correlation Type Options:**
- `event_count` - Count matching events in time window
- `value_count` - Count distinct values (cardinality)
- `temporal` - Multiple events within time window (any order)
- `temporal_ordered` - Event sequence with ordering
- `join` - Correlate by matching field values
- `suppression` - Suppress duplicate alerts
- `baseline_deviation` - Alert on deviation from baseline
- `missing_event` - Alert when expected event is absent

**Common Model Fields:**
- `correlation_type` - The detection type
- `parameters` - Type-specific settings:
  - `time_window` - Duration (e.g., "5m", "1h")
  - `query` - Filter definition with OCSF field conditions
  - `threshold` - Value and comparison operator
  - `group_by` - Fields to aggregate by
  - `field` - Target field for cardinality (value_count)

**Example (event_count):**
```json
{
  "correlation_type": "event_count",
  "parameters": {
    "time_window": "5m",
    "query": {
      "filter": {
        "type": "and",
        "conditions": [
          {"field": ".class_uid", "operator": "eq", "value": 3002},
          {"field": ".status_id", "operator": "eq", "value": 2}
        ]
      }
    },
    "threshold": {"value": 5, "operator": "gte"},
    "group_by": [".actor.user.name", ".src_endpoint.ip"]
  }
}
```

### 2. VIEW (Presentation & Display)
Defines how alerts look and severity classification.

**Key Fields:**
- `title` - Alert title
- `severity` - critical|high|medium|low|informational
- `description` - Template with {{field}} variables
- `category` - Category name (e.g., "Authentication", "Network Security")
- `tags` - Array of tags for classification
- `mitre_attack` - Object with tactics[] and techniques[]
- `fields_order` - Display order for fields
- `priority` - Display priority

**Example:**
```json
{
  "title": "Multiple Failed Login Attempts",
  "severity": "medium",
  "description": "User {{actor.user.name}} from {{src_endpoint.ip}} had {{event_count}} failed login attempts",
  "category": "Authentication",
  "tags": ["authentication", "brute-force"],
  "mitre_attack": {
    "tactics": ["Credential Access"],
    "techniques": ["T1110.001 - Brute Force: Password Guessing"]
  }
}
```

### 3. CONTROLLER (Detection Logic & Evaluation)
Defines how and when the rule is evaluated.

**Key Fields:**
- `detection` - Detection settings:
  - `suppression_window` - Duration before same alert fires again
  - `strict_order` - Enforce sequence order (temporal_ordered)
  - `metadata` - Object with source="builtin" for built-in rules
- `response` - Response actions:
  - `actions` - Array of actions to take
  - `severity_threshold` - Minimum severity to act on
- `query` - Query definition (optional)
- `aggregation_field` - Field for aggregation
- `condition` - Custom condition
- `lookback` - Historical lookback duration
- `evaluation_interval` - How often to evaluate

**Example:**
```json
{
  "detection": {
    "suppression_window": "10m"
  },
  "response": {
    "actions": [],
    "severity_threshold": "medium"
  }
}
```

## Query Filter Language

Rules use OCSF field notation with dot notation for nested access.

### Field Notation
- Use leading dot: `.time`, `.severity`, `.actor.user.name`
- Nested objects: `.src_endpoint.ip`, `.dst_endpoint.port`
- Array access: Direct (e.g., `.tags[0]` not typically used in rules)

### Filter Operators
- **Comparison**: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`
- **Logical**: `and`, `or`, `not`
- **Set**: `in`, `not_in`, `contains`, `not_contains`
- **Null**: `is_null`, `is_not_null`

### Filter Structure - Simple Condition
```json
{
  "filter": {
    "field": ".class_uid",
    "operator": "eq",
    "value": 3002
  }
}
```

### Filter Structure - Compound Condition
```json
{
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".class_uid", "operator": "eq", "value": 3002},
      {"field": ".status_id", "operator": "eq", "value": 2}
    ]
  }
}
```

## OCSF Field Names & Common Classes

### OCSF Class UIDs (event types)
- **3002** - Authentication (login/logout)
- **3001** - Account Change
- **4001** - Network Activity
- **4005** - File Activity
- **1003** - Account Change
- **1007** - User Session
- **6003** - Network Scan
- **2004** - Vulnerability

### Common OCSF Fields Used in Rules
- `.time` - Event timestamp
- `.class_uid` - Event class type
- `.status_id` - Status/result code
- `.severity` - Severity level
- `.actor.user.name` - Username
- `.src_endpoint.ip` - Source IP
- `.src_endpoint.port` - Source port
- `.dst_endpoint.ip` - Destination IP
- `.dst_endpoint.port` - Destination port
- `.dst_endpoint.hostname` - Destination hostname

### Answer: Yes, Rules Use OCSF Field Names
Rules exclusively use OCSF field names with dot notation. No alternative field naming conventions are supported.

## Complete Example Rules

### Example 1: Failed Logins (event_count)
File: `/home/shorton/projects/telhawk-stack/alerting/rules/failed_logins.json`

Detects multiple failed authentication attempts:
- Correlation Type: `event_count`
- Time Window: 5 minutes
- Filter: class_uid=3002 AND status_id=2 (failed auth)
- Threshold: >= 5 failures
- Group By: user.name, src_endpoint.ip
- Alert When: Grouped count exceeds threshold

### Example 2: Port Scanning (value_count)
File: `/home/shorton/projects/telhawk-stack/alerting/rules/port_scanning.json`

Detects scanning of many ports from single source:
- Correlation Type: `value_count` (cardinality)
- Time Window: 10 minutes
- Filter: class_uid=4001 (network activity)
- Field: dst_endpoint.port (count distinct values)
- Threshold: > 20 distinct ports
- Group By: src_endpoint.ip

### Example 3: Privilege Escalation After Failed Login (temporal_ordered)
File: `/home/shorton/projects/telhawk-stack/alerting/rules/privilege_escalation_after_failed_login.json`

Detects suspicious sequence:
- Correlation Type: `temporal_ordered`
- Time Window: 15 minutes
- Max Gap: 10 minutes (between events)
- Sequence:
  1. Failed login (class_uid=3002, status_id=2)
  2. Privilege escalation (class_uid=3001 or 1007)
- Group By: actor.user.name
- Alert When: Sequence detected

## Frontend Types

Location: `/home/shorton/projects/telhawk-stack/web/frontend/src/types/rules.ts`

```typescript
interface DetectionSchemaModel {
  fields?: string[];
  group_by?: string[];
  time_window?: string;
  threshold?: number;
  aggregation?: string;
}

interface DetectionSchemaView {
  title: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'informational';
  priority?: string;
  fields_order?: string[];
  description_template?: string;
  mitre_attack?: {
    tactics?: string[];
    techniques?: string[];
  };
}

interface DetectionSchemaController {
  query: string;
  aggregation_field?: string;
  condition?: string;
  lookback?: string;
  evaluation_interval?: string;
}
```

## API Endpoints

Rules Service exposes JSON:API-compliant endpoints at `/api/v1/schemas`:
- **POST** - Create new rule
- **GET** - List rules (with pagination and filtering)
- **GET /{id}** - Get specific rule
- **PUT /{id}** - Update rule (creates new version)
- **GET /{id}/versions** - Get version history
- **POST /{id}/disable** - Disable rule
- **POST /{id}/enable** - Enable rule
- **POST /{id}/hide** - Hide rule (soft delete)
- **POST /{id}/parameters** - Set active parameter set

## Built-in Rule Protection

Rules with `controller.metadata.source = "builtin"` are protected:
- Cannot be updated (HTTP 403)
- Cannot be disabled (HTTP 403)
- Cannot be deleted (HTTP 403)
- Can be read and viewed by users

## Correlation Support Constraints

From migration `002_correlation_support.up.sql`:

Valid correlation_type values:
- event_count
- value_count
- temporal
- temporal_ordered
- join
- suppression
- baseline_deviation
- missing_event

Database enforces this with CHECK constraint.

## Summary

**1. Storage**: PostgreSQL detection_schemas table with immutable versioning
**2. Structure**: Three-part JSON (model + view + controller)
**3. Detection Types**: 8 correlation types from simple counting to complex sequences
**4. Field Names**: Exclusively OCSF with dot notation
**5. Query Language**: JSON-based filter expressions with logical operators
**6. Examples**: 3 production-ready rules in alerting/rules/ directory
**7. Protection**: Built-in rules marked with metadata.source="builtin"
**8. Templates**: Alert descriptions support {{field}} variable interpolation
