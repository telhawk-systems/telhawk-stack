# TelHawk Query Language - Phase 1 Implementation

This document describes the Phase 1 implementation of the TelHawk canonical query language, which provides a JSON-based query interface that translates to OpenSearch Query DSL.

## Overview

Phase 1 implements the foundational components for the canonical JSON query language:

- **Query Data Structures** (`pkg/model/query.go`) - Canonical JSON query format
- **Translator** (`internal/translator/opensearch.go`) - Converts JSON queries to OpenSearch DSL
- **Validator** (`internal/validator/query.go`) - Validates queries before execution
- **API Endpoint** (`POST /api/v1/query`) - Accepts JSON queries via HTTP

## Architecture

```
Client → POST /api/v1/query → Validator → Translator → OpenSearch
         (JSON Query)         (Validate)   (JSON→DSL)   (Execute)
```

## Using the Query API

### Endpoint

```
POST http://localhost:8082/api/v1/query
Content-Type: application/json
```

### Request Format

The request body should be a JSON query following the canonical format:

```json
{
  "select": [".time", ".severity", ".actor.user.name"],
  "filter": {
    "field": ".severity",
    "operator": "eq",
    "value": "High"
  },
  "timeRange": {
    "last": "1h"
  },
  "limit": 100
}
```

### Response Format

The response follows the existing `SearchResponse` format:

```json
{
  "request_id": "abc123",
  "latency_ms": 45,
  "result_count": 25,
  "total_matches": 150,
  "results": [
    {
      "time": 1736467200,
      "severity": "High",
      "actor": {
        "user": {
          "name": "jsmith"
        }
      }
    }
  ],
  "search_after": [1736467200000],
  "aggregations": {}
}
```

## Query Structure

### Field Projection (`select`)

Specify which OCSF fields to return:

```json
{
  "select": [".time", ".severity", ".actor.user.name", ".src_endpoint.ip"]
}
```

- Field paths use jq-style notation with leading dot (`.`)
- Omit `select` to return all fields
- Reduces network transfer and improves performance

### Filter Expressions (`filter`)

#### Simple Condition

```json
{
  "filter": {
    "field": ".severity",
    "operator": "eq",
    "value": "High"
  }
}
```

#### Supported Operators

- `eq` - Equals
- `ne` - Not equals
- `gt`, `gte`, `lt`, `lte` - Comparison operators
- `in` - Value in array
- `contains` - String contains
- `startsWith` - String starts with
- `endsWith` - String ends with
- `regex` - Regular expression match
- `exists` - Field exists (value: true/false)
- `cidr` - IP in CIDR range

#### Compound Conditions

**AND:**
```json
{
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".class_uid", "operator": "eq", "value": 3002},
      {"field": ".severity_id", "operator": "gte", "value": 4}
    ]
  }
}
```

**OR:**
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

**NOT:**
```json
{
  "filter": {
    "type": "not",
    "condition": {
      "field": ".actor.user.name",
      "operator": "eq",
      "value": "system"
    }
  }
}
```

### Time Range (`timeRange`)

#### Relative Time

```json
{
  "timeRange": {
    "last": "1h"
  }
}
```

Supported values: `15m`, `1h`, `24h`, `7d`, `30d`, `90d`

#### Absolute Time

```json
{
  "timeRange": {
    "start": "2025-01-01T00:00:00Z",
    "end": "2025-01-31T23:59:59Z"
  }
}
```

### Sorting (`sort`)

```json
{
  "sort": [
    {"field": ".severity_id", "order": "desc"},
    {"field": ".time", "order": "desc"}
  ]
}
```

- Order: `asc` or `desc`
- Default: `[{"field": ".time", "order": "desc"}]`

### Aggregations

#### Terms Aggregation

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

#### Date Histogram

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

#### Metrics

```json
{
  "aggregations": [
    {
      "type": "avg",
      "field": ".risk_score",
      "name": "avg_risk"
    }
  ]
}
```

Supported types: `terms`, `date_histogram`, `avg`, `sum`, `min`, `max`, `stats`, `cardinality`

#### Nested Aggregations

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

### Pagination

#### Limit/Offset

```json
{
  "limit": 100,
  "offset": 200
}
```

- Max limit without cursor: 10,000
- Use cursor pagination for larger result sets

#### Cursor-based (Future)

```json
{
  "limit": 100,
  "cursor": "eyJzb3J0IjpbMTczNjQ2NzIwMDAwMF19"
}
```

## Example Queries

See `examples/sample_queries.json` for comprehensive examples including:

1. Simple equality filter
2. Failed authentication hunt (complex AND/NOT logic)
3. Lateral movement detection (CIDR matching)
4. Top users aggregation
5. Events over time histogram
6. Nested aggregations

## Testing

### Unit Tests

```bash
# Test the translator
go test ./internal/translator -v

# Test all packages
go test ./...
```

### Manual Testing with curl

```bash
# Simple query
curl -X POST http://localhost:8082/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{
    "filter": {
      "field": ".severity",
      "operator": "eq",
      "value": "High"
    },
    "timeRange": {"last": "1h"},
    "limit": 10
  }'

# Complex query with aggregations
curl -X POST http://localhost:8082/api/v1/query \
  -H "Content-Type: application/json" \
  -d @examples/sample_queries.json
```

## Validation

All queries are validated before execution to ensure they are well-formed, safe, and performant.

### Validation Limits

| Resource | Limit | Description |
|----------|-------|-------------|
| Select fields | 100 | Maximum fields in select clause |
| Aggregations | 10 | Maximum number of aggregations |
| Sort fields | 10 | Maximum sort specifications |
| Filter depth | 10 | Maximum nesting for compound filters |
| Result size | 10,000 | Maximum results without cursor |

### Quick Validation Rules

- **Field paths** must start with `.` (jq-style): `.severity`, `.actor.user.name`
- **Operators** must be supported: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `contains`, `regex`, `exists`, `cidr`
- **Time ranges** must be valid: relative (`1h`, `7d`) or absolute with start < end
- **Operator values** must match type: `in` requires array, `exists` requires boolean, `regex` must be valid pattern
- **Aggregations** require name, type, and (usually) field
- **Pagination** cannot use both offset and cursor

### Validation Error Format

Validation errors return HTTP 400 with structured details:

```json
{
  "code": "invalid_request",
  "message": "query validation failed: invalid filter: unsupported operator: invalid_op"
}
```

**See [VALIDATION.md](VALIDATION.md) for complete validation reference with examples.**

## Implementation Details

### Translation Process

1. **Validate** - Check query structure and constraints
2. **Translate** - Convert JSON to OpenSearch DSL
3. **Execute** - Submit to OpenSearch
4. **Transform** - Format response

### Field Path Translation

OCSF paths (`.actor.user.name`) → OpenSearch fields (`actor.user.name`)

The leading dot is removed during translation.

### Operator Mapping

| Query Language | OpenSearch DSL |
|----------------|----------------|
| `eq` | `term` |
| `ne` | `bool.must_not.term` |
| `gt/gte/lt/lte` | `range` |
| `in` | `terms` |
| `contains` | `wildcard` with `*value*` |
| `regex` | `regexp` |
| `exists` | `exists` |

### Security Features

- **Injection prevention** - Values are parameterized, never interpolated
- **Field whitelist** - Only OCSF field paths allowed
- **Resource limits** - Max aggregations, result size, timeout
- **RBAC** - User can only query accessible indices (future)

## Next Steps (Phase 2+)

1. **Filter Chip Integration** - UI generates JSON queries from filter chips
2. **Text Syntax Parser** - Parse `severity:high status:failed` → JSON
3. **Saved Searches** - Store JSON queries in database
4. **S3 Cold Storage** - Translate queries to Parquet predicates

## References

- [Query Language Design](../../docs/QUERY_LANGUAGE_DESIGN.md) - Complete design document
- [OCSF Schema 1.1.0](https://schema.ocsf.io/1.1.0/) - Field reference
- [OpenSearch Query DSL](https://opensearch.org/docs/latest/query-dsl/) - Backend query format
