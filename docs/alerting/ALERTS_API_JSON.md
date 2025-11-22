Alerts API (JSON:API)

List Alerts
- GET `/api/v1/alerts`
- Response: `{ "data": [ { "type":"alert", "id":"...", "attributes": { "name":"...", "description":"...", "query":"...", "severity":"high", "schedule": {"interval_minutes":5,"lookback_minutes":15}, "status":"active", "last_triggered_at":"...", "owner":"..." } } ], "links": { "self": "/api/v1/alerts" } }`

Create/Upsert Alert
- POST `/api/v1/alerts`
- Request: `{ "data": { "type":"alert", "attributes": { "id?":"...", "name":"...", "description?":"...", "query":"...", "severity":"...", "schedule": {"interval_minutes":5,"lookback_minutes":15}, "status?":"active", "owner?":"..." } } }`
- Response: `{ "data": { "type":"alert", "id":"...", "attributes": { ... } } }`

Get Alert
- GET `/api/v1/alerts/{id}` → returns `{ "data": { "type":"alert", "id":"...", "attributes": { ... } } }`

Patch Alert
- PATCH `/api/v1/alerts/{id}`
- Request: `{ "data": { "type":"alert", "id":"{id}", "attributes": { "status?":"...", "owner?":"..." } } }`
- Response: updated alert resource

Errors
- JSON:API error objects with `status`, `code`, `title`, `detail?`.

