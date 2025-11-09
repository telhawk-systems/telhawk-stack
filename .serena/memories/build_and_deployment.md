# TelHawk Stack - Build and Deployment Guide

## Building Services

### Standard Build Process

Each service follows the same build pattern:

```bash
cd <service-name>
go build -o ../bin/<service-name> ./cmd/<service-name>
```

### Build All Services

```bash
# Auth Service
cd auth && go build -o ../bin/auth ./cmd/auth

# Ingest Service
cd ingest && go build -o ../bin/ingest ./cmd/ingest

# Core Service
cd core && go build -o ../bin/core ./cmd/core

# Storage Service
cd storage && go build -o ../bin/storage ./cmd/storage

# Query Service
cd query && go build -o ../bin/query ./cmd/query

# Web Service
cd web/backend && go build -o ../../bin/web ./cmd/web

# CLI Tool
cd cli && go build -o ../bin/thawk .
```

### Build Output
Binaries are placed in `bin/` directory at the project root.

### Build with Specific Flags

```bash
# Build with version information
go build -ldflags "-X main.version=1.0.0" -o ../bin/service ./cmd/service

# Build for production (optimized)
go build -ldflags "-s -w" -o ../bin/service ./cmd/service

# Build for different OS/architecture
GOOS=linux GOARCH=amd64 go build -o ../bin/service ./cmd/service
```

## Docker Build

### Docker Compose Build

#### Build All Services
```bash
docker-compose build
```

#### Build Specific Service
```bash
docker-compose build auth
docker-compose build ingest
docker-compose build core
docker-compose build storage
docker-compose build query
docker-compose build web
```

#### Build with No Cache
```bash
# Force complete rebuild
docker-compose build --no-cache

# Rebuild specific service without cache
docker-compose build --no-cache auth
```

#### Build and Start
```bash
# Rebuild and restart all services
docker-compose up -d --build

# Rebuild and restart specific service
docker-compose up -d --build auth
```

### Individual Docker Builds

Each service has its own Dockerfile:

```bash
# Build auth service image
docker build -t telhawk/auth:latest -f auth/Dockerfile .

# Build from service directory
cd auth && docker build -t telhawk/auth:latest .
```

### Multi-Stage Docker Builds

Services use multi-stage builds for smaller images:

1. **Build stage**: Compiles Go binary
2. **Runtime stage**: Minimal image with only the binary and config

Example Dockerfile pattern:
```dockerfile
# Build stage
FROM golang:1.24.2-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /service ./cmd/service

# Runtime stage
FROM alpine:latest
COPY --from=builder /service /usr/local/bin/service
COPY config.yaml /etc/telhawk/service/config.yaml
CMD ["service"]
```

## Deployment with Docker Compose

### Starting the Stack

#### Development Mode (Default)
```bash
# Start all services in background
docker-compose up -d

# Start with logs visible
docker-compose up

# Start specific services
docker-compose up -d auth ingest core
```

#### Production Mode
```bash
# Use environment file
docker-compose --env-file .env.production up -d

# With specific compose file
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### Stopping the Stack

```bash
# Stop all services (keep data)
docker-compose down

# Stop and remove volumes (DELETE ALL DATA)
docker-compose down -v

# Stop specific service
docker-compose stop auth
```

### Viewing Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f auth
docker-compose logs -f ingest

# Last N lines
docker-compose logs --tail=100 auth

# Since timestamp
docker-compose logs --since 2024-01-01T00:00:00 auth
```

### Service Health and Status

```bash
# Check service status
docker-compose ps

# Check service health
docker-compose ps auth

# Detailed service info
docker inspect telhawk-stack-auth-1
```

### Restarting Services

```bash
# Restart all services
docker-compose restart

# Restart specific service
docker-compose restart auth

# Restart with delay
docker-compose restart -t 30 auth
```

## Environment Configuration

### Environment Files

#### .env.example (Template)
Copy and customize for local development:
```bash
cp .env.example .env
# Edit .env with your settings
```

#### Environment Variable Override
```bash
# Override specific variables
AUTH_JWT_SECRET=newsecret docker-compose up -d

# Use different env file
docker-compose --env-file .env.staging up -d
```

### Configuration Precedence

1. Environment variables (highest priority)
2. .env file
3. config.yaml (lowest priority)

### Key Environment Variables

```bash
# Auth Service
AUTH_SERVER_PORT=8080
AUTH_JWT_SECRET=your-secret-key
AUTH_DB_HOST=auth-db
AUTH_DB_PASSWORD=secure-password

# Ingest Service
INGEST_SERVER_PORT=8088
INGEST_AUTH_URL=http://auth:8080
INGEST_CORE_URL=http://core:8090

# Core Service
CORE_SERVER_PORT=8090
CORE_STORAGE_URL=http://storage:8083

# Storage Service
STORAGE_OPENSEARCH_URL=https://opensearch:9200
STORAGE_OPENSEARCH_USERNAME=admin
STORAGE_OPENSEARCH_PASSWORD=TelHawk123!

# Query Service
QUERY_OPENSEARCH_URL=https://opensearch:9200

# TLS Settings (per service)
AUTH_TLS_ENABLED=false
AUTH_TLS_SKIP_VERIFY=true
```

## Certificate Management

### Certificate Generation

Certificates are automatically generated by init containers:

```bash
# View certificate generation logs
docker-compose logs telhawk-certs
docker-compose logs opensearch-certs
```

### Certificate Locations

- **Go services**: `/certs/` (from `telhawk-certs` volume)
- **OpenSearch**: `/certs/` (from `opensearch-certs` volume)

### Regenerate Certificates

```bash
# Remove old certificates
docker volume rm telhawk-stack_telhawk-certs
docker volume rm telhawk-stack_opensearch-certs

# Restart stack (will regenerate)
docker-compose up -d
```

### TLS Configuration

```bash
# Enable TLS for all services
# Set in .env or docker-compose.yml
AUTH_TLS_ENABLED=true
INGEST_TLS_ENABLED=true
CORE_TLS_ENABLED=true
STORAGE_TLS_ENABLED=true
QUERY_TLS_ENABLED=true
WEB_TLS_ENABLED=true

# Skip certificate verification (DEV ONLY)
AUTH_TLS_SKIP_VERIFY=true
```

## Database Operations

### PostgreSQL (auth-db)

#### Connect to Database
```bash
# Using docker-compose
docker-compose exec auth-db psql -U telhawk -d telhawk_auth

# Direct connection (if exposed)
psql -h localhost -p 5432 -U telhawk -d telhawk_auth
```

#### Backup Database
```bash
# Backup
docker-compose exec auth-db pg_dump -U telhawk telhawk_auth > backup.sql

# Restore
docker-compose exec -T auth-db psql -U telhawk telhawk_auth < backup.sql
```

#### Reset Database
```bash
# Stop auth service
docker-compose stop auth

# Remove database volume
docker-compose down -v
docker volume rm telhawk-stack_auth-db-data

# Restart (migrations will run)
docker-compose up -d
```

### OpenSearch

#### Connect to OpenSearch
```bash
# Using curl (from host)
curl -k -u admin:TelHawk123! https://localhost:9200

# List indices
curl -k -u admin:TelHawk123! https://localhost:9200/_cat/indices?v

# Check cluster health
curl -k -u admin:TelHawk123! https://localhost:9200/_cluster/health?pretty
```

#### Backup Data
```bash
# Export index
docker-compose exec opensearch curl -X GET "localhost:9200/telhawk-events-*/_search?pretty" > events.json
```

#### Reset OpenSearch
```bash
# Remove all indices
curl -k -u admin:TelHawk123! -X DELETE https://localhost:9200/telhawk-events-*

# Or remove volume (complete reset)
docker-compose down -v
docker volume rm telhawk-stack_opensearch-data
docker-compose up -d
```

## Scaling Services

### Scale with Docker Compose
```bash
# Scale specific service
docker-compose up -d --scale ingest=3
docker-compose up -d --scale core=2

# View scaled services
docker-compose ps
```

Note: Requires load balancer configuration for proper distribution.

## Monitoring and Health Checks

### Health Endpoints

Each service exposes health endpoints:

```bash
# Auth service
curl http://localhost:8080/healthz

# Ingest service
curl http://localhost:8088/healthz

# Core service
curl http://localhost:8090/readyz

# Storage service
curl http://localhost:8083/healthz

# Query service
curl http://localhost:8082/healthz

# Web service
curl http://localhost:3000/healthz
```

### Metrics Endpoints

Prometheus-compatible metrics:

```bash
# View metrics for a service
curl http://localhost:8080/metrics
curl http://localhost:8088/metrics
```

### Docker Health Checks

```bash
# View health check status
docker inspect --format='{{json .State.Health}}' telhawk-stack-auth-1 | jq

# View health check logs
docker inspect telhawk-stack-auth-1 | jq '.[0].State.Health'
```

## Production Deployment Checklist

Before deploying to production:

- [ ] Change all default passwords (database, OpenSearch, default user)
- [ ] Set secure `AUTH_JWT_SECRET` environment variable
- [ ] Enable TLS (`*_TLS_ENABLED=true` for all services)
- [ ] Set `*_TLS_SKIP_VERIFY=false` (enforce certificate validation)
- [ ] Use external PostgreSQL (not Docker container)
- [ ] Use external OpenSearch cluster
- [ ] Configure proper backup strategy
- [ ] Set up log aggregation
- [ ] Configure monitoring and alerting
- [ ] Review and harden security settings
- [ ] Set resource limits in docker-compose.yml
- [ ] Configure firewall rules
- [ ] Use secrets management (not plain environment variables)
- [ ] Enable PostgreSQL SSL/TLS (`sslmode=require`)
- [ ] Review RBAC and user permissions
- [ ] Test disaster recovery procedures

## Troubleshooting Build Issues

### Common Build Problems

#### Dependency Issues
```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download

# Tidy dependencies
go mod tidy
```

#### Docker Build Issues
```bash
# Clear Docker build cache
docker builder prune

# Rebuild without cache
docker-compose build --no-cache

# Check Docker disk space
docker system df
```

#### Port Conflicts
```bash
# Check ports in use
netstat -tuln | grep -E '8080|8088|8090|8082|8083|3000|9200|5432|6379'

# Change ports in docker-compose.yml or .env
```

## Quick Reference

| Task | Command |
|------|---------|
| Build all Go services | `cd <service> && go build -o ../bin/<service> ./cmd/<service>` |
| Build Docker images | `docker-compose build` |
| Start stack | `docker-compose up -d` |
| Stop stack | `docker-compose down` |
| View logs | `docker-compose logs -f <service>` |
| Rebuild service | `docker-compose up -d --build <service>` |
| Check health | `docker-compose ps` |
| Reset everything | `docker-compose down -v` |
| Scale service | `docker-compose up -d --scale <service>=N` |