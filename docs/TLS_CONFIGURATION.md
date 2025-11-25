# TLS/HTTPS Configuration Guide

## Overview

TelHawk Stack supports HTTPS/TLS for all service-to-service communication. This guide explains how to enable TLS with self-signed certificates (for development) or production certificates.

## Quick Start: Enable HTTPS for All Services

Create a `.env` file in the project root:

```bash
# Enable HTTPS/TLS for all Go services (V2 Architecture)
AUTHENTICATE_TLS_ENABLED=true
INGEST_TLS_ENABLED=true
SEARCH_TLS_ENABLED=true
RESPOND_TLS_ENABLED=true
WEB_TLS_ENABLED=true

# Allow self-signed certificates (development only)
# Set to "false" for production with valid certificates
AUTHENTICATE_TLS_SKIP_VERIFY=true
INGEST_TLS_SKIP_VERIFY=true
SEARCH_TLS_SKIP_VERIFY=true
RESPOND_TLS_SKIP_VERIFY=true

# Update service URLs to use HTTPS
INGEST_AUTH_URL=https://authenticate:8080
WEB_AUTH_SERVICE_URL=https://authenticate:8080
WEB_SEARCH_SERVICE_URL=https://search:8082
WEB_RESPOND_SERVICE_URL=https://respond:8085
```

Then start the stack:

```bash
docker compose up -d
```

## Certificate Management

### Self-Signed Certificates (Development)

TelHawk automatically generates self-signed certificates for development:

1. **Certificate Generator Container** (`telhawk-certs`) runs on first startup
2. Generates certificates for all services: `authenticate`, `ingest`, `search`, `respond`, `web`
3. Stores certificates in Docker volume `telhawk-certs`
4. Certificates are valid for 10 years with proper Subject Alternative Names (SANs)

**Certificate Locations:**
- `/certs/generated/ca.pem` - Certificate Authority
- `/certs/generated/{service}.pem` - Service certificate
- `/certs/generated/{service}-key.pem` - Private key

### Production Certificates

For production, mount your own certificates:

```yaml
services:
  auth:
    volumes:
      - ./certs/production/auth.pem:/certs/production/auth.pem:ro
      - ./certs/production/auth-key.pem:/certs/production/auth-key.pem:ro
      - ./certs/production/ca.pem:/certs/production/ca.pem:ro
```

**Priority Order:**
1. Production certificates at `/certs/production/` (if provided)
2. Generated self-signed certificates at `/certs/generated/`

## Configuration Options

### Per-Service TLS Settings

Each service supports these environment variables:

#### Server TLS (Service Listening)
- `{SERVICE}_TLS_ENABLED` - Enable HTTPS server (default: `false`)
- `{SERVICE}_TLS_CERT_FILE` - Path to certificate file
- `{SERVICE}_TLS_KEY_FILE` - Path to private key file

Examples:
- `AUTHENTICATE_TLS_ENABLED=true`
- `INGEST_TLS_ENABLED=true`
- `SEARCH_TLS_ENABLED=true`
- `RESPOND_TLS_ENABLED=true`
- `WEB_TLS_ENABLED=true`

#### Client TLS (Outgoing Connections)
- `{SERVICE}_TLS_SKIP_VERIFY` - Skip certificate verification (default: `false`)

Examples:
- `AUTHENTICATE_TLS_SKIP_VERIFY=true` - Skip verification when connecting to authenticate service
- `INGEST_TLS_SKIP_VERIFY=true` - Skip verification when connecting to ingest service
- `SEARCH_TLS_SKIP_VERIFY=true` - Skip verification when connecting to search service

### OpenSearch TLS

OpenSearch uses self-signed certificates by default:

```bash
OPENSEARCH_URL=https://opensearch:9200
OPENSEARCH_PASSWORD=TelHawk123!
INGEST_OPENSEARCH_TLS_SKIP_VERIFY=true  # Required for self-signed certs
```

### PostgreSQL TLS

PostgreSQL uses SSL/TLS by default:

```bash
AUTH_DATABASE_POSTGRES_SSLMODE=require  # require, verify-ca, or verify-full
```

## Security Considerations

### Development Environment

For local development, self-signed certificates are acceptable:

```bash
# Enable TLS with self-signed certificates
AUTH_TLS_ENABLED=true
AUTH_TLS_SKIP_VERIFY=true  # Accept self-signed certs
```

### Production Environment

For production, use proper certificates and **disable** `TLS_SKIP_VERIFY`:

```bash
# Enable TLS with production certificates
AUTH_TLS_ENABLED=true
AUTH_TLS_SKIP_VERIFY=false  # Validate certificates (IMPORTANT!)

# Provide production certificates
# Mount to /certs/production/ in each service
```

**Production Checklist:**
- ✅ Use certificates from a trusted CA (Let's Encrypt, DigiCert, etc.)
- ✅ Set `TLS_SKIP_VERIFY=false` for all services
- ✅ Enable `COOKIE_SECURE=true` for web service
- ✅ Use HTTPS URLs for all service-to-service communication
- ✅ Configure proper firewall rules
- ✅ Set up certificate rotation/renewal

## Configuration Examples

### Example 1: Full HTTPS Stack (Development)

`.env` file:
```bash
# Enable HTTPS for all services (V2)
AUTHENTICATE_TLS_ENABLED=true
INGEST_TLS_ENABLED=true
SEARCH_TLS_ENABLED=true
RESPOND_TLS_ENABLED=true
WEB_TLS_ENABLED=true

# Accept self-signed certificates
AUTHENTICATE_TLS_SKIP_VERIFY=true
INGEST_TLS_SKIP_VERIFY=true
SEARCH_TLS_SKIP_VERIFY=true
RESPOND_TLS_SKIP_VERIFY=true

# HTTPS URLs
INGEST_AUTH_URL=https://authenticate:8080
WEB_AUTH_SERVICE_URL=https://authenticate:8080
WEB_SEARCH_SERVICE_URL=https://search:8082
WEB_RESPOND_SERVICE_URL=https://respond:8085
```

### Example 2: Selective HTTPS (Ingest Only)

```bash
# Enable HTTPS for external-facing ingest only
INGEST_TLS_ENABLED=true

# Other services remain HTTP
AUTHENTICATE_TLS_ENABLED=false
SEARCH_TLS_ENABLED=false
```

### Example 3: Production Configuration

```bash
# Enable HTTPS for all services (V2)
AUTHENTICATE_TLS_ENABLED=true
INGEST_TLS_ENABLED=true
SEARCH_TLS_ENABLED=true
RESPOND_TLS_ENABLED=true
WEB_TLS_ENABLED=true

# Validate all certificates (production certs required)
AUTHENTICATE_TLS_SKIP_VERIFY=false
INGEST_TLS_SKIP_VERIFY=false
SEARCH_TLS_SKIP_VERIFY=false
RESPOND_TLS_SKIP_VERIFY=false

# HTTPS URLs
INGEST_AUTH_URL=https://authenticate:8080
WEB_AUTH_SERVICE_URL=https://authenticate:8080
WEB_SEARCH_SERVICE_URL=https://search:8082
WEB_RESPOND_SERVICE_URL=https://respond:8085

# Secure cookies
COOKIE_SECURE=true

# Production credentials
OPENSEARCH_PASSWORD=<strong-password>
AUTHENTICATE_DB_PASSWORD=<strong-password>
AUTHENTICATE_JWT_SECRET=<strong-secret>
```

## Troubleshooting

### Certificate Errors

**Error:** `x509: certificate signed by unknown authority`

**Solution:** Enable `TLS_SKIP_VERIFY` for development or provide trusted certificates.

```bash
INGEST_AUTH_TLS_SKIP_VERIFY=true
```

### Connection Refused

**Error:** Service cannot connect to HTTPS endpoint

**Solution:** Ensure the target service has `TLS_ENABLED=true` and is listening on HTTPS.

### Health Check Failures

Health checks use HTTP by default. When TLS is enabled, update health check URLs:

```yaml
healthcheck:
  test: ["CMD", "wget", "-q", "-O", "-", "https://localhost:8080/healthz"]
```

### Regenerate Certificates

To regenerate self-signed certificates:

```bash
# Remove certificate volume
docker compose down
docker volume rm telhawk-stack_telhawk-certs

# Restart - certificates will be regenerated
docker compose up -d
```

## Architecture

### Certificate Flow

```
┌─────────────────────┐
│  telhawk-certs      │  (runs once, exits)
│  ├─ Check for certs │
│  ├─ Generate if none│
│  └─ Store in volume │
└──────────┬──────────┘
           │
           ▼
    [telhawk-certs volume]
           │
           ├───────────────┬───────────────┬───────────────┐
           ▼               ▼               ▼               ▼
  [authenticate:8080] [ingest:8088] [search:8082] [respond:8085]
                      [web:3000]
```

### TLS Verification

When `TLS_SKIP_VERIFY=false`:
- Client verifies server certificate against CA
- Certificate hostname must match (SANs)
- Certificate must not be expired

When `TLS_SKIP_VERIFY=true`:
- Client accepts any certificate (self-signed OK)
- **WARNING:** Susceptible to man-in-the-middle attacks
- **Use only in development**

## Related Documentation

- [SECURITY_ARCHITECTURE.md](./SECURITY_ARCHITECTURE.md) - Overall security design
- [CONFIGURATION.md](./CONFIGURATION.md) - Complete configuration reference
- [DOCKER.md](../DOCKER.md) - Docker deployment guide

## Support

For issues or questions about TLS configuration:
1. Check service logs: `docker compose logs <service>`
2. Verify certificate validity: `openssl x509 -in /certs/generated/auth.pem -text -noout`
3. Open an issue on GitHub with logs and configuration
