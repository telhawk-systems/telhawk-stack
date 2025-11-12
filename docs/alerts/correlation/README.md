# Correlation System Documentation

**Status**: Design Phase
**Version**: 1.0
**Last Updated**: 2025-11-11

## Overview

This directory contains the complete design specification for TelHawk's correlation system, which extends detection rules to support multi-event correlation, temporal analysis, baseline deviation, and alert suppression.

## Why Correlation?

Traditional SIEM rules detect single events (e.g., "failed login"). Correlation detects **patterns across multiple events**:

- **Multi-stage attacks**: Failed auth → privilege escalation → lateral movement
- **Behavioral anomalies**: User activity 5x higher than normal baseline
- **Alert fatigue reduction**: Suppress duplicate alerts, reduce noise by 90%+
- **Cross-source detection**: Join events from firewall + endpoint + authentication

### Gaps in Sigma

While Sigma provides 7 correlation types (event_count, value_count, temporal, temporal_ordered, value_sum, value_avg, value_percentile), it's missing critical capabilities that TelHawk addresses:

| Capability | Sigma | TelHawk | Impact |
|------------|-------|---------|--------|
| **join** - Cross-event-type correlation | ❌ | ✅ | Multi-stage attack detection |
| **suppression** - Alert deduplication | ❌ | ✅ | Reduce alert fatigue 90%+ |
| **baseline_deviation** - Anomaly detection | ❌ | ✅ | Detect behavioral anomalies |
| **missing_event** - Absence detection | ❌ | ✅ | "Dog didn't bark" scenarios |

## Documentation Structure

### Core Documentation

1. **[CORE_TYPES.md](CORE_TYPES.md)** - Tier 1 Essential Correlation Types (8 types)
   - event_count, value_count, temporal, temporal_ordered
   - join, suppression, baseline_deviation, missing_event
   - Complete parameter definitions and examples

2. **[ADVANCED_TYPES.md](ADVANCED_TYPES.md)** - Tier 2/3 Correlation Types
   - rate/velocity, ratio/proportion
   - outlier/anomaly, geographic, pattern_match, chain/graph
   - Future enhancements and use cases

3. **[PARAMETER_ARCHITECTURE.md](PARAMETER_ARCHITECTURE.md)** - Parameter System Design
   - Hybrid versioning (structural vs. tuning parameters)
   - Parameter sets (dev/prod/custom)
   - Global defaults and inheritance
   - Validation and expressions

4. **[MVC_SCHEMA.md](MVC_SCHEMA.md)** - Model/View/Controller Extensions
   - How correlation fits into detection_schemas
   - JSON examples for all types
   - Template variables and dynamic descriptions

5. **[DATABASE_SCHEMA.md](DATABASE_SCHEMA.md)** - Database Design
   - JSONB structure (no new tables)
   - Migrations and constraints
   - Indexes for performance

6. **[ALERTING_SERVICE.md](ALERTING_SERVICE.md)** - Service Architecture
   - State Manager (Redis)
   - Query Orchestrator
   - Correlation Evaluator
   - Flow diagrams and implementation

### Implementation Guides

7. **[IMPLEMENTATION.md](IMPLEMENTATION.md)** - Implementation Plan
   - 3-phase rollout (6 weeks)
   - Backward compatibility
   - API changes
   - Monitoring and metrics

8. **[TESTING.md](TESTING.md)** - Testing Strategy
   - Unit, integration, and load tests
   - Sample test data for each type
   - Edge cases and benchmarks
   - Rule testing API

9. **[ERROR_HANDLING.md](ERROR_HANDLING.md)** - Error Handling & Reliability
   - Graceful degradation (Redis down scenarios)
   - Query timeout behavior
   - Retry strategies
   - Partial result handling

10. **[PERFORMANCE.md](PERFORMANCE.md)** - Performance & Limits
    - Benchmarks and capacity planning
    - Query complexity limits
    - State storage sizing
    - Optimization strategies

### Integration Documentation

11. **[CASE_INTEGRATION.md](CASE_INTEGRATION.md)** - Case Management Integration
    - Auto-case creation from correlation alerts
    - Alert grouping strategies
    - Correlation-aware case linking

12. **[OCSF_INTEGRATION.md](OCSF_INTEGRATION.md)** - OCSF correlation_uid
    - How to populate correlation_uid during ingestion
    - Using correlation_uid in correlation engine
    - Pre-grouped event correlation

## Quick Start

### Understanding Correlation Types

Start with **[CORE_TYPES.md](CORE_TYPES.md)** to understand the 8 essential correlation types. Each type includes:
- Description and use cases
- Complete parameter reference
- Real-world examples
- Evaluation logic

### Creating Your First Correlation Rule

1. Choose a correlation type from [CORE_TYPES.md](CORE_TYPES.md)
2. Review the MVC schema in [MVC_SCHEMA.md](MVC_SCHEMA.md)
3. Create rule JSON with parameters
4. Test using the testing API (see [TESTING.md](TESTING.md))

**Example**: Simple event_count rule

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
    }
  },
  "view": {
    "title": "Brute Force Login Attempts",
    "severity": "high",
    "description": "User {{user.name}} had {{event_count}} failed logins in {{time_window}}"
  }
}
```

See [MVC_SCHEMA.md](MVC_SCHEMA.md) for complete examples.

## Design Principles

1. **Hybrid Versioning**: Structural parameters versioned, tuning parameters adjustable
2. **Embedded Configuration**: All config in existing JSONB fields (no new tables)
3. **User-Managed Parameter Sets**: Multiple named configurations (dev/prod/strict)
4. **Stateful Correlation**: Redis for persistent state (baselines, suppression, heartbeats)
5. **Backward Compatible**: Existing simple rules continue working

## Key Design Decisions

These were user-approved in the design phase:

✅ **Versioning**: Hybrid (structural versioned, tuning adjustable)
✅ **Storage**: Embed in model/controller JSONB
✅ **Environments**: User-managed parameter sets
✅ **Parameters**: Support arrays and structured data
✅ **State**: Redis for correlation state
✅ **Scope**: 8 Tier 1 essential types initially

## Architecture Overview

```
┌─────────────┐      ┌──────────────────────────────┐      ┌─────────────┐
│  Rules Svc  │─────→│  Correlation Engine          │─────→│  Storage    │
│ (schemas)   │      │                              │      │ (alerts)    │
└─────────────┘      │  ┌─────────────────────┐    │      └─────────────┘
                     │  │ Evaluator           │    │
                     │  │ - Simple rules      │    │
                     │  │ - Correlation rules │    │
                     │  └─────────────────────┘    │
                     │           │                  │
                     │           ▼                  │
                     │  ┌─────────────────────┐    │
                     │  │ State Manager       │◄───┼───► Redis
                     │  │ - Baselines         │    │    (state)
                     │  │ - Suppression cache │    │
                     │  │ - Rate tracking     │    │
                     │  └─────────────────────┘    │
                     │           │                  │
                     │           ▼                  │
                     │  ┌─────────────────────┐    │
                     │  │ Query Orchestrator  │    │
                     │  │ - Single query      │    │
                     │  │ - Multi-query (join)│    │
                     │  │ - Aggregation       │    │
                     │  └─────────────────────┘    │
                     └──────────────┬───────────────┘
                                    │
                                    ▼
                             ┌──────────────┐
                             │  OpenSearch  │
                             │   (events)   │
                             └──────────────┘
```

See [ALERTING_SERVICE.md](ALERTING_SERVICE.md) for detailed architecture.

## Implementation Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| **Phase 1: Foundation** | Weeks 1-2 | State Manager, Query Orchestrator, event_count, suppression |
| **Phase 2: Multi-Event** | Weeks 3-4 | temporal, temporal_ordered, join |
| **Phase 3: Analytics** | Weeks 5-6 | value_count, baseline_deviation, missing_event |

See [IMPLEMENTATION.md](IMPLEMENTATION.md) for complete plan.

## API Overview

### New Endpoints

```bash
# Get correlation types and schemas
GET /api/v1/correlation/types

# Set active parameter set
PUT /api/v1/schemas/{id}/parameters
Body: {"active_parameter_set": "prod"}

# Test correlation rule (dry run)
POST /api/v1/schemas/{id}/test
Body: {
  "time_range": {"from": "2024-01-01T00:00:00Z", "to": "2024-01-02T00:00:00Z"},
  "parameter_set": "dev"
}
```

### Existing Endpoints (Extended)

```bash
# Create detection schema with correlation
POST /api/v1/schemas
Body: {model: {correlation_type: "event_count", ...}, ...}

# Update schema (version created if structural change)
PUT /api/v1/schemas/{id}

# List schemas (includes correlation rules)
GET /api/v1/schemas?correlation_type=join

# Get schema with correlation config
GET /api/v1/schemas/{id}
```

See [MVC_SCHEMA.md](MVC_SCHEMA.md) and [IMPLEMENTATION.md](IMPLEMENTATION.md) for details.

## Common Use Cases

### Brute Force Detection
```json
{"correlation_type": "event_count", "time_window": "5m", "threshold": 10}
```
See: [CORE_TYPES.md#event_count](CORE_TYPES.md#1-event_count)

### Multi-Stage Attack
```json
{"correlation_type": "temporal_ordered", "sequence": ["recon", "exploit", "persistence"]}
```
See: [CORE_TYPES.md#temporal_ordered](CORE_TYPES.md#4-temporal_ordered)

### Compromised Account
```json
{"correlation_type": "join", "left": "failed_auth", "right": "privileged_action"}
```
See: [CORE_TYPES.md#join](CORE_TYPES.md#5-join)

### Behavioral Anomaly
```json
{"correlation_type": "baseline_deviation", "deviation_threshold": 2.0}
```
See: [CORE_TYPES.md#baseline_deviation](CORE_TYPES.md#7-baseline_deviation)

### Alert Deduplication
```json
{"suppression": {"enabled": true, "window": "1h", "key": ["user.name"]}}
```
See: [CORE_TYPES.md#suppression](CORE_TYPES.md#6-suppression)

## Comparison with Other SIEMs

| Feature | Splunk | Elastic SIEM | Chronicle | TelHawk |
|---------|--------|--------------|-----------|---------|
| Event count thresholds | ✅ | ✅ | ✅ | ✅ |
| Temporal correlation | ✅ (transaction) | ✅ (EQL) | ✅ (YARA-L) | ✅ |
| Cross-event joins | ✅ (correlate) | ✅ (EQL sequences) | ✅ (multi-event) | ✅ |
| Baseline deviation | ✅ (ML Toolkit) | ✅ (anomaly detection) | ✅ (UDM) | ✅ |
| Alert suppression | ✅ (throttle) | ✅ (rule actions) | ❌ | ✅ |
| Missing event detection | ✅ (scheduled) | ⚠️ (manual) | ⚠️ (manual) | ✅ |
| Sigma compatibility | ⚠️ (converters) | ✅ (Detection Engine) | ❌ | ✅ (extended) |

**TelHawk's Advantages**:
- Sigma-compatible with extensions
- OCSF-native correlation
- Built-in suppression (not addon)
- Explicit missing_event type

## Next Steps

1. **Review Documentation**: Read [CORE_TYPES.md](CORE_TYPES.md) to understand correlation types
2. **Review Architecture**: Read [ALERTING_SERVICE.md](ALERTING_SERVICE.md) for system design
3. **Review Implementation**: Read [IMPLEMENTATION.md](IMPLEMENTATION.md) for rollout plan
4. **Provide Feedback**: Submit issues or discuss in team meetings

## Contributing

This is a living specification. To propose changes:

1. Identify which doc needs updating
2. Open GitHub issue with proposal
3. Submit PR with changes
4. Update this README if adding new docs

## Questions?

- **Technical questions**: See specific doc (e.g., [ALERTING_SERVICE.md](ALERTING_SERVICE.md))
- **Use case questions**: See [CORE_TYPES.md](CORE_TYPES.md) or [ADVANCED_TYPES.md](ADVANCED_TYPES.md)
- **Implementation questions**: See [IMPLEMENTATION.md](IMPLEMENTATION.md)
- **General questions**: Open GitHub issue or team discussion
