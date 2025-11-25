# TelHawk Stack - Docker Quick Reference

## Starting the Stack

```bash
# Start all services
docker compose up -d

# Start with build
docker compose up -d --build

# View logs
docker compose logs -f

# View logs for specific service
docker compose logs -f ingest
```

## Using the CLI Tool

```bash
# Use the wrapper script (recommended)
./scripts/thawk auth login -u admin -p admin123

# Create HEC token
./scripts/thawk token create --name my-token

# Send event
./scripts/thawk ingest send --message "test event" --token <your-token>

# List detection rules
./scripts/thawk rules list

# List alerts
./scripts/thawk alerts list

# Get help
./scripts/thawk --help

# Optional: Create alias for convenience
alias thawk='./scripts/thawk'
```

## Service Endpoints

- **Auth Service**: http://localhost:8080
- **Ingest Service**: http://localhost:8088
- **OpenSearch**: http://localhost:9200
- **OpenSearch Dashboards**: Not included (add if needed)

## Default Credentials

- **OpenSearch**: 
  - Username: `admin`
  - Password: `TelHawk123!`
  - **⚠️ CHANGE THIS IN PRODUCTION**

## Health Checks

```bash
# Auth service
curl http://localhost:8080/healthz

# Ingest service
curl http://localhost:8088/readyz

# OpenSearch
curl -u admin:TelHawk123! http://localhost:9200/_cluster/health

# All services
docker compose ps
```

## Managing Data

```bash
# Stop services (keep data)
docker compose down

# Stop and remove all data
docker compose down -v

# Backup OpenSearch data
docker compose exec opensearch tar czf /tmp/backup.tar.gz /usr/share/opensearch/data
docker cp telhawk-opensearch:/tmp/backup.tar.gz ./opensearch-backup.tar.gz
```

## Troubleshooting

```bash
# Check service logs
docker compose logs <service-name>

# Restart a service
docker compose restart <service-name>

# Rebuild a service
docker compose up -d --build <service-name>

# Enter a container
docker compose exec <service-name> sh

# Check resource usage
docker stats
```

## Development

```bash
# Rebuild after code changes
docker compose build
docker compose up -d

# Watch logs during development
docker compose logs -f auth ingest
```
