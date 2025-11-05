# TelHawk Stack TODO

## Core Service
- [x] Create normalizer generator (extends tools/ocsf-generator pattern) ✅
  * Generate one normalizer per OCSF class from schema
  * Include intelligent field mapping (common variants → OCSF fields)
  * Auto-generate source type classification patterns
  * Output: core/internal/normalizer/generated/*.go files
- [x] Integrate generated normalizers into pipeline ✅
  * 7 normalizers integrated (Auth, Network, Process, File, DNS, HTTP, Detection)
  * Integration tests passing with real log data (8 test files, 26 test cases)
  * Documentation: docs/NORMALIZATION_INTEGRATION.md
- [x] Expand normalizer patterns to cover all 77 OCSF classes ✅
  * OCSF classes generated: 77 (by ocsf-generator)
  * Normalizers generated: 77 (by enhanced normalizer-generator)
  * All normalizers registered in pipeline
  * Generator now schema-driven (auto-generates from event classes)
  * Custom patterns via sourcetype_patterns.json (optional)
- [x] Implement class-specific validators (can also be generated) ✅
  * BasicValidator checks required OCSF fields
  * Validator chain pattern integrated into pipeline
  * Validators run after normalization before storage
- [x] Persist normalized events to storage once pipeline succeeds ✅
  * Events now stored with automatic retry (3 attempts, exponential backoff)
  * Storage failures properly returned (no silent data loss)
  * Health metrics track storage success rate
  * Tests: TestStoragePersistence, TestStorageRetry, TestStorageFailure
  * Documentation: docs/STORAGE_PERSISTENCE.md
- [x] Capture normalization errors to a dead-letter queue for replay/analysis ✅
  * File-based DLQ at /var/lib/telhawk/dlq
  * Captures normalization and storage failures
  * API endpoints for list and purge operations
  * Full event context preserved for debugging/replay
  * Metrics exposed via health endpoint
  * Documentation: docs/DLQ_AND_BACKPRESSURE.md

## Ingest Service
- [x] Validate HEC tokens against the auth service (✅ Token validation with caching)
- [x] Backpressure + retries when core returns 4xx/5xx during normalization ✅
  * Exponential backoff retry (3 attempts, 100ms initial delay)
  * Retries on 5xx, 429, and network errors
  * No retry on 4xx client errors (except 429)
  * Total retry window: ~700ms
  * Prevents cascade failures
  * Documentation: docs/DLQ_AND_BACKPRESSURE.md
- [x] Forward normalized events to storage service (✅ Complete pipeline: Ingest → Core → Storage → OpenSearch)
- [ ] Implement HEC ack channel
- [ ] Add Redis-backed rate limiting in the ingestion pipeline
- [ ] Expose Prometheus metrics for queue depth and normalization latency

## Query Service
- [x] Replace stubbed search implementation with real OpenSearch queries ✅
  * OpenSearch client with TLS/mTLS support
  * Full query_string syntax with time-based filtering
  * Field projection, pagination (10k results), sorting
  * Query performance tracking and comprehensive error handling
  * Documentation: docs/QUERY_SERVICE_READ_PATH.md, query/README.md
- [x] Add cursor pagination and aggregation support ✅
  * Cursor-based pagination with search_after (no 10k limit)
  * Aggregations: terms, date_histogram, metrics (avg, sum, min, max, stats, cardinality)
  * Combined queries with both results and aggregations
  * Total matches tracking in responses
  * Documentation: docs/QUERY_PAGINATION_AGGREGATIONS.md
- [ ] Implement alert scheduling and notification delivery
- [ ] Connect dashboards to saved query definitions

## Storage Service
- [x] Scaffold OpenSearch client and index lifecycle management
- [x] Define OCSF-aware index templates and ILM policies
- [x] Implement bulk ingestion pipeline fed by normalized events

## Web
- [ ] Bootstrap UI shell with authentication (React, NOT NextJS - avoid magic)
- [ ] Build search console backed by query API
- [ ] Add dashboard visualization components
- [ ] Establish testing strategy for UI rendering validation

## DevOps & Tooling
- [x] Provide docker-compose to run full stack locally ✅
  * Full stack deployment with auth, ingest, core, storage, query services
  * OpenSearch integration with health checks
  * CLI tool (thawk) container for administration
  * Documentation: DOCKER.md
- [ ] Add CI pipeline with linting, gofmt, and go test
- [ ] Publish OpenAPI docs automatically (query/core endpoints)

## Documentation
- [ ] Expand core pipeline docs with class mapping examples
- [x] Document auth integration and token lifecycle ✅
  * Documentation: docs/AUTH_INTEGRATION.md
  * Covers: JWT/refresh tokens, HEC tokens, web UI integration
  * Security patterns: cookie storage, token refresh, rate limiting
  * PostgreSQL schema with audit logging
- [ ] Provide onboarding guide for adding new data sources
