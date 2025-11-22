Events API (JSON:API)

Overview
- Resource: event
- Media type: application/vnd.api+json
- Auth: Authorization: Bearer <JWT> required

List Events (simple filtering)
- GET /api/v1/events
- Query params (examples):
  - filter[query]=severity:high AND class_uid:3002
  - sort=-time
  - page[number]=1&page[size]=50
- Response:
  { "data": [ { "type":"event", "id":"...", "attributes": { ...OCSF doc... } } ],
    "links": { "self": "/api/v1/events?..." },
    "meta": { "total": 1234, "latency_ms": 42, "page": { "number":1, "size":50 } } }

Example:
```
curl -s -H "Authorization: Bearer $TOKEN" -H "Accept: application/vnd.api+json" \
  'http://localhost:8082/api/v1/events?filter[query]=severity:high&sort=-time&page[number]=1&page[size]=10' | jq
```

Canonical Query (complex queries)
- POST /api/v1/events/query
- Request:
  { "data": { "type":"event-query", "attributes": { ...canonical query JSON... } } }
- Response: same collection shape as list; meta may include aggregations and next_cursor

Example:
```
curl -s -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/vnd.api+json" -H "Accept: application/vnd.api+json" \
  -d '{
        "data": {"type":"event-query","attributes": {
          "timeRange":{"last":"24h"},
          "aggregations":[{"type":"terms","field":"severity","name":"severity_count"}],
          "limit": 10,
          "sort":[{"field":"time","order":"desc"}]
        }}
      }' \
  http://localhost:8082/api/v1/events/query | jq
```

Run Saved Search
- POST /api/v1/events/run/{saved_search_id}
- Response: same event collection shape; 409 if the saved search is disabled

Example:
```
curl -s -H "Authorization: Bearer $TOKEN" -H "Accept: application/vnd.api+json" \
  -X POST http://localhost:8082/api/v1/events/run/$SAVED_ID | jq
```

Notes
- Cursor pagination may be surfaced as meta.next_cursor on POST /events/query
- Aggregations, when present, are returned under meta.aggregations
