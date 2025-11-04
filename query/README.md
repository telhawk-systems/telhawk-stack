# Query Service

The Query service provides a RESTful API for searching and analyzing security data stored in OpenSearch.

## Features

- **Real-time search** - Query OCSF-normalized events with flexible search syntax
- **Query string support** - Full OpenSearch query string syntax
- **Time-based filtering** - Filter events by time range
- **Field projection** - Return only specific fields to reduce bandwidth
- **Pagination** - Configurable result limits (up to 10,000 events)
- **Sorting** - Sort results by any field
- **Alert management** - Create, update, and manage security alerts
- **Dashboard definitions** - Pre-built security dashboards

## API Endpoints

### Search
- `POST /api/v1/search` - Execute search queries
  ```json
  {
    "query": "severity:high AND class_name:\"Authentication\"",
    "time_range": {
      "from": "2024-11-01T00:00:00Z",
      "to": "2024-11-03T23:59:59Z"
    },
    "limit": 100,
    "sort": {
      "field": "time",
      "order": "desc"
    },
    "include_fields": ["time", "severity", "class_name", "src_endpoint"]
  }
  ```

### Alerts
- `GET /api/v1/alerts` - List all alerts
- `POST /api/v1/alerts` - Create or update an alert
- `GET /api/v1/alerts/{alertId}` - Get alert by ID
- `PATCH /api/v1/alerts/{alertId}` - Patch alert metadata

### Dashboards
- `GET /api/v1/dashboards` - List all dashboards
- `GET /api/v1/dashboards/{dashboardId}` - Get dashboard by ID

### Export
- `POST /api/v1/export` - Request data export (async job)

### Health
- `GET /healthz` - Service health check

## Query Syntax

The query service supports OpenSearch query string syntax:

```
# Simple field search
severity:high

# Boolean operators
severity:high AND class_name:"Authentication"

# Wildcards
user.name:admin*

# Ranges
time:[1698796800 TO 1698883200]

# Exists
_exists_:threat.name

# Match all
*
```

## Configuration

See `config.yaml` for configuration options:

```yaml
server:
  port: 8082
  read_timeout_seconds: 15
  write_timeout_seconds: 15
  idle_timeout_seconds: 60

opensearch:
  url: "https://opensearch:9200"
  username: "admin"
  password: "TelHawk123!"
  insecure: false
  index: "ocsf-events"
```

Environment variables override config file settings:
- `QUERY_SERVER_PORT`
- `QUERY_OPENSEARCH_URL`
- `QUERY_OPENSEARCH_USERNAME`
- `QUERY_OPENSEARCH_PASSWORD`
- `QUERY_OPENSEARCH_INSECURE`
- `QUERY_OPENSEARCH_INDEX`

## Running

```bash
# With config file
go run cmd/query/main.go -config config.yaml

# With environment variables
export QUERY_OPENSEARCH_URL=https://opensearch:9200
export QUERY_OPENSEARCH_PASSWORD=TelHawk123!
go run cmd/query/main.go
```

## Docker

```bash
docker build -t telhawk-query .
docker run -p 8082:8082 \
  -e QUERY_OPENSEARCH_URL=https://opensearch:9200 \
  -e QUERY_OPENSEARCH_PASSWORD=TelHawk123! \
  telhawk-query
```

## Index Pattern

The service searches across all indices matching the configured pattern:
- Default: `ocsf-events*`
- Matches: `ocsf-events-000001`, `ocsf-events-000002`, etc.

This allows for index rollover while maintaining search continuity.

## Performance

- Maximum result size: 10,000 events per query
- Query timeout: Inherited from OpenSearch cluster settings
- Recommended: Use time ranges to limit search scope
- For large exports, use the export endpoint for async processing

## Examples

### Search for failed logins
```bash
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "class_name:\"Authentication\" AND activity_id:1",
    "limit": 50
  }'
```

### Search with time range
```bash
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "severity:high",
    "time_range": {
      "from": "2024-11-01T00:00:00Z",
      "to": "2024-11-03T23:59:59Z"
    }
  }'
```

### Get specific fields only
```bash
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "*",
    "limit": 10,
    "include_fields": ["time", "class_name", "severity"]
  }'
```
