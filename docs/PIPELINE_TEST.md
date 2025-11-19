# TelHawk Stack Pipeline Integration Test

## Overview

This document describes how to test the complete event pipeline from Ingest → Core → Storage → OpenSearch.

## Pipeline Flow

```
Event Source
    ↓
Ingest Service (HEC endpoint)
    ↓
Core Service (OCSF normalization)
    ↓
Storage Service (OpenSearch indexing)
    ↓
OpenSearch (persistent storage)
```

## Test Procedure

### Prerequisites

Ensure all services are running:

```bash
docker-compose up -d
docker-compose ps
```

All services should show "healthy" status.

### Step 1: Create HEC Token

```bash
# Login first
./scripts/thawk auth login -u admin -p SecurePassword123

# Create token
./scripts/thawk token create --name test-token

# Save the token output for next step
```

### Step 2: Send Test Event

```bash
# Replace <TOKEN> with actual token from Step 1
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Telhawk <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "event": {
      "message": "Pipeline test event",
      "severity": "info",
      "user": "testuser",
      "action": "login"
    },
    "source": "test_application",
    "sourcetype": "json",
    "host": "test-host"
  }'
```

Expected response:
```json
{
  "text": "Success",
  "code": 0
}
```

### Step 3: Verify Event in Ingest Logs

```bash
docker-compose logs --tail=20 ingest
```

Look for:
- `Processing event: id=<uuid> source=test_application`
- `event <uuid> normalized via core service`
- `event <uuid> successfully stored`

### Step 4: Verify Event in Core Logs

```bash
docker-compose logs --tail=20 core
```

Look for normalization activity.

### Step 5: Verify Event in Storage Logs

```bash
docker-compose logs --tail=20 storage
```

Look for indexing operations.

### Step 6: Query OpenSearch Directly

```bash
# Wait a few seconds for indexing
sleep 5

# Search for the event
curl -X GET "http://localhost:9200/telhawk-events-*/_search?pretty" \
  -u admin:TelHawk123! \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "match": {
        "message": "Pipeline test event"
      }
    }
  }'
```

Expected: Should return 1+ hits with your test event.

### Step 7: Send Multiple Events (Batch Test)

```bash
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Telhawk <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"event": {"message": "Event 1"}, "source": "batch_test"}
{"event": {"message": "Event 2"}, "source": "batch_test"}
{"event": {"message": "Event 3"}, "source": "batch_test"}'
```

### Step 8: Verify Event Count

```bash
curl -X GET "http://localhost:9200/telhawk-events-*/_count" \
  -u admin:TelHawk123! \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "match": {
        "source": "batch_test"
      }
    }
  }'
```

Expected: Count should show 3 events.

## Troubleshooting

### Events Not Reaching OpenSearch

1. Check Ingest service logs:
   ```bash
   docker-compose logs ingest | grep ERROR
   ```

2. Check Core service logs:
   ```bash
   docker-compose logs core | grep ERROR
   ```

3. Check Storage service logs:
   ```bash
   docker-compose logs storage | grep ERROR
   ```

4. Verify OpenSearch is healthy:
   ```bash
   curl -u admin:TelHawk123! http://localhost:9200/_cluster/health?pretty
   ```

### Common Issues

**Issue:** "storage client not configured"
- **Solution:** Verify `INGEST_STORAGE_URL` environment variable is set in docker-compose.yml

**Issue:** "core client not configured"
- **Solution:** Verify `INGEST_CORE_URL` environment variable is set in docker-compose.yml

**Issue:** OpenSearch indexing failures
- **Solution:** Check OpenSearch disk space and cluster health

**Issue:** Connection refused to services
- **Solution:** Ensure all services are healthy: `docker-compose ps`

## Performance Testing

### Load Test with Multiple Events

```bash
# Send 100 events rapidly
for i in {1..100}; do
  curl -s -X POST http://localhost:8088/services/collector/event \
    -H "Authorization: Telhawk <TOKEN>" \
    -H "Content-Type: application/json" \
    -d "{\"event\": {\"message\": \"Load test event $i\"}, \"source\": \"loadtest\"}"
done

# Check ingestion stats
curl http://localhost:8088/readyz | jq
```

### Check Pipeline Throughput

Monitor service logs in real-time:

```bash
# Terminal 1
docker-compose logs -f ingest

# Terminal 2
docker-compose logs -f core

# Terminal 3
docker-compose logs -f storage
```

## Success Criteria

✅ **Pipeline is working correctly when:**

1. Events sent to Ingest return HTTP 200 with `"text": "Success"`
2. Ingest logs show "event <id> normalized via core service"
3. Ingest logs show "event <id> successfully stored"
4. Storage logs show successful indexing operations
5. Events are queryable in OpenSearch within 5 seconds
6. All services maintain "healthy" status under load

## Next Steps

Once the pipeline is validated:

1. Configure retention policies in Storage service
2. Set up monitoring and alerting
3. Tune OpenSearch index settings for your workload
4. Configure event enrichment in Core service
5. Set up the Query service for programmatic access
