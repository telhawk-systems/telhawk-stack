# Auth Event Forwarding to Ingest Pipeline

## Overview

The auth service now selectively forwards security-relevant authentication events to the ingest pipeline for centralized analytics, correlation, and search in OpenSearch. This creates a complete audit trail visible in the main TelHawk security platform.

## Architecture

```
┌──────────────┐
│ User Login   │
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────────┐
│         Auth Service                      │
│  ┌────────────────────────────────────┐  │
│  │  Audit Logger                      │  │
│  │  1. Log to PostgreSQL (always)     │  │
│  │  2. Forward to Ingest (selective)  │  │
│  └────────────────────────────────────┘  │
└──────┬───────────────────┬────────────────┘
       │                   │
       │                   │ (only security events)
       ▼                   ▼
┌──────────────┐    ┌──────────────┐
│ PostgreSQL   │    │ Ingest       │
│ audit_log    │    │ Service      │
│ (all events) │    │ (HEC)        │
└──────────────┘    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ Core         │
                    │ (Normalize)  │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ Storage      │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ OpenSearch   │
                    │ (Searchable) │
                    └──────────────┘
```

## Event Selection Strategy

### ✅ Events Forwarded to Ingest (Security-Relevant)

These events are valuable for security analysis and correlation:

- **login** - User login attempts (success/failure)
- **logout** - User logout
- **register** - New user registration
- **token_revoke** - Session revocation
- **hec_token_create** - Admin creates API token
- **hec_token_revoke** - Admin revokes API token
- **user_update** - User account modifications
- **user_delete** - User account deletion
- **password_change** - Password changes

### ❌ Events NOT Forwarded (Operational Noise)

These events are too frequent or would cause infinite loops:

- **token_validate** - Every API call (too noisy)
- **token_refresh** - Every 15 minutes per user (too noisy)
- **hec_token_validate** - Every ingested event (would cause infinite loop!)

### The Infinite Loop Problem

If we forwarded HEC token validation events:

```
1. User logs in → Auth forwards event to ingest
2. Ingest validates HEC token with auth
3. Auth logs "hec_token_validate" → forwards to ingest
4. Ingest validates HEC token with auth
5. Auth logs "hec_token_validate" → forwards to ingest
6. [INFINITE LOOP] 🔥
```

**Solution:** Never forward events that trigger during the forwarding process itself.

## OCSF Mapping

Auth events are normalized to **OCSF Authentication class (3002)**:

```json
{
  "class_uid": 3002,
  "category_uid": 3,
  "activity_id": 1,
  "type_uid": 300201,
  "time": 1730764800000,
  "severity": "Informational",
  "severity_id": 1,
  "status": "Success",
  "status_id": 1,
  "user": {
    "name": "admin",
    "uid": "00000000-0000-0000-0000-000000000001",
    "type": "User",
    "type_id": 1
  },
  "src_endpoint": {
    "ip": "172.29.0.1"
  },
  "metadata": {
    "version": "1.1.0",
    "product": {
      "name": "TelHawk Auth",
      "vendor_name": "TelHawk Systems",
      "version": "1.0.0"
    },
    "log_name": "auth_audit"
  },
  "message": "login success by admin",
  "observables": [
    {
      "name": "source_ip",
      "type": "IP Address",
      "type_id": 2,
      "value": "172.29.0.1"
    }
  ]
}
```

### OCSF Activity IDs

| Auth Action        | OCSF Activity ID | Description           |
|--------------------|------------------|-----------------------|
| login              | 1                | Logon                 |
| logout             | 2                | Logoff                |
| register           | 1                | Logon (user creation) |
| token_revoke       | 2                | Logoff                |
| hec_token_create   | 3                | Authentication Ticket |
| hec_token_revoke   | 99               | Other                 |
| user_update        | 99               | Other                 |
| password_change    | 99               | Other                 |

## Configuration

### Enable Event Forwarding

Set these environment variables in docker compose or production:

```bash
AUTH_INGEST_ENABLED=true
AUTH_INGEST_URL=http://ingest:8082
AUTH_INGEST_HEC_TOKEN=<your-hec-token>
```

### Disabled by Default

Event forwarding is **disabled by default** to prevent issues during initial setup:
- No HEC token required for basic auth functionality
- Can test auth service independently
- Enable when ready to integrate with full stack

### Creating the HEC Token

1. Login as admin user
2. Create HEC token via API (once endpoint is implemented):
   ```bash
   curl -X POST http://localhost:8080/api/v1/auth/hec-tokens \
     -H "Authorization: Bearer <admin-jwt>" \
     -H "Content-Type: application/json" \
     -d '{"name": "auth-service-events"}'
   ```
3. Use returned token in `AUTH_INGEST_HEC_TOKEN`
4. Set `AUTH_INGEST_ENABLED=true`
5. Restart auth service

## Dual Storage

All auth events are **always** stored in both locations:

### 1. PostgreSQL `audit_log` table
- **Purpose**: Auth-specific queries, compliance, debugging
- **Retention**: Configurable (no automatic deletion)
- **Access**: Direct SQL queries or auth API (future)
- **Contains**: ALL events including token validations

### 2. OpenSearch (via Ingest)
- **Purpose**: Security analytics, correlation, dashboards
- **Retention**: ILM policies (e.g., 90 days)
- **Access**: TelHawk query API, search UI
- **Contains**: Only security-relevant events

## Benefits

### 1. Complete Visibility
Security analysts can search for auth events alongside other telemetry:
```
sourcetype=telhawk:auth:audit AND user.name=admin AND status=Failure
```

### 2. Correlation
Link authentication failures with network events, file access, etc.

### 3. Alerting
Create alerts on suspicious auth patterns:
- Multiple failed logins from same IP
- Login from unusual location
- Admin token created outside business hours

### 4. Compliance
Meet audit requirements with immutable PostgreSQL trail + searchable OpenSearch index

### 5. Separation of Concerns
- Auth service remains self-contained
- Ingest forwarding is optional (disabled by default)
- No circular dependencies
- Auth still works if ingest is down (events buffered in PostgreSQL)

## Performance Considerations

### Async Forwarding
Events are forwarded asynchronously in goroutines:
```go
go func() {
    _ = l.ingestClient.ForwardEvent(...)
}()
```

This ensures:
- Auth operations never blocked by ingest latency
- Failed forwards don't break authentication
- High throughput maintained

### Event Volume

With 1000 active users:
- **With selective forwarding**: ~2-5 events/user/day = 2,000-5,000 events/day
- **Without selective forwarding**: ~100-200 events/user/day = 100,000-200,000 events/day

Selective forwarding reduces volume by **98%** while keeping security-relevant data.

## Monitoring

Check if events are flowing:

```bash
# Check PostgreSQL (all events)
docker exec telhawk-auth-db psql -U telhawk -d telhawk_auth \
  -c "SELECT COUNT(*), action FROM audit_log GROUP BY action ORDER BY COUNT DESC;"

# Check OpenSearch (forwarded events only)
curl -X GET "http://localhost:9200/telhawk-*/_search?q=metadata.product.name:TelHawk+Auth&size=0"
```

## Troubleshooting

### Events not appearing in OpenSearch

1. Check if forwarding is enabled:
   ```bash
   docker exec telhawk-auth env | grep AUTH_INGEST
   ```

2. Check auth service logs:
   ```bash
   docker logs telhawk-auth | grep -i ingest
   ```

3. Verify HEC token is valid:
   ```bash
   curl -X POST http://localhost:8082/services/collector \
     -H "Authorization: Telhawk <token>" \
     -d '{"event":"test"}'
   ```

4. Check ingest service logs:
   ```bash
   docker logs telhawk-ingest | grep -i auth
   ```

### All events showing in OpenSearch

If you see token_validate or token_refresh events:
- Bug in `ShouldForwardToIngest()` filter
- Check `auth/internal/models/audit.go`

## Future Enhancements

1. **Configurable event selection** - Allow admins to choose which events to forward
2. **Buffering on ingest failure** - Queue events if ingest is temporarily down
3. **Rate limiting** - Prevent flood if something goes wrong
4. **Event enrichment** - Add geolocation, threat intel to source IPs
5. **Sampling** - For extremely high-volume deployments

## Summary

Auth event forwarding provides complete security visibility while maintaining:
- ✅ No infinite loops (smart filtering)
- ✅ No performance impact (async forwarding)
- ✅ No operational noise (selective events only)
- ✅ No tight coupling (optional feature, disabled by default)
- ✅ Complete audit trail (PostgreSQL always has all events)
