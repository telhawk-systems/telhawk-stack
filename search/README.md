# Query Service

The Query service provides a RESTful API for searching and analyzing security data stored in OpenSearch.

## Features

- **Real-time search** - Query OCSF-normalized events with flexible search syntax
- **Query string support** - Full OpenSearch query string syntax
- **Time-based filtering** - Filter events by time range
- **Field projection** - Return only specific fields to reduce bandwidth
- **Pagination** - Configurable result limits (up to 10,000 events)
- **Sorting** - Sort results by any field
- **Alert scheduling** - Automated alert execution and notification delivery
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

alerting:
  enabled: true
  check_interval_seconds: 30
  webhook_url: "https://your-webhook.example.com/alerts"
  slack_webhook_url: "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
  notification_timeout_seconds: 10
```

Environment variables override config file settings:
- `QUERY_SERVER_PORT`
- `QUERY_OPENSEARCH_URL`
- `QUERY_OPENSEARCH_USERNAME`
- `QUERY_OPENSEARCH_PASSWORD`
- `QUERY_OPENSEARCH_INSECURE`
- `QUERY_OPENSEARCH_INDEX`
- `QUERY_ALERTING_ENABLED`
- `QUERY_ALERTING_CHECK_INTERVAL_SECONDS`
- `QUERY_ALERTING_WEBHOOK_URL`
- `QUERY_ALERTING_SLACK_WEBHOOK_URL`
- `QUERY_ALERTING_NOTIFICATION_TIMEOUT_SECONDS`

## Alert Scheduling

The query service includes a built-in alert scheduler that executes saved queries on a schedule and delivers notifications when results are found. See [ALERT_SCHEDULING.md](../docs/ALERT_SCHEDULING.md) for detailed documentation.

Quick start:

```bash
# Enable alerting
export QUERY_ALERTING_ENABLED=true
export QUERY_ALERTING_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK

# Start query service
go run cmd/query/main.go

# Create an alert
curl -X POST http://localhost:8082/api/v1/alerts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Failed SSH Logins",
    "query": "class_name:\"Authentication\" AND status:\"Failure\"",
    "severity": "medium",
    "schedule": {
      "interval_minutes": 5,
      "lookback_minutes": 15
    },
    "status": "active"
  }'
```

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

- Cursor-based pagination: Deep pagination beyond 10,000 events using `search_after`
- Aggregations: Statistical analysis and grouping without loading all documents
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

### Deep pagination with search_after
```bash
# First request
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "severity:high",
    "limit": 1000,
    "sort": {"field": "time", "order": "desc"}
  }'
# Response includes "search_after": [1698883200, "doc123"]

# Next page using search_after from previous response
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "severity:high",
    "limit": 1000,
    "sort": {"field": "time", "order": "desc"},
    "search_after": [1698883200, "doc123"]
  }'
```

### Aggregations - Count by severity
```bash
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "*",
    "limit": 0,
    "aggregations": {
      "severity_count": {
        "type": "terms",
        "field": "severity",
        "size": 10
      }
    }
  }'
```

### Aggregations - Events over time
```bash
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "class_name:\"Network Activity\"",
    "limit": 0,
    "aggregations": {
      "events_over_time": {
        "type": "date_histogram",
        "field": "time",
        "opts": {
          "interval": "1h"
        }
      }
    }
  }'
```

### Multiple aggregations
```bash
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "severity:high",
    "limit": 100,
    "aggregations": {
      "by_class": {
        "type": "terms",
        "field": "class_name",
        "size": 20
      },
      "avg_duration": {
        "type": "avg",
        "field": "duration"
      },
      "timeline": {
        "type": "date_histogram",
        "field": "time",
        "opts": {
          "interval": "5m"
        }
      }
    }
  }'
```
