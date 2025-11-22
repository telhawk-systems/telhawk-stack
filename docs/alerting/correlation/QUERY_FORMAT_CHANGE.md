# Query Format Change - Before and After

This document shows the architectural fix applied to correlation rules.

## The Problem

Initial correlation design used **raw query strings** (Lucene/OpenSearch syntax):
```json
{
  "controller": {
    "detection": {
      "query": "class_uid:3002 AND status_id:2"  // ← STRING!
    }
  }
}
```

**Why this was wrong:**
- Different query format than saved searches/UI
- Security risk (injection attacks)
- Can't reuse existing filter bar UI
- No validation
- Bypassed existing query infrastructure

## The Solution

Use the **canonical JSON query language** (from `query/pkg/model`):
```json
{
  "controller": {
    "detection": {
      "query": {  // ← STRUCTURED OBJECT!
        "filter": {
          "type": "and",
          "conditions": [
            {"field": ".class_uid", "operator": "eq", "value": 3002},
            {"field": ".status_id", "operator": "eq", "value": 2}
          ]
        }
      }
    }
  }
}
```

## Side-by-Side Comparison

### Simple Condition

**Before (Wrong):**
```json
{
  "query": "severity:High"
}
```

**After (Correct):**
```json
{
  "query": {
    "filter": {
      "field": ".severity",
      "operator": "eq",
      "value": "High"
    }
  }
}
```

### AND Condition

**Before (Wrong):**
```json
{
  "query": "class_uid:3002 AND status_id:2"
}
```

**After (Correct):**
```json
{
  "query": {
    "filter": {
      "type": "and",
      "conditions": [
        {"field": ".class_uid", "operator": "eq", "value": 3002},
        {"field": ".status_id", "operator": "eq", "value": 2}
      ]
    }
  }
}
```

### OR Condition

**Before (Wrong):**
```json
{
  "query": "severity:High OR severity:Critical"
}
```

**After (Correct):**
```json
{
  "query": {
    "filter": {
      "type": "or",
      "conditions": [
        {"field": ".severity", "operator": "eq", "value": "High"},
        {"field": ".severity", "operator": "eq", "value": "Critical"}
      ]
    }
  }
}
```

### Complex Nested

**Before (Wrong):**
```json
{
  "query": "class_uid:3002 AND (status_id:2 OR status_id:3) AND severity:High"
}
```

**After (Correct):**
```json
{
  "query": {
    "filter": {
      "type": "and",
      "conditions": [
        {"field": ".class_uid", "operator": "eq", "value": 3002},
        {
          "type": "or",
          "conditions": [
            {"field": ".status_id", "operator": "eq", "value": 2},
            {"field": ".status_id", "operator": "eq", "value": 3}
          ]
        },
        {"field": ".severity", "operator": "eq", "value": "High"}
      ]
    }
  }
}
```

### Wildcards/Contains

**Before (Wrong):**
```json
{
  "query": "user.name:*admin*"
}
```

**After (Correct):**
```json
{
  "query": {
    "filter": {
      "field": ".user.name",
      "operator": "contains",
      "value": "admin"
    }
  }
}
```

### Ranges

**Before (Wrong):**
```json
{
  "query": "severity_id:>=3"
}
```

**After (Correct):**
```json
{
  "query": {
    "filter": {
      "field": ".severity_id",
      "operator": "gte",
      "value": 3
    }
  }
}
```

### Field Existence

**Before (Wrong):**
```json
{
  "query": "_exists_:user.email"
}
```

**After (Correct):**
```json
{
  "query": {
    "filter": {
      "field": ".user.email",
      "operator": "exists"
    }
  }
}
```

## Field Path Format

All OCSF field paths use **jq-style notation with leading dot**:

| Wrong | Correct |
|-------|---------|
| `user.name` | `.actor.user.name` |
| `severity` | `.severity` |
| `src_endpoint.ip` | `.src_endpoint.ip` |
| `class_uid` | `.class_uid` |

The leading dot (`.`) makes it clear it's a field path, not a string literal.

## Supported Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equals | `{"field": ".severity", "operator": "eq", "value": "High"}` |
| `ne` | Not equals | `{"field": ".status", "operator": "ne", "value": "success"}` |
| `gt` | Greater than | `{"field": ".severity_id", "operator": "gt", "value": 2}` |
| `gte` | Greater than or equal | `{"field": ".severity_id", "operator": "gte", "value": 3}` |
| `lt` | Less than | `{"field": ".count", "operator": "lt", "value": 100}` |
| `lte` | Less than or equal | `{"field": ".count", "operator": "lte", "value": 50}` |
| `in` | In array | `{"field": ".status", "operator": "in", "value": [1, 2, 3]}` |
| `contains` | String contains | `{"field": ".message", "operator": "contains", "value": "error"}` |
| `startsWith` | String starts with | `{"field": ".user.name", "operator": "startsWith", "value": "admin"}` |
| `endsWith` | String ends with | `{"field": ".file.name", "operator": "endsWith", "value": ".exe"}` |
| `regex` | Regular expression | `{"field": ".ip", "operator": "regex", "value": "192\\.168\\..*"}` |
| `exists` | Field exists | `{"field": ".user.email", "operator": "exists"}` |
| `cidr` | IP in CIDR range | `{"field": ".src_endpoint.ip", "operator": "cidr", "value": "10.0.0.0/8"}` |

## Complete Rule Example

### Before (Wrong):
```json
{
  "model": {
    "correlation_type": "event_count",
    "parameters": {
      "time_window": "5m",
      "group_by": ["user.name"]
    }
  },
  "controller": {
    "detection": {
      "query": "class_uid:3002 AND status_id:2 AND severity:High",
      "threshold": 10
    }
  },
  "view": {
    "title": "Brute Force Detected",
    "severity": "high"
  }
}
```

### After (Correct):
```json
{
  "model": {
    "correlation_type": "event_count",
    "parameters": {
      "time_window": "5m",
      "group_by": [".actor.user.name"]
    }
  },
  "controller": {
    "detection": {
      "query": {
        "filter": {
          "type": "and",
          "conditions": [
            {"field": ".class_uid", "operator": "eq", "value": 3002},
            {"field": ".status_id", "operator": "eq", "value": 2},
            {"field": ".severity", "operator": "eq", "value": "High"}
          ]
        }
      },
      "threshold": 10
    }
  },
  "view": {
    "title": "Brute Force Detected",
    "severity": "high"
  }
}
```

## Benefits of Structured Queries

1. **Security** - No injection attacks possible
2. **Validation** - Type checking and field validation
3. **UI Reuse** - Same filter bar for search and rules
4. **Consistency** - One query language for everything
5. **Type Safety** - Proper Go structs instead of string parsing
6. **Future-proof** - Supports advanced features (subqueries, joins, etc.)

## Migration Checklist

If you have existing rules with raw query strings:

- [ ] Identify all rules using string queries
- [ ] Convert each query to structured format
- [ ] Add leading dots to all field paths
- [ ] Use proper operators (`eq`, `contains`, etc.)
- [ ] Test with query validator
- [ ] Update via API

## References

- [Query Language Documentation](../../../query/QUERY_LANGUAGE.md)
- [Query Model Definition](../../../query/pkg/model/query.go)
- [Refactor Notes](REFACTOR_NOTES.md)
- [Core Types with Examples](CORE_TYPES.md)
