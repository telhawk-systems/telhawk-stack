# Dead-Letter Queue and Backpressure Implementation

## Overview

The TelHawk stack now includes production-ready error handling:

- **Dead-Letter Queue (DLQ)** - Captures failed normalization events for analysis and replay
- **Backpressure & Retries** - Handles transient failures with exponential backoff

## Dead-Letter Queue (DLQ)

### Purpose

The DLQ captures events that fail normalization or storage, enabling:
- **Debugging** - Analyze why events failed to process
- **Replay** - Re-process events after fixing normalizers/configuration
- **Auditing** - Ensure no data loss during processing
- **Monitoring** - Track failure rates and patterns

### Architecture

Failed events are written to disk as JSON files with full context:
- Original raw event envelope
- Error message and reason
- Timestamp and attempt count
- Complete metadata for replay

### Configuration

```yaml
# ingest/config.yaml
dlq:
  enabled: true
  base_path: /var/lib/telhawk/dlq
```

Environment variables:
- `INGEST_DLQ_ENABLED=true|false`
- `INGEST_DLQ_BASE_PATH=/custom/path`

### File Structure

```
/var/lib/telhawk/dlq/
├── failed_1699040123_0.json
├── failed_1699040124_1.json
└── failed_1699040125_2.json
```

Each file contains:
```json
{
  "timestamp": "2024-11-04T01:02:03Z",
  "envelope": {
    "id": "evt-12345",
    "source": "firewall",
    "source_type": "cisco_asa",
    "format": "json",
    "payload": "...",
    "attributes": {
      "host": "fw01.example.com"
    },
    "received_at": "2024-11-04T01:02:03Z"
  },
  "error": "no normalizer registered for format=json source_type=cisco_asa",
  "reason": "normalization_failed",
  "attempts": 1,
  "last_attempt": "2024-11-04T01:02:03Z"
}
```

### Failure Reasons

- `normalization_failed` - No normalizer found or normalization error
- `storage_failed` - Failed to persist to OpenSearch
- `validation_failed` - Event failed OCSF validation

### API Endpoints

#### List Failed Events

```bash
GET /api/v1/dlq
```

Response:
```json
{
  "events": [
    {
      "timestamp": "2024-11-04T01:02:03Z",
      "envelope": {...},
      "error": "...",
      "reason": "normalization_failed",
      "attempts": 1,
      "last_attempt": "2024-11-04T01:02:03Z"
    }
  ],
  "count": 1
}
```

Example:
```bash
curl http://localhost:8088/api/v1/dlq
```

#### Purge All Failed Events

```bash
DELETE /api/v1/dlq/purge
```

Response:
```json
{
  "status": "purged"
}
```

Example:
```bash
curl -X DELETE http://localhost:8088/api/v1/dlq/purge
```

#### Health Check (includes DLQ stats)

```bash
GET /healthz
```

Response:
```json
{
  "uptime_seconds": 3600,
  "processed": 125000,
  "failed": 23,
  "stored": 124977,
  "dlq_written": 23,
  "dlq_stats": {
    "enabled": true,
    "written": 23,
    "pending_files": 23,
    "base_path": "/var/lib/telhawk/dlq"
  }
}
```

### Monitoring

Track DLQ metrics via health endpoint:
- `dlq_written` - Total events written to DLQ
- `pending_files` - Current number of files in queue
- `failed` - Total failed events (includes DLQ)

Alert on:
- DLQ write rate increasing
- Pending files accumulating
- Specific failure patterns

### Replay Workflow

1. **Identify Failed Events**
   ```bash
   curl http://localhost:8088/api/v1/dlq | jq '.events[] | {timestamp, reason, error}'
   ```

2. **Analyze Failures**
   ```bash
   # Download and examine
   curl http://localhost:8088/api/v1/dlq > failed_events.json
   jq '.events[] | select(.reason == "normalization_failed")' failed_events.json
   ```

3. **Fix Issue** (e.g., add missing normalizer, fix config)

4. **Replay Events**
   ```bash
   # Extract and replay via HEC
   jq -r '.events[].envelope' failed_events.json | while read envelope; do
     curl -X POST http://localhost:8088/services/collector/event \
       -H "Authorization: Telhawk $TOKEN" \
       -d "$envelope"
   done
   ```

5. **Purge Successfully Replayed**
   ```bash
   curl -X DELETE http://localhost:8088/api/v1/dlq/purge
   ```

### Best Practices

- **Monitor regularly** - Check DLQ stats in health endpoint
- **Set alerts** - Alert when pending_files > 100
- **Investigate patterns** - Group by reason/error to find systematic issues
- **Purge after replay** - Clean up after successful reprocessing
- **Backup before purge** - Save failed events for records
- **Size limits** - Monitor disk usage of DLQ directory

### Docker Configuration

Mount persistent volume for DLQ:
```yaml
# docker-compose.yml
services:
  core:
    volumes:
      - dlq-data:/var/lib/telhawk/dlq
    environment:
      - CORE_DLQ_ENABLED=true
      - CORE_DLQ_BASE_PATH=/var/lib/telhawk/dlq

volumes:
  dlq-data:
    driver: local
```

## Backpressure & Retries

### Purpose

Handle transient failures gracefully:
- Network glitches
- Temporary service unavailability
- Rate limiting
- Server overload (5xx errors)

### Implementation

The ingest service retries failed requests to the ingest service with exponential backoff after normalizing events to OCSF format.

### Retry Strategy

```
Attempt 1: Immediate
Attempt 2: Wait 100ms
Attempt 3: Wait 200ms
Attempt 4: Wait 400ms
```

Total max retry time: ~700ms

### Retryable Errors

- **5xx Server Errors** - ingest service temporarily unavailable
- **429 Rate Limit** - Too many requests, back off
- **Network Errors** - Connection refused, timeout, DNS failure

### Non-Retryable Errors

- **4xx Client Errors** (except 429) - Bad request, invalid payload
  - 400 Bad Request
  - 401 Unauthorized
  - 403 Forbidden
  - 404 Not Found

These errors indicate a problem with the request itself, not transient failure.

### Configuration

Built-in defaults:
- Max retries: 3
- Initial delay: 100ms
- Backoff multiplier: 2x

### Behavior Examples

#### Success on Retry

```
Request 1: 503 Service Unavailable → Wait 100ms
Request 2: 503 Service Unavailable → Wait 200ms
Request 3: 200 OK → Success
```

#### Max Retries Exceeded

```
Request 1: 503 Service Unavailable → Wait 100ms
Request 2: 503 Service Unavailable → Wait 200ms
Request 3: 503 Service Unavailable → Wait 400ms
Request 4: 503 Service Unavailable → Fail
Error: max retries exceeded
```

Event written to DLQ (if enabled)

#### Non-Retryable Error

```
Request 1: 400 Bad Request → Fail immediately
No retries attempted
```

Event written to DLQ (if enabled)

### Flow Diagram

```
[Ingest] → [ingest service]
              ↓
          Success? → [Storage]
              ↓ No
          Retryable?
              ↓ Yes
          Exponential Backoff
              ↓
          Retry (max 3)
              ↓
          Still failing?
              ↓ Yes
          Write to DLQ
```

### Monitoring

Track retry metrics:
- **Success rate** - Percentage of first-attempt successes
- **Retry rate** - How often retries are needed
- **DLQ write rate** - Events that exceed max retries

Recommended alerts:
- Retry rate > 10% (indicates systemic issues)
- DLQ write rate increasing
- ingest service 5xx error rate high

### Load Shedding

Backpressure prevents cascade failures:

1. **Ingest Service**
   - Receives high event rate
   - ingest service slow/unavailable
   - Retries with backoff
   - Slows down request rate automatically

2. **ingest service**
   - Under load, returns 5xx or 429
   - Ingest backs off
   - Gives core time to recover
   - Prevents overload spiral

3. **Recovery**
   - ingest service recovers
   - Retries succeed
   - Normal throughput resumes

### Testing Retry Logic

#### Simulate Transient Failure

```bash
# Stop ingest service
docker-compose stop storage

# Send event via ingest
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Telhawk $TOKEN" \
  -d '{"event": "test"}'

# Restart ingest service (during retry window)
docker-compose start storage

# Check if event succeeded
```

#### Simulate Rate Limiting

```bash
# Send burst of events
for i in {1..1000}; do
  curl -X POST http://localhost:8088/services/collector/event \
    -H "Authorization: Telhawk $TOKEN" \
    -d "{\"event\": \"test-$i\"}" &
done

# Monitor retry behavior
docker-compose logs -f ingest | grep "retry"
```

### Production Configuration

```yaml
# ingest/config.yaml
core:
  url: http://ingest:8088
  timeout_seconds: 10
  max_retries: 3
  retry_delay_ms: 100
```

Environment overrides:
- `INGEST_OPENSEARCH_URL=http://ingest:8088`
- `INGEST_CORE_TIMEOUT_SECONDS=10`

## Combined Workflow

DLQ and retries work together:

1. **Event arrives** at ingest service
2. **Token validated** against auth service
3. **Sent to core** for normalization
4. **Retry if needed** (up to 3 times with backoff)
5. **Success** → stored in OpenSearch
6. **Failure after retries** → written to DLQ
7. **Operator notified** via monitoring/alerts
8. **Analyze failure** via DLQ API
9. **Fix issue** (add normalizer, fix config)
10. **Replay events** from DLQ
11. **Purge DLQ** after successful replay

## Comparison with Alternatives

### vs. Message Queue (Kafka, RabbitMQ)

**DLQ Advantages:**
- Simple file-based storage
- No additional infrastructure
- Easy inspection (JSON files)
- Built-in to ingest service

**Message Queue Advantages:**
- Higher throughput
- Better for multi-consumer scenarios
- Built-in retry mechanisms
- Distributed processing

**TelHawk Choice:** File-based DLQ is simpler for SOC use cases with moderate volume.

### vs. No Error Handling

**Without DLQ/Retries:**
- Data loss on failures
- No visibility into errors
- Manual intervention difficult
- Poor production reliability

**With DLQ/Retries:**
- Zero data loss
- Full error visibility
- Easy replay workflow
- Production-ready

## Performance Impact

### DLQ Overhead

- **Success path**: Zero overhead (no DLQ writes)
- **Failure path**: ~1-5ms to write JSON file
- **Disk usage**: ~1-10KB per failed event
- **Memory**: Minimal (streaming writes)

### Retry Overhead

- **Success path**: Zero overhead (no retries)
- **Failure path**: 100-700ms of backoff delays
- **CPU**: Minimal (exponential backoff is cheap)
- **Memory**: Minimal (reuses request buffer)

### Recommendations

- Monitor DLQ disk usage
- Rotate/archive old DLQ files
- Set alerts for high failure rates
- Tune retry delays based on load

## Security Considerations

### DLQ Access Control

DLQ files may contain sensitive data:
- Restrict filesystem permissions (0644)
- Secure DLQ API endpoints
- Audit DLQ access
- Encrypt at rest if needed

### Replay Authorization

Ensure proper auth for replay:
- Use valid HEC tokens
- Verify source authorization
- Log replay events
- Rate limit replay requests

## Summary

**Dead-Letter Queue:**
- ✅ Captures all normalization failures
- ✅ Enables debugging and replay
- ✅ Prevents data loss
- ✅ Simple file-based storage

**Backpressure & Retries:**
- ✅ Handles transient failures automatically
- ✅ Exponential backoff prevents overload
- ✅ Retries only for retryable errors
- ✅ Integrates with DLQ for persistent failures

Together, they provide production-grade reliability for the TelHawk ingestion pipeline.
