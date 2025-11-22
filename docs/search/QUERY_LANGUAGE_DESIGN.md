# TelHawk Query Language Design

**Last Updated:** 2025-11-09
**Status:** Design Document (Not Yet Implemented)

## Executive Summary

TelHawk uses a **JSON-based query language** as the canonical representation for all search, filtering, and analysis operations. This design choice provides:

1. **Storage abstraction** - Same query language works against OpenSearch (hot/warm), S3 (cold), and future backends
2. **Type safety** - Structured format prevents injection attacks and enables validation
3. **Forward compatibility** - New features (aggregations, joins) extend existing structure
4. **Multi-tier efficiency** - Query structure supports partition pruning, column projection, predicate pushdown

**User-facing syntax options:**
- **Filter chips (UI)** - Visual query builder generates JSON (90% of users)
- **Text syntax** - Hand-written queries like `severity:high user:admin` parse to JSON (power users)
- **Raw JSON** - Direct JSON editing for complex queries (experts, saved searches, APIs)

**All paths converge to canonical JSON**, which backends translate to native query formats.

---

## Core Principles

### 1. JSON is the Constitution

Every query, regardless of how it's authored, translates to a canonical JSON structure:

```json
{
  "select": [".time", ".severity", ".actor.user.name", ".src_endpoint.ip"],
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".class_uid", "operator": "eq", "value": 3002},
      {"field": ".severity_id", "operator": "gte", "value": 4}
    ]
  },
  "timeRange": {
    "start": "2025-01-01T00:00:00Z",
    "end": "2025-01-31T23:59:59Z"
  },
  "sort": [
    {"field": ".time", "order": "desc"}
  ],
  "limit": 100
}
```

### 2. OCSF-First Field References

All field references use **jq-style OCSF paths** starting with `.`:

- `.time` - Event timestamp (OCSF required field)
- `.severity` - Human-readable severity (Info, Low, Medium, High, Critical)
- `.severity_id` - Numeric severity (1-5)
- `.class_uid` - OCSF event class (3002 = Authentication, 4001 = Network, etc.)
- `.actor.user.name` - Nested object navigation
- `.src_endpoint.ip` - Source IP address
- `.attacks[0].tactic.name` - Array indexing for MITRE ATT&CK data

**Why jq-style?**
- Familiar to DevOps/SRE users
- Intuitive for nested JSON structures
- Distinguishes field paths from string literals
- Concise and readable

### 3. Storage Backend Agnostic

The query language is designed to work efficiently across multiple storage tiers:

| Storage Tier | Technology | Query Translation |
|--------------|-----------|-------------------|
| **Hot** (0-7 days) | OpenSearch | JSON → OpenSearch Query DSL |
| **Warm** (8-90 days) | OpenSearch (read-only) | JSON → OpenSearch Query DSL |
| **Cold** (91+ days) | S3 + Parquet | JSON → Partition filters + Parquet predicates |
| **Future** | ClickHouse, DuckDB, etc. | JSON → Native query format |

---

## JSON Query Structure

### Complete Query Schema

```json
{
  "select": ["<field_path>", ...],           // Optional: field projection
  "filter": {<filter_expression>},            // Optional: WHERE clause
  "timeRange": {<time_range>},                // Required for most queries
  "aggregations": [<aggregation_spec>],       // Optional: GROUP BY / stats
  "sort": [<sort_spec>],                      // Optional: ORDER BY
  "limit": <number>,                          // Optional: result limit (default: 100)
  "offset": <number>,                         // Optional: pagination offset
  "cursor": "<cursor_token>"                  // Optional: cursor-based pagination
}
```

### Field Projection (`select`)

**Purpose:** Specify which OCSF fields to return. Enables column pruning in Parquet, reduces network transfer.

**Syntax:**
```json
{
  "select": [".time", ".severity", ".actor.user.name", ".src_endpoint.ip"]
}
```

**Special cases:**
- **Omit `select`** - Return all fields (default behavior)
- **Wildcards** - `.actor.user.*` returns all user fields (future enhancement)
- **Computed fields** - `.severity_id * 10` for calculations (future enhancement)

**OCSF-aware defaults** (when `select` is omitted and event class is filtered):

| Event Class | Default Fields |
|-------------|----------------|
| Authentication (3002) | `.time`, `.severity`, `.actor.user.name`, `.src_endpoint.ip`, `.status`, `.auth_protocol.name` |
| Network (4001) | `.time`, `.severity`, `.src_endpoint.ip`, `.src_endpoint.port`, `.dst_endpoint.ip`, `.dst_endpoint.port`, `.protocol` |
| Process (1007) | `.time`, `.severity`, `.process.name`, `.process.pid`, `.process.cmd_line`, `.actor.user.name` |
| Detection (2004) | `.time`, `.severity`, `.finding.title`, `.attacks[0].tactic.name`, `.attacks[0].technique.name`, `.risk_score` |

### Filter Expressions (`filter`)

**Purpose:** Define which events to include (WHERE clause).

#### Basic Condition

```json
{
  "field": "<field_path>",
  "operator": "<operator>",
  "value": <value>
}
```

**Supported operators:**

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equals | `{"field": ".severity", "operator": "eq", "value": "High"}` |
| `ne` | Not equals | `{"field": ".status", "operator": "ne", "value": "Success"}` |
| `gt` | Greater than | `{"field": ".severity_id", "operator": "gt", "value": 3}` |
| `gte` | Greater than or equal | `{"field": ".severity_id", "operator": "gte", "value": 4}` |
| `lt` | Less than | `{"field": ".risk_score", "operator": "lt", "value": 50}` |
| `lte` | Less than or equal | `{"field": ".risk_score", "operator": "lte", "value": 75}` |
| `in` | In array | `{"field": ".status", "operator": "in", "value": ["Failed", "Locked"]}` |
| `contains` | String contains | `{"field": ".process.cmd_line", "operator": "contains", "value": "mimikatz"}` |
| `startsWith` | String starts with | `{"field": ".file.path", "operator": "startsWith", "value": "/etc/"}` |
| `endsWith` | String ends with | `{"field": ".file.path", "operator": "endsWith", "value": ".exe"}` |
| `regex` | Regular expression | `{"field": ".src_endpoint.ip", "operator": "regex", "value": "^10\\."}` |
| `exists` | Field exists | `{"field": ".actor.user.name", "operator": "exists", "value": true}` |
| `cidr` | IP in CIDR range | `{"field": ".src_endpoint.ip", "operator": "cidr", "value": "192.168.0.0/16"}` |

#### Compound Filters

**AND logic:**
```json
{
  "type": "and",
  "conditions": [
    {"field": ".class_uid", "operator": "eq", "value": 3002},
    {"field": ".severity_id", "operator": "gte", "value": 4},
    {"field": ".status", "operator": "eq", "value": "Failed"}
  ]
}
```

**OR logic:**
```json
{
  "type": "or",
  "conditions": [
    {"field": ".severity", "operator": "eq", "value": "High"},
    {"field": ".severity", "operator": "eq", "value": "Critical"}
  ]
}
```

**NOT logic:**
```json
{
  "type": "not",
  "condition": {"field": ".actor.user.name", "operator": "eq", "value": "system"}
}
```

**Nested logic:**
```json
{
  "type": "and",
  "conditions": [
    {"field": ".class_uid", "operator": "eq", "value": 3002},
    {
      "type": "or",
      "conditions": [
        {"field": ".severity_id", "operator": "eq", "value": 4},
        {"field": ".severity_id", "operator": "eq", "value": 5}
      ]
    }
  ]
}
```

### Time Range (`timeRange`)

**Purpose:** Filter events by timestamp. Critical for partition pruning in S3.

**Absolute time:**
```json
{
  "timeRange": {
    "start": "2025-01-01T00:00:00Z",
    "end": "2025-01-31T23:59:59Z"
  }
}
```

**Relative time:**
```json
{
  "timeRange": {
    "last": "1h"  // Supported: 15m, 1h, 24h, 7d, 30d, 90d
  }
}
```

**Open-ended:**
```json
{
  "timeRange": {
    "start": "2025-01-01T00:00:00Z"  // No end = up to now
  }
}
```

**Translation to storage tiers:**
- **OpenSearch**: Converted to `range` query on `@timestamp` field
- **S3**: Used for partition pruning (`year=2025/month=01/day=15/`)

### Aggregations

**Purpose:** Group events and compute statistics (GROUP BY, COUNT, AVG, etc.).

**Count by field:**
```json
{
  "aggregations": [
    {
      "type": "terms",
      "field": ".actor.user.name",
      "name": "top_users",
      "size": 10
    }
  ]
}
```

**Time histogram:**
```json
{
  "aggregations": [
    {
      "type": "date_histogram",
      "field": ".time",
      "name": "events_over_time",
      "interval": "1h"
    }
  ]
}
```

**Metrics:**
```json
{
  "aggregations": [
    {
      "type": "avg",
      "field": ".risk_score",
      "name": "avg_risk"
    },
    {
      "type": "max",
      "field": ".risk_score",
      "name": "max_risk"
    }
  ]
}
```

**Nested aggregations:**
```json
{
  "aggregations": [
    {
      "type": "terms",
      "field": ".severity",
      "name": "by_severity",
      "size": 5,
      "aggregations": [
        {
          "type": "terms",
          "field": ".actor.user.name",
          "name": "top_users_per_severity",
          "size": 3
        }
      ]
    }
  ]
}
```

**Supported aggregation types:**
- `terms` - Group by discrete values
- `date_histogram` - Group by time buckets
- `avg`, `sum`, `min`, `max` - Numeric metrics
- `stats` - All metrics at once (count, avg, sum, min, max)
- `cardinality` - Unique count

### Sorting (`sort`)

**Purpose:** Order results (ORDER BY).

**Single field:**
```json
{
  "sort": [
    {"field": ".time", "order": "desc"}
  ]
}
```

**Multiple fields:**
```json
{
  "sort": [
    {"field": ".severity_id", "order": "desc"},
    {"field": ".time", "order": "desc"}
  ]
}
```

**Default:** If omitted, defaults to `[{"field": ".time", "order": "desc"}]`

### Pagination

**Limit/Offset (simple pagination):**
```json
{
  "limit": 100,
  "offset": 200  // Skip first 200 results
}
```

**Cursor-based (recommended for deep pagination):**
```json
{
  "limit": 100,
  "cursor": "eyJzb3J0IjpbMTczNjQ2NzIwMDAwMF19"  // Opaque cursor from previous response
}
```

**Response includes next cursor:**
```json
{
  "events": [...],
  "total": 45000,
  "cursor": "eyJzb3J0IjpbMTczNjQ2NzIwMDEwMF19"
}
```

---

## Text Syntax → JSON Translation

### Design Philosophy

Text syntax is **syntactic sugar** that parses to canonical JSON. It should:
- Be intuitive for analysts familiar with Splunk/Elastic
- Support common use cases (80% of queries)
- Gracefully fall back to JSON for complex queries

### Basic Field:Value Syntax

**Text:**
```
severity:high
```

**JSON:**
```json
{
  "filter": {
    "field": ".severity",
    "operator": "eq",
    "value": "High"
  }
}
```

### Multiple Conditions (AND)

**Text:**
```
severity:high status:failed user:jsmith
```

**JSON:**
```json
{
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".severity", "operator": "eq", "value": "High"},
      {"field": ".status", "operator": "eq", "value": "Failed"},
      {"field": ".actor.user.name", "operator": "eq", "value": "jsmith"}
    ]
  }
}
```

### OR Logic

**Text:**
```
severity:high OR severity:critical
```

**JSON:**
```json
{
  "filter": {
    "type": "or",
    "conditions": [
      {"field": ".severity", "operator": "eq", "value": "High"},
      {"field": ".severity", "operator": "eq", "value": "Critical"}
    ]
  }
}
```

### NOT Logic

**Text:**
```
NOT user:system
```

**JSON:**
```json
{
  "filter": {
    "type": "not",
    "condition": {"field": ".actor.user.name", "operator": "eq", "value": "system"}
  }
}
```

### Comparison Operators

**Text:**
```
severity_id>=4 risk_score<50
```

**JSON:**
```json
{
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".severity_id", "operator": "gte", "value": 4},
      {"field": ".risk_score", "operator": "lt", "value": 50}
    ]
  }
}
```

**Supported operators in text syntax:**
- `:` - Equals (default)
- `:!` - Not equals
- `:>` - Greater than
- `:>=` - Greater than or equal
- `:<` - Less than
- `:<=` - Less than or equal

### Wildcards and Patterns

**Text:**
```
file.path:/etc/* cmd_line:*mimikatz*
```

**JSON:**
```json
{
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".file.path", "operator": "startsWith", "value": "/etc/"},
      {"field": ".process.cmd_line", "operator": "contains", "value": "mimikatz"}
    ]
  }
}
```

### CIDR Notation

**Text:**
```
src_ip:192.168.0.0/16
```

**JSON:**
```json
{
  "filter": {
    "field": ".src_endpoint.ip",
    "operator": "cidr",
    "value": "192.168.0.0/16"
  }
}
```

### Field Aliases

Common field name aliases for user convenience:

| Alias | OCSF Path |
|-------|-----------|
| `user` | `.actor.user.name` |
| `src_ip` | `.src_endpoint.ip` |
| `dst_ip` | `.dst_endpoint.ip` |
| `src_port` | `.src_endpoint.port` |
| `dst_port` | `.dst_endpoint.port` |
| `file` | `.file.path` |
| `process` | `.process.name` |
| `cmd` | `.process.cmd_line` |
| `host` | `.device.hostname` |

### Grouping and Precedence

**Text:**
```
severity:high AND (user:admin OR user:root)
```

**JSON:**
```json
{
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".severity", "operator": "eq", "value": "High"},
      {
        "type": "or",
        "conditions": [
          {"field": ".actor.user.name", "operator": "eq", "value": "admin"},
          {"field": ".actor.user.name", "operator": "eq", "value": "root"}
        ]
      }
    ]
  }
}
```

### Complex Example

**Text:**
```
class_uid:3002 status:failed severity_id>=4 (src_ip:10.0.0.0/8 OR src_ip:192.168.0.0/16)
```

**JSON:**
```json
{
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".class_uid", "operator": "eq", "value": 3002},
      {"field": ".status", "operator": "eq", "value": "Failed"},
      {"field": ".severity_id", "operator": "gte", "value": 4},
      {
        "type": "or",
        "conditions": [
          {"field": ".src_endpoint.ip", "operator": "cidr", "value": "10.0.0.0/8"},
          {"field": ".src_endpoint.ip", "operator": "cidr", "value": "192.168.0.0/16"}
        ]
      }
    ]
  }
}
```

---

## Filter Chip → JSON Mapping

### UI Filter Chip State

```javascript
// Frontend state
const activeFilters = [
  { type: 'event_class', value: 3002 },
  { type: 'field', field: 'status', operator: 'eq', value: 'Failed' },
  { type: 'field', field: 'user', operator: 'eq', value: 'jsmith' }
];
```

### Translation to JSON Query

```json
{
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".class_uid", "operator": "eq", "value": 3002},
      {"field": ".status", "operator": "eq", "value": "Failed"},
      {"field": ".actor.user.name", "operator": "eq", "value": "jsmith"}
    ]
  },
  "select": [".time", ".severity", ".actor.user.name", ".src_endpoint.ip", ".status", ".auth_protocol.name"]
}
```

**Note:** `select` clause auto-populated based on event class defaults when event class filter is active.

### Multiple Values for Same Field (OR Logic)

**UI chips:**
```
[Status: Failed] [Status: Locked]
```

**JSON:**
```json
{
  "filter": {
    "type": "or",
    "conditions": [
      {"field": ".status", "operator": "eq", "value": "Failed"},
      {"field": ".status", "operator": "eq", "value": "Locked"}
    ]
  }
}
```

**Optimization:** Equivalent to `{"field": ".status", "operator": "in", "value": ["Failed", "Locked"]}`

---

## Backend Translation

### OpenSearch Query DSL

**Input (JSON query):**
```json
{
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".class_uid", "operator": "eq", "value": 3002},
      {"field": ".severity_id", "operator": "gte", "value": 4}
    ]
  },
  "timeRange": {
    "last": "1h"
  },
  "sort": [{"field": ".time", "order": "desc"}],
  "limit": 100
}
```

**Output (OpenSearch DSL):**
```json
{
  "query": {
    "bool": {
      "must": [
        {"term": {"class_uid": 3002}},
        {"range": {"severity_id": {"gte": 4}}},
        {"range": {"time": {"gte": "now-1h"}}}
      ]
    }
  },
  "sort": [{"time": {"order": "desc"}}],
  "size": 100
}
```

**Field mapping:**
- Remove `.` prefix from field paths
- Map OCSF field names to OpenSearch field names
- Handle nested object paths (`.actor.user.name` → `actor.user.name`)

**Operator mapping:**

| Query Language | OpenSearch DSL |
|----------------|----------------|
| `eq` | `{"term": {field: value}}` |
| `ne` | `{"bool": {"must_not": {"term": {field: value}}}}` |
| `gt/gte/lt/lte` | `{"range": {field: {op: value}}}` |
| `in` | `{"terms": {field: [values]}}` |
| `contains` | `{"wildcard": {field: "*value*"}}` |
| `regex` | `{"regexp": {field: value}}` |
| `exists` | `{"exists": {"field": field}}` |
| `cidr` | `{"term": {field: value}}` (OpenSearch handles CIDR) |

### S3 + Parquet Translation

**Purpose:** Translate query to partition filters + Parquet predicates for cold storage.

**Input (JSON query):**
```json
{
  "select": [".time", ".severity", ".actor.user.name"],
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".class_uid", "operator": "eq", "value": 3002},
      {"field": ".severity_id", "operator": "gte", "value": 4}
    ]
  },
  "timeRange": {
    "start": "2025-01-15T00:00:00Z",
    "end": "2025-01-17T23:59:59Z"
  }
}
```

**Translation steps:**

1. **Partition Pruning** (from `timeRange`):
   - Scan only: `s3://bucket/year=2025/month=01/day=15/`, `day=16/`, `day=17/`
   - Skip all other partitions

2. **Column Projection** (from `select`):
   - Read only: `time`, `severity`, `actor.user.name` columns from Parquet
   - Skip all other columns (massive I/O savings)

3. **Predicate Pushdown** (from `filter`):
   - Parquet row group filter: `class_uid = 3002 AND severity_id >= 4`
   - Only read row groups matching predicates

4. **Query Execution** (pseudo-SQL):
   ```sql
   SELECT time, severity, actor.user.name
   FROM s3://bucket/year=2025/month=01/day=*/
   WHERE class_uid = 3002 AND severity_id >= 4
   ```

**Future implementation:** Use DuckDB, AWS Athena, or Trino for S3 queries.

---

## Complete Examples

### Example 1: Failed Authentication Hunt

**Use case:** Find all failed login attempts from external IPs in the last 24 hours.

**Filter chips (UI):**
```
[🔐 Authentication] [Status: Failed] [Severity: High]
```

**Text syntax:**
```
class_uid:3002 status:failed severity:high NOT src_ip:10.0.0.0/8
```

**JSON (canonical):**
```json
{
  "select": [".time", ".severity", ".actor.user.name", ".src_endpoint.ip", ".status", ".auth_protocol.name"],
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".class_uid", "operator": "eq", "value": 3002},
      {"field": ".status", "operator": "eq", "value": "Failed"},
      {"field": ".severity", "operator": "eq", "value": "High"},
      {
        "type": "not",
        "condition": {"field": ".src_endpoint.ip", "operator": "cidr", "value": "10.0.0.0/8"}
      }
    ]
  },
  "timeRange": {"last": "24h"},
  "sort": [{"field": ".time", "order": "desc"}],
  "limit": 100
}
```

**OpenSearch DSL:**
```json
{
  "query": {
    "bool": {
      "must": [
        {"term": {"class_uid": 3002}},
        {"term": {"status": "Failed"}},
        {"term": {"severity": "High"}},
        {"range": {"time": {"gte": "now-24h"}}}
      ],
      "must_not": [
        {"term": {"src_endpoint.ip": "10.0.0.0/8"}}
      ]
    }
  },
  "_source": ["time", "severity", "actor.user.name", "src_endpoint.ip", "status", "auth_protocol.name"],
  "sort": [{"time": {"order": "desc"}}],
  "size": 100
}
```

### Example 2: Lateral Movement Detection

**Use case:** Find network connections where internal IP appears as both source and destination.

**Text syntax:**
```
class_uid:4001 src_ip:192.168.0.0/16 dst_ip:192.168.0.0/16 dst_port:445 OR dst_port:3389
```

**JSON:**
```json
{
  "select": [".time", ".src_endpoint.ip", ".src_endpoint.port", ".dst_endpoint.ip", ".dst_endpoint.port", ".protocol"],
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".class_uid", "operator": "eq", "value": 4001},
      {"field": ".src_endpoint.ip", "operator": "cidr", "value": "192.168.0.0/16"},
      {"field": ".dst_endpoint.ip", "operator": "cidr", "value": "192.168.0.0/16"},
      {
        "type": "or",
        "conditions": [
          {"field": ".dst_endpoint.port", "operator": "eq", "value": 445},
          {"field": ".dst_endpoint.port", "operator": "eq", "value": 3389}
        ]
      }
    ]
  },
  "timeRange": {"last": "1h"},
  "sort": [{"field": ".time", "order": "desc"}],
  "limit": 500
}
```

### Example 3: MITRE ATT&CK Tactic Analysis

**Use case:** Count detection findings by tactic over the last 7 days.

**JSON (with aggregations):**
```json
{
  "filter": {
    "field": ".class_uid",
    "operator": "eq",
    "value": 2004
  },
  "timeRange": {"last": "7d"},
  "aggregations": [
    {
      "type": "terms",
      "field": ".attacks[0].tactic.name",
      "name": "by_tactic",
      "size": 20,
      "aggregations": [
        {
          "type": "terms",
          "field": ".severity",
          "name": "by_severity",
          "size": 5
        }
      ]
    }
  ]
}
```

**Response:**
```json
{
  "aggregations": {
    "by_tactic": {
      "buckets": [
        {
          "key": "Lateral Movement",
          "count": 145,
          "by_severity": {
            "buckets": [
              {"key": "Critical", "count": 45},
              {"key": "High", "count": 78},
              {"key": "Medium", "count": 22}
            ]
          }
        },
        {
          "key": "Credential Access",
          "count": 89,
          "by_severity": {...}
        }
      ]
    }
  }
}
```

### Example 4: Suspicious Process Activity

**Use case:** Find processes with unusual command lines.

**Text syntax:**
```
class_uid:1007 cmd_line:*mimikatz* OR cmd_line:*invoke-* OR cmd_line:*powershell* -enc
```

**JSON:**
```json
{
  "select": [".time", ".process.name", ".process.cmd_line", ".actor.user.name", ".device.hostname"],
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".class_uid", "operator": "eq", "value": 1007},
      {
        "type": "or",
        "conditions": [
          {"field": ".process.cmd_line", "operator": "contains", "value": "mimikatz"},
          {"field": ".process.cmd_line", "operator": "contains", "value": "invoke-"},
          {"field": ".process.cmd_line", "operator": "contains", "value": "powershell -enc"}
        ]
      }
    ]
  },
  "timeRange": {"last": "24h"},
  "sort": [{"field": ".time", "order": "desc"}],
  "limit": 100
}
```

---

## Implementation Roadmap

### Phase 1: JSON Query Foundation (Current Sprint)

**Goals:**
- Define canonical JSON query structure
- Implement JSON → OpenSearch DSL translator
- Update search service API to accept JSON queries
- Validate with existing OpenSearch queries

**Deliverables:**
- `docs/QUERY_LANGUAGE_DESIGN.md` (this document)
- `query/internal/translator/opensearch.go` - JSON → OpenSearch DSL
- `query/internal/validator/query.go` - JSON schema validation
- Unit tests for translator
- API endpoint: `POST /api/v1/query` accepts JSON body

**Backend changes:**
```go
// query/pkg/model/query.go
type Query struct {
    Select       []string         `json:"select,omitempty"`
    Filter       *FilterExpr      `json:"filter,omitempty"`
    TimeRange    *TimeRange       `json:"timeRange,omitempty"`
    Aggregations []Aggregation    `json:"aggregations,omitempty"`
    Sort         []SortSpec       `json:"sort,omitempty"`
    Limit        int              `json:"limit,omitempty"`
    Offset       int              `json:"offset,omitempty"`
    Cursor       string           `json:"cursor,omitempty"`
}

type FilterExpr struct {
    // Simple condition
    Field    string      `json:"field,omitempty"`
    Operator string      `json:"operator,omitempty"`
    Value    interface{} `json:"value,omitempty"`

    // Compound condition
    Type       string        `json:"type,omitempty"` // "and", "or", "not"
    Conditions []FilterExpr  `json:"conditions,omitempty"`
    Condition  *FilterExpr   `json:"condition,omitempty"` // for NOT
}
```

### Phase 2: Filter Chip Integration (Next Sprint)

**Goals:**
- Update web UI to generate JSON queries from filter chips
- Implement OCSF-aware field defaults per event class
- Replace OpenSearch query_string with JSON queries

**Deliverables:**
- `web/frontend/src/utils/queryBuilder.ts` - Filter chips → JSON
- Updated `SearchConsole.tsx` to use JSON queries
- Event class → default fields mapping
- Remove direct OpenSearch query_string usage from UI

**Frontend changes:**
```typescript
// web/frontend/src/utils/queryBuilder.ts
interface FilterChip {
  type: 'event_class' | 'field';
  field?: string;
  operator?: string;
  value: any;
}

function buildQuery(chips: FilterChip[], timeRange: TimeRange): Query {
  // Translate chips to JSON query
}
```

### Phase 3: Text Syntax Parser (Future)

**Goals:**
- Build parser for text syntax → JSON
- Integrate with search console (text input → JSON → OpenSearch)
- Support both text and JSON input modes

**Deliverables:**
- `query/internal/parser/text.go` - Text syntax parser
- Grammar definition (EBNF or PEG)
- Parser library integration (participle or goyacc)
- API accepts both text and JSON
- UI "Advanced Query" mode with text input

**Technical approach:**
- Use participle (Go parser library) for simple, readable grammar
- Parse to AST, translate to JSON query struct
- Validate JSON, then proceed as normal

### Phase 4: Saved Searches (Future)

**Goals:**
- Store queries as JSON in database
- Users can save, load, share searches
- Saved searches work across UI, API, CLI

**Deliverables:**
- Database schema for saved searches
- API endpoints: `POST /api/v1/searches`, `GET /api/v1/searches/:id`
- UI: Save/Load search buttons in search console
- Share searches with other users (view/edit permissions)

**Database schema:**
```sql
CREATE TABLE saved_searches (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    query JSONB NOT NULL,  -- Canonical JSON query
    is_shared BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

### Phase 5: S3 Cold Storage (Long-term)

**Goals:**
- Translate JSON queries to S3/Parquet queries
- Unified query interface across hot/warm/cold tiers
- Automatic tier selection based on time range

**Deliverables:**
- `storage/internal/translator/parquet.go` - JSON → Parquet predicates
- Integration with DuckDB or AWS Athena
- Partition pruning logic
- Query router (hot vs cold based on time range)

**Architecture:**
```
search service
    ├─ JSON Query Input
    ├─ Time range router
    │   ├─ 0-7 days → OpenSearch (hot)
    │   ├─ 8-90 days → OpenSearch (warm)
    │   └─ 91+ days → S3 + DuckDB (cold)
    └─ Merge results
```

---

## Design Decisions

### Why JSON Over Text-First?

**Advantages:**
- **Type safety** - Structured format prevents injection
- **Validation** - JSON Schema catches errors before execution
- **Extensibility** - Add new operators/features without breaking parser
- **Multi-backend** - Same query works on OpenSearch, S3, future DBs
- **API-friendly** - No ambiguity in programmatic queries

**Disadvantages:**
- **Verbosity** - JSON is more verbose than text syntax
- **User friction** - Power users prefer typing `severity:high` over JSON

**Mitigation:** Text syntax as syntactic sugar for common cases.

### Why jq-Style Field Paths?

**Alternatives considered:**
1. **Dot notation without prefix** - `severity`, `actor.user.name`
   - Problem: Ambiguous with string literals
2. **Bracket notation** - `["severity"]`, `["actor"]["user"]["name"]`
   - Problem: Verbose, unfamiliar
3. **jq-style** - `.severity`, `.actor.user.name`
   - ✅ Chosen: Clear, familiar to DevOps, unambiguous

### Why OCSF-Aware Defaults?

**Problem:** Generic SIEMs show 50+ columns with 90% "N/A" values.

**Solution:** When filtering to a single event class, automatically select relevant fields.

**Example:**
- Query: `{"filter": {"field": ".class_uid", "operator": "eq", "value": 3002}}`
- Auto-inject: `{"select": [".time", ".severity", ".actor.user.name", ".src_endpoint.ip", ".status"]}`

**Benefits:**
- Users get useful results without specifying fields
- Parquet column projection reduces I/O by 10-100x
- Cleaner UI (cards/tables show only relevant data)

### Why Support Both Limit/Offset and Cursor Pagination?

**Limit/Offset:**
- Simple, predictable
- Good for small result sets (<10k events)
- Breaks down at scale (offset=100000 is slow)

**Cursor:**
- Efficient at any scale
- Stateless (cursor is self-contained)
- Required for OpenSearch `search_after`

**Decision:** Support both, recommend cursor for production.

---

## Security Considerations

### Query Validation

All queries must pass validation before execution:

1. **JSON Schema validation** - Ensure structure is valid
2. **Field whitelist** - Only allow OCSF fields (prevent NoSQL injection)
3. **Operator whitelist** - Only supported operators
4. **Resource limits** - Max aggregations, max result size, timeout
5. **RBAC** - User can only query indices they have access to

### Injection Prevention

**Problem:** User-supplied values could contain malicious payloads.

**Mitigation:**
- Values are parameterized, never interpolated into query strings
- OpenSearch queries use structured DSL (not query_string unless explicitly enabled)
- Regex patterns validated before use
- CIDR ranges validated before use

**Example safe translation:**
```go
// User input
filter := FilterExpr{
  Field: ".user.name",
  Operator: "eq",
  Value: "'; DROP TABLE users; --", // Malicious input
}

// Safe translation to OpenSearch
{
  "term": {
    "user.name": "'; DROP TABLE users; --" // Treated as literal string
  }
}
```

### Rate Limiting

Query API endpoints enforce:
- Max queries per minute per user
- Max concurrent queries per user
- Query timeout (30s for search, 60s for aggregations)
- Result size limits (10k events without cursor, unlimited with cursor)

---

## Future Enhancements

### Computed Fields

Allow users to create calculated fields in queries:

```json
{
  "select": [
    ".time",
    ".severity_id",
    {
      "expression": ".severity_id * 10",
      "alias": "risk_score_scaled"
    }
  ]
}
```

### Joins (Limited)

Support simple joins for enrichment:

```json
{
  "filter": {...},
  "join": {
    "type": "left",
    "index": "asset_inventory",
    "left_field": ".device.ip",
    "right_field": ".ip_address",
    "select": [".asset_owner", ".asset_criticality"]
  }
}
```

### Subqueries

Nested queries for advanced use cases:

```json
{
  "filter": {
    "field": ".actor.user.name",
    "operator": "in",
    "value": {
      "subquery": {
        "filter": {"field": ".class_uid", "operator": "eq", "value": 3002},
        "select": [".actor.user.name"],
        "distinct": true
      }
    }
  }
}
```

### Machine Learning Queries

Anomaly detection and pattern matching:

```json
{
  "filter": {...},
  "ml": {
    "type": "anomaly_detection",
    "field": ".network.bytes_transferred",
    "sensitivity": "high"
  }
}
```

---

## References

### Related Documentation

- [UX Design Philosophy](UX_DESIGN_PHILOSOPHY.md) - Filter bar and UI design
- [search service README](../query/README.md) - Current query implementation
- [OCSF Schema 1.1.0](https://schema.ocsf.io/1.1.0/) - Field reference
- [OpenSearch Query DSL](https://opensearch.org/docs/latest/query-dsl/) - Backend query format

### External Inspirations

- **Elasticsearch Query DSL** - Structured query format
- **jq** - Field path syntax
- **GraphQL** - Field selection and nested structures
- **SQL** - Familiar operators and semantics

---

## Change Log

- **2025-11-09**: Initial document created
- **TBD**: Phase 1 implementation complete (JSON → OpenSearch)
- **TBD**: Phase 2 implementation complete (Filter chips → JSON)
- **TBD**: Phase 3 implementation complete (Text syntax parser)
