Dashboards API (JSON:API)

List Dashboards
- GET `/api/v1/dashboards`
- Response: `{ "data": [ { "type":"dashboard", "id":"threat-overview", "attributes": { "name":"Threat Overview", "description":"...", "widgets": [ ... ] } } ], "links": { "self": "/api/v1/dashboards" } }`

Get Dashboard
- GET `/api/v1/dashboards/{id}` → `{ "data": { "type":"dashboard", "id":"{id}", "attributes": { ... } } }`

Notes
- Widgets are opaque JSON payloads consumed by the frontend.
- Errors follow JSON:API error object structure.

