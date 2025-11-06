# TLS Quick Reference

## Enable TLS (Development)

```bash
# Quick setup with helper script
./bin/configure-tls.sh enable
docker-compose up -d

# Or manually create .env
cat > .env << 'EOF'
AUTH_TLS_ENABLED=true
INGEST_TLS_ENABLED=true
CORE_TLS_ENABLED=true
STORAGE_TLS_ENABLED=true
QUERY_TLS_ENABLED=true
WEB_TLS_ENABLED=true

AUTH_TLS_SKIP_VERIFY=true
CORE_TLS_SKIP_VERIFY=true
STORAGE_TLS_SKIP_VERIFY=true

INGEST_AUTH_URL=https://auth:8080
INGEST_CORE_URL=https://core:8090
INGEST_STORAGE_URL=https://storage:8083
CORE_STORAGE_URL=https://storage:8083
WEB_AUTH_SERVICE_URL=https://auth:8080
WEB_QUERY_SERVICE_URL=https://query:8082
WEB_CORE_SERVICE_URL=https://core:8090
EOF
```

## Disable TLS (Default)

```bash
# Quick disable
./bin/configure-tls.sh disable
docker-compose restart

# Or remove .env
rm .env
```

## Check Status

```bash
./bin/configure-tls.sh status
```

## Test HTTPS

```bash
# Test with self-signed cert
curl -k https://localhost:8080/healthz

# Check certificates exist
docker exec telhawk-auth ls -la /certs/generated/
```

## Environment Variables

### Server TLS (Enable HTTPS)
- `AUTH_TLS_ENABLED=true`
- `INGEST_TLS_ENABLED=true`
- `CORE_TLS_ENABLED=true`
- `STORAGE_TLS_ENABLED=true`
- `QUERY_TLS_ENABLED=true`
- `WEB_TLS_ENABLED=true`

### Client TLS (Skip Verification)
- `AUTH_TLS_SKIP_VERIFY=true` - For self-signed auth certs
- `CORE_TLS_SKIP_VERIFY=true` - For self-signed core certs
- `STORAGE_TLS_SKIP_VERIFY=true` - For self-signed storage certs

## Certificate Locations

- **Self-signed:** `/certs/generated/*.pem`
- **Production:** `/certs/production/*.pem`
- **CA Certificate:** `/certs/generated/ca.pem`

## Full Documentation

See: [docs/TLS_CONFIGURATION.md](docs/TLS_CONFIGURATION.md)
