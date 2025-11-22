# search service: Cursor Pagination and Aggregations

## Overview

The search service now supports advanced OpenSearch features for production SOC operations:

- **Cursor-based pagination** - Efficiently paginate through millions of events
- **Aggregations** - Statistical analysis and grouping without loading all documents

## Cursor-Based Pagination

### Problem with Offset Pagination

Traditional offset pagination (`from` + `size`) becomes inefficient for deep pagination:
- OpenSearch must process all skipped documents
- Maximum depth limited to 10,000 results (index.max_result_window)
- Performance degrades with higher offsets

### Solution: search_after

Cursor-based pagination using `search_after`:
- Constant performance regardless of depth
- No arbitrary limits on pagination depth
- More efficient for real-time data

### How It Works

1. **First Request**: Include `sort` to establish consistent ordering
   ```json
   {
     "query": "severity:high",
     "limit": 1000,
     "sort": {
       "field": "time",
       "order": "desc"
     }
   }
   ```

2. **Response** includes `search_after` cursor:
   ```json
   {
     "request_id": "abc123",
     "result_count": 1000,
     "total_matches": 45678,
     "search_after": [1698883200, "doc_id_999"],
     "results": [...]
   }
   ```

3. **Next Request**: Use `search_after` from previous response
   ```json
   {
     "query": "severity:high",
     "limit": 1000,
     "sort": {
       "field": "time",
       "order": "desc"
     },
     "search_after": [1698883200, "doc_id_999"]
   }
   ```

4. **Continue** until `search_after` is not returned (last page)

### Best Practices

- **Always specify sort**: Required for consistent pagination
- **Use unique tiebreaker**: Include document ID in sort for stability
- **Don't modify query**: Keep query consistent across pages
- **Check total_matches**: Know total result count upfront
- **Handle concurrent updates**: New documents may appear during pagination

### Example: Paginate Through All High Severity Events

```bash
#!/bin/bash
QUERY_URL="http://localhost:8082/api/v1/search"
SEARCH_AFTER=""

while true; do
  if [ -z "$SEARCH_AFTER" ]; then
    # First request
    RESPONSE=$(curl -s -X POST "$QUERY_URL" \
      -H "Content-Type: application/json" \
      -d '{
        "query": "severity:high",
        "limit": 1000,
        "sort": {"field": "time", "order": "desc"}
      }')
  else
    # Subsequent requests with search_after
    RESPONSE=$(curl -s -X POST "$QUERY_URL" \
      -H "Content-Type: application/json" \
      -d "{
        \"query\": \"severity:high\",
        \"limit\": 1000,
        \"sort\": {\"field\": \"time\", \"order\": \"desc\"},
        \"search_after\": $SEARCH_AFTER
      }")
  fi
  
  # Process results
  echo "$RESPONSE" | jq '.results[] | {time, class_name, severity}'
  
  # Get next search_after cursor
  SEARCH_AFTER=$(echo "$RESPONSE" | jq '.search_after')
  
  # Exit if no more pages
  if [ "$SEARCH_AFTER" = "null" ]; then
    break
  fi
done
```

## Aggregations

### Supported Aggregation Types

#### 1. Terms Aggregation
Group documents by field values (top N):
```json
{
  "query": "*",
  "limit": 0,
  "aggregations": {
    "top_users": {
      "type": "terms",
      "field": "user.name",
      "size": 20
    }
  }
}
```

Response:
```json
{
  "aggregations": {
    "top_users": {
      "buckets": [
        {"key": "admin", "doc_count": 1523},
        {"key": "john.doe", "doc_count": 891},
        ...
      ]
    }
  }
}
```

#### 2. Date Histogram
Time-series analysis:
```json
{
  "query": "class_name:\"Authentication\"",
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
}
```

Supported intervals: `1m`, `5m`, `10m`, `30m`, `1h`, `12h`, `1d`, `7d`, `30d`

#### 3. Metric Aggregations

**Average**:
```json
{
  "aggregations": {
    "avg_duration": {
      "type": "avg",
      "field": "duration"
    }
  }
}
```

**Sum, Min, Max**:
```json
{
  "aggregations": {
    "total_bytes": {"type": "sum", "field": "bytes_sent"},
    "min_duration": {"type": "min", "field": "duration"},
    "max_duration": {"type": "max", "field": "duration"}
  }
}
```

**Stats** (combined metrics):
```json
{
  "aggregations": {
    "duration_stats": {
      "type": "stats",
      "field": "duration"
    }
  }
}
```

Response:
```json
{
  "aggregations": {
    "duration_stats": {
      "count": 5000,
      "min": 12,
      "max": 9876,
      "avg": 234.5,
      "sum": 1172500
    }
  }
}
```

**Cardinality** (unique count):
```json
{
  "aggregations": {
    "unique_users": {
      "type": "cardinality",
      "field": "user.name"
    }
  }
}
```

### Combining Aggregations with Queries

Get both results AND aggregations:
```json
{
  "query": "severity:high AND class_name:\"Network Activity\"",
  "limit": 100,
  "sort": {"field": "time", "order": "desc"},
  "aggregations": {
    "by_src_ip": {
      "type": "terms",
      "field": "src_endpoint.ip",
      "size": 10
    },
    "by_dst_port": {
      "type": "terms",
      "field": "dst_endpoint.port",
      "size": 10
    },
    "total_bytes": {
      "type": "sum",
      "field": "traffic.bytes"
    },
    "timeline": {
      "type": "date_histogram",
      "field": "time",
      "opts": {"interval": "5m"}
    }
  }
}
```

This returns:
- Top 100 matching events (sorted)
- Top 10 source IPs
- Top 10 destination ports
- Total bytes transferred
- Event timeline (5-minute buckets)

### Aggregation-Only Queries

Set `limit: 0` for analytics without documents:
```json
{
  "query": "*",
  "limit": 0,
  "aggregations": {
    "severity_distribution": {
      "type": "terms",
      "field": "severity",
      "size": 10
    }
  }
}
```

### Performance Considerations

- **Set limit: 0** for aggregation-only queries to skip document retrieval
- **Use filters**: Narrow scope with query filters to reduce aggregation work
- **Limit bucket size**: Keep `size` parameter reasonable (default: 10)
- **Date histogram intervals**: Larger intervals = faster queries
- **Cardinality**: Approximate for high-cardinality fields

## Use Cases

### SOC Dashboard: Security Overview

```json
{
  "query": "*",
  "time_range": {
    "from": "2024-11-04T00:00:00Z",
    "to": "2024-11-04T23:59:59Z"
  },
  "limit": 0,
  "aggregations": {
    "severity_count": {
      "type": "terms",
      "field": "severity",
      "size": 5
    },
    "events_by_class": {
      "type": "terms",
      "field": "class_name",
      "size": 20
    },
    "unique_users": {
      "type": "cardinality",
      "field": "actor.user.name"
    },
    "unique_ips": {
      "type": "cardinality",
      "field": "src_endpoint.ip"
    },
    "timeline": {
      "type": "date_histogram",
      "field": "time",
      "opts": {"interval": "1h"}
    }
  }
}
```

### Threat Hunting: Find Unusual Behavior

```json
{
  "query": "class_name:\"Process Activity\" AND activity_id:1",
  "time_range": {
    "from": "2024-11-01T00:00:00Z",
    "to": "2024-11-04T23:59:59Z"
  },
  "limit": 0,
  "aggregations": {
    "processes_by_user": {
      "type": "terms",
      "field": "actor.user.name",
      "size": 50
    },
    "rare_processes": {
      "type": "terms",
      "field": "process.name",
      "size": 100,
      "opts": {
        "order": {"_count": "asc"}
      }
    }
  }
}
```

### Bulk Export: Download All Events

Use cursor pagination to export millions of events:

```python
import requests
import json

url = "http://localhost:8082/api/v1/search"
search_after = None
total_exported = 0

while True:
    req = {
        "query": "time:[1698796800 TO 1698883200]",
        "limit": 5000,
        "sort": {"field": "time", "order": "asc"}
    }
    
    if search_after:
        req["search_after"] = search_after
    
    resp = requests.post(url, json=req)
    data = resp.json()
    
    # Write to file
    with open(f"export_{total_exported}.json", "w") as f:
        json.dump(data["results"], f)
    
    total_exported += len(data["results"])
    print(f"Exported {total_exported} events...")
    
    search_after = data.get("search_after")
    if not search_after:
        break

print(f"Export complete: {total_exported} total events")
```

## API Reference

### SearchRequest Fields

| Field | Type | Description |
|-------|------|-------------|
| `query` | string | OpenSearch query string syntax |
| `time_range` | object | Time bounds (from, to) |
| `limit` | int | Max results (0-10000, default: 100) |
| `sort` | object | Sort field and order |
| `include_fields` | array | Fields to return |
| `search_after` | array | Cursor from previous response |
| `aggregations` | map | Aggregation definitions |

### SearchResponse Fields

| Field | Type | Description |
|-------|------|-------------|
| `request_id` | string | Unique request identifier |
| `latency_ms` | int | Query execution time |
| `result_count` | int | Number of results returned |
| `total_matches` | int | Total matching documents |
| `results` | array | Event documents |
| `search_after` | array | Cursor for next page (if exists) |
| `aggregations` | map | Aggregation results |

### AggregationRequest Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Aggregation type (terms, date_histogram, avg, sum, min, max, stats, cardinality) |
| `field` | string | Field to aggregate |
| `size` | int | Max buckets for terms aggs (default: 10) |
| `opts` | map | Type-specific options |

## Migration Guide

### Before: Simple Search

```json
{
  "query": "severity:high",
  "limit": 100
}
```

### After: With Pagination

```json
{
  "query": "severity:high",
  "limit": 1000,
  "sort": {"field": "time", "order": "desc"}
}
```

Use `search_after` from response for subsequent pages.

### Before: Manual Counting

Query all events and count in application code.

### After: Use Aggregations

```json
{
  "query": "*",
  "limit": 0,
  "aggregations": {
    "count_by_severity": {
      "type": "terms",
      "field": "severity"
    }
  }
}
```

Aggregations computed by OpenSearch - much faster.
