# Error Handling & Reliability

**Status**: Design Phase
**Version**: 1.0
**Last Updated**: 2025-11-11

## Overview

This document specifies error handling strategies, graceful degradation patterns, and reliability mechanisms for the correlation system.

## Design Principles

1. **Fail Gracefully**: Correlation failures should not block event ingestion or simple rule evaluation
2. **Retry Smartly**: Exponential backoff with jitter for transient failures
3. **Degrade Predictably**: Clear fallback behavior when dependencies unavailable
4. **Alert on Errors**: System health issues trigger internal alerts
5. **Preserve State**: Minimize data loss during failures

---

## Failure Scenarios

### 1. Redis Unavailable

**Impact**: All stateful correlation types affected (baseline_deviation, suppression, missing_event, rate tracking)

#### Behavior

**Option A: Fail Open (Recommended)**
```go
func (sm *StateManager) GetBaseline(ruleID, entityKey string) (*Baseline, error) {
    baseline, err := sm.redis.Get(ctx, key).Result()
    if err == redis.Nil {
        return nil, ErrNoBaseline
    }
    if err != nil {
        log.Warn("Redis unavailable, skipping baseline check", "error", err)
        return nil, ErrRedisUnavailable
    }
    return baseline, nil
}

// In evaluator
baseline, err := stateManager.GetBaseline(ruleID, entityKey)
if err == ErrRedisUnavailable {
    // Skip this correlation rule, continue with others
    log.Warn("Skipping baseline_deviation rule due to Redis unavailability", "rule_id", ruleID)
    return nil // No alert (fail open)
}
```

**Option B: Fail Closed**
```go
// Alert on every evaluation (treat missing baseline as anomaly)
if err == ErrRedisUnavailable {
    // Generate degraded mode alert
    return &Alert{
        Title: fmt.Sprintf("%s (DEGRADED MODE - Redis unavailable)", rule.View.Title),
        Severity: "medium",
        Description: "Unable to verify baseline, alerting conservatively",
    }
}
```

**Configuration**:
```yaml
alerting:
  correlation:
    redis_failure_mode: "fail_open" # or "fail_closed"
    redis_retry_attempts: 3
    redis_retry_backoff: "exponential" # 1s, 2s, 4s
```

**Metrics**:
```
correlation_redis_errors_total{rule_id, type} counter
correlation_degraded_evaluations_total{rule_id} counter
```

---

### 2. OpenSearch Query Timeout

**Impact**: Queries exceed timeout, no events returned

#### Behavior

```go
func (qo *QueryOrchestrator) ExecuteQuery(ctx context.Context, query string, window time.Duration) ([]*Event, error) {
    // Set timeout from config
    queryCtx, cancel := context.WithTimeout(ctx, qo.config.QueryTimeout)
    defer cancel()

    events, err := qo.storageClient.Search(queryCtx, query, window)
    if err == context.DeadlineExceeded {
        // Log slow query
        log.Error("Query timeout", "query", query, "timeout", qo.config.QueryTimeout)

        // Increment metric
        queryTimeoutCounter.WithLabelValues(query).Inc()

        // Return empty result (don't fail entire evaluation)
        return []*Event{}, ErrQueryTimeout
    }

    return events, err
}
```

**Retry Logic**:
```go
func (qo *QueryOrchestrator) ExecuteQueryWithRetry(ctx context.Context, query string, window time.Duration) ([]*Event, error) {
    var events []*Event
    var err error

    for attempt := 1; attempt <= qo.config.MaxRetries; attempt++ {
        events, err = qo.ExecuteQuery(ctx, query, window)

        if err == nil {
            return events, nil
        }

        if err == ErrQueryTimeout {
            // Don't retry timeout (likely slow query, won't help)
            return nil, err
        }

        if !isRetryable(err) {
            return nil, err
        }

        // Exponential backoff with jitter
        backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
        jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
        time.Sleep(backoff + jitter)

        log.Warn("Retrying query", "attempt", attempt, "error", err)
    }

    return nil, fmt.Errorf("query failed after %d attempts: %w", qo.config.MaxRetries, err)
}
```

**Timeout Configuration**:
```yaml
alerting:
  correlation:
    query_timeout: "30s"
    query_max_retries: 3
    slow_query_threshold: "10s" # Log slow queries
```

---

### 3. Query Returns Partial Results

**Impact**: OpenSearch returns partial results due to shard failures

#### Detection

```go
type SearchResponse struct {
    Hits   Hits   `json:"hits"`
    Shards Shards `json:"_shards"`
}

type Shards struct {
    Total      int `json:"total"`
    Successful int `json:"successful"`
    Failed     int `json:"failed"`
}

func (qo *QueryOrchestrator) ExecuteQuery(ctx context.Context, query string, window time.Duration) ([]*Event, error) {
    resp, err := qo.storageClient.Search(ctx, query, window)
    if err != nil {
        return nil, err
    }

    // Check for partial results
    if resp.Shards.Failed > 0 {
        failureRate := float64(resp.Shards.Failed) / float64(resp.Shards.Total)

        if failureRate > 0.5 {
            // More than 50% shards failed - unreliable
            return nil, fmt.Errorf("too many shard failures: %d/%d", resp.Shards.Failed, resp.Shards.Total)
        }

        // Log warning but continue with partial results
        log.Warn("Partial search results", "failed_shards", resp.Shards.Failed, "total_shards", resp.Shards.Total)
        partialResultsCounter.WithLabelValues(query).Inc()
    }

    return resp.Hits.Events, nil
}
```

---

### 4. Rule Parameter Validation Errors

**Impact**: Invalid parameters in rule definition

#### Validation Stages

**1. At Rule Creation (Fail Fast)**
```go
func ValidateDetectionSchema(schema *DetectionSchema) error {
    correlationType := schema.Model["correlation_type"].(string)

    validator, exists := validators[correlationType]
    if !exists {
        return fmt.Errorf("unknown correlation type: %s", correlationType)
    }

    if err := validator.ValidateParameters(schema.Model["parameters"]); err != nil {
        return fmt.Errorf("parameter validation failed: %w", err)
    }

    // Validate parameter sets
    paramSets := schema.Model["parameter_sets"].([]map[string]interface{})
    for _, set := range paramSets {
        if err := validator.ValidateParameters(set); err != nil {
            return fmt.Errorf("parameter set '%s' validation failed: %w", set["name"], err)
        }
    }

    return nil
}
```

**2. At Evaluation Time (Graceful Fallback)**
```go
func (ce *CorrelationEvaluator) evaluateRule(ctx context.Context, schema *DetectionSchema) []*Alert {
    // Validate at runtime (in case rule modified outside system)
    if err := ValidateDetectionSchema(schema); err != nil {
        log.Error("Rule validation failed at evaluation", "rule_id", schema.ID, "error", err)

        // Disable rule automatically
        ce.rulesClient.DisableSchema(ctx, schema.ID)

        // Alert administrators
        ce.sendSystemAlert(&Alert{
            Title: fmt.Sprintf("Rule Disabled: %s", schema.View["title"]),
            Severity: "high",
            Description: fmt.Sprintf("Rule %s disabled due to validation error: %v", schema.ID, err),
        })

        return nil
    }

    // Continue with evaluation...
}
```

---

### 5. State Corruption

**Impact**: Redis state data corrupted or invalid

#### Detection & Recovery

```go
func (sm *StateManager) GetBaseline(ruleID, entityKey string) (*Baseline, error) {
    data, err := sm.redis.Get(ctx, key).Result()
    if err != nil {
        return nil, err
    }

    var baseline Baseline
    if err := json.Unmarshal([]byte(data), &baseline); err != nil {
        // Corrupted data
        log.Error("Corrupted baseline data", "rule_id", ruleID, "entity", entityKey, "error", err)

        // Delete corrupted data
        sm.redis.Del(ctx, key)

        // Start fresh baseline
        return nil, ErrCorruptedBaseline
    }

    // Validate baseline sanity
    if baseline.Count < 0 || baseline.StdDev < 0 || math.IsNaN(baseline.Mean) {
        log.Error("Invalid baseline statistics", "baseline", baseline)
        sm.redis.Del(ctx, key)
        return nil, ErrInvalidBaseline
    }

    return &baseline, nil
}
```

---

### 6. Memory Exhaustion

**Impact**: Redis or application runs out of memory

#### Prevention

**1. TTL on All Keys**
```go
func (sm *StateManager) RecordBaseline(ruleID, entityKey string, baseline *Baseline) error {
    key := fmt.Sprintf("baseline:%s:%s", ruleID, entityKey)
    data, _ := json.Marshal(baseline)

    // Always set TTL (2x baseline window)
    ttl := baseline.Window * 2
    return sm.redis.Set(ctx, key, data, ttl).Err()
}
```

**2. Memory Limits**
```yaml
# redis.conf
maxmemory 10gb
maxmemory-policy allkeys-lru # Evict least recently used
```

**3. Monitoring**
```go
// Periodic memory check
func (sm *StateManager) monitorMemory(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            info, err := sm.redis.Info(ctx, "memory").Result()
            if err != nil {
                log.Error("Failed to get Redis memory info", "error", err)
                continue
            }

            usedMemory := parseUsedMemory(info)
            maxMemory := parseMaxMemory(info)
            usagePercent := float64(usedMemory) / float64(maxMemory) * 100

            redisMemoryUsage.Set(float64(usedMemory))
            redisMemoryUsagePercent.Set(usagePercent)

            if usagePercent > 90 {
                log.Warn("Redis memory usage high", "percent", usagePercent)
            }
        }
    }
}
```

---

## Retry Strategies

### Exponential Backoff with Jitter

```go
type RetryConfig struct {
    MaxAttempts int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Jitter      bool
}

func retry(ctx context.Context, cfg RetryConfig, fn func() error) error {
    var err error

    for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
        err = fn()
        if err == nil {
            return nil
        }

        if !isRetryable(err) {
            return err
        }

        if attempt < cfg.MaxAttempts-1 {
            delay := time.Duration(math.Pow(2, float64(attempt))) * cfg.BaseDelay
            if delay > cfg.MaxDelay {
                delay = cfg.MaxDelay
            }

            if cfg.Jitter {
                jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)
                delay += jitter
            }

            log.Debug("Retrying after delay", "attempt", attempt+1, "delay", delay, "error", err)

            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    }

    return fmt.Errorf("operation failed after %d attempts: %w", cfg.MaxAttempts, err)
}

func isRetryable(err error) bool {
    // Network errors
    if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
        return true
    }

    // Temporary errors
    if e, ok := err.(interface{ Temporary() bool }); ok && e.Temporary() {
        return true
    }

    // Rate limiting
    if errors.Is(err, ErrRateLimited) {
        return true
    }

    return false
}
```

---

## Circuit Breaker Pattern

### Prevent Cascading Failures

```go
type CircuitBreaker struct {
    maxFailures  int
    resetTimeout time.Duration
    state        CircuitState
    failures     int
    lastFailure  time.Time
    mu           sync.Mutex
}

type CircuitState int

const (
    StateClosed CircuitState = iota // Normal operation
    StateOpen                        // Failing, reject requests
    StateHalfOpen                    // Testing if recovered
)

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()

    if cb.state == StateOpen {
        if time.Since(cb.lastFailure) > cb.resetTimeout {
            cb.state = StateHalfOpen
            log.Info("Circuit breaker half-open, testing")
        } else {
            cb.mu.Unlock()
            return ErrCircuitOpen
        }
    }

    cb.mu.Unlock()

    err := fn()

    cb.mu.Lock()
    defer cb.mu.Unlock()

    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()

        if cb.failures >= cb.maxFailures {
            cb.state = StateOpen
            log.Warn("Circuit breaker opened", "failures", cb.failures)
        }

        return err
    }

    // Success
    if cb.state == StateHalfOpen {
        cb.state = StateClosed
        log.Info("Circuit breaker closed, recovered")
    }
    cb.failures = 0

    return nil
}
```

**Usage**:
```go
// Wrap Redis operations
err := redisCircuitBreaker.Call(func() error {
    return stateManager.GetBaseline(ruleID, entityKey)
})

if err == ErrCircuitOpen {
    log.Warn("Skipping Redis operation, circuit open")
    // Fall back to degraded mode
}
```

---

## Health Checks

### Alerting Service Health Endpoint

```go
type HealthStatus struct {
    Status       string            `json:"status"` // healthy, degraded, unhealthy
    Redis        ComponentHealth   `json:"redis"`
    OpenSearch   ComponentHealth   `json:"opensearch"`
    Rules        ComponentHealth   `json:"rules"`
    LastEvalTime time.Time         `json:"last_eval_time"`
    Metrics      map[string]interface{} `json:"metrics"`
}

type ComponentHealth struct {
    Status  string `json:"status"`
    Latency string `json:"latency"`
    Error   string `json:"error,omitempty"`
}

func (s *Service) Health(ctx context.Context) (*HealthStatus, error) {
    status := &HealthStatus{
        Status: "healthy",
        Metrics: make(map[string]interface{}),
    }

    // Check Redis
    start := time.Now()
    if err := s.stateManager.Ping(ctx); err != nil {
        status.Redis = ComponentHealth{Status: "unhealthy", Error: err.Error()}
        status.Status = "degraded"
    } else {
        status.Redis = ComponentHealth{Status: "healthy", Latency: time.Since(start).String()}
    }

    // Check OpenSearch
    start = time.Now()
    if err := s.orchestrator.Ping(ctx); err != nil {
        status.OpenSearch = ComponentHealth{Status: "unhealthy", Error: err.Error()}
        status.Status = "unhealthy" // Critical dependency
    } else {
        status.OpenSearch = ComponentHealth{Status: "healthy", Latency: time.Since(start).String()}
    }

    // Check Rules Service
    start = time.Now()
    if _, err := s.rulesClient.ListSchemas(ctx); err != nil {
        status.Rules = ComponentHealth{Status: "unhealthy", Error: err.Error()}
        status.Status = "degraded"
    } else {
        status.Rules = ComponentHealth{Status: "healthy", Latency: time.Since(start).String()}
    }

    // Add metrics
    status.LastEvalTime = s.evaluator.lastEvalTime
    status.Metrics["active_rules"] = s.evaluator.activeRuleCount
    status.Metrics["alerts_generated"] = s.evaluator.alertsGenerated

    return status, nil
}
```

---

## Monitoring & Alerting

### Key Metrics

```
# Error rates
correlation_errors_total{type, rule_id, error_type} counter
correlation_redis_errors_total{operation} counter
correlation_query_errors_total{query_type} counter

# Degradation
correlation_degraded_mode_active{rule_id} gauge
correlation_partial_results_total counter

# Health
correlation_evaluation_lag_seconds gauge # Time since last successful eval
correlation_redis_latency_seconds histogram
correlation_query_latency_seconds histogram
```

### Alert Rules (Prometheus)

```yaml
groups:
  - name: correlation_health
    rules:
      - alert: CorrelationRedisDown
        expr: correlation_redis_errors_total > 10
        for: 5m
        annotations:
          summary: "Redis unavailable, correlation degraded"

      - alert: CorrelationEvaluationStalled
        expr: time() - correlation_last_evaluation_timestamp > 600
        annotations:
          summary: "Correlation evaluation hasn't run in 10 minutes"

      - alert: CorrelationHighErrorRate
        expr: rate(correlation_errors_total[5m]) > 0.1
        annotations:
          summary: "Correlation error rate > 10%"
```

---

## Summary

| Failure Type | Strategy | Fallback | User Impact |
|--------------|----------|----------|-------------|
| Redis down | Fail open | Skip stateful rules | Some alerts missed (temporary) |
| Query timeout | Skip rule | Log error | Affected rule doesn't alert |
| Partial results | Continue if >50% shards OK | Log warning | Potentially incomplete detection |
| Invalid params | Disable rule | Alert admins | Rule stops alerting |
| State corruption | Delete & rebuild | Start fresh baseline | Temporary false negatives |
| Memory exhaustion | LRU eviction | Oldest baselines dropped | Older entities lose history |

## References

- [TESTING.md](TESTING.md) - Chaos testing scenarios
- [PERFORMANCE.md](PERFORMANCE.md) - Performance limits
- [IMPLEMENTATION.md](IMPLEMENTATION.md) - Implementation phases
