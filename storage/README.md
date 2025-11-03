# Storage Service

The Storage service manages data persistence in OpenSearch with OCSF-aware index templates and lifecycle management.

## Features

- OpenSearch client with connection management
- OCSF-aware index templates and mappings
- Index lifecycle management (ILM) with rollover and retention policies
- Bulk ingestion API for normalized events
- Health and readiness endpoints

## API Endpoints

### Ingest
- `POST /api/v1/ingest` - Ingest normalized events
  ```json
  {
    "events": [
      {
        "time": 1234567890,
        "class_uid": 1001,
        "class_name": "File System Activity",
        ...
      }
    ]
  }
  ```

### Bulk Ingest
- `POST /api/v1/bulk` - Bulk ingest in NDJSON format

### Health
- `GET /healthz` - Service health check
- `GET /readyz` - Service readiness check (includes OpenSearch connectivity)

## Configuration

See `config.yaml` for configuration options:
- Server settings (port, timeouts)
- OpenSearch connection settings
- Index management settings (shards, replicas, retention, rollover)

## Index Management

The service automatically:
- Creates OCSF-aware index templates
- Sets up ISM policies for automatic rollover and deletion
- Creates initial indices with proper aliases
- Manages write aliases for seamless rollover

## Running

```bash
go run cmd/storage/main.go -config config.yaml
```

Or with Docker:
```bash
docker build -t telhawk-storage .
docker run -p 8083:8083 telhawk-storage
```
