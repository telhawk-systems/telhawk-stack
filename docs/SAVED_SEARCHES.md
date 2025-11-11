Saved Searches

- Purpose: Create, version, and run saved OpenSearch queries via the Query service.

API (JSON:API)
- Content-Type: `application/vnd.api+json`
- Resource: `type: saved-search`, `id` = stable UUID v7; `attributes` include `version_id`, `name`, `query`, `filters`, `is_global`, `created_at`, `disabled_at`, `hidden_at`; `relationships.owner`, `relationships.created_by` refer to `user`.

Endpoints
- GET `/api/v1/saved-searches?filter[show_all]=false&page[number]=1&page[size]=20` → list latest per id; default hides hidden; sort active-first then `-created_at`.
- GET `/api/v1/saved-searches/{id}` → get latest version for id.
- POST `/api/v1/saved-searches` → create
  - Body: `{ "data": { "type": "saved-search", "attributes": { "name": "...", "query": { ... }, "filters": { ... }, "is_global": false } } }`
- PATCH `/api/v1/saved-searches/{id}` → update (creates new version)
  - Body: `{ "data": { "id": "<id>", "type": "saved-search", "attributes": { "name?": "...", "query?": { ... }, "filters?": { ... } } } }`
- POST `/api/v1/saved-searches/{id}/disable|enable|hide|unhide` → state transitions (new versions)
- POST `/api/v1/saved-searches/{id}/run` → execute latest; 409 if disabled

Auth
- Mutating endpoints require `Authorization: Bearer <JWT>`; the Query service validates via Auth `POST /api/v1/auth/validate` and uses returned `user_id` for `owner_id` and `created_by`.

Storage
- Migration: `query/migrations/002_saved_searches.up.sql` (immutable versioning: `id` + `version_id` v7, lifecycle: `disabled_at/by`, `hidden_at/by`).
- Query format: OpenSearch Query DSL JSON. Canonical translator also available at `/api/v1/query`.
Pagination
- Parameters: `page[number]` (1-based), `page[size]` (default 20, max 200)
- Response includes `meta.page` and `meta.total` per JSON:API
