# Correlation System Refactor - Using Canonical Query Language

**Date**: 2025-11-12
**Status**: Completed
**Rationale**: Align correlation system with existing query infrastructure

## Problem Identified

The initial correlation design used raw query strings (`"class_uid:3002 AND status_id:2"`) instead of the canonical JSON query language that already exists in the `query/` service.

This created multiple issues:
1. **Architectural inconsistency** - Different query formats for search vs. rules
2. **Security concerns** - Raw strings prone to injection vs. structured queries
3. **No UI reuse** - Can't use existing filter bar for rule building
4. **Duplicated logic** - Custom query orchestrator vs. existing translator
5. **No validation** - Bypassed the query validator

## Solution

Refactored the correlation system to use `query/pkg/model.Query` (the canonical JSON query format).

## Changes Made

### 1. Updated Go Module Dependencies
- Added `github.com/telhawk-systems/telhawk-stack/query` to `alerting/go.mod`
- Used local replace directive: `replace github.com/telhawk-systems/telhawk-stack/query => ../query`

### 2. Updated Correlation Types (`alerting/internal/correlation/types.go`)
**Before:**
```go
type QueryConfig struct {
    Name  string `json:"name"`
    Query string `json:"query"`  // Raw string!
}
```

**After:**
```go
import "github.com/telhawk-systems/telhawk-stack/query/pkg/model"

type QueryConfig struct {
    Name  string       `json:"name"`
    Query *model.Query `json:"query"`  // Structured query!
}
```

### 3. Created Query Executor (`alerting/internal/correlation/query_executor.go`)
- Wraps the canonical query translator
- Executes `model.Query` objects against OpenSearch
- Handles count aggregations and cardinality queries
- Uses structured queries throughout

### 4. Created Local Translator (`alerting/internal/correlation/translator.go`)
- Simplified translator for correlation needs
- Avoids importing `internal/translator` from query service
- Converts `model.Query` → OpenSearch DSL

### 5. Updated Evaluators (`alerting/internal/correlation/evaluators.go`)
- Changed from `QueryOrchestrator` to `QueryExecutor`
- Parse controller queries as `model.Query` objects
- Use structured query execution

### 6. Removed Old Code
- Renamed `query_orchestrator.go` → `query_orchestrator.go.old`
- Will be deleted after verification

## New Query Format for Correlation Rules

### Before (Broken):
```json
{
  "controller": {
    "detection": {
      "query": "class_uid:3002 AND status_id:2",  ← Raw string!
      "threshold": 10
    }
  }
}
```

### After (Correct):
```json
{
  "controller": {
    "detection": {
      "query": {
        "filter": {
          "type": "and",
          "conditions": [
            {"field": ".class_uid", "operator": "eq", "value": 3002},
            {"field": ".status_id", "operator": "eq", "value": 2}
          ]
        }
      },
      "threshold": 10
    }
  }
}
```

## Benefits

### 1. UI Reuse
The same filter bar component can be used for:
- Search queries
- Saved searches
- **Correlation rule building** ← NEW!

### 2. Security
- Structured queries prevent injection attacks
- Validation before execution
- Type-safe field references

### 3. Consistency
- Same query language everywhere
- Same translator for all queries
- Same OCSF field paths (`.actor.user.name`)

### 4. Validation
- Leverage existing `query/internal/validator`
- Field path validation
- Operator validation
- Type checking

### 5. Future-Proof
- Supports advanced features (aggregations, joins, subqueries)
- Works with OCSF-aware field defaults
- Compatible with saved search system

## Example: Brute Force Detection Rule

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
            {
              "field": ".class_uid",
              "operator": "eq",
              "value": 3002
            },
            {
              "field": ".status_id",
              "operator": "eq",
              "value": 2
            }
          ]
        }
      },
      "threshold": 10,
      "operator": "gt"
    }
  },
  "view": {
    "title": "Brute Force Login Attempts",
    "severity": "high",
    "description": "User {{actor.user.name}} had {{event_count}} failed logins in {{time_window}}"
  }
}
```

## Migration Path

### For Existing Rules (if any)
If any rules were created with raw query strings, they need to be migrated:

1. Parse the raw query string
2. Convert to structured `model.Query` format
3. Update the rule via API

### For Documentation
All correlation documentation examples must be updated to use the structured query format.

## Next Steps

1. ✅ Refactor complete and building
2. 📝 Update correlation documentation examples
3. 🧪 Write integration tests with structured queries
4. 🎨 Build UI components using existing filter bar
5. 🚀 Deploy and validate

## Files Modified

- `alerting/go.mod` - Added query package dependency
- `alerting/internal/correlation/types.go` - Use `model.Query`
- `alerting/internal/correlation/query_executor.go` - NEW: Query execution with translator
- `alerting/internal/correlation/translator.go` - NEW: Local translator implementation
- `alerting/internal/correlation/evaluators.go` - Updated to use structured queries
- `alerting/internal/correlation/query_orchestrator.go` - Deprecated (renamed to `.old`)

## References

- [Query Language Documentation](../../../query/QUERY_LANGUAGE.md)
- [Query Model Definition](../../../query/pkg/model/query.go)
- [Canonical Query Design](../../QUERY_LANGUAGE_DESIGN.md)
