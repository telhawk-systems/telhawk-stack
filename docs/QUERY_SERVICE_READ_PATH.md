# Query Service Read Path Implementation

## Overview

The Query Service read path has been fully implemented, replacing all stubbed functionality with real OpenSearch integration. Users can now execute searches against stored OCSF-normalized events with full query string syntax support.

## Architecture

```
┌─────────────┐
│  REST API   │  POST /api/v1/search
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Handler   │  handlers.Search()
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Service   │  QueryService.ExecuteSearch()
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ OS Client   │  OpenSearchClient.Search()
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ OpenSearch  │  ocsf-events* indices
└─────────────┘
```

## Components

### 1. OpenSearch Client (`query/internal/client/opensearch.go`)

- **Purpose**: Manages connections to OpenSearch cluster
- **Features**:
  - TLS/mTLS support with configurable certificate validation
  - Username/password authentication
  - Health check on initialization
  - Connection pooling via opensearch-go client
  - Index pattern support (wildcards)

### 2. Configuration (`query/internal/config/config.go`)

**New Configuration Fields:**
```go
type OpenSearchConfig struct {
    URL      string  // OpenSearch endpoint
    Username string  // Admin username
    Password string  // Admin password
    Insecure bool    // Skip TLS verification
    Index    string  // Base index pattern
}
```

**Environment Variables:**
- `QUERY_OPENSEARCH_URL` - Override OpenSearch endpoint
- `QUERY_OPENSEARCH_USERNAME` - Override username
- `QUERY_OPENSEARCH_PASSWORD` - Override password
- `QUERY_OPENSEARCH_INSECURE` - Set to "true" to skip TLS verification
- `QUERY_OPENSEARCH_INDEX` - Override index pattern

### 3. Search Implementation (`query/internal/service/service.go`)

**ExecuteSearch() Method:**
- Accepts `SearchRequest` with query, time range, filters, sort, pagination
- Builds OpenSearch DSL query from request parameters
- Executes search against `{index}*` pattern (e.g., `ocsf-events*`)
- Applies field projection if `include_fields` specified
- Returns results with latency metrics

**buildOpenSearchQuery() Helper:**
- Converts simplified search request to OpenSearch DSL
- Supports:
  - `match_all` for wildcard queries (`*`)
  - `query_string` for text searches
  - `range` filters for time-based queries
  - `bool` queries with `must` clauses
  - Sort specifications

## Query Syntax

The query service supports OpenSearch query string syntax:

### Basic Queries
```
severity:high                           # Field match
class_name:"Authentication"             # Exact phrase
user.name:admin*                        # Wildcard
severity:(high OR critical)             # Boolean OR
severity:high AND class_name:Auth*      # Boolean AND
NOT severity:low                        # Negation
```

### Field Existence
```
_exists_:threat.name                    # Field must exist
NOT _exists_:error                      # Field must not exist
```

### Ranges
```
time:[1698796800 TO 1698883200]        # Inclusive range
severity_id:{1 TO 5}                   # Exclusive range
time:>=1698796800                      # Greater than or equal
```

### Wildcards and Regex
```
user.name:admin*                       # Wildcard
user.name:/joh?n(ath[oa]n)/           # Regex
```

## Request/Response Examples

### Simple Search
```bash
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "severity:high",
    "limit": 50
  }'
```

### Time-Bounded Search
```bash
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "class_name:Authentication",
    "time_range": {
      "from": "2024-11-01T00:00:00Z",
      "to": "2024-11-03T23:59:59Z"
    },
    "limit": 100
  }'
```

### Field Projection
```bash
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "*",
    "limit": 10,
    "include_fields": ["time", "class_name", "severity", "message"]
  }'
```

### Sorted Results
```bash
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "severity:high",
    "sort": {
      "field": "time",
      "order": "desc"
    },
    "limit": 100
  }'
```

### Response Format
```json
{
  "request_id": "a3b7c9d1-e5f7-4a8b-9c2d-1e3f5a7b9c1d",
  "latency_ms": 142,
  "result_count": 23,
  "results": [
    {
      "time": 1699022400,
      "class_uid": 3002,
      "class_name": "Authentication",
      "activity_id": 1,
      "severity": "High",
      "severity_id": 4,
      "user": {
        "name": "admin",
        "uid": "1001"
      },
      "src_endpoint": {
        "ip": "192.168.1.100",
        "port": 54321
      }
    }
  ]
}
```

## Performance Characteristics

- **Max results per query**: 10,000 events
- **Default limit**: 100 events
- **Index pattern**: Searches all matching indices (e.g., `ocsf-events-000001`, `ocsf-events-000002`)
- **Latency**: Typically 50-500ms depending on query complexity and result set size
- **Timeout**: Inherits from OpenSearch cluster settings (default 30s)

## Error Handling

The service handles the following error scenarios:

1. **Connection Failures**: Returns 500 with "search request" error
2. **Query Syntax Errors**: Returns 400 with OpenSearch error message
3. **Timeout**: Returns 500 with timeout error
4. **Index Not Found**: Returns empty result set (not an error)

## Testing

Unit tests validate:
- Query building logic (`TestBuildOpenSearchQuery`)
- ID generation (`TestGenerateID`)
- Configuration parsing
- Error handling

Integration tests require:
- Running OpenSearch instance
- Populated `ocsf-events*` indices

## Deployment

### Configuration File (`config.yaml`)
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

### Docker Deployment
```bash
docker build -t telhawk-query .
docker run -p 8082:8082 \
  -e QUERY_OPENSEARCH_URL=https://opensearch:9200 \
  -e QUERY_OPENSEARCH_PASSWORD=TelHawk123! \
  telhawk-query
```

### Docker Compose Integration
```yaml
query:
  build: ./query
  ports:
    - "8082:8082"
  environment:
    - QUERY_OPENSEARCH_URL=https://opensearch:9200
    - QUERY_OPENSEARCH_USERNAME=admin
    - QUERY_OPENSEARCH_PASSWORD=${OPENSEARCH_PASSWORD}
    - QUERY_OPENSEARCH_INSECURE=true
  depends_on:
    - opensearch
```

## What's Still Stubbed

The following features remain in-memory and are not persisted to OpenSearch:

1. **Alerts**: Alert definitions are stored in memory only
2. **Dashboards**: Dashboard definitions are stored in memory only
3. **Export**: Export jobs return stub responses

These can be implemented in future iterations as they are lower priority than the core search functionality.

## Future Enhancements

- [ ] Add query result caching (Redis)
- [ ] Implement cursor-based pagination for large result sets
- [ ] Add aggregation support (stats, terms, date histograms)
- [ ] Implement saved searches
- [ ] Add query performance analytics
- [ ] Support multiple index patterns per query
- [ ] Implement field suggestions/autocomplete
- [ ] Add query validation endpoint

## Related Documentation

- [Storage Service](../storage/README.md) - Backend storage layer
- [OCSF Schema](https://schema.ocsf.io/) - Event schema reference
- [OpenSearch Query DSL](https://opensearch.org/docs/latest/query-dsl/) - Query syntax reference
