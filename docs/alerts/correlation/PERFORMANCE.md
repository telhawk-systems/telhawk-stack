# Performance & Capacity Planning

**Status**: Design Phase
**Version**: 1.0
**Last Updated**: 2025-11-11

## Overview

This document specifies performance benchmarks, capacity limits, and optimization strategies for the correlation system.

## Target Benchmarks

### Evaluation Performance

| Metric | Target | Stretch Goal |
|--------|--------|--------------|
| Event ingestion rate | 10,000 events/sec | 50,000 events/sec |
| Concurrent correlation rules | 1,000 active | 5,000 active |
| Evaluation cycle time | < 60s | < 30s |
| event_count evaluation | < 1s per rule | < 500ms per rule |
| join evaluation | < 5s per rule | < 2s per rule |
| baseline_deviation evaluation | < 2s per rule | < 1s per rule |
| Alert generation latency | < 5s from event | < 2s from event |

### Resource Limits

| Resource | Limit | Rationale |
|----------|-------|-----------|
| Redis memory | 10GB | ~1M entities with baselines |
| OpenSearch query timeout | 30s | Prevent hung queries |
| Max events per query | 100,000 | Memory/performance balance |
| Baseline history size | 10,080 samples | 7 days at 1 min intervals |
| Suppression cache TTL | 24h max | Prevent infinite growth |
| Concurrent evaluations | 100 goroutines | CPU/memory balance |

## Performance by Correlation Type

### 1. event_count

**Query Pattern**: Single aggregation query

```
POST /telhawk-events-*/_search
{
  "size": 0,
  "query": {"bool": {"must": [...], "filter": {"range": {"time": {...}}}}},
  "aggs": {"by_entity": {"terms": {"field": "user.name"}, "aggs": {"count": {"value_count": {"field": "_id"}}}}}
}
```

**Performance**:
- **Query time**: 100-500ms (1M events)
- **Memory**: < 10MB per rule
- **Throughput**: 100+ rules/second

**Optimization**:
- Use aggregations instead of fetching all events
- Index `user.name` as keyword
- Use date math in range queries

---

### 2. join

**Query Pattern**: Two queries + in-memory join

```
Query 1: Fetch left events (failed auth)
Query 2: Fetch right events (file delete)
Join in memory by user.name
```

**Performance**:
- **Query time**: 200ms-2s per side
- **Join time**: 10-100ms (1000 events each side)
- **Memory**: ~1MB per 1000 events
- **Throughput**: 20-50 rules/second

**Bottlenecks**:
- Fetching large result sets (>10K events)
- Complex join conditions (multiple fields)
- Wide time windows (>1h)

**Optimization**:
- Limit result size with `max_events_per_window`
- Use scroll API for large result sets
- Index join fields as keywords
- Consider bloom filters for large joins

---

### 3. baseline_deviation

**Query Pattern**: Aggregation + Redis state lookup

```
1. Query OpenSearch for recent events (5m window)
2. Aggregate by entity
3. Fetch baseline from Redis
4. Calculate deviation
5. Update baseline in Redis
```

**Performance**:
- **Query time**: 100-500ms
- **Redis lookup**: 1-5ms per entity
- **Calculation**: < 1ms per entity
- **Redis update**: 1-5ms per entity
- **Throughput**: 50-100 rules/second

**State Size**:
```
Baseline: ~2KB per entity (10,080 samples)
1M entities = 2GB Redis memory
```

**Optimization**:
- Pipeline Redis operations (batch lookups)
- Use Welford's algorithm (online stats, no array storage)
- Compress baseline data (reduce to mean/stddev only)
- TTL old baselines

---

### 4. suppression

**Performance**:
- **Redis lookup**: 1-5ms per alert
- **Redis write**: 1-5ms per alert
- **Overhead**: < 10ms per alert

**State Size**:
```
Suppression entry: ~500 bytes
1M active suppressions = 500MB Redis memory
```

---

## Capacity Planning

### Small Deployment (1K events/sec)

```yaml
# Recommended resources
alerting:
  replicas: 2
  cpu: 2 cores
  memory: 4GB

redis:
  memory: 2GB
  persistence: RDB (snapshot)

opensearch:
  nodes: 3
  cpu: 4 cores each
  memory: 16GB each
  storage: 500GB
```

**Capacity**:
- 1,000 events/sec = 86M events/day
- 500 correlation rules
- 100K entities with baselines

---

### Medium Deployment (10K events/sec)

```yaml
alerting:
  replicas: 5
  cpu: 4 cores
  memory: 8GB

redis:
  memory: 10GB
  persistence: AOF (append-only)
  replicas: 2 (HA)

opensearch:
  nodes: 6
  cpu: 8 cores each
  memory: 32GB each
  storage: 5TB
```

**Capacity**:
- 10,000 events/sec = 864M events/day
- 1,000-2,000 correlation rules
- 1M entities with baselines

---

### Large Deployment (50K events/sec)

```yaml
alerting:
  replicas: 10
  cpu: 8 cores
  memory: 16GB

redis:
  cluster: true
  nodes: 6
  memory: 64GB total
  persistence: AOF + RDB

opensearch:
  nodes: 12+
  cpu: 16 cores each
  memory: 64GB each
  storage: 20TB+
```

**Capacity**:
- 50,000 events/sec = 4.3B events/day
- 5,000+ correlation rules
- 10M entities with baselines

---

## Optimization Strategies

### 1. Query Optimization

**Use Aggregations**:
```go
// Bad: Fetch all events
events := fetchEvents(query, timeWindow)
count := len(events)

// Good: Use aggregation
resp := aggregateEvents(query, timeWindow, []string{"user.name"})
count := resp.Aggregations["by_user"].DocCount
```

**Date Math in Queries**:
```json
{
  "query": {
    "range": {
      "time": {
        "gte": "now-5m",
        "lt": "now"
      }
    }
  }
}
```

**Field Filtering**:
```json
{
  "_source": ["user.name", "src_endpoint.ip", "time"],
  "query": {...}
}
```

---

### 2. Redis Optimization

**Pipeline Operations**:
```go
pipe := redis.Pipeline()
for _, entity := range entities {
    pipe.Get(ctx, fmt.Sprintf("baseline:%s:%s", ruleID, entity))
}
results, err := pipe.Exec(ctx)
```

**Compress Baselines** (Welford's algorithm):
```go
// Instead of storing all samples (2KB)
type Baseline struct {
    Samples []float64 // 10,080 * 8 bytes = 80KB
}

// Store only statistics (200 bytes)
type Baseline struct {
    Count   int64
    Mean    float64
    M2      float64 // Sum of squared differences
    Min     float64
    Max     float64
}

func (b *Baseline) Update(value float64) {
    b.Count++
    delta := value - b.Mean
    b.Mean += delta / float64(b.Count)
    delta2 := value - b.Mean
    b.M2 += delta * delta2
}

func (b *Baseline) StdDev() float64 {
    return math.Sqrt(b.M2 / float64(b.Count))
}
```

---

### 3. Concurrent Evaluation

**Goroutine Pool**:
```go
func (ce *CorrelationEvaluator) evaluateConcurrent(schemas []*DetectionSchema) {
    semaphore := make(chan struct{}, ce.config.MaxConcurrency)
    var wg sync.WaitGroup

    for _, schema := range schemas {
        wg.Add(1)
        go func(s *DetectionSchema) {
            defer wg.Done()
            semaphore <- struct{}{}        // Acquire
            defer func() { <-semaphore }() // Release

            ce.evaluateRule(ctx, s)
        }(schema)
    }

    wg.Wait()
}
```

---

### 4. Caching

**Rule Caching**:
```go
type RuleCache struct {
    cache *ristretto.Cache
    ttl   time.Duration
}

func (rc *RuleCache) GetSchemas(ctx context.Context) ([]*DetectionSchema, error) {
    if schemas, found := rc.cache.Get("active_schemas"); found {
        return schemas.([]*DetectionSchema), nil
    }

    schemas, err := rc.rulesClient.ListSchemas(ctx)
    if err != nil {
        return nil, err
    }

    rc.cache.SetWithTTL("active_schemas", schemas, 1, rc.ttl)
    return schemas, nil
}
```

---

## Monitoring

### Key Metrics

```
# Evaluation performance
correlation_evaluation_duration_seconds{type, rule_id} histogram
correlation_query_duration_seconds{type} histogram
correlation_baseline_calculation_duration_seconds histogram

# Throughput
correlation_rules_evaluated_total counter
correlation_events_processed_total counter
correlation_alerts_generated_total counter

# Resource usage
correlation_redis_operations_total{operation} counter
correlation_redis_latency_seconds{operation} histogram
correlation_opensearch_query_size_bytes histogram
correlation_memory_usage_bytes gauge
correlation_goroutines_active gauge

# Limits
correlation_query_events_fetched{rule_id} histogram
correlation_query_timeout_total counter
correlation_max_events_exceeded_total counter
```

### Dashboard Queries

**Average Evaluation Time by Type**:
```promql
rate(correlation_evaluation_duration_seconds_sum[5m])
/
rate(correlation_evaluation_duration_seconds_count[5m])
```

**Redis Hit Rate**:
```promql
rate(correlation_redis_hits_total[5m])
/
(rate(correlation_redis_hits_total[5m]) + rate(correlation_redis_misses_total[5m]))
```

**Query Size Distribution**:
```promql
histogram_quantile(0.95, 
  rate(correlation_opensearch_query_size_bytes_bucket[5m])
)
```

---

## Load Testing

### Test Scenarios

#### Scenario 1: High Rule Count

```bash
# Create 5000 correlation rules
for i in {1..5000}; do
    curl -X POST http://rules:8084/api/v1/schemas \
      -H "Content-Type: application/json" \
      -d "{
        \"model\": {\"correlation_type\": \"event_count\", \"parameters\": {\"time_window\": \"5m\"}},
        \"controller\": {\"detection\": {\"query\": \"class_uid:$((3000 + RANDOM % 1000))\", \"threshold\": 10}},
        \"view\": {\"title\": \"Rule $i\", \"severity\": \"medium\"}
      }"
done

# Monitor evaluation time
watch 'curl -s http://alerting:8085/metrics | grep correlation_evaluation_duration'
```

#### Scenario 2: High Event Volume

```bash
# Generate 50K events/sec for 5 minutes
./tools/event-seeder/event-seeder \
  -token $HEC_TOKEN \
  -count 15000000 \
  -interval 0 \
  -batch-size 1000 \
  -workers 50

# Monitor lag
watch 'curl -s http://alerting:8085/metrics | grep correlation_evaluation_lag'
```

#### Scenario 3: Complex Joins

```bash
# Create 100 join rules with wide time windows
for i in {1..100}; do
    create_join_rule "rule_$i" "1h"
done

# Generate correlated events
generate_correlated_events 10000

# Monitor join performance
watch 'curl -s http://alerting:8085/metrics | grep "correlation_evaluation_duration.*join"'
```

---

## Troubleshooting

### Slow Evaluation

**Symptoms**: `correlation_evaluation_duration_seconds` > 60s

**Investigation**:
```bash
# Check slow queries
curl http://alerting:8085/metrics | grep correlation_query_duration | sort -n

# Check Redis latency
redis-cli --latency-history

# Check OpenSearch slow logs
curl http://opensearch:9200/_cluster/settings?include_defaults=true | jq '.defaults.search.slowlog'
```

**Solutions**:
- Reduce rule count or time windows
- Increase OpenSearch resources
- Add caching layer
- Optimize queries (use aggregations)

---

### High Memory Usage

**Symptoms**: `correlation_memory_usage_bytes` growing unbounded

**Investigation**:
```bash
# Check Redis memory
redis-cli INFO memory

# Check baseline count
redis-cli DBSIZE

# Check largest keys
redis-cli --bigkeys
```

**Solutions**:
- Reduce baseline history size
- Set TTLs on all keys
- Use Welford's algorithm (compress baselines)
- Increase Redis maxmemory + LRU eviction

---

## References

- [TESTING.md](TESTING.md) - Load test procedures
- [ERROR_HANDLING.md](ERROR_HANDLING.md) - Performance degradation handling
- [IMPLEMENTATION.md](IMPLEMENTATION.md) - Performance goals per phase
