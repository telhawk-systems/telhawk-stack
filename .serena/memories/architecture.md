# TelHawk Stack - Architecture

## Service Communication Flow
```
External Sources → Ingest (8088) → Core (8090) → Storage (8083) → OpenSearch (9200)
                       ↓                                                    ↑
                   Auth (8080)                                              |
                                                                             |
                                    Query (8082) ←---------------------------+
                                        ↓
                                    Web (3000)
```

## Event Pipeline (Core Data Flow)

### 1. Ingestion (ingest service - port 8088)
- Receives raw events via HEC endpoint `/services/collector/event`
- Validates HEC token via auth service (with 5-min caching)
- IP-based and token-based rate limiting (Redis-backed)
- Forwards to core service for normalization
- Retry with exponential backoff (3 attempts, ~700ms total)
- Supports HEC ack channel for event tracking

**Key Files**: `ingest/internal/handlers/hec.go`

### 2. Normalization (core service - port 8090)
- Registry pattern matches raw event format/source_type to normalizer
- 77 auto-generated normalizers (one per OCSF class) in `core/internal/normalizer/generated/`
- HECNormalizer as fallback for generic HEC events
- Validation chain ensures OCSF compliance
- Failed events → Dead Letter Queue (file-based at `/var/lib/telhawk/dlq`)
- Successful events → forwarded to storage service

**Key Files**:
- `core/internal/pipeline/pipeline.go`: Orchestrates normalization
- `core/internal/normalizer/normalizer.go`: Registry and interfaces
- `core/internal/normalizer/registry.go`: Normalizer registration

### 3. Storage (storage service - port 8083)
- Bulk indexing with automatic retry (3 attempts, exponential backoff)
- Index pattern: `telhawk-events-YYYY.MM.DD`
- OCSF-optimized field mappings

**Key Files**: `storage/internal/client/opensearch.go`

### 4. Query (query service - port 8082)
- SPL-subset query language support
- Time-based filtering and aggregations
- Cursor-based pagination
- Direct OpenSearch integration

**Key Files**: `query/internal/service/service.go`

## Authentication Flow

1. User login (`POST /api/v1/auth/login`) → JWT access token + refresh token
2. Access token used in `Authorization: Bearer <token>` header
3. Token validation endpoint (`POST /api/v1/auth/validate`) called by other services
4. Refresh tokens stored in PostgreSQL sessions table with revocation
5. HEC tokens stored separately with user association
6. All auth events forwarded to ingest as OCSF Authentication events (class_uid: 3002)

**Key Files**:
- `auth/internal/repository/postgres.go`: Database operations
- `auth/pkg/tokens/jwt.go`: JWT generation/validation
- `auth/migrations/001_init.up.sql`: Database schema

## OCSF Normalization Architecture

### Code Generation Approach
1. **OCSF Schema** (`ocsf-schema/`): Git submodule tracking OCSF 1.1.0
2. **Generator** (`tools/normalizer-generator/`): Reads schema, generates Go code
3. **Generated Code** (`core/internal/normalizer/generated/`): 77 normalizer files
4. **Runtime**: Registry matches events to normalizers via source_type patterns

### Normalizer Responsibilities
- Field mapping (common variants → OCSF standard fields)
- Event classification (category_uid, class_uid, activity_id, type_uid)
- Metadata enrichment (product info, timestamps, severity)

**Key Files**:
- `tools/normalizer-generator/main.go`: Code generator
- `common/ocsf/ocsf/event.go`: Base OCSF event structure

## Configuration Management

All services follow consistent pattern:
- **YAML config** embedded at `/etc/telhawk/<service>/config.yaml`
- **Environment variables** override YAML (12-factor app)
- **Viper library** for config loading
- **No CLI arguments** for configuration

Environment variable naming: `<SERVICE>_<SECTION>_<KEY>`
Examples:
- `AUTH_SERVER_PORT=8080`
- `INGEST_AUTH_URL=http://auth:8080`
- `QUERY_OPENSEARCH_PASSWORD=secret`

## Database Architecture

### PostgreSQL (auth service)
- UUID primary keys
- Timestamp tracking (created_at, updated_at)
- Trigger-based updated_at automation
- Foreign keys with CASCADE delete
- JSONB for flexible metadata
- Tables: users, sessions, hec_tokens, audit_log

### OpenSearch
- Daily time-based indices: `telhawk-events-YYYY.MM.DD`
- OCSF-optimized mappings (nested objects for actors, devices)
- Query pattern: `telhawk-events-*`
- Retention via index lifecycle policies

## Error Handling

### Dead Letter Queue (DLQ)
- File-based storage at `/var/lib/telhawk/dlq`
- Captures normalization and storage failures
- Preserves full event context
- API: `GET /dlq/list`, `POST /dlq/purge`

### Retry Strategy
- Ingest → Core: 3 attempts, exponential backoff
- Core → Storage: 3 attempts, exponential backoff
- Retries on 5xx, 429, network errors
- No retry on 4xx (except 429)

### Rate Limiting
- Redis-backed sliding window algorithm
- IP-based (pre-auth) and token-based (post-auth)
- Returns HTTP 429 when exceeded
- Graceful degradation if Redis unavailable

## TLS/Certificate Management

### Certificate Generation
- Two init containers create certificates before services start
- `telhawk-certs`: For Go services
- `opensearch-certs`: For OpenSearch

### Certificate Storage
- `telhawk-certs:/certs` - mounted read-only to Go services
- `opensearch-certs:/certs` - mounted read-only to OpenSearch

### TLS Configuration
- `<SERVICE>_TLS_ENABLED=true/false`: Enable TLS per service
- `<SERVICE>_TLS_SKIP_VERIFY=true/false`: Skip verification (dev only)