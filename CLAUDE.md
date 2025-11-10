# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Important Instructions

**Git Commits:** When creating git commits, DO NOT add the "Co-Authored-By: Claude" signature or the "Generated with Claude Code" footer. Keep commit messages clean and professional without AI attribution.

## Project Overview

TelHawk Stack is a lightweight, OCSF-compliant SIEM (Security Information and Event Management) platform built in Go. It provides Splunk-compatible event collection with OpenSearch as the backend storage engine. The system is designed as a microservices architecture where events flow through multiple services: ingestion → normalization → storage → query/web.

## Development Commands

### Building and Testing

```bash
# Build all services
cd <service-name> && go build -o ../bin/<service-name> ./cmd/<service-name>

# Examples:
cd auth && go build -o ../bin/auth ./cmd/auth
cd ingest && go build -o ../bin/ingest ./cmd/ingest
cd cli && go build -o ../bin/thawk .

# Test all modules
go test ./...

# Test specific module
cd core && go test ./...

# Run tests with coverage
go test -cover ./...
```

### Docker Operations

```bash
# Start the full stack (default development mode)
docker-compose up -d

# View logs
docker-compose logs -f

# Rebuild all services
docker-compose build

# Rebuild specific service
docker-compose build <service-name>

# Rebuild and restart
docker-compose up -d --build

# Stop all services
docker-compose down

# Stop and remove volumes (deletes all data)
docker-compose down -v

# Check service health
docker-compose ps

# Run CLI tool
docker-compose run --rm thawk <command>
# Example: docker-compose run --rm thawk auth login -u admin -p admin123
```

### Internal API Access (Development Tools)

TelHawk provides two tools for accessing internal services that are not exposed externally:

#### Quick API Calls: `scripts/curl.sh`

For single curl commands, use the curl wrapper:

```bash
# Make GET request to rules service
./scripts/curl.sh http://rules:8084/api/v1/schemas

# Pretty print JSON with jq
./scripts/curl.sh -s http://rules:8084/api/v1/schemas | jq '.'

# Create a detection rule
./scripts/curl.sh -X POST http://rules:8084/api/v1/schemas \
  -H "Content-Type: application/json" \
  -d @rule-definition.json

# Check service health
./scripts/curl.sh http://auth:8080/healthz
./scripts/curl.sh http://query:8082/healthz
./scripts/curl.sh http://rules:8084/healthz
./scripts/curl.sh http://alerting:8085/healthz

# Query OpenSearch directly (requires credentials)
./scripts/curl.sh -u admin:TelHawk123! -k https://opensearch:9200/_cat/indices

# All standard curl options work
./scripts/curl.sh -v -X GET http://core:8090/healthz
```

#### Running Scripts: `scripts/script_exec.sh`

For complex operations or automation, use the persistent devtools container:

```bash
# Start the devtools container (one-time, stays running)
docker-compose --profile devtools up -d devtools

# Run a bash script with access to internal services
./scripts/script_exec.sh tmp/my_script.sh
```

**Available tools in scripts:**
- `bash` - Full bash shell
- `curl` - HTTP client
- `jq` - JSON processor
- `wget` - File downloader
- Access to all internal services (auth, rules, query, core, storage, opensearch, etc.)

**Example script** (`tmp/create_detection_rules.sh`):
```bash
#!/bin/bash
# Create multiple detection rules

RULES_API="http://rules:8084/api/v1/schemas"

for severity in critical high medium; do
  curl -s -X POST "$RULES_API" \
    -H "Content-Type: application/json" \
    -d "{
      \"model\": {...},
      \"view\": {\"severity\": \"$severity\", ...},
      \"controller\": {...}
    }" | jq -r '.id'
done
```

**File locations:**
- Scripts in `./tmp/` are accessible at `/tmp/` in the container (read/write)
- Scripts in `./scripts/` are accessible at `/scripts/` in the container (read-only)

**How it works:** Both tools use an Alpine-based Docker image (`telhawk-stack-devtools`) with curl, bash, jq, and wget. The image is automatically built and connected to the TelHawk internal network.

### Database Migrations

The auth service uses golang-migrate for database schema management:

```bash
# Migrations are automatically run on auth service startup
# Migration files: auth/migrations/*.sql

# To manually run migrations:
cd auth
# View migration status
migrate -database "postgres://telhawk:password@localhost:5432/telhawk_auth?sslmode=disable" -path migrations version

# Apply migrations
migrate -database "postgres://telhawk:password@localhost:5432/telhawk_auth?sslmode=disable" -path migrations up

# Rollback last migration
migrate -database "postgres://telhawk:password@localhost:5432/telhawk_auth?sslmode=disable" -path migrations down 1
```

### Generating Test Data (Event Seeder)

The event seeder generates realistic OCSF events for development and testing:

```bash
# 1. Create a HEC token (via web UI at http://localhost:3000/tokens or via API):
curl -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  -c /tmp/cookies.txt

curl -b /tmp/cookies.txt -X POST http://localhost:3000/api/auth/api/v1/hec/tokens \
  -H "Content-Type: application/json" \
  -d '{"name":"Event Seeder"}'

# Note the token value from the response

# 2. Build and run the seeder:
cd tools/event-seeder
go build
./event-seeder -token YOUR_HEC_TOKEN -count 1000 -time-spread 1h -interval 0

# Common seeder use cases:

# Quick development dataset (100 events):
./event-seeder -token YOUR_TOKEN

# Dashboard population (1000 events over last hour):
./event-seeder -token YOUR_TOKEN -count 1000 -time-spread 1h -interval 0

# Load testing (50k events, fast as possible):
./event-seeder -token YOUR_TOKEN -count 50000 -interval 0 -batch-size 100

# Specific event types only:
./event-seeder -token YOUR_TOKEN -types auth,detection -count 500

# Historical data (week of events):
./event-seeder -token YOUR_TOKEN -count 10000 -time-spread 168h -interval 0
```

**Event Types Generated:**
- Authentication (3002): Login attempts, MFA, logout, password changes
- Network Activity (4001): TCP/UDP/ICMP connections, firewall events
- Process Activity (1007): Process launches with command lines
- File Activity (4006): File operations (create, read, update, delete)
- DNS Activity (4003): DNS queries with various record types
- HTTP Activity (4002): HTTP requests with realistic status codes
- Detection Finding (2004): Security alerts with MITRE ATT&CK tactics

See `tools/event-seeder/README.md` for full documentation.

## Architecture

### Service Communication Flow

```
External Sources → Ingest (8088*) → Core (internal) → Storage (internal) → OpenSearch (localhost)
                       ↓                                                           ↑
                   Auth (internal)                                                 |
                                                                                    |
                                    Query (internal) ←------------------------------+
                                        ↓
                                    Web (3000*)

* = Externally exposed ports
All other services are internal to Docker network only
```

**Security Posture:**
- **Exposed Services:** Only `web` (3000) and `ingest` (8088) accept external connections
- **Internal Services:** `auth`, `query`, `core`, `storage` are only accessible via Docker network
- **Localhost-Only:** OpenSearch (9200, 9600) and Redis (6379) bound to 127.0.0.1
- **Access Methods:**
  - End users → Web UI (port 3000)
  - Event sources → Ingest HEC endpoint (port 8088)
  - Admin operations → `thawk` CLI tool (uses internal network)
- **No Direct Service Access:** Cannot bypass web UI authentication by calling internal services directly

### Service Descriptions

- **auth (internal:8080)**: JWT-based authentication, user management, HEC token generation/validation, RBAC. Uses PostgreSQL for persistence. **Internal only** - accessed via web UI or thawk CLI.
- **rules (internal:8084)**: Detection schema management service. Stores detection rules using MVC pattern (Model/View/Controller) with immutable versioning. Uses PostgreSQL for persistence. **Internal only** - accessed via web UI and alerting service.
- **alerting (internal:8085)**: Evaluation engine and case management. Polls OpenSearch for events, evaluates against detection schemas, generates alerts, manages investigation cases. Uses PostgreSQL for case data, OpenSearch for alerts. **Internal only** - accessed via web UI.
- **ingest (external:8088)**: Splunk HEC-compatible ingestion endpoint. Validates tokens via auth service, forwards raw events to core for normalization. **Externally exposed** for event collection.
- **core (internal:8090)**: OCSF normalization engine. Converts raw events to OCSF-compliant format using auto-generated normalizers (77 OCSF classes). Implements validation, DLQ for failed events, and forwards to storage. **Internal only**.
- **storage (internal:8083)**: OpenSearch abstraction layer. Handles bulk indexing, index lifecycle management, and data persistence. **Internal only**.
- **query (internal:8082)**: Query API with OpenSearch integration. Supports SPL-subset, time-based filtering, aggregations, cursor-based pagination. **Internal only** - accessed via web UI.
- **web (external:3000)**: React-based frontend with search console, event table, OCSF field inspection. **Externally exposed** as primary user interface.
- **cli (thawk)**: Cobra-based CLI for authentication, HEC token management, event ingestion, and search queries. Uses Docker network for internal service access.

### Supporting Services

- **opensearch (9200, 9600)**: Primary datastore with TLS/mTLS security
- **auth-db (PostgreSQL)**: Stores users, sessions, HEC tokens, audit logs
- **rules-db (PostgreSQL)**: Stores detection schemas with immutable versioning
- **alerting-db (PostgreSQL)**: Stores cases and case-alert associations
- **redis (6379)**: Rate limiting (sliding window algorithm), future caching

## Code Architecture

### Event Pipeline (Ingest → Core → Storage)

1. **Ingest Service** receives raw events via HEC endpoint (`/services/collector/event`)
   - Validates HEC token via auth service (with 5-min caching)
   - IP-based and token-based rate limiting (Redis-backed)
   - Forwards to core service for normalization
   - Implements retry with exponential backoff (3 attempts, ~700ms total)
   - Supports HEC ack channel for event tracking

2. **Core Service** normalizes events to OCSF format
   - Registry pattern matches raw event format/source_type to normalizer
   - 77 auto-generated normalizers (one per OCSF class) in `core/internal/normalizer/generated/`
   - HECNormalizer as fallback for generic HEC events
   - Validation chain ensures OCSF compliance
   - Failed events → Dead Letter Queue (file-based at `/var/lib/telhawk/dlq`)
   - Successful events → forwarded to storage service

3. **Storage Service** persists to OpenSearch
   - Bulk indexing with automatic retry (3 attempts, exponential backoff)
   - Index pattern: `telhawk-events-YYYY.MM.DD`
   - OCSF-optimized field mappings

**Key Files:**
- `core/internal/pipeline/pipeline.go`: Orchestrates normalization and validation
- `core/internal/normalizer/normalizer.go`: Registry and interface definitions
- `ingest/internal/handlers/hec.go`: HEC endpoint implementation
- `storage/internal/client/opensearch.go`: OpenSearch bulk operations

### Authentication Flow

1. User login (`POST /api/v1/auth/login`) → returns JWT access token + refresh token
2. Access token used in `Authorization: Bearer <token>` header
3. Token validation endpoint (`POST /api/v1/auth/validate`) called by other services
4. Refresh tokens stored in PostgreSQL sessions table with revocation support
5. HEC tokens stored separately with user association
6. All auth events forwarded to ingest service as OCSF Authentication events (class_uid: 3002)

**Key Files:**
- `auth/internal/repository/postgres.go`: Database operations
- `auth/pkg/tokens/jwt.go`: JWT generation and validation
- `auth/migrations/001_init.up.sql`: Database schema

### OCSF Normalization

The system uses a code generator approach for OCSF compliance:

1. **OCSF Schema** (`ocsf-schema/`): Git submodule tracking OCSF 1.1.0 schema
2. **Generator** (`tools/normalizer-generator/`): Reads OCSF schema, generates Go normalizers
3. **Generated Code** (`core/internal/normalizer/generated/`): One file per OCSF class (77 total)
4. **Runtime**: Registry matches events to normalizers based on source_type patterns

Each normalizer implements:
- Field mapping (common variants → OCSF standard fields)
- Event classification (category_uid, class_uid, activity_id, type_uid)
- Metadata enrichment (product info, timestamps, severity)

**Key Files:**
- `tools/normalizer-generator/main.go`: Code generator
- `core/internal/normalizer/registry.go`: Normalizer registration
- `core/pkg/ocsf/event.go`: Base OCSF event structure

### Configuration Management

All services follow a consistent pattern:
- **YAML config file** embedded in Docker images at `/etc/telhawk/<service>/config.yaml`
- **Environment variables** override YAML settings (12-factor app compliant)
- **Viper** library for config loading
- **No CLI arguments** for configuration

Environment variable naming: `<SERVICE>_<SECTION>_<KEY>`
Examples:
- `AUTH_SERVER_PORT=8080`
- `INGEST_AUTH_URL=http://auth:8080`
- `QUERY_OPENSEARCH_PASSWORD=secret`

**Key Files:**
- `auth/config.yaml`, `ingest/config.yaml`, etc.: Default configurations
- `docker-compose.yml`: Shows environment variable overrides
- `.env.example`: Template for local overrides

## TLS/Certificate Management

The stack uses mutual TLS (mTLS) for service-to-service communication:

1. **Certificate Generation**: Two init containers create certificates before services start
   - `telhawk-certs`: Generates certs for Go services (auth, ingest, core, storage, query, web)
   - `opensearch-certs`: Generates certs for OpenSearch

2. **Certificate Storage**: Shared Docker volumes
   - `telhawk-certs:/certs` - mounted read-only to all Go services
   - `opensearch-certs:/certs` - mounted read-only to OpenSearch

3. **TLS Configuration**: Controlled via environment variables
   - `<SERVICE>_TLS_ENABLED=true/false`: Enable TLS for each service
   - `<SERVICE>_TLS_SKIP_VERIFY=true/false`: Skip cert verification (dev only)

**Key Files:**
- `certs/generator/Dockerfile`: Go service certificate generator
- `opensearch/cert-generator/Dockerfile`: OpenSearch certificate generator
- `docs/TLS_CONFIGURATION.md`: Detailed TLS setup guide

## Important Implementation Details

### Database Schema Patterns

**PostgreSQL - Immutability Pattern**:
The system follows an immutable database pattern for audit trails and versioning:

**Core Principles**:
- **UUID v7**: All new IDs use `uuid.NewV7()` for time-ordered UUIDs (better B-tree performance than random UUIDs)
- **Lifecycle Timestamps**: Use `disabled_at`, `deleted_at`, `hidden_at` instead of boolean flags for audit trails
- **Append-Only Pattern**: INSERT for new content, UPDATE only for lifecycle timestamps
- **No Physical Deletes**: Soft delete with `deleted_at` timestamp and `deleted_by` user reference
- **Immutable Versioning**: Content changes create new rows with same stable ID but new version ID

**Auth Service (users, hec_tokens, sessions)**:
- UUIDs: UUID v7 primary keys
- Lifecycle: `created_at`, `disabled_at`, `disabled_by`, `deleted_at`, `deleted_by`
- No `enabled` boolean or `updated_at` timestamp
- Helper methods: `IsActive()` checks lifecycle state
- Example:
  ```sql
  CREATE TABLE users (
      id UUID PRIMARY KEY,
      username VARCHAR(255) NOT NULL UNIQUE,
      created_at TIMESTAMP NOT NULL DEFAULT NOW(),
      disabled_at TIMESTAMP,
      disabled_by UUID REFERENCES users(id),
      deleted_at TIMESTAMP,
      deleted_by UUID REFERENCES users(id)
  );
  ```

**Rules Service (detection_schemas)**:
- Immutable versioning: Same `id` (stable) + new `version_id` (version-specific) per update
- Window functions calculate version numbers on read (no race conditions)
- Lifecycle: `created_at`, `disabled_at`, `disabled_by`, `hidden_at`, `hidden_by`
- Server-generated UUIDs: Users NEVER provide `id` or `version_id` in requests
- POST creates new rule (server generates both IDs), PUT creates new version (reuses `id`, new `version_id`)
- Example:
  ```sql
  CREATE TABLE detection_schemas (
      id UUID NOT NULL,                 -- Stable identifier (groups versions)
      version_id UUID PRIMARY KEY,      -- Version-specific UUID (UUID v7)
      model JSONB NOT NULL,
      view JSONB NOT NULL,
      controller JSONB NOT NULL,
      created_at TIMESTAMP NOT NULL DEFAULT NOW(),
      disabled_at TIMESTAMP,
      disabled_by UUID,
      hidden_at TIMESTAMP,
      hidden_by UUID
  );
  ```

**Alerting Service (cases, case_alerts)**:
- Cases: UUID v7 IDs, mutable status field (open, in_progress, resolved, closed)
- Case lifecycle: `created_at`, `updated_at`, `closed_at`, `closed_by`
- Case-Alert junction: Links cases to alerts (OpenSearch documents)
- Foreign keys: `detection_schema_id` (stable) and `detection_schema_version_id` (version)

**OpenSearch**:
- Daily time-based indices: `telhawk-events-YYYY.MM.DD`, `telhawk-alerts-YYYY.MM.DD`
- OCSF-optimized mappings (nested objects for actors, devices, etc.)
- Retention managed via index lifecycle policies
- Query pattern: `telhawk-events-*` for searches across all indices

### Error Handling and Reliability

**Dead Letter Queue (DLQ)**:
- File-based storage at `/var/lib/telhawk/dlq`
- Captures normalization and storage failures
- Preserves full event context for debugging/replay
- API endpoints: `GET /dlq/list`, `POST /dlq/purge`
- Metrics exposed via health endpoint

**Retry Strategy**:
- Ingest → Core: 3 attempts, exponential backoff (~700ms total)
- Core → Storage: 3 attempts, exponential backoff
- Retries on 5xx, 429, network errors
- No retry on 4xx client errors (except 429)

**Rate Limiting**:
- Redis-backed sliding window algorithm
- IP-based (pre-auth) and token-based (post-auth)
- Returns HTTP 429 when exceeded
- Graceful degradation if Redis unavailable

### Observability

**Health Checks**:
- All services expose `/healthz` or `/readyz` endpoints
- Docker health checks with configurable retries
- Dependencies managed via `depends_on` with health conditions

**Metrics** (Prometheus format at `/metrics`):
- Event processing: `events_total`, `normalization_duration`, `storage_duration`
- Queue depth: `queue_depth`, `queue_capacity`
- Rate limiting: `rate_limit_hits_total`
- Acks: `acks_pending`, `acks_completed_total`

## Testing

### Test Organization

- Unit tests alongside source files: `*_test.go`
- Integration tests in dedicated files: `*_integration_test.go`
- Test data and fixtures in `testdata/` directories

### Key Test Files

- `core/internal/pipeline/integration_test.go`: End-to-end normalization pipeline
- `ingest/internal/handlers/hec_handler_test.go`: HEC endpoint tests
- `query/internal/service/service_test.go`: Query API tests

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -v ./core/internal/pipeline -run TestNormalization

# Run with race detection
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## CLI Tool (thawk)

The CLI uses Cobra for command structure:

```bash
# Authentication
thawk auth login -u <username> -p <password>
thawk auth whoami
thawk auth logout

# HEC token management
thawk token create --name <token-name>
thawk token list
thawk token revoke <token-id>

# Event ingestion
thawk ingest send --message "event text" --token <hec-token>

# Search queries
thawk search --query "severity:high" --from 1h
```

**Key Files:**
- `cli/cmd/root.go`: Root command and global flags
- `cli/cmd/auth.go`, `cli/cmd/token.go`, etc.: Subcommands
- `cli/internal/config/config.go`: CLI configuration (~/.thawk/config.yaml)

## Code Generation

### OCSF Normalizer Generator

Location: `tools/normalizer-generator/`

Regenerate normalizers after OCSF schema updates:

```bash
cd tools/normalizer-generator
go run main.go

# Output: core/internal/normalizer/generated/*.go (77 files)
```

The generator:
- Reads OCSF schema from `ocsf-schema/` directory
- Generates one normalizer per event class
- Creates intelligent field mappings
- Outputs registration code for the normalizer registry

## Common Development Patterns

### Adding a New Service

1. Create service directory with standard structure:
   ```
   newservice/
   ├── cmd/newservice/main.go
   ├── internal/
   │   ├── config/config.go
   │   ├── handlers/handlers.go
   │   └── ...
   ├── Dockerfile
   ├── config.yaml
   └── go.mod
   ```

2. Add to `docker-compose.yml` with health checks and dependencies
3. Add TLS certificate generation if needed
4. Follow environment variable naming: `NEWSERVICE_SECTION_KEY`

### Adding Database Migrations

1. Create numbered migration files in `auth/migrations/`:
   - `NNN_description.up.sql`
   - `NNN_description.down.sql`

2. Migrations run automatically on auth service startup
3. Use PostgreSQL best practices: indexes, constraints, comments

### Adding a New OCSF Normalizer (Custom)

If you need custom normalization logic beyond the generated code:

1. Create file in `core/internal/normalizer/` (not in `generated/`)
2. Implement the `Normalizer` interface
3. Register in `core/internal/normalizer/registry.go`

Example:
```go
type CustomNormalizer struct{}

func (n *CustomNormalizer) Normalize(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error) {
    // Custom logic
}

func (n *CustomNormalizer) Matches(envelope *model.RawEventEnvelope) bool {
    // Match condition
}
```

## Security Considerations

**Network Exposure:**
- **Minimal Attack Surface**: Only web (3000) and ingest (8088) exposed externally
- **No Direct Service Access**: Auth, query, core, and storage services ONLY accessible via Docker network
- **Self-Registration Disabled**: User accounts must be created by administrators (registration endpoint disabled)
- **Admin Access**: All administrative operations require authentication via web UI or thawk CLI

**Authentication & Authorization:**
- JWT secrets MUST be set via `AUTH_JWT_SECRET` environment variable
- HEC tokens are random UUIDs, stored hashed in database
- All auth events forwarded to SIEM for audit trail (nonrepudiation)
- Audit log table captures all authentication/authorization events with HMAC signatures

**Transport Security:**
- TLS MUST be enabled in production (`*_TLS_ENABLED=true`)
- PostgreSQL uses SSL/TLS in production (`sslmode=require`)
- OpenSearch uses TLS with client certificates

**Operational Security:**
- Default passwords in `docker-compose.yml` MUST be changed for production
- Rate limiting prevents abuse of ingestion endpoints
- Dead Letter Queue captures failed events for forensic analysis

## Documentation

Key documentation files:
- `README.md`: Overview, quick start, architecture
- `docs/CONFIGURATION.md`: Complete configuration reference
- `docs/TLS_CONFIGURATION.md`: TLS/certificate setup
- `docs/CLI_CONFIGURATION.md`: CLI usage guide
- `DOCKER.md`: Docker commands and troubleshooting
- `TODO.md`: Development roadmap and recent accomplishments
- Individual service READMEs in service directories

## Web Frontend

The web UI is a React application with:
- **Backend**: Go server in `web/backend/` (port 3000)
- **Frontend**: React app in `web/frontend/`
- **Build**: Frontend built and served as static files by backend
- **Features**: Search console, event table, OCSF field inspection, severity color-coding

Frontend development:
```bash
cd web/frontend
npm install
npm start  # Development server
npm run build  # Production build
```

## Default Credentials

**Database (development)**:
- PostgreSQL: `telhawk:telhawk-auth-dev@auth-db:5432/telhawk_auth`
- OpenSearch: `admin:TelHawk123!`

**Default User (created by migration)**:
- Username: `admin`
- Password: `admin123`
- Email: `admin@telhawk.local`
- Roles: `[admin]`

**IMPORTANT**: Change all default passwords before production deployment.
