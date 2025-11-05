# TelHawk Security Architecture

## Core Security Principles

### 1. No Demo Credentials - EVER
**Policy:** TelHawk NEVER uses demo credentials from any component.

- OpenSearch demo credentials (admin:admin) are NEVER used
- All credentials must be configurable via environment variables
- Credentials can be rotated by changing env vars and restarting
- Default credentials are strong and documented

### 2. SSL/TLS Everywhere
**Policy:** All internal service communication uses TLS with proper certificates.

#### Certificate Generation Strategy
OpenSearch requires SSL certificates before it can start. We use a two-stage approach:

1. **cert-generator** - Sidecar container that runs ONCE before OpenSearch
   - Checks if certificates already exist in volume
   - If not, generates self-signed certificates with proper SANs
   - Stores certificates in shared volume
   - Exits after generation (no health check needed - it's a one-shot job)

2. **opensearch** - Main database container
   - Depends on cert-generator completion
   - Uses certificates from shared volume
   - Updates admin credentials from env vars on EVERY boot
   - Credentials are rotatable without regenerating certs

#### Certificate Priority
1. **Production certs** (mounted at `/certs/production/`) - Use if provided
2. **Generated self-signed certs** (in `/certs/generated/`) - Generated if no production certs
3. **NEVER demo certs** - Demo installer is never used

### 3. Credential Rotation
**Policy:** Credentials must be rotatable without data loss.

- Change `OPENSEARCH_ADMIN_USER` or `OPENSEARCH_PASSWORD` env vars
- Restart container
- Credentials are updated automatically
- No volume deletion required
- No data loss

### 4. Health Checks
**Policy:** All long-running services MUST have health checks.

**Require health checks:**
- opensearch
- auth
- auth-db
- storage
- query
- core
- ingest
- web

**No health check needed:**
- cert-generator (one-shot init container, exits after completion)
- CLI tools
- Job containers

Health checks must:
- Use credentials from environment variables (no hardcoded passwords)
- Have appropriate timeouts (90s+ for OpenSearch)
- Retry appropriately (10-20 retries for slow-starting services)

## OpenSearch Security Configuration

### Environment Variables

| Variable | Default | Purpose | Rotatable |
|----------|---------|---------|-----------|
| `OPENSEARCH_ADMIN_USER` | `admin` | Admin username | Yes |
| `OPENSEARCH_PASSWORD` | `TelHawk123!` | Admin password | Yes |

### Architecture

```
┌─────────────────────┐
│  cert-generator     │  (runs once, exits)
│  ├─ Check for certs │
│  ├─ Generate if none│
│  └─ Store in volume │
└──────────┬──────────┘
           │
           ▼
    [Certs Volume]
           │
           ▼
┌──────────┴──────────┐
│    opensearch       │  (long-running)
│  ├─ Load certs      │
│  ├─ Start service   │
│  └─ Update creds    │  <── Uses env vars, runs on EVERY boot
└─────────────────────┘
```

### Security Features

1. **No default admin:admin access** - Always replaced on first boot
2. **Self-signed certs with proper SANs** - Not demo certs
3. **Credential rotation** - Update env vars, restart, credentials updated
4. **Health check uses env vars** - No hardcoded passwords in health checks

## Production Deployment

### Providing Production Certificates

Mount your certificates to `/certs/production/`:

```yaml
opensearch:
  volumes:
    - ./certs/opensearch.pem:/certs/production/opensearch.pem:ro
    - ./certs/opensearch-key.pem:/certs/production/opensearch-key.pem:ro
    - ./certs/root-ca.pem:/certs/production/root-ca.pem:ro
```

### Rotating Credentials

```bash
# Update environment variable
export OPENSEARCH_PASSWORD="NewSecurePassword123!"

# Restart OpenSearch
docker-compose restart opensearch

# Credentials are automatically updated
```

### Security Checklist

- [ ] Change default `OPENSEARCH_PASSWORD` from `TelHawk123!`
- [ ] Consider changing `OPENSEARCH_ADMIN_USER` from `admin`
- [ ] Provide production certificates (or accept self-signed for dev)
- [ ] Enable firewall rules (only internal services access OpenSearch)
- [ ] Review audit logs in OpenSearch
- [ ] Set up credential rotation schedule
- [ ] Never use demo credentials
- [ ] All services have health checks (except one-shot init containers)

## Why This Approach?

### Why separate cert-generator container?
- OpenSearch requires certs BEFORE it starts
- Can't generate certs inside OpenSearch container (chicken-and-egg)
- Sidecar pattern allows cert generation without demo credentials
- Certs persist in volume, only generated once
- Clean separation of concerns

### Why update credentials on every boot?
- Allows credential rotation without volume deletion
- No data loss when rotating credentials
- Credentials are configuration, not data
- Simple operational model

### Why no demo credentials?
- Demo credentials are a security anti-pattern
- Makes penetration testing harder (no known defaults)
- Forces operators to think about security from day one
- TelHawk is a security product - we set the example

## Related Documentation

- [AUTH_INTEGRATION.md](./AUTH_INTEGRATION.md) - Auth service and JWT tokens
- [QUERY_SERVICE_READ_PATH.md](./QUERY_SERVICE_READ_PATH.md) - Query service security
- [DLQ_AND_BACKPRESSURE.md](./DLQ_AND_BACKPRESSURE.md) - Pipeline security
