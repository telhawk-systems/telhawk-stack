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

1. **Environment variables** (e.g., `AUTHENTICATE_SERVER_PORT=9090`)
2. **Config file** (e.g., `/etc/telhawk/authenticate/config.yaml`)
3. **Built-in defaults** (hardcoded in service)

## Service Configuration

### authenticate (Authentication & RBAC)

**Config file**: `/etc/telhawk/authenticate/config.yaml`
**Env prefix**: `AUTHENTICATE_`

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

**Environment overrides:**
```bash
AUTHENTICATE_SERVER_PORT=8080
AUTHENTICATE_AUTH_JWT_SECRET="my-production-secret-key"
AUTHENTICATE_DATABASE_TYPE=postgres
AUTHENTICATE_DATABASE_POSTGRES_HOST=db.example.com
AUTHENTICATE_DATABASE_POSTGRES_PASSWORD=secret
```

---

### ingest (Event Ingestion + Storage)

**Config file**: `/etc/telhawk/ingest/config.yaml`
**Env prefix**: `INGEST_`

```yaml
server:
  port: 8088
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

authenticate:
  url: http://authenticate:8080
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

**Environment overrides:**
```bash
INGEST_SERVER_PORT=8088
INGEST_AUTHENTICATE_URL=http://authenticate:8080
INGEST_OPENSEARCH_URL=https://opensearch-cluster:9200
INGEST_OPENSEARCH_USERNAME=admin
INGEST_OPENSEARCH_PASSWORD=MySecurePassword123!
INGEST_OPENSEARCH_TLS_SKIP_VERIFY=false
INGEST_INGESTION_RATE_LIMIT_REQUESTS=50000
```

---

### search (Query API + Correlation)

**Config file**: `/etc/telhawk/search/config.yaml`
**Env prefix**: `SEARCH_`

```yaml
server:
  port: 8082

opensearch:
  url: https://opensearch:9200
  username: admin
  password: ""

auth:
  url: http://authenticate:8080

logging:
  level: info
  format: json
```

**Environment overrides:**
```bash
SEARCH_SERVER_PORT=8082
SEARCH_OPENSEARCH_URL=https://opensearch:9200
SEARCH_OPENSEARCH_PASSWORD=MySecurePassword123!
SEARCH_AUTH_URL=http://authenticate:8080
```

---

### respond (Detection Rules + Alerting + Cases)

**Config file**: `/etc/telhawk/respond/config.yaml`
**Env prefix**: `RESPOND_`

```yaml
server:
  port: 8085

database:
  postgres:
    host: auth-db
    port: 5432
    database: telhawk_respond
    user: telhawk
    password: ""

auth:
  url: http://authenticate:8080

logging:
  level: info
  format: json
```

**Environment overrides:**
```bash
RESPOND_SERVER_PORT=8085
RESPOND_DATABASE_POSTGRES_HOST=db.example.com
RESPOND_DATABASE_POSTGRES_PASSWORD=secret
RESPOND_AUTH_URL=http://authenticate:8080
```

---

### web (Frontend UI + API Gateway)

**Config file**: `/etc/telhawk/web/config.yaml`
**Env prefix**: `WEB_`

```yaml
server:
  port: 3000

services:
  authenticate_url: http://authenticate:8080
  search_url: http://search:8082
  respond_url: http://respond:8085
  ingest_url: http://ingest:8088

logging:
  level: info
  format: json
```

**Environment overrides:**
```bash
WEB_SERVER_PORT=3000
WEB_SERVICES_AUTHENTICATE_URL=http://authenticate:8080
WEB_SERVICES_SEARCH_URL=http://search:8082
WEB_SERVICES_RESPOND_URL=http://respond:8085
```

---

## Docker Compose Configuration

The `docker-compose.yml` demonstrates environment variable usage:

```yaml
services:
  authenticate:
    environment:
      - AUTHENTICATE_SERVER_PORT=8080
      - AUTHENTICATE_LOGGING_LEVEL=info
      - AUTHENTICATE_AUTH_JWT_SECRET=${AUTHENTICATE_JWT_SECRET:-change-this-in-production}

  ingest:
    environment:
      - INGEST_SERVER_PORT=8088
      - INGEST_AUTHENTICATE_URL=http://authenticate:8080
      - INGEST_OPENSEARCH_URL=https://opensearch:9200
      - INGEST_OPENSEARCH_PASSWORD=${OPENSEARCH_PASSWORD:-TelHawk123!}

  search:
    environment:
      - SEARCH_SERVER_PORT=8082
      - SEARCH_AUTH_URL=http://authenticate:8080

  respond:
    environment:
      - RESPOND_SERVER_PORT=8085
      - RESPOND_AUTH_URL=http://authenticate:8080
```

### Using .env File

Create a `.env` file in the same directory as `docker-compose.yml`:

```bash
# .env
AUTHENTICATE_JWT_SECRET=my-super-secret-jwt-key-change-in-production
OPENSEARCH_PASSWORD=MyStrongPassword123!
```

Docker Compose automatically loads `.env` and substitutes variables.

## Kubernetes Configuration

### ConfigMaps for YAML Files

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: authenticate-config
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
  name: authenticate-secrets
type: Opaque
stringData:
  AUTHENTICATE_AUTH_JWT_SECRET: "production-jwt-secret-key"
  AUTHENTICATE_DATABASE_POSTGRES_PASSWORD: "db-password"
```

### Deployment with ConfigMap and Secrets

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: authenticate
spec:
  template:
    spec:
      containers:
      - name: authenticate
        image: telhawk/authenticate:latest
        envFrom:
        - secretRef:
            name: authenticate-secrets
        volumeMounts:
        - name: config
          mountPath: /etc/telhawk/authenticate
      volumes:
      - name: config
        configMap:
          name: authenticate-config
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
# Authenticate service
docker-compose logs authenticate | grep "Starting"

# Ingest service
docker-compose logs ingest | grep -A 3 "Starting"

# Search service
docker-compose logs search | grep "Starting"
```

## Troubleshooting

### Config File Not Found
Services fallback to defaults if config file is missing. Check logs for warnings.

### Environment Variables Not Applied
Ensure environment variable names match the config structure:
- Use uppercase
- Prefix with service name (AUTHENTICATE_, INGEST_, SEARCH_, RESPOND_, WEB_)
- Separate nested keys with underscores
- Example: `auth.jwt_secret` → `AUTHENTICATE_AUTH_JWT_SECRET`

### Invalid Configuration Values
Services will log errors and fail to start. Check logs:
```bash
docker-compose logs <service>
```
