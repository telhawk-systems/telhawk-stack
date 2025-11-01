# TelHawk Stack

A lightweight, OCSF-compliant SIEM (Security Information and Event Management) platform built in Go. TelHawk Stack provides Splunk-compatible event collection with OpenSearch as the backend storage engine, designed for security engineers who need enterprise-grade security monitoring without enterprise licensing costs.

## Overview

TelHawk Stack is a monorepo containing multiple Go services that work together to provide a complete SIEM solution:

- **Event ingestion** with Splunk HEC compatibility
- **OCSF normalization** for standardized security data
- **OpenSearch storage** for cost-effective log retention and search
- **Query API** for programmatic data access
- **Web interface** for security operations

## Architecture

```
┌─────────────┐
│   Sources   │  (TelHawk Proxy, Syslog, Splunk HEC clients, etc.)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   ingest/   │  HTTP Event Collector (HEC) compatible ingestion server
└──────┬──────┘
       │
       ▼
┌─────────────┐
│    core/    │  OCSF normalization, event processing, routing
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  storage/   │  OpenSearch client, indexing, retention management
└──────┬──────┘
       │
       ▼
┌─────────────┐
│OpenSearch DB│
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   query/    │  Query API service (REST/GraphQL)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│    web/     │  Web UI for dashboards, search, and alerting
└─────────────┘
```

## Projects

### `/auth` - Authentication & Authorization Service
**Purpose:** Centralized authentication and RBAC for all TelHawk Stack services

**Responsibilities:**
- User registration and authentication
- JWT-based access and refresh token management
- HEC token generation and validation for ingestion
- Role-based access control (RBAC)
- Session management and revocation

**Key Features:**
- JWT authentication with bcrypt password hashing
- Multiple user roles (admin, analyst, viewer, ingester)
- HEC token management for Splunk-compatible ingestion
- Token validation API for other services
- In-memory storage (development) with PostgreSQL planned

**API Endpoints:**
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/validate` - Validate token (used by other services)
- `POST /api/v1/auth/revoke` - Revoke refresh token

### `/core` - Core SIEM Engine
**Purpose:** Central event processing and OCSF normalization engine

**Responsibilities:**
- OCSF (Open Cybersecurity Schema Framework) event schema definitions
- Event normalization and enrichment
- Event routing and filtering logic
- Common business logic shared across services

**Key Features:**
- Converts raw events to OCSF-compliant format
- Field mapping and data transformation
- Event classification and categorization
- Enrichment with threat intelligence, GeoIP, etc.

### `/ingest` - Event Ingestion Service
**Purpose:** Multi-protocol event ingestion server with Splunk HEC compatibility

**Responsibilities:**
- Splunk HTTP Event Collector (HEC) API endpoint
- Syslog (RFC 5424) ingestion
- JSON/NDJSON batch ingestion
- Input validation and authentication
- Rate limiting and backpressure management

**API Compatibility:**
- `/services/collector/event` - Splunk HEC event endpoint
- `/services/collector/raw` - Splunk HEC raw endpoint
- `/services/collector/health` - Health check endpoint

**Key Features:**
- Token-based authentication (HEC token compatible)
- TLS/mTLS support
- High-throughput ingestion (designed for 10k+ events/sec)
- Graceful degradation under load

### `/storage` - Storage Abstraction Layer
**Purpose:** OpenSearch client and storage management

**Responsibilities:**
- OpenSearch index lifecycle management
- Bulk indexing operations
- Index templates and mappings
- Data retention and rollover policies
- Query optimization

**Key Features:**
- Automatic index creation with OCSF-optimized mappings
- Hot/warm/cold data tiering
- Configurable retention policies
- Compression and optimization
- Snapshot/restore support

### `/query` - Query API Service
**Purpose:** Programmatic access to security data

**Responsibilities:**
- RESTful query API
- SPL (Search Processing Language) subset support
- Saved searches and alerts
- Data aggregation and analytics
- Export capabilities (CSV, JSON)

**API Endpoints:**
- `POST /api/v1/search` - Execute searches
- `GET /api/v1/alerts` - Manage alerts
- `GET /api/v1/dashboards` - Dashboard definitions
- `POST /api/v1/export` - Data export

**Key Features:**
- SPL-compatible query syntax (subset)
- Time-based queries and aggregations
- Field-based filtering
- Real-time and historical search
- Query caching

### `/web` - Web Interface
**Purpose:** Frontend for security analysts and SOC teams

**Responsibilities:**
- Search interface with SPL support
- Security dashboards and visualizations
- Alert management UI
- User management and RBAC
- Investigation workspace

**Key Features:**
- Real-time event streaming
- Customizable dashboards
- Saved searches and reports
- Dark mode (SOC-friendly)
- Mobile-responsive design

### `/common` - Shared Libraries
**Purpose:** Common utilities and libraries used across services

**Responsibilities:**
- Configuration management
- Logging and observability
- Authentication/authorization middleware
- OpenTelemetry instrumentation
- Shared data structures

**Includes:**
- HTTP server utilities
- Database connection pooling
- Metrics and tracing
- Error handling patterns
- Validation helpers

### `/cli` - Command-Line Interface (`thawk`)
**Purpose:** Terminal interface for TelHawk Stack operations

**Responsibilities:**
- Authentication and session management
- HEC token creation and management
- Event ingestion from command line
- Search query execution
- Alert and dashboard management

**Key Features:**
- Cobra-based CLI with subcommands
- Multiple profile support (~/.thawk/config.yaml)
- Color-coded output for SOC readability
- JSON/YAML/Table output formats
- Shell completion (bash, zsh, fish)

**Commands:**
- `thawk auth login` - Authenticate with TelHawk Stack
- `thawk token create` - Create HEC tokens
- `thawk ingest send` - Send events
- `thawk search` - Execute SPL queries
- `thawk alert list` - Manage alerts

## Getting Started

### Prerequisites
- Go 1.21+
- Docker and Docker Compose (optional, for OpenSearch)

### Quick Start (5 Minutes)

#### 1. Build Everything
```bash
cd telhawk-stack

# Build auth service
cd auth && go build -o ../bin/auth ./cmd/auth && cd ..

# Build ingest service
cd ingest && go build -o ../bin/ingest ./cmd/ingest && cd ..

# Build CLI tool
cd cli && go build -o ../bin/thawk . && cd ..

# Add to PATH (optional)
export PATH=$PATH:$(pwd)/bin
```

#### 2. Start Services
```bash
# Terminal 1: Start auth service (port 8080)
./bin/auth

# Terminal 2: Start ingest service (port 8088)
./bin/ingest
```

#### 3. Create Your First User
```bash
# Register admin user
thawk auth login -u admin -p SecurePassword123 --auth-url http://localhost:8080
# If user doesn't exist, it will be created automatically
# (Note: In production, add proper registration endpoint)

# Verify login
thawk auth whoami
```

#### 4. Create HEC Token for Ingestion
```bash
# Create token for data ingestion
thawk token create --name my-first-token

# Output will show:
# ✓ HEC token created: abc123xyz...
# Use this token with:
#   curl -H 'Authorization: Splunk abc123xyz...' ...
```

#### 5. Send Your First Event
```bash
# Using thawk CLI
thawk ingest send \
  --message "User login successful" \
  --token <your-hec-token-from-step-4> \
  --source application \
  --sourcetype auth_log

# Or using curl
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Splunk <your-hec-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "event": {
      "message": "Security alert detected",
      "severity": "high",
      "user": "analyst1"
    },
    "source": "security_app",
    "sourcetype": "json"
  }'
```

#### 6. Check Ingestion Stats
```bash
curl http://localhost:8088/readyz

# Response:
{
  "status": "ready",
  "stats": {
    "total_events": 1,
    "total_bytes": 234,
    "successful_events": 1,
    "failed_events": 0,
    "last_event": "2024-11-01T12:34:56Z"
  }
}
```

### Full Stack with OpenSearch (Optional)

For persistent storage, add OpenSearch:

```bash
# Start OpenSearch
docker run -d \
  --name opensearch \
  -p 9200:9200 \
  -e "discovery.type=single-node" \
  -e "OPENSEARCH_INITIAL_ADMIN_PASSWORD=Admin123!" \
  opensearchproject/opensearch:2

# Verify OpenSearch is running
curl -u admin:Admin123! http://localhost:9200
```

Then configure services to use OpenSearch (coming soon in storage service).

## OCSF Compliance

TelHawk Stack implements the [Open Cybersecurity Schema Framework (OCSF)](https://schema.ocsf.io/) for event normalization. All ingested events are converted to OCSF format before storage, ensuring:

- **Standardized security data** across all sources
- **Interoperability** with other OCSF-compliant tools
- **Future-proof** schema evolution
- **Rich security context** with standardized fields

### Supported OCSF Classes
- Network Activity
- Authentication
- System Activity
- Application Activity
- Detection Finding
- Security Finding

## Splunk Compatibility

The ingestion service implements the Splunk HTTP Event Collector (HEC) protocol, allowing you to:

- Use existing Splunk forwarders and clients
- Migrate from Splunk with minimal reconfiguration
- Test with Splunk Universal Forwarder
- Leverage Splunk client libraries

### HEC Features
- ✅ Event endpoint (`/services/collector/event`)
- ✅ Raw endpoint (`/services/collector/raw`)
- ✅ Token authentication
- ✅ Source/sourcetype/host metadata
- ✅ Indexed field extraction
- ⚠️ Ack support (planned)

## Development

### Project Structure
```
telhawk-stack/
├── core/           # OCSF normalization engine
├── ingest/         # Event ingestion service
├── query/          # Query API service
├── storage/        # OpenSearch storage layer
├── web/            # Web UI
├── common/         # Shared libraries
├── config/         # Configuration files
├── docker/         # Docker and compose files
├── docs/           # Documentation
└── scripts/        # Build and deployment scripts
```

### Building Individual Services
Each service can be built independently:

```bash
cd ingest
go build -o ../bin/ingest ./cmd/ingest

cd ../query
go build -o ../bin/query ./cmd/query
```

### Running Tests
```bash
# Test all modules
go test ./...

# Test specific module
cd core && go test ./...
```

## Configuration

Each service has its own configuration file in `config/`:

- `config/ingest.yaml` - Ingestion service settings
- `config/core.yaml` - Core engine configuration
- `config/storage.yaml` - OpenSearch connection settings
- `config/query.yaml` - Query API settings
- `config/web.yaml` - Web UI configuration

## Deployment

### Docker Compose (Development)
```bash
docker-compose up -d
```

### Kubernetes (Production)
```bash
kubectl apply -f k8s/
```

## Roadmap

- [x] Project structure and modules
- [x] Auth service with JWT and RBAC
- [x] CLI tool (`thawk`) with Cobra
- [x] Nonrepudiation strategy with HMAC signatures
- [x] Ingestion service with HEC support
- [ ] Core OCSF schema implementation
- [ ] OpenSearch storage layer
- [ ] Basic query API
- [ ] Web UI prototype
- [ ] SPL query support
- [ ] Alert engine
- [ ] Threat intelligence enrichment
- [ ] Machine learning anomaly detection

## License

MIT License - see LICENSE file for details

## Contributing

Contributions welcome! Please read CONTRIBUTING.md for guidelines.

## Related Projects

- [telhawk-proxy](https://github.com/telhawk-systems/telhawk-proxy) - Telemetry collection proxy (natural data source for TelHawk Stack)

## Contact

For questions or support, open an issue on GitHub.
