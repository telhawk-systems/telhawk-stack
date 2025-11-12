# Advanced Correlation Types (Tier 2/3)

**Status**: Design Phase - Future Enhancements
**Version**: 1.0
**Last Updated**: 2025-11-11

## Overview

This document describes advanced correlation types beyond the Tier 1 essentials. These are planned for future phases after the core 8 types are implemented and validated.

## Tier 2: Important Correlation Types

### 1. rate/velocity

**Description**: Alert when rate of change exceeds threshold (different from absolute count).

**Use Cases**:
- Brute force acceleration (attacks speeding up)
- DDoS ramp-up detection (traffic increasing rapidly)
- Data exfiltration velocity (bandwidth usage accelerating)

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `rate_threshold` | float | Yes | - | Events per time unit |
| `rate_unit` | string | Yes | - | per_second, per_minute, per_hour |
| `acceleration_threshold` | float | No | 2.0 | Rate multiplier (2.0 = doubling) |
| `measurement_window` | duration | Yes | - | Window to measure rate |
| `group_by` | array[string] | No | [] | Grouping fields |

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "rate",
    "parameters": {
      "rate_threshold": 10,
      "rate_unit": "per_minute",
      "acceleration_threshold": 2.0,
      "measurement_window": "5m",
      "group_by": ["src_endpoint.ip"]
    }
  },
  "controller": {
    "detection": {
      "query": "class_uid:4001",
      "operator": "gt"
    }
  },
  "view": {
    "title": "Accelerating Network Traffic",
    "severity": "high",
    "description": "{{src_endpoint.ip}} traffic rate increased {{acceleration}}x to {{current_rate}}/min"
  }
}
```

**Key Difference from event_count**:
- event_count: "Alert when count > 100"
- rate: "Alert when current rate > previous rate * 2"

---

### 2. ratio/proportion

**Description**: Alert when ratio between two event types exceeds threshold.

**Use Cases**:
- Authentication failure rate ((failed / total) > 30%)
- Success/failure ratio anomaly
- Normal vs suspicious activity proportions
- Error rate monitoring

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `numerator_query` | object | Yes | - | Query for numerator events |
| `denominator_query` | object | Yes | - | Query for denominator events |
| `ratio_threshold` | float | Yes | - | Minimum ratio to alert (0.0-1.0) |
| `operator` | string | No | "gt" | gt, gte, lt, lte |
| `time_window` | duration | Yes | - | Measurement window |
| `group_by` | array[string] | No | [] | Grouping fields |
| `min_denominator` | integer | No | 10 | Minimum denominator to calculate ratio |

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "ratio",
    "parameters": {
      "numerator_query": {
        "name": "failed_logins",
        "query": "class_uid:3002 AND status_id:2"
      },
      "denominator_query": {
        "name": "total_logins",
        "query": "class_uid:3002"
      },
      "time_window": "10m",
      "group_by": ["user.name"],
      "min_denominator": 5
    }
  },
  "controller": {
    "detection": {
      "ratio_threshold": 0.3,
      "operator": "gt"
    }
  },
  "view": {
    "title": "High Authentication Failure Rate",
    "severity": "medium",
    "description": "User {{user.name}} has {{ratio_percent}}% failed logins ({{numerator}}/{{denominator}})"
  }
}
```

---

### 3. Sigma's Statistical Types (value_sum, value_avg, value_percentile)

While deprioritized in Tier 1, these Sigma types are still valuable for certain use cases.

#### value_sum

**Description**: Alert when sum of numeric field exceeds threshold.

**Use Cases**:
- Total bytes transferred > 1GB
- Total failed attempts across all users
- Aggregate resource consumption

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `field` | string | Yes | - | Numeric field to sum |
| `threshold` | number | Yes | - | Sum threshold |
| `time_window` | duration | Yes | - | Aggregation window |
| `group_by` | array[string] | No | [] | Grouping fields |

#### value_avg

**Description**: Alert when average of numeric field exceeds threshold.

**Use Cases**:
- Average response time > 500ms
- Average file size accessed unusually large
- Mean resource consumption anomaly

#### value_percentile

**Description**: Alert when percentile value exceeds threshold.

**Use Cases**:
- 95th percentile response time > 1s
- Top 5% of users by activity
- Outlier detection (beyond 99th percentile)

---

## Tier 3: Advanced Correlation Types

### 4. outlier/anomaly (Statistical)

**Description**: Advanced statistical outlier detection using z-score, IQR, or machine learning.

**Use Cases**:
- Detect statistical anomalies in any numeric field
- IQR-based outlier detection (resistant to extreme values)
- Z-score normalization across entities

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `field` | string | Yes | - | Numeric field to analyze |
| `method` | string | Yes | - | zscore, iqr, percentile, ml |
| `threshold` | float | Yes | - | Method-specific threshold |
| `baseline_window` | duration | Yes | - | Historical learning period |
| `group_by` | array[string] | Yes | - | Per-entity analysis |

**Methods**:
- **zscore**: Standard deviations from mean (threshold = 2.0, 3.0, etc.)
- **iqr**: Interquartile range multiplier (threshold = 1.5, 3.0)
- **percentile**: Beyond Nth percentile (threshold = 95, 99)
- **ml**: Machine learning model (requires training)

**Example** (IQR method):
```json
{
  "model": {
    "correlation_type": "outlier",
    "parameters": {
      "field": "bytes_transferred",
      "method": "iqr",
      "threshold": 1.5,
      "baseline_window": "7d",
      "group_by": ["user.name"]
    }
  },
  "controller": {
    "detection": {
      "query": "class_uid:4005"
    }
  },
  "view": {
    "title": "Outlier File Transfer Volume",
    "severity": "medium",
    "description": "{{user.name}} transferred {{bytes}} bytes ({{iqr_multiplier}}x IQR above normal)"
  }
}
```

---

### 5. geographic

**Description**: Geographic correlation and impossible travel detection.

**Use Cases**:
- Impossible travel (login from US then China in 1 hour)
- Geofencing violations (access from unauthorized country)
- Distance-based anomalies

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `max_distance_km` | float | No | - | Maximum distance between events |
| `max_velocity_kmh` | float | No | 900 | Maximum travel speed (default: airplane) |
| `time_window` | duration | Yes | - | Time window for travel calc |
| `entity_field` | string | Yes | - | Field identifying entity (user, device) |
| `location_fields` | object | Yes | - | Lat/lon or city/country fields |

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "geographic",
    "parameters": {
      "max_velocity_kmh": 900,
      "time_window": "4h",
      "entity_field": "user.name",
      "location_fields": {
        "latitude": "src_endpoint.location.lat",
        "longitude": "src_endpoint.location.lon",
        "city": "src_endpoint.location.city",
        "country": "src_endpoint.location.country"
      }
    }
  },
  "controller": {
    "detection": {
      "query": "class_uid:3002"
    }
  },
  "view": {
    "title": "Impossible Travel Detected",
    "severity": "critical",
    "description": "{{user.name}} logged in from {{location_2}} {{time_diff}} after {{location_1}} ({{distance_km}} km, impossible at {{required_velocity}} km/h)"
  }
}
```

---

### 6. pattern_match (Sequence Patterns)

**Description**: Advanced sequence pattern matching with wildcards, optionals, and branching.

**Use Cases**:
- Attack patterns with variable steps (A → * → B → C)
- Optional intermediate steps (A → [B] → C)
- Branching detection (A → (B OR C) → D)
- Regular expressions over event sequences

**Pattern Syntax**:
- `*` : Zero or more events (wildcard)
- `[...]` : Optional step
- `(A|B)` : Either A or B
- `{N,M}` : Repetition (N to M times)

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "pattern_match",
    "parameters": {
      "time_window": "1h",
      "group_by": ["user.name"],
      "pattern": "recon → * → (exploit|privilege_escalation) → * → exfiltration"
    }
  },
  "controller": {
    "detection": {
      "pattern_queries": {
        "recon": "class_uid:6003",
        "exploit": "class_uid:2004",
        "privilege_escalation": "class_uid:3004",
        "exfiltration": "class_uid:6003 AND activity_id:3"
      }
    }
  },
  "view": {
    "title": "Attack Pattern Matched",
    "severity": "critical",
    "description": "User {{user.name}} executed attack pattern: {{matched_pattern}}"
  }
}
```

---

### 7. chain/graph (Multi-Hop Correlation)

**Description**: Multi-hop relationship traversal and graph-based correlation.

**Use Cases**:
- Attack path reconstruction (User A → compromised → User B → accessed → Server C)
- Lateral movement tracking
- Infection chain detection
- Supply chain analysis

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `max_hops` | integer | Yes | - | Maximum graph traversal depth |
| `time_window` | duration | Yes | - | Maximum time for chain |
| `relationships` | array[object] | Yes | - | Relationship definitions |
| `start_query` | object | Yes | - | Starting node query |
| `end_query` | object | No | - | Target node query (optional) |

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "chain",
    "parameters": {
      "max_hops": 5,
      "time_window": "24h",
      "start_query": {
        "name": "initial_compromise",
        "query": "class_uid:2004"
      },
      "relationships": [
        {
          "type": "compromised",
          "from_field": "src_endpoint.ip",
          "to_field": "dst_endpoint.ip"
        },
        {
          "type": "accessed",
          "from_field": "user.name",
          "to_field": "dst_endpoint.hostname"
        }
      ]
    }
  },
  "controller": {
    "detection": {
      "min_hops": 3
    }
  },
  "view": {
    "title": "Multi-Stage Attack Chain",
    "severity": "critical",
    "description": "Attack chain detected: {{chain_path}}"
  }
}
```

---

## Implementation Priority

| Type | Priority | Complexity | Value | Recommended Phase |
|------|----------|------------|-------|-------------------|
| rate/velocity | High | Medium | High | Phase 4 |
| ratio/proportion | High | Low | High | Phase 4 |
| value_sum/avg | Medium | Low | Medium | Phase 5 |
| outlier/anomaly | High | High | High | Phase 5 |
| geographic | Medium | Medium | Medium | Phase 6 |
| pattern_match | High | Very High | High | Phase 7 |
| chain/graph | Medium | Very High | Medium | Phase 8 |

## Dependencies

### Tier 2 Dependencies:
- **rate/velocity**: Requires time-series state tracking in Redis
- **ratio/proportion**: Requires multi-query orchestration (same as join)

### Tier 3 Dependencies:
- **outlier/anomaly**: Requires statistical libraries (gonum/stats)
- **geographic**: Requires GeoIP database and distance calculation
- **pattern_match**: Requires pattern matching engine (regex over events)
- **chain/graph**: Requires graph database or traversal engine

## Next Steps

1. **Complete Tier 1** (8 types) - Validate architecture works
2. **Add rate/velocity** - Most requested advanced type
3. **Add ratio/proportion** - High value, low complexity
4. **Evaluate demand** for Tier 3 types based on user feedback

## References

- **Splunk**: `correlate`, `transaction`, `stats` commands
- **Elastic SIEM**: EQL sequences, machine learning jobs
- **Chronicle**: YARA-L multi-event rules, UDM
- **Sigma**: Original 7 correlation types specification
