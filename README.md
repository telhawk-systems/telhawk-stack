# TelHawk Stack

A lightweight, OCSF-compliant SIEM (Security Information and Event Management) platform built in Go. TelHawk Stack provides Splunk-compatible event collection with OpenSearch as the backend storage engine, designed for security engineers who need enterprise-grade security monitoring without enterprise licensing costs.

## Overview

OCSF‑compatible SIEM built in Go with Splunk‑compatible ingestion, OpenSearch storage, and a web UI. See `docs/README.md` for full documentation.
 
## Docs
- Local development: `docs/LOCAL_DEVELOPMENT.md`
- Configuration: `docs/CONFIGURATION.md`
- Services & architecture: `docs/SERVICES.md`
- Production considerations: `docs/PRODUCTION.md`
- Splunk compatibility (HEC + ACK): `docs/SPLUNK_COMPATIBILITY.md`
- Prometheus metrics: `docs/PROMETHEUS_METRICS.md`
- Helper scripts: `docs/HELPER_SCRIPTS.md`
- UX design philosophy: `docs/UX_DESIGN_PHILOSOPHY.md`
 
## Project Structure

```
├── auth/         # Authentication & RBAC
├── core/         # OCSF normalization and processing
├── ingest/       # Splunk HEC‑compatible ingestion
├── query/        # Query API service
├── storage/      # OpenSearch client and lifecycle
├── web/          # Frontend UI
├── cli/          # CLI (thawk)
├── docs/         # Documentation (see docs/README.md)
└── scripts/      # Helper scripts
```

## Architecture
See the high‑level diagram and flow in `docs/SERVICES.md`.

## Services
See `docs/SERVICES.md` for microservice descriptions and architecture.

Quick links:
- Splunk compatibility: `docs/SPLUNK_COMPATIBILITY.md`
- Prometheus metrics: `docs/PROMETHEUS_METRICS.md`

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

### CLI (`thawk`)
Keep handy:
- `thawk auth login`
- `thawk token create`
- `thawk ingest send`
- `thawk search`
More in `cli/README.md` and `docs/CLI_CONFIGURATION.md`.

## Getting Started
For setup and local usage, see `docs/LOCAL_DEVELOPMENT.md`. For configuration, production, metrics, helper scripts, and compatibility, see `docs/README.md`.

#### 8. Stop the Stack
```bash
# Stop all services
docker-compose down

# Stop and remove volumes (deletes all data)
docker-compose down -v
```

## Documentation

See `docs/README.md` for configuration, services, production, metrics, helper scripts, and compatibility notes.

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

### Running Tests
```bash
go test ./...
```

## Configuration
See `docs/CONFIGURATION.md`.

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

Refer to `docs/CONFIGURATION.md` for enterprise‑grade configuration management and quick configuration.

## Deployment

See `docs/PRODUCTION.md` for production considerations. Docker Compose usage is standard; consult `docker-compose.yml` as needed.

## Roadmap

Active ML work continues; other major items shipped. See `docs/todo/` for current planning notes.

## License

TelHawk Stack is licensed under the **TelHawk Systems Source Available License (TSSAL) v1.0**.

**What this means:**
- ✅ **Free to use** - Run TelHawk in your own environment at no cost
- ✅ **Source available** - View and study the code
- ✅ **Modify for yourself** - Customize for your internal needs
- ❌ **No distribution** - Cannot distribute binaries or create public forks
- ❌ **No commercial hosting** - Cannot offer as a service to others

See the [LICENSE](LICENSE) file for full terms.

## Contributing

**Note:** This project does not accept external contributions or pull requests from forks. Development is handled internally by the TelHawk team.

For bug reports and feature requests, please open an issue on GitHub.

### Git Hooks and Testing

**Install Git hooks** (recommended - run once after cloning):

```bash
./scripts/install-hooks.sh
```

This installs a pre-commit hook that automatically formats Go code with `gofmt` before each commit.

**Run pre-push checks** before committing changes:

```bash
./scripts/pre-push.sh
```

This script runs the same checks as CI:
- Code formatting (gofmt)
- Go module tidiness
- Static analysis (go vet)
- Unit tests with race detector
- Linting (golangci-lint, if installed)

For detailed information about CI, Git hooks, and local development checks, see [docs/CI_DEVELOPMENT.md](docs/CI_DEVELOPMENT.md).

## Related Projects

- [telhawk-proxy](https://github.com/telhawk-systems/telhawk-proxy) - Telemetry collection proxy (natural data source for TelHawk Stack)

## Contact

- **Issues & Bugs**: Open an issue on GitHub
