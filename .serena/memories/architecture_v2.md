# TelHawk Stack - Architecture V2 (Proposed)

## Service Consolidation

### Current → Proposed
```
Current (7 services)          Proposed (5 services)
─────────────────────         ─────────────────────
ingest                   →    ingest (+ storage merged)
storage                  →    (merged into ingest)
query                    →    search (+ correlation engine)
auth                     →    authenticate (renamed)
rules                    →    respond (+ alerting merged)
alerting                 →    (merged into respond)
web                      →    web (unchanged)
```

## Proposed Architecture Diagram

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
```

## Service Responsibilities

| Service | Purpose | Storage | Auth Model |
|---------|---------|---------|------------|
| `ingest` | Event ingestion + OpenSearch writes | OpenSearch (write) | HEC token validation |
| `search` | Ad-hoc queries + correlation evaluation | OpenSearch (read) | User/service auth |
| `authenticate` | Identity, sessions, tokens | PostgreSQL | N/A (is the auth) |
| `respond` | Cases, alerts, rules, workflows | PostgreSQL | User auth |
| `web` | UI, API gateway, async query orchestration | Stateless | Session cookies |

## Message Broker (NATS)

### Why NATS?
- Sub-millisecond latency (~200μs)
- Go-native with excellent client
- Lightweight (~20MB memory)
- JetStream for persistence
- Single-datacenter scale: 10M+ msg/sec

### Message Subjects
```
search.jobs.query           # Ad-hoc search requests
search.jobs.correlate       # Correlation evaluation requests
search.results.query.{id}   # Ad-hoc query results
search.results.correlate    # Correlation matches
respond.alerts.created      # New alert notifications
respond.alerts.updated      # Alert status changes
respond.cases.*             # Case lifecycle events
```

### Broker Abstraction Interfaces
```go
type EventPublisher interface {
    Publish(ctx context.Context, subject string, data []byte) error
    Close() error
}

type EventSubscriber interface {
    Subscribe(subject string, handler MessageHandler) (Subscription, error)
    QueueSubscribe(subject, queue string, handler MessageHandler) (Subscription, error)
    Close() error
}
```

## Authentication Boundaries

```
EXTERNAL (Auth Required)           INTERNAL (Trusted Network)
────────────────────────           ────────────────────────────
User → web (session/JWT)           web → search (no auth)
Agent → ingest (HEC token)         respond → search via NATS (no auth)
API client → services (JWT)        * → NATS (network trust)
                                   * → OpenSearch (network trust)
                                   * → PostgreSQL (credentials)
```

**Principle**: Auth at the edge only. Internal traffic is trusted.

## Storage Boundaries

### OpenSearch (Time-Series)
- Events: `telhawk-events-YYYY.MM.DD`
- Owned by: `ingest` (write), `search` (read)

### PostgreSQL (Relational)
- Users, sessions, tokens → `authenticate`
- Rules, alerts, cases → `respond`

### NATS JetStream (Messages)
- Job queues with acks
- Temporary storage (hours/days)

## Service Naming Convention

V2 uses **verbs** for service names:
- `ingest` - ingests events
- `search` - searches events
- `authenticate` - authenticates users
- `respond` - responds to alerts/incidents
- `web` - serves web UI

## Key Design Decisions

1. **Why merge ingest + storage?**
   - Storage is stateless OpenSearch client
   - Always used together (tight coupling)
   - Simplifies retry/backpressure logic

2. **Why merge rules + alerting → respond?**
   - Both use PostgreSQL (same ops concerns)
   - Same domain (detection → response)
   - Natural relationships (rules → alerts → cases)

3. **Why add NATS?**
   - Async queries (long-running searches)
   - Correlation job scheduling
   - Real-time notifications to web
   - Decouples producers from consumers

4. **Why query becomes search?**
   - Better verb form
   - Expanded scope (ad-hoc + correlation)
   - Owns all OpenSearch reads

## Reference
Full details: `docs/ARCHITECTURE_V2.md`
