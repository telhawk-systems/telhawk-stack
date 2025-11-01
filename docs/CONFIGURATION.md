# TelHawk Stack Configuration Guide

## Configuration Philosophy

TelHawk Stack follows enterprise-grade configuration best practices:

1. **YAML config files** - Default configuration in `/etc/telhawk/<service>/config.yaml`
2. **Environment variable overrides** - Any config value can be overridden via env vars
3. **No command-line arguments for config** - Only `-config` flag to specify config file path

This approach ensures:
- ✅ Configuration as code (YAML in version control)
- ✅ Environment-specific overrides (dev, staging, prod)
- ✅ 12-factor app compliance
- ✅ Kubernetes/Docker-friendly

## Configuration Priority

Configuration values are resolved in this order (highest to lowest priority):

1. **Environment variables** (e.g., `AUTH_SERVER_PORT=9090`)
2. **Config file** (e.g., `/etc/telhawk/auth/config.yaml`)
3. **Built-in defaults** (hardcoded in service)

## Auth Service Configuration

### Config File Location
- Default: `/etc/telhawk/auth/config.yaml`
- Custom: `auth -config /path/to/config.yaml`

### Sample Configuration

```yaml
server:
  port: 8080
  read_timeout: 15s
  write_timeout: 15s
  idle_timeout: 60s

auth:
  jwt_secret: "change-this-in-production"
  access_token_ttl: 15m
  refresh_token_ttl: 168h  # 7 days

database:
  type: memory  # memory or postgres
  postgres:
    host: localhost
    port: 5432
    database: telhawk_auth
    user: telhawk
    password: ""
    sslmode: disable

logging:
  level: info  # debug, info, warn, error
  format: json  # json or text
```

### Environment Variable Overrides

All config values can be overridden using environment variables with the `AUTH_` prefix:

```bash
# Override server port
AUTH_SERVER_PORT=9090

# Override JWT secret
AUTH_AUTH_JWT_SECRET="my-production-secret-key"

# Override database type
AUTH_DATABASE_TYPE=postgres

# Override nested postgres config
AUTH_DATABASE_POSTGRES_HOST=db.example.com
AUTH_DATABASE_POSTGRES_PORT=5432
AUTH_DATABASE_POSTGRES_PASSWORD=secret
```

## Ingest Service Configuration

### Config File Location
- Default: `/etc/telhawk/ingest/config.yaml`
- Custom: `ingest -config /path/to/config.yaml`

### Sample Configuration

```yaml
server:
  port: 8088
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

auth:
  url: http://localhost:8080
  token_validation_cache_ttl: 5m

opensearch:
  url: https://localhost:9200
  username: admin
  password: ""
  tls_skip_verify: true
  index_prefix: telhawk
  bulk_batch_size: 1000
  bulk_flush_interval: 5s

ingestion:
  max_event_size: 1048576  # 1MB
  rate_limit_enabled: true
  rate_limit_requests: 10000
  rate_limit_window: 1m

logging:
  level: info
  format: json
```

### Environment Variable Overrides

All config values can be overridden using environment variables with the `INGEST_` prefix:

```bash
# Override server port
INGEST_SERVER_PORT=8088

# Override auth URL
INGEST_AUTH_URL=http://auth-service:8080

# Override OpenSearch connection
INGEST_OPENSEARCH_URL=https://opensearch-cluster:9200
INGEST_OPENSEARCH_USERNAME=admin
INGEST_OPENSEARCH_PASSWORD=MySecurePassword123!
INGEST_OPENSEARCH_TLS_SKIP_VERIFY=false

# Override rate limiting
INGEST_INGESTION_RATE_LIMIT_REQUESTS=50000
INGEST_INGESTION_MAX_EVENT_SIZE=2097152  # 2MB
```

## Docker Compose Configuration

The `docker-compose.yml` demonstrates environment variable usage:

```yaml
services:
  auth:
    environment:
      - AUTH_SERVER_PORT=8080
      - AUTH_LOGGING_LEVEL=info
      - AUTH_AUTH_JWT_SECRET=${AUTH_JWT_SECRET:-change-this-in-production}

  ingest:
    environment:
      - INGEST_SERVER_PORT=8088
      - INGEST_AUTH_URL=http://auth:8080
      - INGEST_OPENSEARCH_URL=https://opensearch:9200
      - INGEST_OPENSEARCH_PASSWORD=${OPENSEARCH_PASSWORD:-TelHawk123!}
```

### Using .env File

Create a `.env` file in the same directory as `docker-compose.yml`:

```bash
# .env
AUTH_JWT_SECRET=my-super-secret-jwt-key-change-in-production
OPENSEARCH_PASSWORD=MyStrongPassword123!
```

Docker Compose automatically loads `.env` and substitutes variables.

## Kubernetes Configuration

### ConfigMaps for YAML Files

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: auth-config
data:
  config.yaml: |
    server:
      port: 8080
    auth:
      jwt_secret: "will-be-overridden-by-secret"
    logging:
      level: info
```

### Secrets for Sensitive Data

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: auth-secrets
type: Opaque
stringData:
  AUTH_AUTH_JWT_SECRET: "production-jwt-secret-key"
  AUTH_DATABASE_POSTGRES_PASSWORD: "db-password"
```

### Deployment with ConfigMap and Secrets

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: auth
spec:
  template:
    spec:
      containers:
      - name: auth
        image: telhawk/auth:latest
        envFrom:
        - secretRef:
            name: auth-secrets
        volumeMounts:
        - name: config
          mountPath: /etc/telhawk/auth
      volumes:
      - name: config
        configMap:
          name: auth-config
```

## Configuration Best Practices

### Development
- Use default config.yaml files
- Override with environment variables for local testing
- Never commit secrets to config files

### Staging/Production
- Store config.yaml in ConfigMaps (Kubernetes) or volumes (Docker)
- Use Secrets/environment variables for sensitive data:
  - JWT secrets
  - Database passwords
  - OpenSearch credentials
  - API keys
- Enable TLS/SSL (`tls_skip_verify: false`)
- Set appropriate timeouts and limits
- Use structured logging (`format: json`)

### Security
- ⚠️ **Change default secrets in production**
- ⚠️ **Never commit passwords to Git**
- ✅ Use secrets management (Vault, AWS Secrets Manager, etc.)
- ✅ Rotate credentials regularly
- ✅ Use least-privilege database users

## Viewing Current Configuration

Services log their configuration on startup:

```bash
# Auth service
docker-compose logs auth | grep "Starting Auth"
# Output: Starting Auth service on port 8080
# Output: Loaded config from: /etc/telhawk/auth/config.yaml

# Ingest service
docker-compose logs ingest | grep -A 3 "Starting Ingest"
# Output: Starting Ingest service on port 8088
# Output: Loaded config from: /etc/telhawk/ingest/config.yaml
# Output: Auth URL: http://auth:8080
# Output: OpenSearch URL: https://opensearch:9200
```

## Troubleshooting

### Config File Not Found
Services fallback to defaults if config file is missing. Check logs for warnings.

### Environment Variables Not Applied
Ensure environment variable names match the config structure:
- Use uppercase
- Prefix with service name (AUTH_, INGEST_)
- Separate nested keys with underscores
- Example: `auth.jwt_secret` → `AUTH_AUTH_JWT_SECRET`

### Invalid Configuration Values
Services will log errors and fail to start. Check logs:
```bash
docker-compose logs <service>
```
