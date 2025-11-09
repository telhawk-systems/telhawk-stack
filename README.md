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
- Normalization pipeline documentation: see `docs/core-pipeline.md`

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
- Docker and Docker Compose

**Note:** You don't need Go installed - all services are built and run in Docker containers with OpenSearch included by default.

### Quick Start (2 Minutes)

#### 1. Start the Stack
```bash
cd telhawk-stack

# Start all services (auth, ingest, opensearch)
docker-compose up -d

# Watch logs
docker-compose logs -f
```

#### 2. Wait for Services to be Ready
```bash
# Check service health
docker-compose ps

# All services should show "healthy" status
```

#### 3. Create Your First User
```bash
# Using the CLI tool via Docker
docker-compose run --rm thawk auth login -u admin -p SecurePassword123
# If user doesn't exist, it will be created automatically

# Verify login
docker-compose run --rm thawk auth whoami
```

**Optional:** For convenience, create an alias:
```bash
alias thawk='docker-compose run --rm thawk'
thawk auth whoami  # Now you can use it like a local command
```

#### 4. Create HEC Token for Ingestion
```bash
# Create token for data ingestion
docker-compose run --rm thawk token create --name my-first-token

# Output will show:
# ✓ HEC token created: abc123xyz...
# Use this token with:
#   curl -H 'Authorization: Telhawk abc123xyz...' ...
```

#### 5. Send Your First Event

> **Note:** The Authorization header accepts either `Splunk` or `Telhawk` before the token for compatibility.

```bash
# Using thawk CLI
docker-compose run --rm thawk ingest send \
  --message "User login successful" \
  --token <your-hec-token-from-step-4> \
  --source application \
  --sourcetype auth_log

# Or using curl directly
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Telhawk <your-hec-token>" \
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

#### 6. Check Service Health
```bash
# Check ingestion service
curl http://localhost:8088/readyz

# Check OpenSearch cluster
curl -u admin:TelHawk123! http://localhost:9200/_cluster/health

# View all container status
docker-compose ps
```

#### 7. Stop the Stack
```bash
# Stop all services
docker-compose down

# Stop and remove volumes (deletes all data)
docker-compose down -v
```

## Documentation

Comprehensive guides for configuration, deployment, and usage:

- **[Configuration Guide](docs/CONFIGURATION.md)** - Complete service configuration reference (YAML + environment variables)
- **[TLS Configuration](docs/TLS_CONFIGURATION.md)** - Enable HTTPS/TLS with self-signed or production certificates
- **[CLI Configuration](docs/CLI_CONFIGURATION.md)** - TelHawk CLI (`thawk`) configuration and usage
- **[Docker Quick Reference](DOCKER.md)** - Docker and docker-compose commands, troubleshooting

### Additional Resources

- **OCSF Compliance** - See section below for OCSF schema implementation details
- **[UX Design Philosophy](docs/UX_DESIGN_PHILOSOPHY.md)** - Why TelHawk's interface works like a CRM, not a traditional SIEM. If you've used any modern CRM or ticketing system, you'll know how to triage events - no SIEM-specific training required.
- **API Documentation** - Coming soon
- **Web UI Guide** - Coming soon

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
├── auth/           # Authentication service (+ Dockerfile)
├── cli/            # CLI tool (thawk) (+ Dockerfile)
├── core/           # OCSF normalization engine
├── ingest/         # Event ingestion service (+ Dockerfile)
├── query/          # Query API service
├── storage/        # OpenSearch storage layer
├── web/            # Web UI
├── common/         # Shared libraries
├── docs/           # Documentation
├── docker-compose.yml  # Complete stack orchestration
└── bin/            # Compiled binaries (local dev only)
```

### Local Development (Without Docker)

If you want to develop locally with Go installed:

```bash
# Build individual services
cd auth && go build -o ../bin/auth ./cmd/auth && cd ..
cd ingest && go build -o ../bin/ingest ./cmd/ingest && cd ..
cd cli && go build -o ../bin/thawk . && cd ..

# Run locally (requires separate OpenSearch instance)
./bin/auth &
./bin/ingest &
```

### Running Tests
```bash
# Test all modules
go test ./...

# Test specific module
cd core && go test ./...
```

### Rebuilding Docker Images
```bash
# Rebuild all services
docker-compose build

# Rebuild specific service
docker-compose build auth

# Rebuild and restart
docker-compose up -d --build
```

## Configuration

TelHawk Stack uses **enterprise-grade configuration management**:

- **YAML configuration files** for defaults
- **Environment variable overrides** for deployment-specific settings
- **No CLI arguments** for configuration (12-factor compliant)

### Quick Configuration

Each service has a `config.yaml` file embedded in its Docker image at `/etc/telhawk/<service>/config.yaml`.

Override any setting via environment variables:

```bash
# Auth service
AUTH_SERVER_PORT=8080
AUTH_AUTH_JWT_SECRET="your-secret-key"

# Ingest service  
INGEST_SERVER_PORT=8088
INGEST_AUTH_URL=http://auth:8080
INGEST_OPENSEARCH_URL=https://opensearch:9200
INGEST_OPENSEARCH_PASSWORD="YourPassword"
```

See [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md) for complete configuration guide.

### Docker Compose Configuration

The `docker-compose.yml` demonstrates proper configuration:

```yaml
services:
  auth:
    environment:
      - AUTH_SERVER_PORT=8080
      - AUTH_AUTH_JWT_SECRET=${AUTH_JWT_SECRET:-default-secret}
  
  ingest:
    environment:
      - INGEST_OPENSEARCH_PASSWORD=${OPENSEARCH_PASSWORD:-TelHawk123!}
```

Create a `.env` file for local overrides:

```bash
# .env
AUTH_JWT_SECRET=my-local-dev-secret
OPENSEARCH_PASSWORD=DevPassword123!
```

## Deployment

### Docker Compose (Development & Production)
```bash
# Start the full stack
docker-compose up -d

# View logs
docker-compose logs -f

# Scale specific services
docker-compose up -d --scale ingest=3

# Update services
docker-compose pull
docker-compose up -d
```

### Production Considerations
- Change default OpenSearch password in docker-compose.yml
- **Enable TLS/SSL for all services** - See [TLS Configuration Guide](docs/TLS_CONFIGURATION.md)
- Configure proper resource limits
- Set up external volumes for data persistence
- Implement backup strategy for OpenSearch data
- Use secrets management for sensitive credentials

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

See [TODO.md](TODO.md) for the detailed backlog.

## License

TelHawk Stack is licensed under the **TelHawk Systems Source Available License (TSSAL) v1.0**.

**What this means:**
- ✅ **Free to use** - Run TelHawk in your own environment at no cost
- ✅ **Source available** - View and study the code
- ✅ **Modify for yourself** - Customize for your internal needs
- ❌ **No distribution** - Cannot distribute binaries or create public forks
- ❌ **No commercial hosting** - Cannot offer as a service to others

For commercial licensing, distribution rights, or hosting as a service, contact: **licensing@telhawk.com**

See the [LICENSE](LICENSE) file for full terms.

## Contributing

**Note:** This project does not accept external contributions or pull requests from forks. Development is handled internally by the TelHawk team.

For bug reports and feature requests, please open an issue on GitHub.

### Running Tests and CI Checks

Before committing changes, run the pre-push checks locally:

```bash
./scripts/pre-push.sh
```

This script runs the same checks as CI:
- Code formatting (gofmt)
- Go module tidiness
- Static analysis (go vet)
- Unit tests with race detector
- Linting (golangci-lint, if installed)

For detailed information about CI and local development checks, see [docs/CI_DEVELOPMENT.md](docs/CI_DEVELOPMENT.md).

## Commercial Support & Services

TelHawk Systems offers:
- **Enterprise Support** - SLA-backed support and maintenance
- **Managed Hosting** - Fully managed SIEM in your cloud or ours
- **Custom Development** - Integrations and custom features
- **Training & Consulting** - Security operations best practices

Contact: **sales@telhawk.com**

## Related Projects

- [telhawk-proxy](https://github.com/telhawk-systems/telhawk-proxy) - Telemetry collection proxy (natural data source for TelHawk Stack)

## Contact

- **Issues & Bugs**: Open an issue on GitHub
- **Commercial Licensing**: licensing@telhawk.com
- **Sales & Support**: sales@telhawk.com
- **General Inquiries**: hello@telhawk.com
