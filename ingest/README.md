# Ingest Service

Splunk HTTP Event Collector (HEC) compatible ingestion service for TelHawk Stack.

## Features

- **Splunk HEC compatibility** - Drop-in replacement for Splunk HEC
- **Multiple formats** - JSON events, raw data, NDJSON batches
- **Token authentication** - HEC token validation via auth service
- **High throughput** - Buffered queue with backpressure
- **Nonrepudiation** - Event signatures for tamper detection
- **Standards compliant** - Follows Splunk HEC API specification

## API Endpoints

### Event Endpoint
```bash
POST /services/collector/event
Authorization: Splunk <hec-token>
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
Authorization: Splunk <hec-token>
Content-Type: text/plain

Apr 30 14:39:21 firewall %ASA-6-302013: Built inbound TCP connection
```

### Batch Ingestion (NDJSON)
```bash
POST /services/collector/event
Authorization: Splunk <hec-token>
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
  -H "Authorization: Splunk abc123..." \
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
- `CORE_SERVICE_URL` - Core service URL for OCSF normalization
- `STORAGE_SERVICE_URL` - Storage service URL for indexing

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
  -H "Authorization: Splunk test-token" \
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
    -H "Authorization: Splunk test-token" \
    -H "Content-Type: text/plain" \
    --data-binary @-
```

### Batch ingestion
```bash
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Splunk test-token" \
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
3. **Core** → OCSF normalization
4. **Storage** → OpenSearch indexing
5. **Query** → SPL searching

## Next Steps

- [ ] Implement auth service token validation
- [ ] Add core service integration for OCSF
- [ ] Add storage service integration
- [ ] Implement ack mechanism
- [ ] Add Prometheus metrics
- [ ] Add rate limiting
- [ ] Add compression support (gzip)
