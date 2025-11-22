# Services

Purpose: Orient engineers to each microservice and how they fit together.

## Architecture (High-Level)

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
    │  └────┬────┘    └────┬────┘   └────┬────┘   └────┬────┘  │
    │       │              │             │             │        │
    │       ↓              ↓             ↓             │        │
    │  ┌─────────────────────┐    ┌───────────┐       │        │
    │  │     OpenSearch      │    │ PostgreSQL│       │        │
    │  └─────────────────────┘    └───────────┘       │        │
    │                                    ↑             │        │
    │                             ┌──────┴─────┐      │        │
    │                             │authenticate│←─────┘        │
    │                             └────────────┘               │
    └──────────────────────────────────────────────────────────┘
                         Internal Network
```

## Service Summary

| Service | Purpose | Port | Storage |
|---------|---------|------|---------|
| `authenticate` | Identity, sessions, tokens | 8080 | PostgreSQL |
| `ingest` | Event ingestion + OpenSearch writes | 8088 | OpenSearch (write) |
| `search` | Ad-hoc queries + correlation evaluation | 8082 | OpenSearch (read) |
| `respond` | Cases, alerts, rules, workflows | 8085 | PostgreSQL |
| `web` | UI, API gateway | 3000 | Stateless |

---

## authenticate (Authentication & RBAC)

**Purpose**: Central authentication and authorization for all services; token and session management.

**Responsibilities**:
- User authentication (login, logout, sessions)
- JWT token generation and validation
- HEC token management
- Role-based access control (admin, analyst, viewer, ingester)
- Audit logging (auth events → `ingest`)

**Key endpoints**:
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/validate` - Token validation (used by other services)
- `GET /api/v1/hec/tokens` - HEC token management

**Related docs**: [CONFIGURATION.md](CONFIGURATION.md), [auth/AUTH_INTEGRATION.md](auth/AUTH_INTEGRATION.md)

---

## ingest (Event Ingestion + Storage)

**Purpose**: Receive events via HEC, normalize to OCSF, write directly to OpenSearch.

**Responsibilities**:
- Receive events via HEC endpoint (`/services/collector/event`)
- Validate HEC tokens via `authenticate` service
- Normalize events to OCSF format (77 auto-generated normalizers)
- Write directly to OpenSearch (bulk indexing)
- Manage Dead Letter Queue for failed events (`/var/lib/telhawk/dlq`)
- Rate limiting (IP-based and token-based)

**Key endpoints**:
- `POST /services/collector/event` - HEC event ingestion
- `POST /services/collector/raw` - Raw event ingestion
- `GET /healthz` - Health check

**Notes**: Writes normalized events directly to OpenSearch.

**Related docs**: [ingest/SPLUNK_COMPATIBILITY.md](ingest/SPLUNK_COMPATIBILITY.md), [ocsf/OCSF_COVERAGE.md](ocsf/OCSF_COVERAGE.md), [ocsf/NORMALIZER_GENERATION.md](ocsf/NORMALIZER_GENERATION.md), [PROMETHEUS_METRICS.md](PROMETHEUS_METRICS.md)

---

## search (Query API + Correlation)

**Purpose**: Execute ad-hoc searches and correlation evaluations against OpenSearch.

**Responsibilities**:
- Execute ad-hoc searches (user-initiated)
- Execute correlation scans (scheduled, triggered by `respond`)
- Aggregate results, time-windowed analysis
- Saved searches management
- Publish results to NATS for async consumers

**Key endpoints**:
- `POST /api/v1/search` - Execute searches
- `GET /api/v1/saved-searches` - Saved search management
- `GET /api/v1/dashboards` - Dashboard definitions

**Notes**: Ad-hoc queries and correlation evaluation are fundamentally the same operation - querying OpenSearch with structured criteria.

**Related docs**: [search/QUERY_API.md](search/QUERY_API.md), [search/QUERY_LANGUAGE_DESIGN.md](search/QUERY_LANGUAGE_DESIGN.md), [search/SAVED_SEARCHES.md](search/SAVED_SEARCHES.md)

---

## respond (Detection Rules + Alerting + Cases)

**Purpose**: Manage detection rules, alerts, and case workflows.

**Responsibilities**:
- Detection rule storage (CRUD, versioning, lifecycle)
- Alert management (create, acknowledge, close)
- Case management (triage, investigation, resolution)
- Correlation scheduling (triggers `search` service)
- Notification/action dispatch

**Key endpoints**:
- `GET /api/v1/schemas` - Detection rule management
- `GET /api/v1/alerts` - Alert management
- `GET /api/v1/cases` - Case management

**Notes**: Rules define detections, alerting executes them - same domain, single service.

**Related docs**: [alerting/ALERTING_ARCHITECTURE.md](alerting/ALERTING_ARCHITECTURE.md), [alerting/ALERTING_API.md](alerting/ALERTING_API.md), [alerting/DETECTION_RULES_ROADMAP.md](alerting/DETECTION_RULES_ROADMAP.md)

---

## web (Frontend UI + API Gateway)

**Purpose**: Serve the React frontend and act as API gateway.

**Responsibilities**:
- Serve frontend UI (React app)
- API gateway / reverse proxy to backend services
- Async query orchestration (submit → poll → results)
- Session management (cookies)

**Key features**:
- Search console with query builder
- Event table with OCSF field inspection
- Alert management dashboard
- Detection rule editor
- User administration

**Related docs**: `docs/UX_DESIGN_PHILOSOPHY.md`

---

## Supporting Infrastructure

### OpenSearch
- **Port**: 9200 (HTTPS)
- **Purpose**: Time-series event storage
- **Index pattern**: `telhawk-events-YYYY.MM.DD`
- **Owned by**: `ingest` (write), `search` (read)

### PostgreSQL
- **Port**: 5432
- **Purpose**: Relational data (users, rules, alerts, cases)
- **Databases**: `telhawk_auth` (authenticate), `telhawk_respond` (respond)

### NATS
- **Port**: 4222
- **Purpose**: Async message broker for service communication
- **Key subjects**: `search.jobs.*`, `search.results.*`, `respond.alerts.*`

### Redis
- **Port**: 6379
- **Purpose**: Rate limiting, caching

---

## Cross-Cutting Concerns

- [Configuration](CONFIGURATION.md)
- [Production Deployment](PRODUCTION.md)
- [Prometheus Metrics](PROMETHEUS_METRICS.md)
- [Splunk Compatibility](ingest/SPLUNK_COMPATIBILITY.md)
- [Helper Scripts](HELPER_SCRIPTS.md)
- [TLS Configuration](TLS_CONFIGURATION.md)

