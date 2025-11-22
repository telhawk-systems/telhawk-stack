# TelHawk Stack - Project Overview

## Purpose
TelHawk Stack is a lightweight, OCSF-compliant SIEM (Security Information and Event Management) platform built in Go. It provides Splunk-compatible event collection with OpenSearch as the backend storage engine.

## Technology Stack
- **Primary Language**: Go 1.24.2
- **Storage**: OpenSearch (events), PostgreSQL (users, rules, alerts, cases)
- **Message Broker**: NATS (async communication)
- **Caching/Rate Limiting**: Redis
- **Frontend**: React (web UI)
- **Containerization**: Docker, Docker Compose
- **Configuration**: Viper (YAML + environment variables)
- **CLI Framework**: Cobra
- **Database Migrations**: golang-migrate

## Architecture Type
Microservices architecture (V2 - 5 services):
**Ingestion → OpenSearch ← Search**
**Respond (rules/alerts/cases) ↔ Search**
**Authenticate → All services**

## Services (5 main + CLI)

| Service | Port | Purpose |
|---------|------|---------|
| `authenticate` | 8080 | JWT auth, user management, HEC tokens, RBAC |
| `ingest` | 8088 | Splunk HEC-compatible ingestion + OpenSearch writes |
| `search` | 8082 | Query API, correlation evaluation |
| `respond` | 8085 | Detection rules, alerts, case management |
| `web` | 3000 | React-based UI and API gateway |
| `cli` (thawk) | - | Command-line tool |

## Supporting Infrastructure
- **OpenSearch** (9200): Primary event datastore with TLS
- **PostgreSQL** (5432): Users, rules, alerts, cases
- **NATS** (4222): Async message broker
- **Redis** (6379): Rate limiting and caching

## Key Features
- Splunk HEC-compatible ingestion
- OCSF 1.1.0 compliance (77 event classes)
- Code-generated normalizers
- Dead Letter Queue for failed events
- JWT-based authentication with RBAC
- Rate limiting (IP-based and token-based)
- Detection rules with immutable versioning
- Alert and case management
- TLS/mTLS support for service communication
- Docker-based deployment

## Repository Structure
```
telhawk-stack/
├── authenticate/   # Authentication service
├── ingest/         # HEC ingestion + OpenSearch storage
├── search/         # Query API service (formerly 'query')
├── respond/        # Detection rules, alerts, cases
├── web/            # React frontend + Go backend
├── cli/            # Command-line tool (thawk)
├── common/         # Shared Go code (OCSF types, messaging)
├── tools/          # Code generators and utilities
├── docs/           # Documentation
├── certs/          # Certificate generation
├── opensearch/     # OpenSearch configuration
└── auth-db/        # PostgreSQL setup
```

## V1 → V2 Migration

The V2 architecture consolidated 7 services into 5:

| V1 | V2 | Notes |
|----|----|----|
| auth | authenticate | Renamed |
| ingest | ingest | Now includes storage |
| storage | _(merged)_ | Merged into ingest |
| core | _(merged)_ | Merged into ingest |
| query | search | Renamed |
| rules | respond | Merged with alerting |
| alerting | respond | Merged with rules |
| web | web | Unchanged |

Old directories (auth/, rules/, alerting/, query/, storage/, core/) may still exist but are deprecated.
