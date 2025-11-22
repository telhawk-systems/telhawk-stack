Query API (JSON:API)

Search
- POST `/api/v1/search`
- Request:
  - Headers: `Accept: application/vnd.api+json`, `Content-Type: application/vnd.api+json`
  - Auth: `Authorization: Bearer <JWT>` required
  - Body: `{ "data": { "type": "search", "attributes": { "query": "...", "time_range?": {"from":"...","to":"..."}, "limit?": 100, "sort?": {"field":"time","order":"desc"}, "include_fields?": ["..."], "search_after?": [...], "aggregations?": {...} } } }`
- Response: `{ "data": { "type":"search-result", "id":"<request_id>", "attributes": { "result_count":N, "total_matches":N, "latency_ms":N, "results": [ ... events ... ], "search_after?": [...] } } }`

Canonical Query
- POST `/api/v1/query`
- Request:
  - Headers: JSON:API
  - Auth: `Authorization: Bearer <JWT>` required
  - Body: `{ "data": { "type": "query", "attributes": { ... canonical query JSON ... } } }`
- Response: same as Search, plus `opensearch_query` in attributes for debugging.

Saved Searches
- See `docs/SAVED_SEARCHES.md` (JSON:API, versioned, cursor pagination).

Export
- POST `/api/v1/export`
- Request: `{ "data": { "type": "export", "attributes": { "query":"...", "time_range?":{...}, "format":"ndjson", "compression?": "gzip", "notification_channel?": "webhook:..." } } }`
- Response: `{ "data": { "type":"export-job", "id":"<job_id>", "attributes": { "status":"pending", "expires_at":"..." } } }`
