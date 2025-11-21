# OCSF Integration & correlation_uid

**Status**: Design Phase
**Version**: 1.0
**Last Updated**: 2025-11-11

## Overview

This document specifies how TelHawk integrates with the OCSF `correlation_uid` field to support pre-grouped event correlation and cross-system event tracking.

## OCSF metadata.correlation_uid Field

The OCSF schema includes a `correlation_uid` field in the metadata object:

```go
// ocsf/objects/metadata.go
type Metadata struct {
    CorrelationUid string `json:"correlation_uid,omitempty"`
    // ... other fields
}
```

**Purpose**: Identifies related events across different event types, sources, and time windows.

## Use Cases

### 1. Pre-Grouped Events from Source Systems

Some security tools emit events with correlation IDs already assigned:

- **EDR Platforms**: Group process creation, file access, network connection into single "incident"
- **SIEM Systems**: Forward correlated events from upstream SIEM with correlation ID
- **Application Logs**: Distributed tracing IDs (OpenTelemetry trace_id)

**Example**: CrowdStrike Falcon detection

```json
{
  "class_uid": 2004,
  "activity_id": 1,
  "metadata": {
    "correlation_uid": "inc_abc123_crowdstrike",
    "product": {
      "name": "CrowdStrike Falcon",
      "vendor_name": "CrowdStrike"
    }
  },
  "process": {...}
}
```

### 2. Distributed Tracing

Link events across microservices using OpenTelemetry trace IDs:

```json
{
  "class_uid": 4002,
  "metadata": {
    "correlation_uid": "trace:a1b2c3d4e5f6",
    "product": {"name": "My API Gateway"}
  },
  "http_request": {...}
}
```

```json
{
  "class_uid": 4005,
  "metadata": {
    "correlation_uid": "trace:a1b2c3d4e5f6",
    "product": {"name": "Database Service"}
  },
  "file": {...}
}
```

### 3. Manual Correlation Tagging

Security analysts can tag events during investigation:

```bash
# Tag events as related
curl -X POST http://query:8083/api/v1/events/tag \
  -d '{
    "event_ids": ["event-1", "event-2", "event-3"],
    "correlation_uid": "investigation-2024-01-15-001"
  }'
```

---

## Populating correlation_uid

### Server-Side Generation (Recommended)

TelHawk generates correlation UIDs during normalization:

```go
// core/internal/normalizer/normalizer.go
func (n *BaseNormalizer) Normalize(raw map[string]interface{}) (*ocsf.Event, error) {
    event := &ocsf.Event{
        Metadata: &ocsf.Metadata{
            Product:  n.product,
            Version:  n.version,
            // ... other fields
        },
    }

    // Generate correlation UID if not present
    if raw["correlation_uid"] == nil {
        event.Metadata.CorrelationUid = generateCorrelationUID(raw)
    } else {
        event.Metadata.CorrelationUid = raw["correlation_uid"].(string)
    }

    return event, nil
}

func generateCorrelationUID(raw map[string]interface{}) string {
    // Strategy 1: Use source-provided correlation ID
    if srcCorrelationID, ok := raw["incident_id"]; ok {
        return fmt.Sprintf("src:%v", srcCorrelationID)
    }

    // Strategy 2: Use distributed tracing ID
    if traceID, ok := raw["trace_id"]; ok {
        return fmt.Sprintf("trace:%v", traceID)
    }

    // Strategy 3: Use OpenTelemetry convention
    if spanCtx, ok := raw["span_context"]; ok {
        return fmt.Sprintf("otel:%v", spanCtx.(map[string]interface{})["trace_id"])
    }

    // Strategy 4: Generate from entity + time window
    // Group events by entity in 1-minute buckets
    entity := extractPrimaryEntity(raw) // user, IP, hostname
    timeBucket := time.Unix(int64(raw["time"].(float64)), 0).Truncate(1 * time.Minute)
    return fmt.Sprintf("auto:%s:%d", entity, timeBucket.Unix())
}
```

### Client-Side Submission

Clients can provide correlation UID in HEC payload:

```bash
curl -X POST http://ingest:8088/services/collector/event \
  -H "Authorization: Splunk $HEC_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "event": {
      "sourcetype": "my_app",
      "correlation_uid": "session-xyz789",
      "user": "alice",
      "action": "file_delete"
    }
  }'
```

---

## Using correlation_uid in Correlation Rules

### Pre-Grouped Correlation

Create rules that leverage existing correlation UIDs:

```json
{
  "model": {
    "correlation_type": "pre_grouped",
    "parameters": {
      "time_window": "1h",
      "group_by": ["metadata.correlation_uid"],
      "min_event_types": 3
    }
  },
  "controller": {
    "detection": {
      "query": "metadata.correlation_uid:*",
      "event_type_queries": [
        {"name": "process", "query": "class_uid:1007"},
        {"name": "file", "query": "class_uid:4005"},
        {"name": "network", "query": "class_uid:4001"}
      ]
    }
  },
  "view": {
    "title": "Multi-Stage Activity Detected",
    "severity": "high",
    "description": "Correlation ID {{metadata.correlation_uid}} includes {{event_type_count}} event types"
  }
}
```

### Enhancing join Correlation

Use correlation_uid to optimize join queries:

```go
func (qo *QueryOrchestrator) ExecuteJoinOptimized(ctx context.Context, left, right Query, conditions []JoinCondition, window time.Duration) ([]*JoinedEvent, error) {
    // First, try correlation_uid join (fast)
    correlationJoin := &JoinCondition{
        LeftField:  "metadata.correlation_uid",
        RightField: "metadata.correlation_uid",
        Operator:   "eq",
    }

    quickResults := qo.executeJoinByField(ctx, left, right, correlationJoin, window)
    if len(quickResults) > 0 {
        return quickResults, nil
    }

    // Fall back to field-based join (slower)
    return qo.executeJoinByField(ctx, left, right, conditions[0], window)
}
```

---

## OpenSearch Mapping

Ensure correlation_uid is indexed as keyword for exact matching:

```json
{
  "mappings": {
    "properties": {
      "metadata": {
        "properties": {
          "correlation_uid": {
            "type": "keyword",
            "index": true
          }
        }
      }
    }
  }
}
```

---

## Query Patterns

### Find All Events in Correlation Group

```bash
GET /telhawk-events-*/_search
{
  "query": {
    "term": {
      "metadata.correlation_uid": "inc_abc123_crowdstrike"
    }
  },
  "sort": [{"time": "asc"}]
}
```

### Count Event Types per Correlation

```bash
GET /telhawk-events-*/_search
{
  "size": 0,
  "query": {
    "exists": {"field": "metadata.correlation_uid"}
  },
  "aggs": {
    "by_correlation": {
      "terms": {"field": "metadata.correlation_uid"},
      "aggs": {
        "event_types": {
          "cardinality": {"field": "class_uid"}
        }
      }
    }
  }
}
```

### Find Correlations with Multiple Event Types

```bash
GET /telhawk-events-*/_search
{
  "size": 0,
  "aggs": {
    "by_correlation": {
      "terms": {"field": "metadata.correlation_uid"},
      "aggs": {
        "event_types": {"cardinality": {"field": "class_uid"}},
        "having": {
          "bucket_selector": {
            "buckets_path": {"event_types": "event_types"},
            "script": "params.event_types >= 3"
          }
        }
      }
    }
  }
}
```

---

## Correlation UID Strategies

### Strategy 1: Session-Based

Group events by user session:

```
correlation_uid = "session:{session_id}"
```

**Use case**: Track user activity within single session

### Strategy 2: Transaction-Based

Group events by business transaction:

```
correlation_uid = "txn:{transaction_id}"
```

**Use case**: E-commerce checkout, API request flow

### Strategy 3: Incident-Based

Group events by security incident:

```
correlation_uid = "inc:{incident_id}:{source_system}"
```

**Use case**: EDR-detected incidents, SIEM alerts

### Strategy 4: Trace-Based

Group events by distributed trace:

```
correlation_uid = "trace:{trace_id}"
```

**Use case**: Microservices observability

### Strategy 5: Time-Window-Based

Auto-generate for events from same entity in time window:

```
correlation_uid = "auto:{entity}:{time_bucket}"
```

**Use case**: When source doesn't provide correlation ID

---

## Cross-System Correlation

### Scenario: EDR + Firewall + Auth Logs

**EDR Event** (CrowdStrike):
```json
{
  "class_uid": 1007,
  "metadata": {
    "correlation_uid": "inc_abc123_crowdstrike",
    "product": {"name": "CrowdStrike Falcon"}
  },
  "process": {"name": "malware.exe"}
}
```

**Firewall Event** (Palo Alto):
```json
{
  "class_uid": 4001,
  "metadata": {
    "correlation_uid": "inc_abc123_crowdstrike", // Match EDR incident
    "product": {"name": "Palo Alto NGFW"}
  },
  "src_endpoint": {"ip": "10.0.1.100"}
}
```

**Auth Event** (Active Directory):
```json
{
  "class_uid": 3002,
  "metadata": {
    "correlation_uid": "inc_abc123_crowdstrike", // Match same incident
    "product": {"name": "Active Directory"}
  },
  "user": {"name": "jdoe"}
}
```

**Result**: All three events automatically grouped by correlation_uid, enabling cross-system investigation.

---

## API for Correlation Management

### Tag Events with correlation_uid

```
POST /api/v1/events/correlate
Authorization: Bearer <token>
Content-Type: application/json

{
  "event_ids": [
    "event-1",
    "event-2",
    "event-3"
  ],
  "correlation_uid": "investigation-2024-01-15-001"
}
```

### Get Events by correlation_uid

```
GET /api/v1/events?correlation_uid=inc_abc123_crowdstrike
Authorization: Bearer <token>
```

Response:
```json
{
  "correlation_uid": "inc_abc123_crowdstrike",
  "event_count": 12,
  "event_types": [1007, 4001, 4005, 3002],
  "time_span": "15m30s",
  "events": [...]
}
```

---

## Implementation Phases

### Phase 1: Passive Support (Current)

- Field exists in OCSF schema ✓
- Not populated during ingestion
- Not used in correlation rules

### Phase 2: Population

- Implement server-side generation strategies
- Accept client-provided correlation UIDs
- Add to normalization pipeline

### Phase 3: Correlation Integration

- Add `pre_grouped` correlation type
- Optimize join queries using correlation_uid
- Build correlation UID index

### Phase 4: Advanced Features

- Cross-system correlation tagging UI
- Automatic correlation UID generation for common patterns
- Correlation graph visualization

---

## Configuration

```yaml
core:
  normalization:
    correlation_uid:
      enabled: true
      strategy: "auto" # auto, trace, session, none
      auto_time_bucket: "1m"
      prefix: "telhawk"

alerting:
  correlation:
    use_correlation_uid: true
    prefer_correlation_uid_join: true # Try correlation_uid before field join
```

---

## References

- [OCSF Schema](https://github.com/ocsf/ocsf-schema) - metadata.correlation_uid definition
- [OpenTelemetry Tracing](https://opentelemetry.io/docs/concepts/signals/traces/) - trace_id conventions
- [CORE_TYPES.md](CORE_TYPES.md) - join correlation type
- `common/ocsf/ocsf/objects/metadata.go:11` - correlation_uid field definition
