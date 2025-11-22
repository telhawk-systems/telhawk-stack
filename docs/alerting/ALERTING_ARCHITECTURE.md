# TelHawk Alerting System Architecture

## Overview

The TelHawk alerting system implements a detection-as-code approach using **Detection Schemas** - JSON definitions that combine data modeling, presentation rules, and detection logic in a single immutable, versioned structure.

## Core Concepts

### Detection Schema (Model-View-Controller Pattern)

A Detection Schema consists of three JSON sections that define detection logic:

**API Request (Creating New Rule)**:
```json
{
  "model": {
    "fields": ["src_endpoint.ip", "dst_endpoint.port", "severity", "actor.user.name"],
    "group_by": ["src_endpoint.ip"],
    "time_window": "5m",
    "threshold": 10,
    "aggregation": "count"
  },
  "view": {
    "title": "SSH Brute Force Attempt",
    "severity": "high",
    "priority": "P2",
    "fields_order": ["src_endpoint.ip", "actor.user.name", "count", "time_range"],
    "description_template": "{{count}} failed SSH login attempts from {{src_endpoint.ip}} for user {{actor.user.name}}",
    "mitre_attack": {
      "tactics": ["TA0006"],
      "techniques": ["T1110.001"]
    },
    "response_guidance": "1. Verify if IP is known/authorized\n2. Check for successful logins from same IP\n3. Consider IP blocking if threshold exceeded"
  },
  "controller": {
    "query": "class_uid:3002 AND activity_id:1 AND dst_endpoint.port:22 AND status_id:2",
    "aggregation_field": "src_endpoint.ip",
    "condition": "count > 10",
    "lookback": "5m",
    "evaluation_interval": "1m"
  }
}
```

**API Response (Returned by Server)**:
```json
{
  "id": "018d3c3a-0000-7000-8000-rule00001",
  "version_id": "018d3c3a-7890-7000-8000-123456789abc",
  "version": 1,
  "created_by": "018d3c3a-1111-7000-8000-admin000001",
  "created_at": "2025-01-09T12:00:00Z",
  "disabled_at": null,
  "hidden_at": null,
  "model": { ... },
  "view": { ... },
  "controller": { ... }
}
```

**To Create a New Version** (PUT to existing rule's URL):
```http
PUT /api/v1/schemas/{id}
{
  "model": { ... },
  "view": { ... },
  "controller": { ... }
}
```
Server reuses `id` from URL, generates new `version_id`.

**Important**: Users NEVER provide `id` or `version_id` in request bodies. These are server-generated UUIDs.

### Model Section

**Purpose**: Define the data structure and aggregation logic.

**Fields**:
- `fields` (array): OCSF fields to extract from matching events
- `group_by` (array): Fields to group events by (aggregation keys)
- `time_window` (duration): Time window for aggregation (e.g., "5m", "1h", "24h")
- `threshold` (integer): Minimum count to trigger alert
- `aggregation` (string): Aggregation type - "count", "sum", "avg", "max", "min"
- `aggregation_field` (string, optional): Field to aggregate (for sum/avg/max/min)

**Use Cases**:
- Simple threshold: `threshold: 10, aggregation: count` = "More than 10 events"
- Volume-based: `aggregation: sum, aggregation_field: bytes_transferred, threshold: 1000000` = "More than 1MB transferred"
- Rate-based: `time_window: "5m", threshold: 100` = "More than 100 events in 5 minutes"

### View Section

**Purpose**: Define how alerts are presented to analysts.

**Fields**:
- `title` (string): Human-readable alert name
- `severity` (string): "critical", "high", "medium", "low", "informational"
- `priority` (string): "P1", "P2", "P3", "P4" (for case prioritization)
- `fields_order` (array): Ordered list of fields to display in alert details
- `description_template` (string): Go template for alert description (supports {{field}} interpolation)
- `mitre_attack` (object, optional):
  - `tactics` (array): MITRE ATT&CK tactic IDs (e.g., ["TA0006"])
  - `techniques` (array): MITRE ATT&CK technique IDs (e.g., ["T1110.001"])
- `response_guidance` (string): Step-by-step analyst instructions
- `color` (string, optional): Hex color for UI highlighting (defaults based on severity)
- `tags` (array, optional): Custom tags for categorization

### Controller Section

**Purpose**: Define detection logic and evaluation criteria.

**Fields**:
- `query` (string): OpenSearch query DSL or simple query string (OCSF field-based)
- `aggregation_field` (string): Field to group by (matches model.group_by[0] typically)
- `condition` (string): Boolean expression (e.g., "count > 10", "sum > 1000")
- `lookback` (duration): How far back to search (e.g., "5m", "1h")
- `evaluation_interval` (duration): How often to run this rule (default: 1m)
- `enabled` (boolean): Whether rule is actively evaluated (default: true)

**Query Syntax**:
- OCSF fields: `class_uid:3002 AND activity_id:1`
- Wildcards: `actor.user.name:admin*`
- Ranges: `severity_id:[3 TO 5]`
- Negation: `NOT status_id:1`
- Complex: `(class_uid:3002 OR class_uid:4001) AND severity:high`

## Data Models

### Detection Schema Storage (PostgreSQL)

**Table**: `detection_schemas`

```sql
CREATE TABLE detection_schemas (
    id UUID NOT NULL,                 -- Stable rule identifier (never changes)
    version_id UUID PRIMARY KEY,      -- Version-specific UUID (UUID v7)
    model JSONB NOT NULL,
    view JSONB NOT NULL,
    controller JSONB NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Lifecycle timestamps (immutable pattern)
    disabled_at TIMESTAMP,
    disabled_by UUID,
    hidden_at TIMESTAMP,
    hidden_by UUID
);

CREATE INDEX idx_schemas_id_created ON detection_schemas(id, created_at DESC);
CREATE INDEX idx_schemas_id ON detection_schemas(id);
CREATE INDEX idx_schemas_version_id ON detection_schemas(version_id);
```

**Immutability Rules**:
- Rule content (model/view/controller) is **never updated**
- To modify a rule: Create new row with same `id` but new `version_id` (UUID v7)
- Lifecycle changes (disable/hide) update only timestamp fields
- Version numbers are **calculated** using window functions (no race conditions)
- Alerts reference specific version via `version_id` for historical accuracy
- `id` groups all versions of the same logical rule (stable identifier)
- Rule title (in `view.title`) can change between versions

**Versioning Example**:
```sql
-- Version numbers calculated on read
SELECT
    id,
    version_id,
    view->>'title' as title,
    ROW_NUMBER() OVER (PARTITION BY id ORDER BY created_at) as version
FROM detection_schemas
WHERE id = '018d3c3a-0000-7000-8000-rule00001'
ORDER BY created_at DESC;

-- Results:
-- version_id: 018d3c3a-3333-..., title: "SSH Brute Force Detection", version: 3 (active)
-- version_id: 018d3c3a-2222-..., title: "SSH Brute Force Attempt", version: 2 (disabled)
-- version_id: 018d3c3a-1111-..., title: "SSH Brute Force", version: 1 (hidden)
```

### Alert Storage (OpenSearch)

**Index Pattern**: `telhawk-alerts-YYYY.MM.DD`

**Document Structure**:
```json
{
  "alert_id": "018d3c3a-7890-7000-8000-123456789abc",
  "detection_schema_id": "018d3c3a-0000-7000-8000-rule00001",
  "detection_schema_version_id": "018d3c3a-1234-7000-8000-abcdef123456",
  "detection_schema_title": "SSH Brute Force Attempt",
  "case_id": "018d3c3a-5678-7000-8000-fedcba654321",

  "title": "SSH Brute Force Attempt",
  "description": "15 failed SSH login attempts from 192.168.1.100 for user admin",
  "severity": "high",
  "priority": "P2",
  "status": "open",

  "triggered_at": "2025-01-09T12:00:00Z",
  "event_count": 15,
  "matched_events": [
    "018d3c3a-1111-7000-8000-event0001",
    "018d3c3a-1112-7000-8000-event0002"
  ],

  "fields": {
    "src_endpoint.ip": "192.168.1.100",
    "actor.user.name": "admin",
    "dst_endpoint.port": 22,
    "count": 15,
    "time_range": {
      "start": "2025-01-09T11:55:00Z",
      "end": "2025-01-09T12:00:00Z"
    }
  },

  "mitre_attack": {
    "tactics": ["TA0006"],
    "techniques": ["T1110.001"]
  },

  "metadata": {
    "evaluation_duration_ms": 45,
    "matched_query": "class_uid:3002 AND activity_id:1...",
    "aggregation_key": "192.168.1.100"
  }
}
```

**Mapping**:
```json
{
  "mappings": {
    "properties": {
      "alert_id": {"type": "keyword"},
      "detection_schema_id": {"type": "keyword"},
      "detection_schema_name": {"type": "keyword"},
      "detection_schema_version": {"type": "integer"},
      "case_id": {"type": "keyword"},
      "title": {"type": "text"},
      "description": {"type": "text"},
      "severity": {"type": "keyword"},
      "priority": {"type": "keyword"},
      "status": {"type": "keyword"},
      "triggered_at": {"type": "date"},
      "event_count": {"type": "integer"},
      "matched_events": {"type": "keyword"},
      "fields": {"type": "object", "enabled": true},
      "mitre_attack": {
        "properties": {
          "tactics": {"type": "keyword"},
          "techniques": {"type": "keyword"}
        }
      },
      "metadata": {"type": "object", "enabled": true}
    }
  }
}
```

### Cases Storage (PostgreSQL)

**Table**: `cases`

```sql
CREATE TABLE cases (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    severity VARCHAR(20) NOT NULL,
    priority VARCHAR(10) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    created_by UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    assigned_to UUID,

    -- Lifecycle timestamps
    resolved_at TIMESTAMP,
    resolved_by UUID,
    closed_at TIMESTAMP,
    closed_by UUID,

    metadata JSONB
);
```

**Table**: `case_alerts` (Many-to-Many)

```sql
CREATE TABLE case_alerts (
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    alert_id UUID NOT NULL,                    -- OpenSearch alert document ID
    detection_schema_id UUID NOT NULL,         -- Stable rule ID for grouping
    detection_schema_version_id UUID NOT NULL, -- Specific version that triggered alert
    added_at TIMESTAMP NOT NULL DEFAULT NOW(),
    added_by UUID,
    PRIMARY KEY (case_id, alert_id)
);

CREATE INDEX idx_case_alerts_schema_id ON case_alerts(detection_schema_id);
CREATE INDEX idx_case_alerts_version_id ON case_alerts(detection_schema_version_id);
```

**Status Workflow**:
```
open → investigating → resolved → closed
  ↓          ↓            ↓
  └─────────└────────────└──> false_positive (terminal state)
```

## Service Architecture

### Respond Service (Port 8085)

The `respond` service consolidates detection rules, alerting, and case management into a single service.

**Responsibilities**:
- CRUD operations for Detection Schemas
- Schema versioning and lifecycle management
- Schema validation
- Periodic evaluation of Detection Schemas
- Query OpenSearch for events
- Apply aggregation and threshold logic
- Create alerts in OpenSearch
- Manage cases in PostgreSQL
- Auto-group related alerts into cases

**API Endpoints - Detection Schemas**:
```
POST   /api/v1/schemas          Create new schema (auto-version)
GET    /api/v1/schemas          List active schemas
GET    /api/v1/schemas/:id      Get latest version
GET    /api/v1/schemas/:id/versions  Get version history
PUT    /api/v1/schemas/:id/disable   Disable schema
PUT    /api/v1/schemas/:id/enable    Re-enable schema
DELETE /api/v1/schemas/:id      Soft delete (hide)
GET    /api/v1/schemas/:id/test      Test schema against historical data
```

**API Endpoints - Alerts and Cases**:
```
GET    /api/v1/alerts           List alerts (paginated, filterable)
GET    /api/v1/alerts/:id       Get alert details
POST   /api/v1/alerts/:id/test  Test rule against historical data (replay)
PUT    /api/v1/alerts/:id       Update alert status

GET    /api/v1/cases            List cases
POST   /api/v1/cases            Create case manually
GET    /api/v1/cases/:id        Get case details
PUT    /api/v1/cases/:id        Update case (assign, change status)
POST   /api/v1/cases/:id/alerts Link alert to case
DELETE /api/v1/cases/:id/alerts/:alert_id  Unlink alert
```

**Database**: `respond-db` (PostgreSQL)

**Components**:

1. **Schema Cache**:
   - In-memory cache of active Detection Schemas
   - TTL: 5 minutes (configurable)
   - Refresh on schema changes
   - Pre-compiled query templates

2. **Evaluation Scheduler**:
   - Runs every 60 seconds (configurable)
   - Evaluates all enabled schemas
   - Parallel evaluation with worker pool
   - Rate limiting per schema

3. **Query Builder**:
   - Converts `controller.query` to OpenSearch DSL
   - Applies time window from `model.time_window`
   - Supports aggregations (terms, sum, avg, etc.)
   - Handles field mappings

4. **Threshold Evaluator**:
   - Parses `controller.condition` expressions
   - Evaluates boolean conditions
   - Supports: `>`, `<`, `>=`, `<=`, `==`, `!=`
   - Supports logical operators: `AND`, `OR`

5. **Alert Generator**:
   - Renders `view.description_template` with field values
   - Creates alert document in OpenSearch
   - Links to case (auto-create or existing)
   - Deduplication (don't re-alert same condition within window)

6. **Case Manager**:
   - Auto-creates cases for new alerts
   - Groups related alerts (configurable grouping rules)
   - Updates case metadata (alert count, severity, etc.)

## Evaluation Flow

### Step 1: Load Active Schemas
```
Respond Service startup:
1. Load all active Detection Schemas from database
2. Cache in memory with TTL
3. Pre-compile query templates
4. Start evaluation scheduler
```

### Step 2: Scheduled Evaluation
```
Every 1 minute (configurable):
1. For each cached schema:
   a. Check if evaluation_interval elapsed
   b. Build OpenSearch query from controller.query
   c. Apply time window (now - lookback to now)
   d. Execute aggregation query
   e. Evaluate threshold condition
   f. If triggered → create alert
```

### Step 3: Query Building
```
Input:  controller.query = "class_uid:3002 AND dst_endpoint.port:22"
        model.time_window = "5m"
        model.group_by = ["src_endpoint.ip"]

Output: OpenSearch DSL
{
  "query": {
    "bool": {
      "must": [
        {"term": {"class_uid": 3002}},
        {"term": {"dst_endpoint.port": 22}},
        {"range": {"time": {"gte": "now-5m", "lte": "now"}}}
      ]
    }
  },
  "aggs": {
    "grouped": {
      "terms": {"field": "src_endpoint.ip.keyword", "size": 1000},
      "aggs": {
        "count": {"value_count": {"field": "_id"}}
      }
    }
  },
  "size": 0
}
```

### Step 4: Threshold Evaluation
```
Input:  aggregation_result = {
          "192.168.1.100": {"count": 15},
          "192.168.1.101": {"count": 3}
        }
        controller.condition = "count > 10"

Output: Triggered groups = ["192.168.1.100"]
```

### Step 5: Alert Creation
```
For each triggered group:
1. Render view.description_template with field values
2. Create alert document in OpenSearch (telhawk-alerts-*)
3. Check for existing case (grouping logic):
   - Same detection_schema_name within 24 hours?
   - Same src_endpoint.ip within 1 hour?
   - Manual case assignment?
4. Create new case or link to existing
5. Update case metadata (alert count, latest alert time)
```

### Step 6: Deduplication
```
Before creating alert:
1. Check for duplicate alert in last N minutes (configurable)
2. Query: same detection_schema_id + same aggregation_key + triggered_at within window
3. If duplicate exists → skip alert creation (or update existing)
```

## Example Detection Schemas

### 1. Failed Login Spike
```json
{
  "id": "018d3c3a-rule-7000-8000-failedlogin1",
  "model": {
    "fields": ["actor.user.name", "src_endpoint.ip", "status_detail"],
    "group_by": ["actor.user.name"],
    "time_window": "5m",
    "threshold": 5,
    "aggregation": "count"
  },
  "view": {
    "title": "Multiple Failed Login Attempts",
    "severity": "medium",
    "priority": "P3",
    "description_template": "User {{actor.user.name}} had {{count}} failed login attempts from {{src_endpoint.ip}}"
  },
  "controller": {
    "query": "class_uid:3002 AND activity_id:1 AND status_id:2",
    "aggregation_field": "actor.user.name",
    "condition": "count >= 5",
    "lookback": "5m"
  }
}
```

### 2. Large Data Exfiltration
```json
{
  "id": "018d3c3a-rule-7000-8000-datatransfer",
  "model": {
    "fields": ["src_endpoint.ip", "dst_endpoint.ip", "traffic.bytes"],
    "group_by": ["src_endpoint.ip", "dst_endpoint.ip"],
    "time_window": "1h",
    "threshold": 10737418240,
    "aggregation": "sum",
    "aggregation_field": "traffic.bytes"
  },
  "view": {
    "title": "Large Data Transfer Detected",
    "severity": "high",
    "priority": "P2",
    "description_template": "{{src_endpoint.ip}} transferred {{traffic.bytes_human}} to {{dst_endpoint.ip}} in 1 hour",
    "mitre_attack": {
      "tactics": ["TA0010"],
      "techniques": ["T1048"]
    }
  },
  "controller": {
    "query": "class_uid:4001 AND traffic.bytes:>0",
    "aggregation_field": "traffic.bytes",
    "condition": "sum > 10737418240",
    "lookback": "1h"
  }
}
```

### 3. Unusual Process Execution
```json
{
  "id": "018d3c3a-rule-7000-8000-suspiciousproc",
  "model": {
    "fields": ["process.name", "process.parent_process.name", "device.hostname"],
    "group_by": ["process.name", "process.parent_process.name"],
    "time_window": "24h",
    "threshold": 1,
    "aggregation": "count"
  },
  "view": {
    "title": "Suspicious Process Execution",
    "severity": "high",
    "priority": "P1",
    "description_template": "Unusual process chain: {{process.parent_process.name}} spawned {{process.name}} on {{device.hostname}}",
    "response_guidance": "1. Verify process legitimacy\n2. Check process hash against VirusTotal\n3. Isolate host if malicious"
  },
  "controller": {
    "query": "class_uid:1007 AND process.parent_process.name:(cmd.exe OR powershell.exe) AND process.name:(certutil.exe OR bitsadmin.exe OR mshta.exe)",
    "aggregation_field": "process.name",
    "condition": "count >= 1",
    "lookback": "24h"
  }
}
```

## Testing and Validation

### Rule Testing (Replay)

API endpoint: `POST /api/v1/alerts/:schema_id/test`

**Request**:
```json
{
  "time_range": {
    "from": "2025-01-09T00:00:00Z",
    "to": "2025-01-09T23:59:59Z"
  },
  "dry_run": true
}
```

**Response**:
```json
{
  "schema_id": "018d3c3a-7890-7000-8000-123456789abc",
  "schema_name": "brute_force_ssh",
  "time_range": {
    "from": "2025-01-09T00:00:00Z",
    "to": "2025-01-09T23:59:59Z"
  },
  "would_trigger": true,
  "trigger_count": 3,
  "triggers": [
    {
      "triggered_at": "2025-01-09T08:15:00Z",
      "aggregation_key": "192.168.1.100",
      "event_count": 15,
      "fields": {"src_endpoint.ip": "192.168.1.100", "count": 15}
    },
    {
      "triggered_at": "2025-01-09T14:30:00Z",
      "aggregation_key": "10.0.0.50",
      "event_count": 12,
      "fields": {"src_endpoint.ip": "10.0.0.50", "count": 12}
    }
  ],
  "total_events_matched": 1247,
  "evaluation_duration_ms": 125
}
```

### Schema Validation

When creating/updating Detection Schemas:

1. **JSON Schema Validation**: Ensure model/view/controller structure is correct
2. **Query Syntax Validation**: Parse controller.query to ensure valid OpenSearch query
3. **Field Existence Check**: Verify referenced OCSF fields exist in schema
4. **Template Validation**: Ensure view.description_template references valid fields
5. **Condition Parsing**: Validate controller.condition is parseable boolean expression

## Performance Considerations

### Caching Strategy
- Detection Schemas cached in-memory (5-minute TTL)
- Alert deduplication cache (Redis or in-memory, 1-hour TTL)
- Case grouping cache (recent cases, 24-hour window)

### Query Optimization
- Use OpenSearch aggregations (not post-processing)
- Limit lookback windows (default max: 24h)
- Pre-compile query templates
- Parallel evaluation with worker pool

### Scalability
- Horizontal scaling: Multiple Respond Service instances
- Distributed work queue (uses NATS message broker)
- Partition by schema priority (high-priority rules evaluated first)

## Configuration

### Respond Service

`respond/config.yaml`:
```yaml
server:
  port: 8085

database:
  postgres:
    host: respond-db
    port: 5432
    database: telhawk_respond
    user: telhawk
    password: ${RESPOND_DB_PASSWORD}
    sslmode: require

opensearch:
  url: https://opensearch:9200
  username: admin
  password: ${OPENSEARCH_PASSWORD}
  alerts_index_prefix: telhawk-alerts

validation:
  max_time_window: 24h
  max_threshold: 100000
  allowed_aggregations:
    - count
    - sum
    - avg
    - max
    - min

evaluation:
  enabled: true
  interval: 1m
  worker_pool_size: 10
  max_concurrent_schemas: 50
  query_timeout: 30s

deduplication:
  enabled: true
  window: 1h
  cache_type: memory

case_grouping:
  enabled: true
  time_window: 24h
  group_by_schema: true
  group_by_field: src_endpoint.ip
```
