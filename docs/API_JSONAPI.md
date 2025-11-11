JSON:API Conventions (TelHawk)

- Content negotiation
  - Requests: `Content-Type: application/vnd.api+json` for POST/PATCH
  - Responses: `Content-Type: application/vnd.api+json`
  - Accept must allow `application/vnd.api+json`

- Envelope
  - Single resource: `{ "data": { "type", "id", "attributes", "relationships?" }, "links?": { "self", ... } }`
  - Collections: `{ "data": [ ... ], "links?": { "self", "next?" }, "meta?": { ... } }`
  - Errors: `{ "errors": [{ "status", "code", "title", "detail?" }] }`

- Pagination
  - Offset: `page[number]`, `page[size]` + `links.self`, `meta.total`
  - Cursor (when exposed): `page[cursor]`, `page[size]` + `links.next`

See endpoint references below for resource shapes and examples.

