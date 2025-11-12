## Core Correlation Types

### Tier 1 Essentials (8 Types)

#### 1. event_count

**Description**: Alert when number of matching events exceeds threshold within time window.

**Use Cases**:
- Brute force detection (10+ failed logins in 5 minutes)
- DDoS detection (1000+ requests in 1 minute)
- Excessive file access (100+ files accessed in 10 minutes)

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `time_window` | duration | Yes | - | Lookback window (e.g., "5m", "1h") |
| `threshold` | integer | Yes | - | Minimum event count to trigger |
| `operator` | string | No | "gt" | Comparison operator: gt, gte, lt, lte, eq, ne |
| `group_by` | array[string] | No | [] | Fields to group by (per-entity counting) |

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "event_count",
    "parameters": {
      "time_window": "5m",
      "group_by": ["user.name", "src_endpoint.ip"]
    },
    "parameter_sets": [
      {"name": "dev", "threshold": 5},
      {"name": "prod", "threshold": 10},
      {"name": "strict", "threshold": 3}
    ]
  },
  "controller": {
    "detection": {
      "query": "class_uid:3002 AND status_id:2",
      "threshold": 10,
      "operator": "gt"
    }
  },
  "view": {
    "title": "Brute Force Login Attempts",
    "severity": "high",
    "description": "User {{user.name}} from {{src_endpoint.ip}} had {{event_count}} failed logins in {{time_window}}"
  }
}
```

---

#### 2. value_count

**Description**: Alert when number of distinct values (cardinality) exceeds threshold.

**Use Cases**:
- Password spray (1 user tries to login as 50+ different users)
- Port scanning (1 source hits 100+ destination ports)
- Data exfiltration (1 session accesses 1000+ unique files)

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `time_window` | duration | Yes | - | Lookback window |
| `field` | string | Yes | - | Field to count distinct values |
| `threshold` | integer | Yes | - | Minimum distinct value count |
| `operator` | string | No | "gt" | Comparison operator |
| `group_by` | array[string] | No | [] | Grouping fields |

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "value_count",
    "parameters": {
      "time_window": "10m",
      "field": "dst_endpoint.port",
      "group_by": ["src_endpoint.ip"]
    }
  },
  "controller": {
    "detection": {
      "query": "class_uid:4001",
      "threshold": 100,
      "operator": "gt"
    }
  },
  "view": {
    "title": "Port Scanning Detected",
    "severity": "high",
    "description": "{{src_endpoint.ip}} scanned {{distinct_count}} ports in {{time_window}}"
  }
}
```

---

#### 3. temporal

**Description**: Alert when multiple events occur within time proximity (unordered).

**Use Cases**:
- Suspicious activity cluster (failed login AND file delete AND network connection within 5 min)
- Co-occurrence detection (A and B both happen, any order)
- Multi-indicator detection

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `time_window` | duration | Yes | - | Maximum time span for events |
| `queries` | array[object] | Yes | - | List of event queries to match |
| `min_matches` | integer | No | all | Minimum queries that must match |
| `group_by` | array[string] | No | [] | Correlation key fields |

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "temporal",
    "parameters": {
      "time_window": "5m",
      "group_by": ["user.name"],
      "queries": [
        {"name": "failed_auth", "query": "class_uid:3002 AND status_id:2"},
        {"name": "file_delete", "query": "class_uid:4005 AND activity_id:5"},
        {"name": "external_conn", "query": "class_uid:4001 AND dst_endpoint.type:Internet"}
      ]
    }
  },
  "controller": {
    "detection": {
      "min_matches": 3
    }
  },
  "view": {
    "title": "Suspicious Activity Cluster",
    "severity": "critical",
    "description": "User {{user.name}} exhibited multiple suspicious behaviors in {{time_window}}"
  }
}
```

---

#### 4. temporal_ordered

**Description**: Alert when events occur in specific sequence within time window.

**Use Cases**:
- Attack chain detection (recon → exploit → persistence)
- Multi-stage attack (privilege escalation → lateral movement → exfiltration)
- Workflow violations (action without prior approval)

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `time_window` | duration | Yes | - | Maximum time between first and last |
| `sequence` | array[object] | Yes | - | Ordered list of event queries |
| `max_gap` | duration | No | time_window | Maximum time between consecutive events |
| `group_by` | array[string] | No | [] | Correlation key fields |

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "temporal_ordered",
    "parameters": {
      "time_window": "30m",
      "max_gap": "10m",
      "group_by": ["user.name"],
      "sequence": [
        {"step": 1, "name": "recon", "query": "class_uid:6003"},
        {"step": 2, "name": "exploit", "query": "class_uid:2004"},
        {"step": 3, "name": "persistence", "query": "class_uid:1003"}
      ]
    }
  },
  "controller": {
    "detection": {
      "strict_order": true
    }
  },
  "view": {
    "title": "Attack Chain Detected",
    "severity": "critical",
    "description": "User {{user.name}} executed attack sequence: recon → exploit → persistence"
  }
}
```

---

#### 5. join

**Description**: Correlate events from different types/sources by matching field values.

**Use Cases**:
- Failed auth followed by successful privileged action (compromised service account)
- DNS query matched with network connection (C2 detection)
- User creation followed by privilege escalation
- Cross-dataset correlation (firewall + endpoint + auth)

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `time_window` | duration | Yes | - | Maximum time between joined events |
| `left_query` | object | Yes | - | First event query |
| `right_query` | object | Yes | - | Second event query |
| `join_conditions` | array[object] | Yes | - | Field matching conditions |
| `join_type` | string | No | "inner" | inner, left, any |

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "join",
    "parameters": {
      "time_window": "10m",
      "left_query": {
        "name": "failed_auth",
        "query": "class_uid:3002 AND status_id:2"
      },
      "right_query": {
        "name": "file_delete",
        "query": "class_uid:4005 AND activity_id:5"
      },
      "join_conditions": [
        {
          "left_field": "user.name",
          "right_field": "actor.user.name",
          "operator": "eq"
        }
      ],
      "join_type": "inner"
    },
    "parameter_sets": [
      {"name": "dev", "time_window": "30m"},
      {"name": "prod", "time_window": "10m"}
    ]
  },
  "controller": {
    "detection": {
      "order": "right_after_left"
    }
  },
  "view": {
    "title": "Post-Auth-Failure Privilege Action",
    "severity": "critical",
    "description": "User {{user.name}} performed privileged action after failed auth"
  }
}
```

---

#### 6. suppression

**Description**: Alert deduplication and throttling to prevent alert fatigue.

**Use Cases**:
- Prevent re-alerting on same condition
- Group similar alerts into single incident
- Rate limiting for noisy rules
- Reduce duplicate alerts by 90%+

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `window` | duration | Yes | - | Suppression window duration |
| `key` | array[string] | Yes | - | Fields to group alerts by |
| `max_alerts` | integer | No | 1 | Max alerts per window per key |
| `reset_on_change` | array[string] | No | [] | Fields that reset suppression if changed |

**Example Rule**:
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
      "query": "class_uid:3002 AND status_id:2",
      "threshold": 10,
      "operator": "gt"
    },
    "suppression": {
      "enabled": true,
      "window": "1h",
      "key": ["user.name", "src_endpoint.ip"],
      "max_alerts": 1,
      "reset_on_change": ["severity"]
    }
  },
  "view": {
    "title": "Brute Force Login Attempts",
    "severity": "high",
    "description": "User {{user.name}} had {{event_count}} failed logins (suppressed for 1h)"
  }
}
```

**Suppression State** (Redis):
```
Key: suppression:<rule_id>:<key_hash>
Value: {
  "first_alert_time": 1699564800,
  "last_alert_time": 1699564800,
  "alert_count": 1,
  "suppression_context": {"user.name": "admin", "src_endpoint.ip": "10.0.1.5"}
}
TTL: suppression.window duration
```

---

#### 7. baseline_deviation

**Description**: Alert when current behavior deviates from learned historical baseline.

**Use Cases**:
- User logs in at unusual time (normally 9-5, now at 3 AM)
- Process memory usage 5x higher than baseline
- File access volume 10x normal
- First-seen behavior detection

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `baseline_window` | duration | Yes | - | Historical learning period (e.g., "7d") |
| `comparison_window` | duration | Yes | - | Current period to compare (e.g., "5m") |
| `field` | string | Yes | - | Field to measure |
| `deviation_threshold` | float | Yes | - | Standard deviations from baseline |
| `sensitivity` | string | No | "medium" | low=3σ, medium=2σ, high=1.5σ |
| `group_by` | array[string] | Yes | - | Per-entity baselines |
| `min_baseline_samples` | integer | No | 100 | Minimum samples to establish baseline |

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "baseline_deviation",
    "parameters": {
      "baseline_window": "7d",
      "comparison_window": "5m",
      "field": "event_count",
      "group_by": ["user.name"],
      "min_baseline_samples": 50
    },
    "parameter_sets": [
      {"name": "dev", "sensitivity": "low", "deviation_threshold": 3.0},
      {"name": "prod", "sensitivity": "medium", "deviation_threshold": 2.0},
      {"name": "strict", "sensitivity": "high", "deviation_threshold": 1.5}
    ]
  },
  "controller": {
    "detection": {
      "query": "class_uid:4005",
      "deviation_threshold": 2.0
    }
  },
  "view": {
    "title": "Abnormal File Access Volume",
    "severity": "medium",
    "description": "User {{user.name}} accessed {{current_count}} files vs baseline {{baseline_avg}} ({{deviation}}σ)"
  }
}
```

**Baseline State** (Redis):
```
Key: baseline:<rule_id>:<entity_hash>
Value: {
  "samples": [120, 115, 125, ...],  // Rolling window
  "count": 1008,
  "sum": 121000,
  "sum_squares": 14762000,
  "mean": 120.0,
  "stddev": 5.2,
  "last_updated": 1699564800
}
TTL: baseline_window * 2
```

---

#### 8. missing_event

**Description**: Alert when expected event does NOT occur within expected interval.

**Use Cases**:
- Heartbeat monitoring (endpoint agent silent for 10 minutes)
- Scheduled job didn't run (backup expected every 24h)
- Log source went silent (no logs from firewall in 5 minutes)
- "Dog didn't bark" detection

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `expected_interval` | duration | Yes | - | Expected event frequency |
| `grace_period` | duration | No | 0 | Allow delay before alerting |
| `entity_field` | string | Yes | - | Field identifying entity |
| `alert_after_missing` | integer | No | 1 | Alert after N missed intervals |

**Example Rule**:
```json
{
  "model": {
    "correlation_type": "missing_event",
    "parameters": {
      "expected_interval": "5m",
      "grace_period": "30s",
      "entity_field": "device.hostname",
      "alert_after_missing": 2
    },
    "parameter_sets": [
      {"name": "dev", "expected_interval": "10m"},
      {"name": "prod", "expected_interval": "5m"}
    ]
  },
  "controller": {
    "detection": {
      "query": "metadata.product.name:'TelHawk Agent' AND class_uid:6001"
    }
  },
  "view": {
    "title": "Agent Heartbeat Missing",
    "severity": "medium",
    "description": "No heartbeat from {{device.hostname}} for {{missing_duration}}"
  }
}
```

**Missing Event State** (Redis):
```
Key: heartbeat:<rule_id>:<entity_hash>
Value: {
  "entity": "webserver-01",
  "last_seen": 1699564800,
  "missed_count": 0,
  "expected_next": 1699565100
}
TTL: expected_interval * (alert_after_missing + 1)
```

---

