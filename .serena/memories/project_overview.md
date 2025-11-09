# TelHawk Stack - Project Overview

## Purpose
TelHawk Stack is a lightweight, OCSF-compliant SIEM (Security Information and Event Management) platform built in Go. It provides Splunk-compatible event collection with OpenSearch as the backend storage engine.

## Technology Stack
- **Primary Language**: Go 1.24.2
- **Storage**: OpenSearch (primary datastore), PostgreSQL (auth service)
- **Caching/Rate Limiting**: Redis
- **Frontend**: React (web UI)
- **Containerization**: Docker, Docker Compose
- **Configuration**: Viper (YAML + environment variables)
- **CLI Framework**: Cobra
- **Database Migrations**: golang-migrate

## Architecture Type
Microservices architecture where events flow through multiple services:
**Ingestion → Normalization → Storage → Query/Web**

## Key Services (6 main + CLI)
1. **auth** (port 8080): JWT authentication, user management, HEC token management, RBAC
2. **ingest** (port 8088): Splunk HEC-compatible ingestion endpoint
3. **core** (port 8090): OCSF normalization engine (77 event classes)
4. **storage** (port 8083): OpenSearch abstraction layer
5. **query** (port 8082): Query API with SPL-subset support
6. **web** (port 3000): React-based search console and event viewer
7. **cli** (thawk): Command-line tool for auth, token management, ingestion, search

## Supporting Services
- **opensearch** (9200, 9600): Primary datastore with TLS
- **auth-db** (PostgreSQL): User/session/token storage
- **redis** (6379): Rate limiting and caching

## Key Features
- Splunk HEC-compatible ingestion
- OCSF 1.1.0 compliance (77 event classes)
- Code-generated normalizers
- Dead Letter Queue for failed events
- JWT-based authentication with RBAC
- Rate limiting (IP-based and token-based)
- TLS/mTLS support for service communication
- Docker-based deployment

## Repository Structure
```
telhawk-stack/
├── auth/           # Authentication service
├── ingest/         # HEC ingestion service
├── core/           # OCSF normalization engine
├── storage/        # OpenSearch storage layer
├── query/          # Query API service
├── web/            # React frontend + Go backend
├── cli/            # Command-line tool (thawk)
├── common/         # Shared Go code
├── tools/          # Code generators and utilities
├── docs/           # Documentation
├── certs/          # Certificate generation
├── opensearch/     # OpenSearch configuration
└── auth-db/        # PostgreSQL setup
```