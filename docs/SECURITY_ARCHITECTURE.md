# TelHawk Security Architecture

## Core Security Principles

### 1. No Demo Credentials - EVER
**Policy:** TelHawk NEVER uses demo credentials from any component.

- OpenSearch demo credentials (admin:admin) are NEVER used
- All credentials must be configurable via environment variables
- Credentials can be rotated by changing env vars and restarting
- Default credentials are strong and documented

### 2. SSL/TLS Everywhere
**Policy:** All internal service communication can use TLS with proper certificates.

**Current Status (V2 Architecture):**
- ✅ **OpenSearch (9200, 9600):** HTTPS enabled with self-signed certificates
- ✅ **PostgreSQL (5432):** SSL/TLS enabled with self-signed certificates
- ✅ **Authenticate (8080):** TLS support with feature flag (`AUTHENTICATE_TLS_ENABLED`)
- ✅ **Ingest (8088):** TLS support with feature flag (`INGEST_TLS_ENABLED`)
- ✅ **Search (8082):** TLS support with feature flag (`SEARCH_TLS_ENABLED`)
- ✅ **Respond (8085):** TLS support with feature flag (`RESPOND_TLS_ENABLED`)
- ✅ **Web (3000):** TLS support with feature flag (`WEB_TLS_ENABLED`)

**Implementation:** TLS is disabled by default but can be enabled via environment variables. See `docs/TLS_CONFIGURATION.md` for details.

#### Certificate Generation Strategy
TelHawk Stack uses automated certificate generation for both OpenSearch and Go services:

**OpenSearch Certificates:**
1. **opensearch-certs** - Sidecar container for OpenSearch
   - Checks if certificates already exist in volume
   - If not, generates self-signed certificates with proper SANs
   - Stores certificates in `opensearch-certs` volume
   - Exits after generation (no health check needed - it's a one-shot job)

2. **opensearch** - Main database container
   - Depends on opensearch-certs completion
   - Uses certificates from shared volume
   - Updates admin credentials from env vars on EVERY boot
   - Credentials are rotatable without regenerating certs

**Go Service Certificates:**
1. **telhawk-certs** - Certificate generator for all Go services
   - Generates certificates for: authenticate, ingest, search, respond, web
   - Creates self-signed CA and service certificates
   - Stores certificates in `telhawk-certs` volume
   - Each certificate includes proper Subject Alternative Names (SANs)
   - Certificates valid for 10 years

2. **Services** - All Go services can use certificates
   - Mount `telhawk-certs` volume as read-only
   - TLS disabled by default (enable with `{SERVICE}_TLS_ENABLED=true`)
   - Support for TLS_SKIP_VERIFY flag (development only)

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
- authenticate
- authenticate-db
- respond-db
- search
- respond
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
- [DLQ_AND_BACKPRESSURE.md](./DLQ_AND_BACKPRESSURE.md) - Pipeline security
- [TLS_CONFIGURATION.md](./TLS_CONFIGURATION.md) - TLS setup guide
