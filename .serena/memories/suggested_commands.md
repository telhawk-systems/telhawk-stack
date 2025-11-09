# TelHawk Stack - Suggested Commands

## Building Services

### Build All Services
```bash
# Build each service individually
cd auth && go build -o ../bin/auth ./cmd/auth
cd ingest && go build -o ../bin/ingest ./cmd/ingest
cd core && go build -o ../bin/core ./cmd/core
cd storage && go build -o ../bin/storage ./cmd/storage
cd query && go build -o ../bin/query ./cmd/query
cd web/backend && go build -o ../../bin/web ./cmd/web
cd cli && go build -o ../bin/thawk .
```

### Build Specific Service
```bash
cd <service-name> && go build -o ../bin/<service-name> ./cmd/<service-name>
```

## Testing

### Run All Tests
```bash
go test ./...
```

### Test Specific Module
```bash
cd <service-name> && go test ./...
```

### Run Tests with Coverage
```bash
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Tests with Verbose Output
```bash
go test -v ./...
```

### Run Specific Test
```bash
go test -v ./core/internal/pipeline -run TestNormalization
```

### Run with Race Detection
```bash
go test -race ./...
```

## Docker Operations

### Start Full Stack (Development Mode)
```bash
docker-compose up -d
```

### View Logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f <service-name>
```

### Rebuild Services
```bash
# Rebuild all
docker-compose build

# Rebuild specific service
docker-compose build <service-name>

# Rebuild and restart
docker-compose up -d --build
```

### Stop Services
```bash
# Stop all
docker-compose down

# Stop and remove volumes (deletes all data)
docker-compose down -v
```

### Check Service Health
```bash
docker-compose ps
```

### Run CLI Tool in Container
```bash
docker-compose run --rm thawk <command>

# Examples:
docker-compose run --rm thawk auth login -u admin -p admin123
docker-compose run --rm thawk token list
```

## Database Migrations

### Automatic Migration (on auth service startup)
Migrations run automatically when auth service starts.

### Manual Migration Operations
```bash
cd auth

# View migration status
migrate -database "postgres://telhawk:password@localhost:5432/telhawk_auth?sslmode=disable" -path migrations version

# Apply migrations
migrate -database "postgres://telhawk:password@localhost:5432/telhawk_auth?sslmode=disable" -path migrations up

# Rollback last migration
migrate -database "postgres://telhawk:password@localhost:5432/telhawk_auth?sslmode=disable" -path migrations down 1
```

## Code Generation

### Regenerate OCSF Normalizers
```bash
cd tools/normalizer-generator
go run main.go

# Output: core/internal/normalizer/generated/*.go (77 files)
```

## CLI Tool (thawk) Commands

### Authentication
```bash
thawk auth login -u <username> -p <password>
thawk auth whoami
thawk auth logout
```

### HEC Token Management
```bash
thawk token create --name <token-name>
thawk token list
thawk token revoke <token-id>
```

### Event Ingestion
```bash
thawk ingest send --message "event text" --token <hec-token>
```

### Search Queries
```bash
thawk search --query "severity:high" --from 1h
```

## Web Frontend Development

### Development Server
```bash
cd web/frontend
npm install
npm start
```

### Production Build
```bash
cd web/frontend
npm run build
```

## Standard Go Commands

### Format Code
```bash
go fmt ./...
```

### Tidy Dependencies
```bash
go mod tidy
```

### Download Dependencies
```bash
go mod download
```

### Verify Dependencies
```bash
go mod verify
```

### Vendor Dependencies
```bash
go mod vendor
```

## Linux System Commands

Standard Linux commands are available:
- `ls`, `cd`, `pwd`: Directory navigation
- `grep`, `find`: File searching
- `cat`, `less`, `head`, `tail`: File viewing
- `git`: Version control
- `curl`, `wget`: HTTP requests
- `docker`, `docker-compose`: Container management