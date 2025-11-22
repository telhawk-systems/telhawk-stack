# Architecture V2 - Service Consolidation

This document describes the proposed architecture evolution for TelHawk Stack. The goal is to consolidate services by domain responsibility, introduce asynchronous messaging, and clarify storage boundaries.

## Current vs Proposed Architecture

### Current (7 services)
```
ingest → storage → OpenSearch ← query
   ↓                              ↓
  auth                           web
   ↑
rules ← alerting
```

### Proposed (5 services)
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

## Service Responsibilities

| Service | Purpose | Storage | Auth Model |
|---------|---------|---------|------------|
| `ingest` | Event ingestion + OpenSearch writes | OpenSearch (write) | HEC token validation |
| `search` | Ad-hoc queries + correlation evaluation | OpenSearch (read) | User/service auth |
| `authenticate` | Identity, sessions, tokens | PostgreSQL | N/A (is the auth service) |
| `respond` | Cases, alerts, rules, workflows | PostgreSQL | User auth |
| `web` | UI, API gateway, async query orchestration | Stateless | Session cookies |

## Service Details

### ingest

**Consolidates**: Current `ingest` + `storage` services

**Responsibilities**:
- Receive events via HEC endpoint (`/services/collector/event`)
- Validate HEC tokens via `authenticate` service
- Normalize events to OCSF format (77 auto-generated normalizers)
- Write directly to OpenSearch (bulk indexing)
- Manage Dead Letter Queue for failed events
- Rate limiting (IP-based and token-based)

**Why merge ingest + storage?**
- Storage is stateless - just an OpenSearch client
- Ingest always forwards to storage (tight coupling)
- Single service simplifies retry/backpressure logic
- No scenario where you run one without the other

**Ports**: 8088 (HEC)

---

### search

**Consolidates**: Current `query` service + correlation engine from `alerting`

**Responsibilities**:
- Execute ad-hoc searches (user-initiated)
- Execute correlation scans (scheduled, triggered by `respond`)
- Aggregate results, time-windowed analysis
- Publish results to NATS for async consumers

**Key insight**: Both ad-hoc queries and correlation evaluation are fundamentally the same operation - querying OpenSearch with structured criteria. The difference is trigger (user vs schedule) and output (results vs alerts).

**Message patterns**:
```
Subscribe: search.jobs.query      (ad-hoc query requests)
Subscribe: search.jobs.correlate  (correlation evaluation requests)
Publish:   search.results.*       (query/correlation results)
```

**Ports**: 8082 (API)

---

### authenticate

**Same as**: Current `auth` service (renamed for verb consistency)

**Responsibilities**:
- User authentication (login, logout, sessions)
- JWT token generation and validation
- HEC token management
- Role-based access control
- Audit logging (auth events → `ingest`)

**Why separate?**
- Security boundary - isolated failure domain
- Different operational concerns (secrets, key rotation)
- Rarely changes once working
- Clear API contract used by all other services

**Ports**: 8080 (API)

---

### respond

**Consolidates**: Current `rules` + `alerting` services + case management

**Responsibilities**:
- Detection rule storage (CRUD, versioning, lifecycle)
- Alert management (create, acknowledge, close)
- Case management (triage, investigation, resolution)
- Correlation scheduling (triggers `search` service)
- Notification/action dispatch

**Why merge rules + alerting?**
- Both are relational data (PostgreSQL)
- Rules define detections, alerting executes them - same domain
- Cases link to alerts link to rules - natural relationships
- Same operational concerns (backups, migrations)

**Message patterns**:
```
Publish:   search.jobs.correlate  (schedule correlation jobs)
Subscribe: search.results.correlate (receive correlation results)
Publish:   respond.alerts.created (new alert notifications)
Publish:   respond.cases.*        (case lifecycle events)
```

**Ports**: 8085 (API)

---

### web

**Same as**: Current `web` service

**Responsibilities**:
- Serve frontend UI (React app)
- API gateway / reverse proxy
- Async query orchestration (submit → poll → results)
- Session management (cookies)

**Changes from current**:
- Queries become asynchronous (submit job, poll for status)
- WebSocket support for real-time updates
- Subscribes to NATS for push notifications

**Message patterns**:
```
Subscribe: search.results.{query_id}  (async query results)
Subscribe: respond.alerts.created     (real-time alert notifications)
```

**Ports**: 3000 (HTTP)

---

## Message Broker Architecture

### Why NATS?

| Requirement | NATS Capability |
|-------------|-----------------|
| Low latency | Sub-millisecond (~200μs) |
| Go-native | First-class Go client |
| Lightweight | ~20MB memory footprint |
| Persistence | JetStream for durable queues |
| Scale | 10M+ msg/sec single node |
| Ops simplicity | Single binary, minimal config |

### Alternative considered

- **Redis Streams**: Already have Redis, but weaker messaging semantics
- **Kafka**: Overkill for single-datacenter, high ops burden
- **RabbitMQ**: Mature but Erlang dependency, more complex

### Message Subjects

```
# Query jobs (web/respond → search)
search.jobs.query           # Ad-hoc search requests
search.jobs.correlate       # Correlation evaluation requests

# Query results (search → web/respond)
search.results.query.{id}   # Ad-hoc query results
search.results.correlate    # Correlation matches

# Alert lifecycle (respond → web)
respond.alerts.created      # New alert created
respond.alerts.updated      # Alert status changed

# Case lifecycle (respond → web)
respond.cases.created       # New case opened
respond.cases.updated       # Case status changed
respond.cases.assigned      # Case assigned to analyst
```

### Broker Abstraction

Services interact with NATS through interfaces, not directly:

```go
// EventPublisher publishes messages to a subject
type EventPublisher interface {
    Publish(ctx context.Context, subject string, data []byte) error
    Close() error
}

// EventSubscriber subscribes to messages
type EventSubscriber interface {
    Subscribe(subject string, handler MessageHandler) (Subscription, error)
    QueueSubscribe(subject, queue string, handler MessageHandler) (Subscription, error)
    Close() error
}

type MessageHandler func(msg *Message) error

type Message struct {
    Subject string
    Data    []byte
    Reply   string  // for request/reply pattern
}

type Subscription interface {
    Unsubscribe() error
}
```

This allows swapping NATS for another broker without changing service code.

---

## Authentication Boundaries

```
┌────────────────────────────────────────────────────────────────┐
│                     EXTERNAL (Auth Required)                   │
│                                                                │
│   User → web (session cookie / JWT)                            │
│   Agent → ingest (HEC token)                                   │
│   API client → any service (JWT Bearer token)                  │
│                                                                │
├────────────────────────────────────────────────────────────────┤
│                     INTERNAL (Trusted Network)                 │
│                                                                │
│   web → search (service-to-service, no auth)                   │
│   web → respond (service-to-service, no auth)                  │
│   respond → search (via NATS, no auth)                         │
│   search → respond (via NATS, no auth)                         │
│   * → NATS (network-level trust)                               │
│   * → OpenSearch (network-level trust)                         │
│   * → PostgreSQL (connection credentials)                       │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

**Principle**: Auth happens at the edge. Internal service-to-service communication is trusted. NATS does not authenticate messages - it's inside the trust boundary.

---

## Storage Boundaries

### OpenSearch (Time-Series / Events)
- Event data (`telhawk-events-YYYY.MM.DD`)
- High write volume, append-only
- Time-based retention
- Full-text search, aggregations
- **Owned by**: `ingest` (write), `search` (read)

### PostgreSQL (Relational / Config)
- Users, sessions, tokens (`authenticate`)
- Detection rules with versioning (`respond`)
- Alerts with lifecycle state (`respond`)
- Cases with relationships (`respond`)
- **Owned by**: `authenticate`, `respond`

### NATS JetStream (Message Persistence)
- Job queues with acknowledgment
- Replay capability for failed consumers
- Temporary storage (hours/days, not permanent)
- **Owned by**: All services (pub/sub)

---

## Data Flow Diagrams

### Event Ingestion

```
External Agent
     │
     ▼
┌─────────┐    validate token    ┌──────────────┐
│ ingest  │ ──────────────────▶  │ authenticate │
└────┬────┘                      └──────────────┘
     │
     │ normalize (OCSF)
     │ validate
     ▼
┌─────────────┐
│ OpenSearch  │
└─────────────┘
```

### Ad-hoc Query (Async)

```
User
  │
  ▼
┌─────┐  submit query   ┌────────┐  search.jobs.query   ┌────────┐
│ web │ ─────────────▶  │  NATS  │ ─────────────────▶   │ search │
└──┬──┘                 └────────┘                      └───┬────┘
   │                         │                              │
   │  poll status            │  search.results.query.{id}   │
   │◀────────────────────────┼──────────────────────────────┘
   │                         │
   ▼                         │
 Results                     │
                             ▼
                        OpenSearch
```

### Correlation / Alerting

```
┌─────────┐  schedule tick
│ respond │ ──────────────┐
└────┬────┘               │
     │                    ▼
     │           ┌────────────────┐
     │           │ search.jobs.   │
     │           │ correlate      │
     │           └───────┬────────┘
     │                   │
     │                   ▼
     │              ┌────────┐
     │              │ search │
     │              └───┬────┘
     │                  │
     │                  │ query OpenSearch
     │                  │ evaluate rules
     │                  ▼
     │           ┌────────────────┐
     │           │ search.results.│
     │           │ correlate      │
     │           └───────┬────────┘
     │                   │
     ◀───────────────────┘
     │
     │ create alerts
     │ update cases
     ▼
┌────────────┐
│ PostgreSQL │
└────────────┘
```

---

## Migration Path

### Phase 1: Consolidate Storage into Ingest
1. Move OpenSearch client code from `storage/` to `ingest/`
2. Remove `storage` service from docker-compose
3. Update ingest to write directly to OpenSearch

### Phase 2: Add NATS Infrastructure
1. Add NATS to docker-compose
2. Create `common/messaging/` with `EventPublisher`/`EventSubscriber` interfaces
3. Implement NATS adapter

### Phase 3: Consolidate Rules + Alerting → Respond
1. Create `respond/` service directory
2. Migrate rules storage from `rules/`
3. Migrate alert management from `alerting/`
4. Add case management tables
5. Integrate correlation scheduler with NATS

### Phase 4: Refactor Query → Search
1. Rename `query/` to `search/`
2. Add NATS job consumer
3. Move correlation evaluation logic from alerting
4. Implement async result publishing

### Phase 5: Update Web for Async
1. Add NATS subscriber for results
2. Implement query polling endpoint
3. Add WebSocket for real-time updates

### Phase 6: Rename Auth → Authenticate
1. Rename service directory
2. Update all service references
3. Update docker-compose and configs

---

## Service Naming Conventions

| Pattern | Examples | When to use |
|---------|----------|-------------|
| Verb | `ingest`, `search`, `authenticate`, `respond` | Services that perform actions |
| Plural noun | `rules`, `alerts`, `cases` | Data-centric services (discouraged in V2) |
| Gerund | `alerting`, `ingesting` | Avoid - awkward naming |

V2 standardizes on **verbs** for service names since services are processes that do things.

---

## Open Questions

1. **Correlation scheduling**: Should `respond` own the scheduler, or should there be a lightweight `schedule` service?

2. **WebSocket gateway**: Should `web` handle WebSocket connections directly, or delegate to a dedicated service?

3. **Multi-tenancy**: If we add tenants later, which services need tenant-awareness? (Likely: `ingest`, `search`, `respond`)

4. **Metrics aggregation**: Where do Prometheus metrics get scraped from? Each service, or a central collector?
