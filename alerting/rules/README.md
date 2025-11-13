# Builtin Detection Rules

This directory contains JSON files that define builtin detection rules. These rules are automatically imported when the alerting service starts.

## Features

- **Automatic Import**: Rules are imported on service startup
- **Content Hash Detection**: Changes are detected and rules are updated automatically
- **Protection**: Builtin rules cannot be modified or deleted through the API
- **Deterministic UUIDs**: Rule IDs are generated from rule names for consistency

## Rule Format

Each rule file must contain:

```json
{
  "name": "rule_identifier",
  "description": "Human-readable description",
  "model": {
    "correlation_type": "event_count|value_count|temporal|temporal_ordered|join",
    "parameters": {
      // Correlation-specific parameters
    }
  },
  "view": {
    "title": "Alert Title",
    "severity": "critical|high|medium|low|informational",
    "description": "Alert description with {{template}} variables",
    "category": "Category Name",
    "tags": ["tag1", "tag2"],
    "mitre_attack": {
      "tactics": ["Tactic Name"],
      "techniques": ["T1234 - Technique Name"]
    }
  },
  "controller": {
    "detection": {
      "suppression_window": "10m"
    },
    "response": {
      "actions": [],
      "severity_threshold": "medium"
    }
  }
}
```

## Correlation Types

### event_count

Counts matching events within a time window.

**Example**: Multiple failed logins
```json
{
  "correlation_type": "event_count",
  "parameters": {
    "time_window": "5m",
    "query": {
      "filter": {
        "field": ".class_uid",
        "operator": "eq",
        "value": 3002
      }
    },
    "threshold": {
      "value": 5,
      "operator": "gte"
    },
    "group_by": [".actor.user.name"]
  }
}
```

### value_count

Counts distinct values of a field (cardinality).

**Example**: Port scanning detection
```json
{
  "correlation_type": "value_count",
  "parameters": {
    "time_window": "10m",
    "query": {
      "filter": {
        "field": ".class_uid",
        "operator": "eq",
        "value": 4001
      }
    },
    "field": ".dst_endpoint.port",
    "threshold": {
      "value": 20,
      "operator": "gt"
    },
    "group_by": [".src_endpoint.ip"]
  }
}
```

### temporal

Multiple different event types within a time window (any order).

**Example**: Suspicious activity cluster
```json
{
  "correlation_type": "temporal",
  "parameters": {
    "time_window": "5m",
    "group_by": [".actor.user.name"],
    "queries": [
      {
        "name": "failed_login",
        "query": { "filter": {...} }
      },
      {
        "name": "file_delete",
        "query": { "filter": {...} }
      }
    ]
  }
}
```

### temporal_ordered

Event sequence detection with ordering.

**Example**: Attack chain (recon → exploit → persistence)
```json
{
  "correlation_type": "temporal_ordered",
  "parameters": {
    "time_window": "15m",
    "max_gap": "10m",
    "group_by": [".actor.user.name"],
    "sequence": [
      {
        "step": 1,
        "name": "recon",
        "query": { "filter": {...} }
      },
      {
        "step": 2,
        "name": "exploit",
        "query": { "filter": {...} }
      }
    ]
  }
}
```

### join

Correlate events from different queries by matching field values.

**Example**: DNS lookup + suspicious connection
```json
{
  "correlation_type": "join",
  "parameters": {
    "time_window": "5m",
    "queries": [
      {
        "name": "dns_query",
        "query": { "filter": {...} }
      },
      {
        "name": "network_connection",
        "query": { "filter": {...} }
      }
    ],
    "join_conditions": [
      {
        "left_field": ".query.hostname",
        "right_field": ".dst_endpoint.hostname",
        "operator": "eq"
      }
    ]
  }
}
```

## Adding New Rules

1. Create a new JSON file in this directory
2. Follow the rule format above
3. Use a descriptive filename (e.g., `suspicious_powershell.json`)
4. Rebuild and restart the alerting service

The rule will be automatically imported with a deterministic UUID based on the `name` field.

## OCSF Class UIDs

Common OCSF event classes for detection rules:

- **3002**: Authentication (login/logout)
- **3001**: Account Change
- **4001**: Network Activity
- **4005**: File Activity
- **1003**: Account Change
- **1007**: User Session
- **6003**: Network Scan
- **2004**: Vulnerability

See [OCSF Schema](https://schema.ocsf.io/) for complete list.

## Template Variables

Alert titles and descriptions support template variables:

- `{{actor.user.name}}`: Username
- `{{src_endpoint.ip}}`: Source IP address
- `{{dst_endpoint.ip}}`: Destination IP
- `{{event_count}}`: Number of events matched
- `{{distinct_count}}`: Number of distinct values

Any OCSF field can be used in templates with dot notation.

## Import Behavior

- **First Start**: All rules are imported
- **Subsequent Starts**: Only changed rules are updated
- **Content Hash**: SHA-256 hash of model + view + controller
- **Idempotent**: Safe to restart service multiple times
- **Error Handling**: Import failures are logged but don't prevent service startup

## Protection

Builtin rules are protected from user modifications:

- **Cannot Update**: Returns HTTP 403 Forbidden
- **Cannot Disable**: Returns HTTP 403 Forbidden
- **Cannot Delete**: Returns HTTP 403 Forbidden
- **Metadata**: Marked with `source: "builtin"` in controller metadata

Users can create their own rules through the API which can be freely modified.
