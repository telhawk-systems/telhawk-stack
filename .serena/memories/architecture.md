# TelHawk Stack - Architecture (V2)

## Service Overview (5 Services)

```
                    ┌─────────────────────────────────────────┐
                    │                  NATS                   │
                    │         (Message Broker)                │
                    └──────┬─────────────┬───────────────────┘
                           │             │
    ┌──────────────────────┼─────────────┼──────────────────────┐
    │                      ↓             ↓                      │
    │  ┌─────────┐    ┌─────────┐   ┌─────────┐   ┌─────────┐  │
    │  │ ingest  │    │ search  │   │ respond │   │   web   │  │
    │  │  8088   │    │  8082   │   │  8085   │   │  3000   │  │
    │  └────┬────┘    └────┬────┘   └────┬────┘   └────┬────┘  │
    │       │              │             │             │        │
    │       ↓              ↓             ↓             │        │
    │  ┌─────────────────────┐    ┌───────────┐       │        │
    │  │     OpenSearch      │    │ PostgreSQL│       │        │
    │  │        9200         │    │    5432   │       │        │
    │  └─────────────────────┘    └───────────┘       │        │
    │                                    ↑             │        │
    │                             ┌──────┴─────┐      │        │
    │                             │authenticate│←─────┘        │
    │                             │    8080    │               │
    │                             └────────────┘               │
    └──────────────────────────────────────────────────────────┘
                         Internal Network
```

## Services

| Service | Purpose | Port | Storage |
|---------|---------|------|---------|
| `authenticate` | Identity, sessions, JWT/HEC tokens | 8080 | PostgreSQL |
| `ingest` | Event ingestion + OpenSearch writes | 8088 | OpenSearch (write) |
| `search` | Ad-hoc queries + correlation | 8082 | OpenSearch (read) |
| `respond` | Detection rules, alerts, cases | 8085 | PostgreSQL |
| `web` | Frontend UI + API gateway | 3000 | Stateless |

## Service Details

### authenticate (Authentication & RBAC)
- User authentication (login, logout, sessions)
- JWT token generation and validation
- HEC token management
- Role-based access control (admin, analyst, viewer, ingester)
- Audit logging (auth events → ingest)

**Key Files**: `authenticate/internal/repository/postgres.go`, `authenticate/pkg/tokens/jwt.go`

### ingest (Event Ingestion + Storage)
- Receives raw events via HEC endpoint `/services/collector/event`
- Validates HEC token via authenticate service (with 5-min caching)
- IP-based and token-based rate limiting (Redis-backed)
- Normalizes events to OCSF format (77 auto-generated normalizers)
- Writes directly to OpenSearch (bulk indexing)
- Dead Letter Queue for failed events (`/var/lib/telhawk/dlq`)

**Key Files**: `ingest/internal/handlers/hec_handler.go`, `ingest/internal/storage/opensearch.go`

### search (Query API + Correlation)
- Execute ad-hoc searches (user-initiated)
- Execute correlation scans (scheduled, triggered by respond)
- Saved searches management
- Aggregate results, time-windowed analysis

**Key Files**: `search/internal/service/search.go`, `search/internal/translator/opensearch.go`

### respond (Detection Rules + Alerting + Cases)
- Detection rule storage (CRUD, versioning, lifecycle)
- Alert management (create, acknowledge, close)
- Case management (triage, investigation, resolution)
- Correlation scheduling (triggers search service)

**Key Files**: `respond/internal/handlers/handlers.go`, `respond/internal/repository/postgres.go`

### web (Frontend UI + API Gateway)
- Serve frontend UI (React app)
- API gateway / reverse proxy to backend services
- Async query orchestration
- Session management (cookies)

**Key Files**: `web/backend/internal/server/router.go`

## Event Pipeline (Ingest → OpenSearch)

1. **HEC Endpoint** receives raw events
2. **Token Validation** via authenticate service (cached 5 min)
3. **Rate Limiting** via Redis (IP-based and token-based)
4. **OCSF Normalization**: Registry matches events to normalizers
   - 77 auto-generated normalizers in `ingest/internal/normalizer/generated/`
   - HECNormalizer as fallback
5. **Validation Chain** ensures OCSF compliance
6. **OpenSearch Write** (bulk indexing)
   - Index pattern: `telhawk-events-YYYY.MM.DD`

## Authentication Flow

1. User login (`POST /api/v1/auth/login`) → JWT access token + refresh token
2. Access token used in `Authorization: Bearer <token>` header
3. Token validation endpoint (`POST /api/v1/auth/validate`) called by other services
4. Refresh tokens stored in PostgreSQL sessions table
5. HEC tokens stored separately with user association

## Configuration

Environment variable naming: `<SERVICE>_<SECTION>_<KEY>`
- `AUTHENTICATE_SERVER_PORT=8080`
- `AUTHENTICATE_AUTH_JWT_SECRET=secret`
- `INGEST_AUTHENTICATE_URL=http://authenticate:8080`
- `SEARCH_OPENSEARCH_PASSWORD=secret`
- `RESPOND_DATABASE_POSTGRES_HOST=auth-db`

## Database Architecture

### PostgreSQL
- `authenticate`: users, sessions, hec_tokens, audit_log
- `respond`: detection_schemas (rules), alerts, cases

### OpenSearch
- Daily indices: `telhawk-events-YYYY.MM.DD`
- OCSF-optimized mappings
- Query pattern: `telhawk-events-*`

## V1 → V2 Migration Reference

| V1 Service | V2 Service | Notes |
|------------|------------|-------|
| `auth` | `authenticate` | Renamed |
| `ingest` | `ingest` | Now includes storage |
| `storage` | _(merged)_ | Merged into ingest |
| `core` | _(merged)_ | Merged into ingest |
| `query` | `search` | Renamed |
| `rules` | `respond` | Merged with alerting |
| `alerting` | `respond` | Merged with rules |
| `web` | `web` | Unchanged |
