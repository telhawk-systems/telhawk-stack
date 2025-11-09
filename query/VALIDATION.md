# Query Validation Reference

This document describes all validation rules enforced by the TelHawk query validator before queries are executed against OpenSearch.

## Overview

The query validator ensures that all queries are:
- **Well-formed** - Correct structure and syntax
- **Safe** - Protected against resource exhaustion
- **Secure** - No injection vulnerabilities
- **Performant** - Reasonable limits to prevent system overload

Validation occurs **before translation** to OpenSearch DSL, catching errors early and providing clear error messages.

## Validation Flow

```
Client Query → JSON Parse → Validator → Translator → OpenSearch
                               ↓
                         ValidationError
                         (HTTP 400 with details)
```

## Configuration Limits

The validator enforces the following limits (configurable via `NewQueryValidator()`):

| Limit | Default | Description |
|-------|---------|-------------|
| `maxAggregations` | 10 | Maximum number of aggregations per query |
| `maxResultSize` | 10,000 | Maximum results without cursor pagination |
| `maxFilterDepth` | 10 | Maximum nesting depth for compound filters |
| `maxSelectFields` | 100 | Maximum fields in select clause |
| `maxSortFields` | 10 | Maximum sort specifications |

## Validation Rules

### 1. Query Structure

#### Nil Query
```
ERROR: query cannot be nil
```

A query object must be provided.

### 2. Field Paths (`select`)

#### Field Path Format
All OCSF field paths must start with `.` (jq-style notation):

**Valid:**
- `.severity`
- `.actor.user.name`
- `.attacks[0].tactic.name`

**Invalid:**
- `severity` ❌ (missing leading dot)
- `` ❌ (empty)
- `.actor..user` ❌ (double dots)
- `.actor.` ❌ (trailing dot)

#### Maximum Select Fields
```json
{
  "select": [...100 fields...]  // ✓ Valid
}
```

```json
{
  "select": [...101 fields...]  // ✗ ERROR: too many select fields
}
```

### 3. Filter Expressions

#### Simple Conditions

**Field Validation:**
```json
{
  "filter": {
    "field": ".severity",     // ✓ Must start with '.'
    "operator": "eq",          // ✓ Must be supported operator
    "value": "High"            // ✓ Must match operator requirements
  }
}
```

**Supported Operators:**
- `eq`, `ne` - Equality/inequality
- `gt`, `gte`, `lt`, `lte` - Numeric comparisons
- `in` - Array membership
- `contains`, `startsWith`, `endsWith` - String matching
- `regex` - Regular expression
- `exists` - Field existence
- `cidr` - IP range matching

**Operator-Specific Validation:**

| Operator | Value Type | Example | Validation |
|----------|-----------|---------|------------|
| `eq`, `ne`, `gt`, `gte`, `lt`, `lte` | Any | `"High"`, `5` | Value cannot be nil |
| `in` | Array | `["High", "Critical"]` | Must be array |
| `exists` | Boolean | `true`, `false` | Must be boolean |
| `regex` | String | `"^test.*"` | Must be valid regex pattern |
| `cidr` | String | `"192.168.0.0/16"` | Must contain `/` |

**Validation Examples:**

```json
// ✓ Valid: Array for 'in' operator
{
  "field": ".status",
  "operator": "in",
  "value": ["Failed", "Locked"]
}

// ✗ Invalid: Non-array for 'in'
{
  "field": ".status",
  "operator": "in",
  "value": "Failed"  // ERROR: value for 'in' operator must be an array
}

// ✓ Valid: Regex pattern
{
  "field": ".process.cmd_line",
  "operator": "regex",
  "value": "^/usr/bin/.*"
}

// ✗ Invalid: Bad regex
{
  "field": ".process.cmd_line",
  "operator": "regex",
  "value": "[invalid"  // ERROR: invalid regex pattern
}

// ✓ Valid: CIDR notation
{
  "field": ".src_endpoint.ip",
  "operator": "cidr",
  "value": "10.0.0.0/8"
}

// ✗ Invalid: Missing slash
{
  "field": ".src_endpoint.ip",
  "operator": "cidr",
  "value": "10.0.0.0"  // ERROR: invalid CIDR notation: must contain /
}
```

#### Compound Conditions

**AND/OR Filters:**
```json
{
  "type": "and",
  "conditions": [
    {"field": ".severity", "operator": "eq", "value": "High"},
    {"field": ".status", "operator": "eq", "value": "Failed"}
  ]
}
```

**Validation:**
- Must have at least one condition
- All conditions are recursively validated
- Maximum nesting depth enforced (default: 10 levels)

```json
// ✗ Invalid: Empty conditions
{
  "type": "and",
  "conditions": []  // ERROR: and filter requires at least one condition
}
```

**NOT Filter:**
```json
{
  "type": "not",
  "condition": {
    "field": ".actor.user.name",
    "operator": "eq",
    "value": "system"
  }
}
```

**Validation:**
- Must have a condition (not null)
- Condition is recursively validated

```json
// ✗ Invalid: Null condition
{
  "type": "not",
  "condition": null  // ERROR: NOT filter requires a condition
}
```

#### Maximum Filter Depth

Prevents deeply nested filters that could cause performance issues or stack overflow:

```json
// ✓ Valid: 3 levels deep
{
  "type": "and",
  "conditions": [
    {
      "type": "or",
      "conditions": [
        {
          "type": "not",
          "condition": {"field": ".test", "operator": "eq", "value": "test"}
        }
      ]
    }
  ]
}

// ✗ Invalid: 11+ levels deep
// ERROR: filter nesting too deep: 11 (max: 10)
```

### 4. Time Range

#### Relative Time

**Format:** `<number><unit>`

**Supported:**
- `15m`, `30m`, `45m` - Minutes
- `1h`, `2h`, `24h` - Hours
- `7d`, `30d`, `90d` - Days

```json
{
  "timeRange": {
    "last": "1h"  // ✓ Valid
  }
}
```

**Invalid Formats:**
```json
{"last": "1"}        // ✗ ERROR: invalid relative time format
{"last": "1x"}       // ✗ ERROR: invalid relative time format
{"last": "1 hour"}   // ✗ ERROR: invalid relative time format
{"last": "-1h"}      // ✗ ERROR: invalid relative time format
```

#### Absolute Time

```json
{
  "timeRange": {
    "start": "2025-01-01T00:00:00Z",
    "end": "2025-01-31T23:59:59Z"
  }
}
```

**Validation:**
- Start time must be before end time
- RFC3339 format required

```json
// ✗ Invalid: Start after end
{
  "timeRange": {
    "start": "2025-01-31T00:00:00Z",
    "end": "2025-01-01T00:00:00Z"
  }
  // ERROR: start time cannot be after end time
}
```

#### Mutual Exclusivity

```json
// ✗ Invalid: Both absolute and relative
{
  "timeRange": {
    "start": "2025-01-01T00:00:00Z",
    "last": "1h"
  }
  // ERROR: time range cannot specify both absolute and relative times
}

// ✗ Invalid: Neither specified
{
  "timeRange": {}
  // ERROR: time range must specify either start/end or last
}
```

### 5. Aggregations

#### Maximum Aggregations

```json
{
  "aggregations": [
    {...},  // 1
    {...},  // 2
    ...
    {...}   // 10 - ✓ Valid
  ]
}

// ✗ More than 10
// ERROR: too many aggregations: 11 (max: 10)
```

#### Required Fields

**Terms Aggregation:**
```json
{
  "type": "terms",
  "field": ".actor.user.name",  // ✓ Required
  "name": "top_users",           // ✓ Required
  "size": 10                     // ✓ Required, must be > 0
}

// ✗ Invalid: Missing field
{"type": "terms", "name": "agg", "size": 10}
// ERROR: terms aggregation requires a field

// ✗ Invalid: Invalid size
{"type": "terms", "field": ".user", "name": "agg", "size": 0}
// ERROR: terms aggregation size must be > 0
```

**Date Histogram:**
```json
{
  "type": "date_histogram",
  "field": ".time",       // ✓ Required
  "name": "over_time",    // ✓ Required
  "interval": "1h"        // ✓ Required
}

// ✗ Invalid: Missing interval
{"type": "date_histogram", "field": ".time", "name": "agg"}
// ERROR: date_histogram aggregation requires an interval
```

**Metric Aggregations:**
```json
{
  "type": "avg",          // ✓ avg, sum, min, max, stats, cardinality
  "field": ".risk_score", // ✓ Required
  "name": "avg_risk"      // ✓ Required
}

// ✗ Invalid: Missing name
{"type": "avg", "field": ".risk_score"}
// ERROR: aggregation name cannot be empty
```

#### Nested Aggregations

Nested aggregations are validated recursively and count towards the total aggregation limit:

```json
{
  "aggregations": [
    {
      "type": "terms",
      "field": ".severity",
      "name": "by_severity",
      "size": 5,
      "aggregations": [  // ✓ Nested aggregations allowed
        {
          "type": "terms",
          "field": ".user",
          "name": "top_users",
          "size": 3
        }
      ]
    }
  ]
}
```

### 6. Sorting

#### Maximum Sort Fields

```json
{
  "sort": [
    {"field": ".time", "order": "desc"},
    {"field": ".severity_id", "order": "desc"},
    ...  // Up to 10 total
  ]
}

// ✗ More than 10
// ERROR: too many sort fields: 11 (max: 10)
```

#### Sort Order Validation

```json
// ✓ Valid orders
{"field": ".time", "order": "asc"}
{"field": ".time", "order": "desc"}
{"field": ".time"}  // order is optional, defaults to desc

// ✗ Invalid order
{"field": ".time", "order": "ascending"}
// ERROR: invalid order: ascending (must be 'asc' or 'desc')
```

### 7. Pagination

#### Limit Validation

```json
{"limit": 100}       // ✓ Valid
{"limit": 10000}     // ✓ Valid (max without cursor)
{"limit": 0}         // ✓ Valid (uses default: 100)

{"limit": -1}        // ✗ ERROR: limit cannot be negative
{"limit": 20000}     // ✗ ERROR: limit exceeds maximum (use cursor pagination)
```

#### Cursor Pagination

```json
// ✓ Large limits allowed with cursor
{
  "limit": 20000,
  "cursor": "eyJzb3J0IjpbMTczNjQ2NzIwMDAwMF19"
}
```

#### Offset Validation

```json
{"offset": 100}      // ✓ Valid
{"offset": 0}        // ✓ Valid (no offset)

{"offset": -1}       // ✗ ERROR: offset cannot be negative
```

#### Mutual Exclusivity

```json
// ✗ Invalid: Both offset and cursor
{
  "offset": 100,
  "cursor": "abc123"
}
// ERROR: cannot use both offset and cursor pagination
```

## Error Response Format

When validation fails, the API returns HTTP 400 with a structured error:

```json
{
  "code": "invalid_request",
  "message": "query validation failed: invalid filter: unsupported operator: bad_op"
}
```

Error messages provide:
- **Context** - Which part of the query failed
- **Reason** - Why it failed
- **Guidance** - What's expected (when applicable)

### Example Error Messages

```
Field path validation:
✗ "invalid select: invalid field no_dot: field path must start with '.'"

Operator validation:
✗ "invalid filter: unsupported operator: bad_op"

Value validation:
✗ "invalid filter: value for 'in' operator must be an array"
✗ "invalid filter: invalid regex pattern: error parsing regexp: missing closing ]: `[invalid`"

Time range validation:
✗ "invalid time range: invalid relative time format: bad_time"
✗ "invalid time range: start time cannot be after end time"

Aggregation validation:
✗ "invalid aggregations: aggregation 0 (top_users): terms aggregation requires a field"
✗ "invalid aggregations: too many aggregations: 11 (max: 10)"

Pagination validation:
✗ "invalid pagination: limit 20000 exceeds maximum 10000 (use cursor pagination for large result sets)"
✗ "invalid pagination: cannot use both offset and cursor pagination"

Nesting depth:
✗ "invalid filter: filter nesting too deep: 11 (max: 10)"
```

## Testing Validation

### Unit Tests

Comprehensive validator tests are in `internal/validator/query_test.go`:

```bash
# Run validator tests
go test ./internal/validator -v

# Test specific validation
go test ./internal/validator -run TestValidateOperators -v
```

### Manual Testing

```bash
# Test with invalid query
curl -X POST http://localhost:8082/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{
    "filter": {
      "field": "no_dot",
      "operator": "eq",
      "value": "test"
    }
  }'

# Response:
# HTTP/1.1 400 Bad Request
# {
#   "code": "invalid_request",
#   "message": "query validation failed: invalid select: invalid field no_dot: field path must start with '.'"
# }
```

## Best Practices

### 1. Start Simple
Begin with basic queries and add complexity incrementally:

```json
// Start with this
{
  "filter": {"field": ".severity", "operator": "eq", "value": "High"},
  "timeRange": {"last": "1h"}
}

// Then add more conditions
{
  "filter": {
    "type": "and",
    "conditions": [
      {"field": ".severity", "operator": "eq", "value": "High"},
      {"field": ".status", "operator": "eq", "value": "Failed"}
    ]
  },
  "timeRange": {"last": "1h"}
}
```

### 2. Use Field Projection
Specify `select` to reduce network transfer and improve performance:

```json
{
  "select": [".time", ".severity", ".actor.user.name"],
  ...
}
```

### 3. Limit Aggregations
Keep aggregation count reasonable (< 5 typically sufficient):

```json
{
  "aggregations": [
    {"type": "terms", "field": ".severity", "name": "by_severity", "size": 5}
  ]
}
```

### 4. Use Cursor Pagination
For large result sets, use cursor-based pagination:

```json
// First request
{
  "limit": 1000
}

// Subsequent requests
{
  "limit": 1000,
  "cursor": "eyJzb3J0IjpbMTczNjQ2NzIwMDAwMF19"
}
```

### 5. Validate Locally
Catch errors early by validating query structure before sending:

```typescript
// Frontend validation example
function isValidFieldPath(path: string): boolean {
  return path.startsWith('.') &&
         !path.includes('..') &&
         !path.endsWith('.');
}
```

## Security Considerations

The validator provides multiple layers of security:

1. **Injection Prevention**
   - Field paths validated (no SQL/NoSQL injection)
   - Operators whitelisted
   - Regex patterns validated before use
   - CIDR ranges validated

2. **Resource Protection**
   - Maximum result size prevents memory exhaustion
   - Maximum aggregations prevents CPU overload
   - Filter depth limit prevents stack overflow
   - Field limits prevent excessive data transfer

3. **Type Safety**
   - Operator-specific value validation
   - Prevents type confusion attacks
   - Structured queries (not string interpolation)

## Performance Impact

Validation adds minimal overhead:
- **Typical query:** < 1ms validation time
- **Complex query:** 1-5ms validation time
- **Benefit:** Prevents expensive OpenSearch queries

The validator catches most errors before they reach OpenSearch, saving:
- Network round trips
- OpenSearch query parsing
- Partial query execution
- Error handling overhead

## Summary

The query validator ensures that all queries are well-formed, safe, and performant before execution. It provides:

- ✓ Comprehensive validation rules
- ✓ Clear error messages
- ✓ Protection against malformed queries
- ✓ Resource exhaustion prevention
- ✓ Security guarantees

All validation is enforced **before** translation to OpenSearch DSL, catching errors early and providing a better user experience.
