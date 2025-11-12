## MVC Schema Extensions

### Model Section

The `model` section describes **what data** to collect and **how to aggregate** it.

```json
{
  "model": {
    "correlation_type": "join",
    "parameters": {
      "time_window": "10m",
      "left_query": {"name": "auth", "query": "class_uid:3002"},
      "right_query": {"name": "file", "query": "class_uid:4005"},
      "join_conditions": [
        {"left_field": "user.name", "right_field": "actor.user.name", "operator": "eq"}
      ]
    },
    "parameter_sets": [
      {"name": "dev", "time_window": "30m"},
      {"name": "prod", "time_window": "10m"}
    ],
    "active_parameter_set": "prod",
    "aggregation": {
      "method": "join",
      "output_fields": ["user.name", "left.status", "right.file_path"]
    }
  }
}
```

**Fields**:
- `correlation_type` (string, required): Type from Tier 1 list
- `parameters` (object, required): Base parameters for correlation type
- `parameter_sets` (array, optional): Named parameter variants
- `active_parameter_set` (string, optional): Currently active set name
- `aggregation` (object, optional): Output field configuration

### Controller Section

The `controller` section describes **when to alert** and **evaluation rules**.

```json
{
  "controller": {
    "detection": {
      "threshold": 10,
      "operator": "gt",
      "query": "class_uid:3002 AND status_id:2",
      "min_matches": 3,
      "strict_order": true
    },
    "suppression": {
      "enabled": true,
      "window": "1h",
      "key": ["user.name", "src_endpoint.ip"],
      "max_alerts": 1,
      "reset_on_change": ["severity"]
    },
    "evaluation": {
      "max_events_per_window": 10000,
      "timeout": "30s",
      "retry_on_error": true
    }
  }
}
```

**Fields**:
- `detection` (object, required): Core detection logic
  - `query` (string): Base OpenSearch query
  - `threshold` (number): Alert threshold
  - `operator` (string): Comparison operator
  - Type-specific fields (`min_matches`, `strict_order`, etc.)
- `suppression` (object, optional): Alert deduplication config
- `evaluation` (object, optional): Performance/reliability settings

### View Section

The `view` section describes **how to present** alerts to users.

```json
{
  "view": {
    "title": "Brute Force Login Attempts",
    "severity": "high",
    "description": "User {{user.name}} from {{src_endpoint.ip}} had {{event_count}} failed logins in {{time_window}}",
    "tags": ["authentication", "brute_force", "mitre:T1110"],
    "references": [
      {"name": "MITRE ATT&CK", "url": "https://attack.mitre.org/techniques/T1110/"}
    ],
    "recommended_actions": [
      "Review authentication logs for user {{user.name}}",
      "Check if {{src_endpoint.ip}} is known malicious",
      "Consider temporary account lockout"
    ]
  }
}
```

**Template Variables**:
Variables from correlation results injected into strings:
- Event fields: `{{user.name}}`, `{{src_endpoint.ip}}`
- Computed values: `{{event_count}}`, `{{distinct_count}}`, `{{deviation}}`
- Parameters: `{{time_window}}`, `{{threshold}}`

---

