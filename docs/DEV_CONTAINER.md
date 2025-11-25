# Development Container Setup

This guide covers the simplified development environment using a single long-running container with bind mounts.

## Overview

The dev container provides:
- **Bind-mounted source code** - edit locally, run in container
- **Go 1.25.4** - latest version
- **Node.js 24.x** - latest version for frontend development
- **Hot reload** - Vite dev server for frontend
- **Service manager** - `thawk serve` daemon with auto-restart
- **Simple service control** - `thawk start/stop/restart/status`
- **Debugging support** - Delve debugger on port 2345

## Quick Start

### 1. First Time Setup

If this is your first time or you have existing postgres volumes:

```bash
# Clean slate - removes all volumes including databases
docker compose -f docker-compose.dev.yml down -v

# Start fresh
docker compose -f docker-compose.dev.yml up -d
```

The init script will automatically create:
- `telhawk_auth` database
- `telhawk_respond` database
- Default admin user (username: `admin`, password: `admin123`)

### 2. Start the dev environment (subsequent times)

```bash
docker compose -f docker-compose.dev.yml up -d
```

This starts:
- `telhawk-dev` - Development container (Ubuntu + Go + Node.js)
- `opensearch` - Event storage
- `postgres` - Single PostgreSQL instance with multiple databases
- `nats` - Message broker
- `redis` - Rate limiting

### 3. Enter the dev container

```bash
docker exec -it telhawk-dev bash
```

### 4. Manage services with thawk

The `thawk serve` daemon starts automatically and manages all services:

```bash
# Start all services (auto-rebuilds if needed)
thawk start

# Start a single service
thawk start ingest

# Check what's running
thawk status

# Restart a service (rebuilds + restarts)
thawk restart ingest

# Stop a service
thawk stop ingest

# View service logs
tail -f /var/log/telhawk/ingest.log
```

### 5. Start frontend dev server (optional)

For hot-reload frontend development:

```bash
./scripts/dev/run-frontend.sh
```

Frontend will be available at http://localhost:5173

## Architecture

```
Host Machine                  Docker Network
┌─────────────┐              ┌──────────────────────────────┐
│             │              │  telhawk-dev container       │
│  Source     │              │  ┌─────────────────────────┐ │
│  Code    ───┼─bind mount──┼─>│ /app (your code)        │ │
│  ./        │              │  │ - authenticate :8080    │ │
│             │              │  │ - ingest      :8088    │ │
│  Editors   │              │  │ - search      :8082    │ │
│  IDE       │              │  │ - respond     :8085    │ │
│  etc.      │              │  │ - web backend :3000    │ │
│             │              │  │ - vite        :5173    │ │
└─────────────┘              │  └─────────────────────────┘ │
                             │                              │
                             │  Infrastructure:             │
                             │  - opensearch :9200          │
                             │  - auth-db    :5432          │
                             │  - respond-db :5432          │
                             │  - nats       :4222          │
                             │  - redis      :6379          │
                             └──────────────────────────────┘
```

## Development Workflows

### Backend Development

Edit Go files on your host machine, then:

```bash
# Inside container:
cd /app/ingest
go run ./cmd/ingest

# Or rebuild and restart:
./scripts/dev/build.sh
./scripts/dev/stop-all.sh
./scripts/dev/start-all.sh
```

### Frontend Development

Edit React/TypeScript files on your host, Vite auto-reloads:

```bash
# Inside container:
./scripts/dev/run-frontend.sh
# Now edit files in web/frontend/ on your host - changes appear instantly
```

### Running Tests

```bash
# Inside container:
cd /app/authenticate
go test ./...

# With coverage:
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Using the CLI

```bash
# From host (uses wrapper script):
./scripts/thawk auth login -u admin -p admin123
./scripts/thawk rules list

# Inside container (direct):
/app/bin/thawk auth login -u admin -p admin123
```

### Debugging with Delve

```bash
# Inside container:
cd /app/ingest
dlv debug ./cmd/ingest --headless --listen=:2345 --api-version=2

# Connect from your IDE to localhost:2345
```

## Helper Scripts

All scripts are in `scripts/dev/`:

| Script | Purpose |
|--------|---------|
| `build.sh` | Build all services |
| `run.sh <service>` | Run single service in foreground |
| `start-all.sh` | Start all services in background |
| `stop-all.sh` | Stop all background services |
| `run-frontend.sh` | Start Vite dev server with hot reload |

## Logs

When using `start-all.sh`, logs are written to:
- `/var/log/telhawk/authenticate.log`
- `/var/log/telhawk/ingest.log`
- `/var/log/telhawk/search.log`
- `/var/log/telhawk/respond.log`
- `/var/log/telhawk/web.log`

View with:
```bash
tail -f /var/log/telhawk/ingest.log
```

## Port Mappings

| Port | Service |
|------|---------|
| 8080 | authenticate |
| 8082 | search |
| 8085 | respond |
| 8088 | ingest (HEC) |
| 3000 | web backend |
| 5173 | Vite dev server (frontend) |
| 2345 | Delve debugger |
| 9200 | OpenSearch |
| 5432 | PostgreSQL |
| 4222 | NATS |
| 6379 | Redis |

All bound to `127.0.0.1` for security.

## vs Production Docker Setup

| Feature | Dev Container | Production |
|---------|---------------|------------|
| Build time | Instant (no rebuild) | Minutes (full rebuild) |
| Code changes | Edit locally, instant | Requires rebuild |
| Debugging | Easy (Delve, logs) | Limited |
| Services | All in one container | Separate containers |
| TLS | Disabled | Enabled |
| Go version | 1.25.4 | 1.23.4 (older) |
| Node.js | 24.x | N/A |
| Hot reload | Yes (frontend) | No |

## Troubleshooting

### Container won't start
```bash
# Check logs
docker compose -f docker-compose.dev.yml logs dev

# Rebuild container
docker compose -f docker-compose.dev.yml build --no-cache dev
docker compose -f docker-compose.dev.yml up -d
```

### Services can't connect to databases
```bash
# Check database health
docker compose -f docker-compose.dev.yml ps

# Restart databases
docker compose -f docker-compose.dev.yml restart auth-db respond-db
```

### Port already in use
```bash
# Stop production stack first
docker compose down

# Or change ports in docker-compose.dev.yml
```

### Go dependencies out of sync
```bash
# Inside container:
cd /app
go mod tidy
go mod download
```

## Cleaning Up

```bash
# Stop everything
docker compose -f docker-compose.dev.yml down

# Remove volumes (clears databases)
docker compose -f docker-compose.dev.yml down -v

# Remove images
docker compose -f docker-compose.dev.yml down --rmi all
```
