## Implementation Considerations

### Phase 1: Foundation (Weeks 1-2)

**Goals**: Basic correlation infrastructure

1. **State Manager**
   - Redis client with baseline storage
   - Suppression cache implementation
   - Heartbeat tracking

2. **Query Orchestrator**
   - Single query execution
   - Event aggregation helpers
   - OpenSearch client improvements

3. **Simple Correlation Types**
   - `event_count` - proves aggregation works
   - `suppression` - proves state management works
   - Validate entire pipeline end-to-end

### Phase 2: Multi-Event Correlation (Weeks 3-4)

4. **Temporal Correlation**
   - `temporal` - unordered event matching
   - `temporal_ordered` - sequence detection
   - Multi-query orchestration

5. **Join Correlation**
   - `join` - cross-event-type correlation
   - Field matching logic
   - Performance optimization

### Phase 3: Advanced Analytics (Weeks 5-6)

6. **Statistical Correlation**
   - `value_count` - cardinality counting
   - `baseline_deviation` - anomaly detection
   - Baseline learning algorithms

7. **Absence Detection**
   - `missing_event` - heartbeat monitoring
   - Scheduled expectation tracking

### Backward Compatibility

**Existing simple rules continue to work**:
- If `correlation_type` is null/missing, treat as simple rule
- Existing evaluator handles simple rules
- New evaluator handles correlation rules

**Migration path**:
```json
// Before (simple rule)
{
  "model": {},
  "controller": {"detection": {"query": "..."}},
  "view": {}
}

// After (correlation rule)
{
  "model": {
    "correlation_type": "event_count",
    "parameters": {"time_window": "5m"}
  },
  "controller": {
    "detection": {"query": "...", "threshold": 10}
  },
  "view": {}
}
```

### API Changes

**New Endpoints**:

```
# Get correlation types and parameter schemas
GET /api/v1/correlation/types
Response: {
  "types": [
    {
      "name": "event_count",
      "description": "...",
      "parameter_schema": {...},
      "example": {...}
    },
    ...
  ]
}

# Set active parameter set
PUT /api/v1/schemas/{id}/parameters
Body: {"active_parameter_set": "prod"}

# Test correlation rule (dry run)
POST /api/v1/schemas/{id}/test
Body: {
  "time_range": {"from": "...", "to": "..."},
  "parameter_set": "dev"
}
```

**Existing Endpoints** (no breaking changes):
- `POST /api/v1/schemas` - create with correlation config
- `PUT /api/v1/schemas/{id}` - update (creates new version if structural)
- `GET /api/v1/schemas` - list includes correlation rules
- `GET /api/v1/schemas/{id}` - retrieve with correlation config

### Validation Logic

**At Rule Creation**:
```go
func ValidateDetectionSchema(schema *DetectionSchema) error {
    correlationType := schema.Model["correlation_type"]
    if correlationType == nil {
        return nil // Simple rule, no validation needed
    }

    // Get validator for type
    validator := GetCorrelationValidator(correlationType)
    if validator == nil {
        return fmt.Errorf("unknown correlation type: %s", correlationType)
    }

    // Validate parameters
    params := schema.Model["parameters"]
    if err := validator.ValidateParameters(params); err != nil {
        return fmt.Errorf("invalid parameters: %w", err)
    }

    // Validate parameter sets
    paramSets := schema.Model["parameter_sets"]
    for _, set := range paramSets {
        if err := validator.ValidateParameters(set); err != nil {
            return fmt.Errorf("invalid parameter set %s: %w", set["name"], err)
        }
    }

    return nil
}
```

### Monitoring and Metrics

**Prometheus Metrics**:
```
# Evaluation performance
correlation_evaluation_duration_seconds{type, rule_id}
correlation_evaluation_errors_total{type, rule_id}

# State metrics
correlation_baseline_count{rule_id}
correlation_suppression_active_count{rule_id}
correlation_heartbeat_tracked_count{rule_id}

# Alert metrics
correlation_alerts_generated_total{type, rule_id, severity}
correlation_alerts_suppressed_total{rule_id}

# Query performance
correlation_query_duration_seconds{type, rule_id}
correlation_query_events_fetched_total{type, rule_id}
```

**Health Checks**:
- Redis connectivity check
- Baseline staleness check (warn if not updated recently)
- Query timeout monitoring

### Testing Strategy

**Unit Tests**:
- Each correlation type evaluator
- State manager operations
- Parameter validation logic

**Integration Tests**:
- End-to-end correlation evaluation
- Redis state persistence
- Multi-query orchestration

**Load Tests**:
- 1000+ concurrent rules
- 10,000+ events/sec ingestion
- Baseline storage scaling
- Suppression cache performance

---

