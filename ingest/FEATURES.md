# Ingest Service - Rate Limiting, Ack Channel, and Metrics

This document describes the three key features added to the ingest service: Redis-backed rate limiting, HEC acknowledgement channel, and Prometheus metrics.

## Features

### 1. Redis-Backed Rate Limiting

**Purpose:** Protect the ingestion pipeline from being overwhelmed by limiting request rates per IP address and per authenticated token.

**Architecture:**
- **Two-tier rate limiting:**
  - **IP-based:** Applied BEFORE token validation to protect against unauthenticated DoS attacks
  - **Token-based:** Applied AFTER authentication to prevent abuse by valid tokens
- **Redis sliding window algorithm:** Uses sorted sets for precise rate limiting across distributed instances
- **Atomic operations:** Lua scripts ensure race-free rate limit checks

**Configuration:**
```yaml
# config.yaml
redis:
  url: redis://localhost:6379/0
  enabled: true

ingestion:
  rate_limit_enabled: true
  rate_limit_requests: 10000  # requests per window
  rate_limit_window: 1m       # time window
```

**Environment Variables:**
```bash
INGEST_REDIS_URL=redis://redis:6379/0
INGEST_REDIS_ENABLED=true
INGEST_INGESTION_RATE_LIMIT_ENABLED=true
INGEST_INGESTION_RATE_LIMIT_REQUESTS=10000
INGEST_INGESTION_RATE_LIMIT_WINDOW=1m
```

**Behavior:**
- If Redis is unavailable, service degrades gracefully (logs warning and continues without rate limiting)
- Rate limit exceeded returns HTTP 429 (Too Many Requests)
- Separate rate limit keys for IP (`ip:<address>`) and token (`token:<id>`)

**Metrics:**
- `telhawk_ingest_rate_limit_hits_total{token}` - Counter of rate limit violations per token

### 2. HEC Acknowledgement Channel

**Purpose:** Provide guaranteed delivery semantics compatible with Splunk HEC ack protocol.

**Architecture:**
- **In-memory ack manager** with configurable TTL
- **Per-event tracking:** Each event gets a unique ack ID
- **Automatic cleanup:** Expired acks removed after TTL
- **Status tracking:** Pending, Success, Failed

**Configuration:**
```yaml
# config.yaml
ack:
  enabled: true
  ttl: 10m  # how long to keep ack records
```

**Environment Variables:**
```bash
INGEST_ACK_ENABLED=true
INGEST_ACK_TTL=10m
```

**Usage:**

1. **Client sends event with ack request:**
```bash
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Splunk <token>" \
  -H "X-Splunk-Request-Channel: <channel-id>" \
  -d '{"event": "test"}'
```

Response includes ack ID:
```json
{
  "text": "Success",
  "code": 0,
  "ackId": "550e8400-e29b-41d4-a716-446655440000"
}
```

2. **Client queries ack status:**
```bash
curl -X POST http://localhost:8088/services/collector/ack \
  -H "Content-Type: application/json" \
  -d '{"acks": ["550e8400-e29b-41d4-a716-446655440000"]}'
```

Response:
```json
{
  "acks": {
    "550e8400-e29b-41d4-a716-446655440000": true
  }
}
```

**Metrics:**
- `telhawk_ingest_acks_pending` - Gauge of pending acknowledgements
- `telhawk_ingest_acks_completed_total` - Counter of completed acknowledgements

### 3. Prometheus Metrics

**Purpose:** Observable ingestion pipeline with detailed metrics for monitoring and alerting.

**Endpoint:** `GET /metrics`

**Available Metrics:**

#### Event Ingestion
- `telhawk_ingest_events_total{endpoint, status}` - Total events received
  - `endpoint`: "event" or "raw"
  - `status`: "accepted", "rate_limited", "token_rate_limited", "queue_full"
- `telhawk_ingest_event_bytes_total` - Total bytes of event data received

#### Queue Metrics
- `telhawk_ingest_queue_depth` - Current number of events in queue
- `telhawk_ingest_queue_capacity` - Maximum queue capacity

#### Normalization
- `telhawk_ingest_normalization_duration_seconds` - Histogram of normalization latency
- `telhawk_ingest_normalization_errors_total` - Count of normalization failures

#### Storage
- `telhawk_ingest_storage_duration_seconds` - Histogram of storage operation latency
- `telhawk_ingest_storage_errors_total` - Count of storage failures

#### Rate Limiting
- `telhawk_ingest_rate_limit_hits_total{token}` - Rate limit violations by token

#### Acknowledgements
- `telhawk_ingest_acks_pending` - Number of pending acks
- `telhawk_ingest_acks_completed_total` - Total completed acks

**Example Prometheus Queries:**

```promql
# Event ingestion rate
rate(telhawk_ingest_events_total{status="accepted"}[5m])

# Queue utilization percentage
(telhawk_ingest_queue_depth / telhawk_ingest_queue_capacity) * 100

# 95th percentile normalization latency
histogram_quantile(0.95, rate(telhawk_ingest_normalization_duration_seconds_bucket[5m]))

# Error rate
rate(telhawk_ingest_normalization_errors_total[5m]) + rate(telhawk_ingest_storage_errors_total[5m])
```

## Testing

### Rate Limiting Test
```bash
# Spam requests to trigger rate limit
for i in {1..100}; do
  curl -X POST http://localhost:8088/services/collector/event \
    -H "Authorization: Splunk test-token" \
    -d '{"event": "test"}' &
done
wait

# Check metrics
curl http://localhost:8088/metrics | grep rate_limit_hits
```

### Ack Channel Test
```bash
# Send event
response=$(curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Splunk test-token" \
  -d '{"event": "test"}' -s)

# Extract ack ID (if returned in response)
ack_id=$(echo $response | jq -r '.ackId')

# Query ack status
curl -X POST http://localhost:8088/services/collector/ack \
  -H "Content-Type: application/json" \
  -d "{\"acks\": [\"$ack_id\"]}"
```

### Metrics Test
```bash
# View all metrics
curl http://localhost:8088/metrics

# Filter specific metrics
curl http://localhost:8088/metrics | grep telhawk_ingest_queue
```

## Production Considerations

### Rate Limiting
- **Redis high availability:** Use Redis Sentinel or Cluster for production
- **Rate limit tuning:** Adjust based on expected traffic and system capacity
- **IP extraction:** Configure trusted proxy headers (X-Forwarded-For) if behind load balancer

### Acknowledgements
- **TTL tuning:** Balance between memory usage and client polling needs
- **Persistence:** Current implementation is in-memory; consider Redis backing for HA
- **Cleanup frequency:** Runs every 1 minute; adjust if needed for higher throughput

### Metrics
- **Cardinality:** Be cautious with high-cardinality labels (token IDs)
- **Retention:** Configure Prometheus retention policies
- **Alerting:** Set up alerts for queue depth, error rates, and latency

## Troubleshooting

### Rate Limiting Issues
```bash
# Check Redis connectivity
redis-cli -u redis://localhost:6379 PING

# View rate limit keys
redis-cli -u redis://localhost:6379 KEYS "ratelimit:*"

# Check specific rate limit
redis-cli -u redis://localhost:6379 ZCARD "ratelimit:ip:192.168.1.100"
```

### Ack Channel Issues
```bash
# Check pending acks
curl http://localhost:8088/metrics | grep acks_pending

# Check ack completion rate
curl http://localhost:8088/metrics | grep acks_completed
```

### Performance Issues
```bash
# Check queue depth
curl http://localhost:8088/metrics | grep queue_depth

# Check normalization latency
curl http://localhost:8088/metrics | grep normalization_duration

# Check storage latency
curl http://localhost:8088/metrics | grep storage_duration
```
