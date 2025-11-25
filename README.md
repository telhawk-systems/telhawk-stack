# TelHawk Stack

A lightweight, OCSF-compliant SIEM (Security Information and Event Management) platform built in Go. TelHawk Stack provides Splunk-compatible event collection with OpenSearch as the backend storage engine, designed for security engineers who need enterprise-grade security monitoring without enterprise licensing costs.

## Overview

OCSF-compliant SIEM built in Go with Splunk-compatible ingestion, OpenSearch storage, and a web UI. See `docs/README.md` for full documentation.

## Architecture

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

## Services

| Service | Purpose | Port |
|---------|---------|------|
| `authenticate` | Identity, sessions, JWT/HEC tokens | 8080 |
| `ingest` | Event ingestion + OpenSearch writes | 8088 |
| `search` | Ad-hoc queries + correlation | 8082 |
| `respond` | Detection rules, alerts, cases | 8085 |
| `web` | Frontend UI + API gateway | 3000 |

See `docs/SERVICES.md` for detailed service descriptions.

## Project Structure

```
├── authenticate/ # Authentication & RBAC service
├── ingest/       # Splunk HEC-compatible ingestion + OCSF normalization
├── search/       # Query API service (formerly 'query')
├── respond/      # Detection rules, alerting, case management
├── web/          # Frontend UI (React) + Go backend
├── cli/          # CLI tool (thawk)
├── common/       # Shared libraries (OCSF types, messaging, utilities)
├── docs/         # Documentation (see docs/README.md)
└── scripts/      # Helper scripts
```

## Documentation

- **Getting Started**: `docs/LOCAL_DEVELOPMENT.md`
- **Configuration**: `docs/CONFIGURATION.md`
- **Services & Architecture**: `docs/SERVICES.md`
- **Production**: `docs/PRODUCTION.md`
- **Splunk Compatibility**: `docs/SPLUNK_COMPATIBILITY.md`
- **Helper Scripts**: `docs/HELPER_SCRIPTS.md`

See `docs/README.md` for the full documentation index.

## Quick Start

```bash
# Start the stack
docker compose up -d

# Wait for services to be healthy
docker compose ps

# Access the web UI
open http://localhost:3000

# Default credentials: admin / admin123
```

## CLI (thawk)

```bash
# Authentication
./scripts/thawk auth login -u admin -p admin123
./scripts/thawk auth whoami

# Detection rules
./scripts/thawk rules list
./scripts/thawk rules get <rule-id>

# Alerts and cases
./scripts/thawk alerts list

# Event seeding (for development)
./scripts/thawk seeder run --from-rules ./alerting/dist/rules/

# All commands
./scripts/thawk --help
```

See `docs/CLI_CONFIGURATION.md` for detailed CLI usage.

## Development

### Building Services

```bash
# Build a single service
cd authenticate && go build -o ../bin/authenticate ./cmd/authenticate

# Build CLI
cd cli && go build -o ../bin/thawk .

# Run tests
go test ./...
```

### Running Tests

```bash
# Test all modules
go test ./...

# Test with verbose output
go test -v ./...

# Run pre-push checks (same as CI)
./scripts/pre-push.sh
```

## Configuration

TelHawk Stack uses **enterprise-grade configuration management**:

- **YAML configuration files** for defaults
- **Environment variable overrides** for deployment-specific settings
- **No CLI arguments** for configuration (12-factor compliant)

Each service has a `config.yaml` file. Override via environment variables:

```bash
# Authenticate service
AUTHENTICATE_SERVER_PORT=8080
AUTHENTICATE_AUTH_JWT_SECRET="your-secret-key"

# Ingest service
INGEST_SERVER_PORT=8088
INGEST_AUTH_URL=http://authenticate:8080
INGEST_OPENSEARCH_URL=https://opensearch:9200
```

See `docs/CONFIGURATION.md` for complete configuration guide.

## License

TelHawk Stack is licensed under the **TelHawk Systems Source Available License (TSSAL) v1.0**.

- ✅ **Free to use** - Run TelHawk in your own environment at no cost
- ✅ **Source available** - View and study the code
- ✅ **Modify for yourself** - Customize for your internal needs
- ❌ **No distribution** - Cannot distribute binaries or create public forks
- ❌ **No commercial hosting** - Cannot offer as a service to others

See the [LICENSE](LICENSE) file for full terms.

## Contributing

**Note:** This project does not accept external contributions or pull requests from forks. Development is handled internally by the TelHawk team.

For bug reports and feature requests, please open an issue on GitHub.

## Related Projects

- [telhawk-proxy](https://github.com/telhawk-systems/telhawk-proxy) - Telemetry collection proxy (natural data source for TelHawk Stack)

## Contact

- **Issues & Bugs**: Open an issue on GitHub
