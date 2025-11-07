# HEC Token Validation Testing Guide

## Overview

This guide demonstrates the complete HEC token validation flow:
1. Create a user account
2. Generate HEC token
3. Validate token via Auth service
4. Use token to ingest events
5. Verify authentication works end-to-end

## Prerequisites

```bash
# Start all services
docker-compose up -d

# Verify all services are healthy
docker-compose ps
```

## Test Procedure

### Step 1: Create User Account

```bash
# Create admin user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "TestPassword123!",
    "roles": ["admin"]
  }'
```

Expected response:
```json
{
  "id": "user-uuid",
  "username": "testuser",
  "email": "test@example.com",
  "roles": ["admin"],
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Step 2: Login and Get Access Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "TestPassword123!"
  }'
```

Save the `access_token` from the response.

### Step 3: Create HEC Token

Using the CLI tool:
```bash
docker-compose run --rm thawk auth login -u testuser -p TestPassword123!
docker-compose run --rm thawk token create --name test-ingest-token
```

Or using curl with access token:
```bash
# Note: You'll need to add a token creation endpoint or use the CLI
```

**Save the HEC token** - it will look like: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`

### Step 4: Validate HEC Token Directly

Test the auth service validation endpoint:

```bash
curl -X POST http://localhost:8080/api/v1/auth/validate-hec \
  -H "Content-Type: application/json" \
  -d '{
    "token": "YOUR-HEC-TOKEN-HERE"
  }'
```

Expected valid response:
```json
{
  "valid": true,
  "token_id": "token-uuid",
  "token_name": "test-ingest-token",
  "user_id": "user-uuid"
}
```

Expected invalid response:
```json
{
  "valid": false
}
```

### Step 5: Test Ingestion with Valid Token

```bash
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Telhawk YOUR-HEC-TOKEN-HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "event": {
      "message": "Test event with validated HEC token",
      "severity": "info"
    },
    "source": "test_app",
    "sourcetype": "json"
  }'
```

Expected response:
```json
{
  "text": "Success",
  "code": 0
}
```

Check ingest logs for validation:
```bash
docker-compose logs ingest | grep -A2 "HEC token validated"
```

### Step 6: Test Ingestion with Invalid Token

```bash
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Telhawk invalid-token-12345" \
  -H "Content-Type: application/json" \
  -d '{
    "event": {"message": "This should fail"}
  }'
```

Expected response (401 Unauthorized):
```json
{
  "text": "Unauthorized",
  "code": 3
}
```

Check ingest logs:
```bash
docker-compose logs ingest | grep "HEC token validation failed"
```

### Step 7: Test Ingestion without Token

```bash
curl -X POST http://localhost:8088/services/collector/event \
  -H "Content-Type: application/json" \
  -d '{
    "event": {"message": "No token provided"}
  }'
```

Expected response (401 Unauthorized):
```json
{
  "text": "Unauthorized",
  "code": 3
}
```

## Token Caching

The ingest service caches token validation results for performance:
- **Cache TTL:** Configurable via `INGEST_AUTH_TOKEN_VALIDATION_CACHE_TTL` (default: 5 minutes)
- **First validation:** Calls auth service (slower ~10-50ms)
- **Cached validations:** Instant (<1ms)
- **Cache invalidation:** Automatic after TTL expires

Test caching:
```bash
# First request - hits auth service
time curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Telhawk YOUR-TOKEN" \
  -d '{"event": {"test": 1}}'

# Subsequent requests - served from cache (faster)
time curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Telhawk YOUR-TOKEN" \
  -d '{"event": {"test": 2}}'
```

## Monitoring Token Validation

### Check Auth Service Logs

```bash
# View token validation audit logs
docker-compose logs auth | grep "hec_token_validate"
```

### Check Ingest Service Logs

```bash
# View token validation in ingest
docker-compose logs ingest | grep "HEC token"
```

### Check Validation Stats

```bash
# Get ingest service stats
curl http://localhost:8088/readyz | jq
```

## Troubleshooting

### Issue: Token validation always fails

**Check:**
1. Auth service is running: `curl http://localhost:8080/healthz`
2. Auth URL is correct in ingest config
3. Token exists in auth database
4. Token is enabled and not expired

**Debug:**
```bash
# Check ingest can reach auth
docker exec telhawk-ingest wget -O- http://auth:8080/healthz

# Validate token directly
curl -X POST http://localhost:8080/api/v1/auth/validate-hec \
  -d '{"token": "YOUR-TOKEN"}'
```

### Issue: Slow ingestion performance

**Possible causes:**
1. Cache TTL too short - increase `token_validation_cache_ttl`
2. Auth service slow to respond
3. Network latency between containers

**Solution:**
```bash
# Increase cache TTL (in docker-compose.yml)
INGEST_AUTH_TOKEN_VALIDATION_CACHE_TTL=10m
```

### Issue: "auth client not configured"

**Check:**
- `INGEST_AUTH_URL` is set in environment
- Auth service is accessible from ingest container

```bash
docker-compose logs ingest | grep "Auth URL"
```

## Security Considerations

1. **Always use HTTPS** in production for auth communication
2. **Rotate HEC tokens** regularly
3. **Monitor failed validation attempts** for potential attacks
4. **Set appropriate cache TTL** to balance performance and security
5. **Use TLS/mTLS** between services in production

## Performance Benchmarks

Expected validation performance:
- **First validation (uncached):** 10-50ms
- **Cached validation:** <1ms
- **Cache memory:** ~100 bytes per token
- **Recommended cache TTL:** 5-10 minutes

Test with load:
```bash
# Send 100 events with same token (tests caching)
for i in {1..100}; do
  curl -s -X POST http://localhost:8088/services/collector/event \
    -H "Authorization: Telhawk YOUR-TOKEN" \
    -d '{"event": {"test": '$i'}}'
done

# Check stats
curl http://localhost:8088/readyz | jq '.stats'
```

## Next Steps

Once token validation is working:
1. Enable token rotation policies
2. Set up monitoring for failed validations
3. Configure rate limiting per token
4. Implement token usage tracking
5. Add audit logging for compliance
