# Correlation Testing Guide

**Status**: Design Phase
**Version**: 1.0
**Last Updated**: 2025-11-11

## Overview

This document provides comprehensive testing strategies, sample data, and scenarios for validating the correlation system.

## Testing Strategy

### Test Levels

1. **Unit Tests** - Individual correlation type evaluators
2. **Integration Tests** - End-to-end correlation with Redis and OpenSearch
3. **Load Tests** - Performance under high event volumes
4. **Chaos Tests** - Reliability during failures (Redis down, OpenSearch slow)

---

## Unit Testing

### Test Structure

```go
func TestEventCountEvaluator(t *testing.T) {
    tests := []struct{
        name string
        schema *DetectionSchema
        events []*Event
        expected []*Alert
    }{
        {
            name: "triggers_on_threshold",
            schema: &DetectionSchema{...},
            events: generateEvents(15), // Above threshold
            expected: []*Alert{{...}},
        },
        {
            name: "no_alert_below_threshold",
            schema: &DetectionSchema{...},
            events: generateEvents(5), // Below threshold
            expected: nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            evaluator := &EventCountEvaluator{}
            alerts := evaluator.Evaluate(ctx, tt.schema, orch, state)
            assert.Equal(t, tt.expected, alerts)
        })
    }
}
```

### Sample Test Data

#### OCSF Authentication Event (Failed Login)
```json
{
  "class_uid": 3002,
  "category_uid": 3,
  "type_uid": 300201,
  "activity_id": 1,
  "status_id": 2,
  "time": 1699564800,
  "user": {
    "name": "admin",
    "uid": "1000"
  },
  "src_endpoint": {
    "ip": "10.0.1.100",
    "port": 54321
  },
  "dst_endpoint": {
    "ip": "10.0.1.5",
    "port": 22
  },
  "metadata": {
    "product": {
      "name": "SSH Server",
      "vendor_name": "OpenSSH"
    },
    "version": "1.1.0"
  }
}
```

#### OCSF File Activity Event
```json
{
  "class_uid": 4005,
  "category_uid": 4,
  "type_uid": 400505,
  "activity_id": 5,
  "time": 1699564850,
  "actor": {
    "user": {
      "name": "admin",
      "uid": "1000"
    }
  },
  "file": {
    "path": "/etc/passwd",
    "name": "passwd",
    "type_id": 1
  },
  "metadata": {
    "product": {
      "name": "Endpoint Agent"
    },
    "version": "1.1.0"
  }
}
```

---

## Integration Testing

### Test Scenarios by Correlation Type

#### 1. event_count Integration Test

**Scenario**: Brute force detection

**Setup**:
```go
// Create rule
schema := createEventCountSchema("brute_force", 10, "5m")

// Generate events
events := []Event{}
for i := 0; i < 15; i++ {
    events = append(events, createFailedAuthEvent("admin", "10.0.1.100"))
}

// Index events in OpenSearch
for _, event := range events {
    indexEvent(event)
}
```

**Execute**:
```go
// Run evaluator
evaluator.Run(ctx, 1*time.Second) // Single evaluation cycle
```

**Verify**:
```go
// Check alert generated
alerts := getAlertsFromOpenSearch()
assert.Len(t, alerts, 1)
assert.Equal(t, "Brute Force Login Attempts", alerts[0].Title)
assert.Equal(t, 15, alerts[0].Metadata["event_count"])
```

---

#### 2. join Integration Test

**Scenario**: Failed auth followed by privileged action

**Setup**:
```go
schema := createJoinSchema(
    "auth_then_privilege",
    "class_uid:3002 AND status_id:2",  // Failed auth
    "class_uid:4005 AND activity_id:5", // File delete
    "10m",
)

// Generate correlated events
events := []Event{
    createFailedAuthEvent("admin", "10.0.1.100", time.Now()),
    createFileDeleteEvent("admin", "/etc/passwd", time.Now().Add(2*time.Minute)),
}

indexEvents(events)
```

**Verify**:
```go
alerts := getAlertsFromOpenSearch()
assert.Len(t, alerts, 1)
assert.Contains(t, alerts[0].MatchedEvents, events[0].ID)
assert.Contains(t, alerts[0].MatchedEvents, events[1].ID)
```

---

#### 3. baseline_deviation Integration Test

**Scenario**: Abnormal file access volume

**Setup**:
```go
// Build baseline over 7 days
for day := 0; day < 7; day++ {
    // Normal: 100 files/day
    for i := 0; i < 100; i++ {
        event := createFileAccessEvent("admin", time.Now().AddDate(0, 0, -7+day))
        indexEvent(event)
    }
}

// Run evaluator to build baseline
evaluator.Run(ctx, 1*time.Second)

// Anomalous day: 500 files
for i := 0; i < 500; i++ {
    event := createFileAccessEvent("admin", time.Now())
    indexEvent(event)
}
```

**Verify**:
```go
evaluator.Run(ctx, 1*time.Second)
alerts := getAlertsFromOpenSearch()
assert.Len(t, alerts, 1)
assert.Greater(t, alerts[0].Metadata["deviation"], 2.0) // > 2 standard deviations
```

---

#### 4. suppression Integration Test

**Scenario**: Alert deduplication

**Setup**:
```go
schema := createEventCountSchemaWithSuppression(
    "brute_force",
    10,
    "5m",
    "1h", // 1-hour suppression
)

// Generate 3 waves of attacks (15 events each)
for wave := 0; wave < 3; wave++ {
    for i := 0; i < 15; i++ {
        event := createFailedAuthEvent("admin", "10.0.1.100", time.Now().Add(time.Duration(wave*10)*time.Minute))
        indexEvent(event)
    }
    evaluator.Run(ctx, 1*time.Second)
}
```

**Verify**:
```go
alerts := getAlertsFromOpenSearch()
assert.Len(t, alerts, 1) // Only 1 alert despite 3 waves (suppressed)

// Check Redis suppression state
key := fmt.Sprintf("suppression:%s:%s", schema.ID, hashKey(map[string]string{"user.name": "admin"}))
val, err := redisClient.Get(ctx, key).Result()
assert.NoError(t, err)
assert.Contains(t, val, "alert_count")
```

---

## Load Testing

### Performance Benchmarks

#### Scenario: High Event Volume

**Setup**:
- 1000 correlation rules (various types)
- 10,000 events/second ingestion rate
- 5-minute evaluation cycle

**Metrics to Track**:
```
correlation_evaluation_duration_seconds{type="event_count"} < 1.0s
correlation_evaluation_duration_seconds{type="join"} < 5.0s
correlation_query_events_fetched_total{type="event_count"} < 100,000
correlation_baseline_count{rule_id="*"} < 1,000,000

# Resource usage
redis_memory_usage_bytes < 10GB
opensearch_search_duration_seconds < 2.0s
```

**Load Test Script**:
```bash
#!/bin/bash
# Generate high event volume
for i in {1..10000}; do
    curl -s -X POST http://ingest:8088/services/collector/event \
      -H "Authorization: Splunk $HEC_TOKEN" \
      -d "{\"event\": $(generate_ocsf_event)}" &

    if (( $i % 100 == 0 )); then
        wait
    fi
done

# Monitor metrics
while true; do
    curl -s http://alerting:8085/metrics | grep correlation_
    sleep 5
done
```

---

## Edge Cases

### 1. Empty Result Set

**Test**: Rule matches no events

```go
func TestEventCount_NoEvents(t *testing.T) {
    schema := createEventCountSchema("test", 10, "5m")
    events := []*Event{} // Empty

    alerts := evaluator.Evaluate(ctx, schema, orch, state)
    assert.Nil(t, alerts)
}
```

### 2. Single Event

**Test**: Rule with threshold=1, single event

```go
func TestEventCount_SingleEvent(t *testing.T) {
    schema := createEventCountSchema("test", 1, "5m")
    events := []*Event{createEvent()}

    alerts := evaluator.Evaluate(ctx, schema, orch, state)
    assert.Len(t, alerts, 1)
}
```

### 3. Boundary Conditions

**Test**: Exactly at threshold (should not alert with "gt" operator)

```go
func TestEventCount_ExactThreshold(t *testing.T) {
    schema := createEventCountSchema("test", 10, "5m")
    schema.Controller["detection"].(map[string]interface{})["operator"] = "gt"
    events := generateEvents(10) // Exactly 10

    alerts := evaluator.Evaluate(ctx, schema, orch, state)
    assert.Nil(t, alerts) // Should NOT alert (not > 10)
}
```

### 4. Time Window Edge

**Test**: Events exactly at window boundary

```go
func TestEventCount_WindowBoundary(t *testing.T) {
    now := time.Now()
    schema := createEventCountSchema("test", 10, "5m")

    events := []*Event{
        createEvent(now.Add(-5*time.Minute)), // Exactly 5m ago
        createEvent(now.Add(-5*time.Minute - 1*time.Second)), // Just outside
    }

    // Should only count first event
}
```

### 5. Malformed Parameters

**Test**: Invalid parameter values

```go
func TestValidation_InvalidTimeWindow(t *testing.T) {
    schema := &DetectionSchema{
        Model: map[string]interface{}{
            "correlation_type": "event_count",
            "parameters": map[string]interface{}{
                "time_window": "invalid", // Not a duration
            },
        },
    }

    err := ValidateDetectionSchema(schema)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid time_window")
}
```

---

## Rule Testing API

### Endpoint: `POST /api/v1/schemas/{id}/test`

**Request**:
```json
{
  "time_range": {
    "from": "2024-01-01T00:00:00Z",
    "to": "2024-01-02T00:00:00Z"
  },
  "parameter_set": "dev",
  "dry_run": true
}
```

**Response**:
```json
{
  "schema_id": "rule-uuid",
  "schema_title": "Brute Force Detection",
  "time_range": {
    "from": "2024-01-01T00:00:00Z",
    "to": "2024-01-02T00:00:00Z"
  },
  "would_trigger": true,
  "trigger_count": 3,
  "triggers": [
    {
      "triggered_at": "2024-01-01T10:15:00Z",
      "aggregation_key": "user.name=admin,src_ip=10.0.1.100",
      "event_count": 15,
      "fields": {
        "user.name": "admin",
        "src_endpoint.ip": "10.0.1.100"
      }
    }
  ],
  "total_events_matched": 45,
  "evaluation_duration_ms": 234
}
```

### Test Workflow

```bash
# 1. Create test rule
RULE_ID=$(curl -X POST http://rules:8084/api/v1/schemas \
  -H "Content-Type: application/json" \
  -d @brute_force_rule.json | jq -r '.id')

# 2. Generate test data
./tools/event-seeder/event-seeder -token $HEC_TOKEN -count 1000 -types auth

# 3. Test rule
curl -X POST "http://rules:8084/api/v1/schemas/$RULE_ID/test" \
  -H "Content-Type: application/json" \
  -d '{
    "time_range": {"from": "1h", "to": "now"},
    "parameter_set": "dev",
    "dry_run": true
  }' | jq .

# 4. Adjust parameters based on results
curl -X PUT "http://rules:8084/api/v1/schemas/$RULE_ID/parameters" \
  -H "Content-Type: application/json" \
  -d '{"active_parameter_set": "prod"}'
```

---

## Continuous Testing

### GitHub Actions Workflow

```yaml
name: Correlation Tests

on:
  push:
    paths:
      - 'alerting/**'
      - 'rules/**'
  pull_request:

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run unit tests
        run: |
          cd alerting
          go test -v -race -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: codecov/codecov-action@v3

  integration-tests:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:7
        ports:
          - 6379:6379
      opensearch:
        image: opensearchproject/opensearch:2.11.0
        ports:
          - 9200:9200
        env:
          discovery.type: single-node
    steps:
      - uses: actions/checkout@v3
      - name: Run integration tests
        run: |
          cd alerting
          go test -v -tags=integration ./internal/evaluator

  load-tests:
    runs-on: ubuntu-latest
    if: github.event_name == 'pull_request'
    steps:
      - uses: actions/checkout@v3
      - name: Run load tests
        run: |
          docker-compose up -d
          ./tests/load/run_load_test.sh
          docker-compose down
```

---

## Test Checklist

Before marking correlation feature complete:

- [ ] All unit tests passing (>90% coverage)
- [ ] All integration tests passing
- [ ] Load test meets performance benchmarks
- [ ] Chaos tests (Redis down, OpenSearch slow) pass gracefully
- [ ] All 8 correlation types tested end-to-end
- [ ] Parameter validation tested for all types
- [ ] Suppression tested (deduplication works)
- [ ] Baseline learning tested (statistics correct)
- [ ] Missing event detection tested
- [ ] Rule testing API works
- [ ] Documentation includes test examples
- [ ] Sample rules tested in production-like environment

## References

- [IMPLEMENTATION.md](IMPLEMENTATION.md) - Implementation phases
- [PERFORMANCE.md](PERFORMANCE.md) - Performance benchmarks
- [ERROR_HANDLING.md](ERROR_HANDLING.md) - Failure scenarios
