# Ingest Service

Splunk HTTP Event Collector (HEC) compatible ingestion service with built-in OCSF normalization for TelHawk Stack.

## Features

- **Splunk HEC compatibility** - Drop-in replacement for Splunk HEC
- **OCSF normalization** - 77 auto-generated normalizers transform raw events to OCSF format
- **Multiple formats** - JSON events, raw data, NDJSON batches
- **Token authentication** - HEC token validation via auth service
- **Validation chain** - Ensures OCSF compliance before storage
- **Dead Letter Queue** - Failed events stored at `/var/lib/telhawk/dlq`
- **High throughput** - Buffered queue with backpressure
- **Nonrepudiation** - Event signatures for tamper detection
- **Standards compliant** - Follows Splunk HEC API specification

## API Endpoints

### Event Endpoint
```bash
POST /services/collector/event
Authorization: Telhawk <hec-token>
Content-Type: application/json

{
  "time": 1617187200.0,
  "host": "web-server-01",
  "source": "apache",
  "sourcetype": "access_log",
  "index": "web",
  "event": {
    "message": "User login successful",
    "user": "analyst1",
    "ip": "192.168.1.100"
  }
}
```

### Raw Endpoint
```bash
POST /services/collector/raw?source=firewall&sourcetype=cisco_asa
Authorization: Telhawk <hec-token>
Content-Type: text/plain

Apr 30 14:39:21 firewall %ASA-6-302013: Built inbound TCP connection
```

### Batch Ingestion (NDJSON)
```bash
POST /services/collector/event
Authorization: Telhawk <hec-token>
Content-Type: application/json

{"event": {"message": "Event 1"}}
{"event": {"message": "Event 2"}}
{"event": {"message": "Event 3"}}
```

### Health Check
```bash
GET /services/collector/health
GET /healthz
```

### Readiness Check
```bash
GET /readyz

Response:
{
  "status": "ready",
  "stats": {
    "total_events": 12345,
    "total_bytes": 987654,
    "successful_events": 12300,
    "failed_events": 45,
    "last_event": "2024-11-01T12:34:56Z"
  }
}
```

## HEC Token Authentication

Tokens are validated against the auth service:

```bash
# Create HEC token
thawk token create --name production-ingester

# Use token in requests
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Telhawk abc123..." \
  -d '{"event": {"message": "Test"}}'
```

## Event Format

### Standard HEC Event
```json
{
  "time": 1617187200.0,          // Unix timestamp (optional)
  "host": "web-01",               // Hostname (optional)
  "source": "app_logs",           // Source identifier (optional)
  "sourcetype": "json",           // Source type (optional)
  "index": "main",                // Target index (optional, default: main)
  "event": {                      // Event data (required)
    "severity": "high",
    "message": "Security alert"
  },
  "fields": {                     // Indexed fields (optional)
    "env": "production",
    "datacenter": "us-east-1"
  }
}
```

### Internal Event Representation
After ingestion, events are converted to:
```json
{
  "id": "uuid",
  "timestamp": "2024-11-01T12:34:56.789Z",
  "host": "web-01",
  "source": "app_logs",
  "sourcetype": "json",
  "source_ip": "192.168.1.100",
  "index": "main",
  "event": {...},
  "fields": {...},
  "raw": "...",
  "hec_token_id": "token-uuid",
  "signature": "hmac-signature"
}
```

## Error Codes

Matches Splunk HEC error codes:

- `0` - Success
- `4` - Invalid authorization (401)
- `5` - No data (400)
- `6` - Invalid data format (400)
- `9` - Server is busy (503)

## Configuration

Environment variables:
- `INGEST_PORT` - Server port (default: 8088)
- `INGEST_QUEUE_SIZE` - Event queue size (default: 10000)
- `AUTH_SERVICE_URL` - Auth service URL for token validation
- `STORAGE_SERVICE_URL` - Storage service URL for indexing
- `INGEST_DLQ_PATH` - Dead letter queue path (default: /var/lib/telhawk/dlq)

## Building

```bash
cd ingest
go build -o ../bin/ingest ./cmd/ingest
```

## Running

```bash
./bin/ingest
# Listens on :8088
```

## Testing

### Send test event
```bash
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Telhawk test-token" \
  -d '{
    "event": {
      "message": "Test security event",
      "severity": "high"
    },
    "source": "test",
    "sourcetype": "json"
  }'
```

### Send raw data
```bash
echo "Security alert detected at 12:34:56" | \
  curl -X POST http://localhost:8088/services/collector/raw \
    -H "Authorization: Telhawk test-token" \
    -H "Content-Type: text/plain" \
    --data-binary @-
```

### Batch ingestion
```bash
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Telhawk test-token" \
  -d '{"event": {"msg": "Event 1"}}
{"event": {"msg": "Event 2"}}
{"event": {"msg": "Event 3"}}'
```

## Splunk Universal Forwarder Compatibility

Configure Splunk Universal Forwarder to send to TelHawk:

```ini
[httpout:telhawk]
httpEventCollectorToken = <your-hec-token>
uri = http://localhost:8088/services/collector/event
```

## Performance

- **Queue size**: 10,000 events (configurable)
- **Backpressure**: Returns 503 when queue full
- **Throughput**: Designed for 10k+ events/sec
- **Batch support**: NDJSON for efficient bulk ingestion

## Nonrepudiation

Each event receives:
- Unique ID (UUID)
- Ingestion timestamp
- Source IP capture
- HMAC signature
- HEC token ID tracking

See [Nonrepudiation Strategy](../docs/nonrepudiation.md) for details.

## Integration with TelHawk Stack

1. **Ingestion** → Events received via HEC
2. **Auth** → Token validation
3. **Normalization** → Transform raw events to OCSF format (77 normalizers)
4. **Validation** → Ensure OCSF compliance
5. **Storage** → Forward normalized events to storage service
6. **OpenSearch** → Storage service indexes to OpenSearch
7. **Query** → SPL searching via query service

## OCSF Normalization

The ingest service includes built-in OCSF normalization:

- **77 auto-generated normalizers** - One per OCSF event class
- **Registry pattern** - Matches raw events to normalizers based on source_type
- **Field mapping** - Common field variants → OCSF standard fields
- **Event classification** - Automatic category_uid, class_uid, activity_id assignment
- **Validation chain** - Ensures events meet OCSF schema requirements
- **Dead Letter Queue** - Failed normalizations captured for analysis

### Normalizer Generation

Normalizers are generated from the OCSF schema:

```bash
# Regenerate normalizers when OCSF schema updates
cd tools/normalizer-generator
go run main.go
# Output: ingest/internal/normalizer/generated/*.go
```

See `docs/NORMALIZER_GENERATION.md` for details.

## Key Files

- `ingest/internal/pipeline/pipeline.go` - Orchestrates normalization and validation
- `ingest/internal/normalizer/normalizer.go` - Normalizer interface and registry
- `ingest/internal/normalizer/generated/` - Auto-generated OCSF normalizers (77 files)
- `ingest/internal/dlq/dlq.go` - Dead Letter Queue implementation
- `ingest/internal/handlers/hec.go` - HEC endpoint implementation
- `common/ocsf/` - Shared OCSF event structures and types

## Related Documentation

- `docs/SERVICES.md` - Service architecture overview
- `docs/NORMALIZER_GENERATION.md` - Normalizer generation strategy
- `docs/NORMALIZATION_INTEGRATION.md` - Integration guide with examples
- `docs/OCSF_COVERAGE.md` - OCSF schema coverage details
- `docs/SPLUNK_COMPATIBILITY.md` - HEC compatibility details
