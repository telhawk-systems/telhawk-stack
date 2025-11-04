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
- [ ] Implement class-specific validators (can also be generated)
- [ ] Add enrichment hooks (GeoIP, threat intel) post-normalization
- [x] Persist normalized events to storage once pipeline succeeds ✅
  * Events now stored with automatic retry (3 attempts, exponential backoff)
  * Storage failures properly returned (no silent data loss)
  * Health metrics track storage success rate
  * Tests: TestStoragePersistence, TestStorageRetry, TestStorageFailure
  * Documentation: docs/STORAGE_PERSISTENCE.md
- [ ] Capture normalization errors to a dead-letter queue for replay/analysis

## Ingest Service
- [x] Validate HEC tokens against the auth service (✅ Token validation with caching)
- [ ] Backpressure + retries when core returns 4xx/5xx during normalization
- [x] Forward normalized events to storage service (✅ Complete pipeline: Ingest → Core → Storage → OpenSearch)
- [ ] Implement HEC ack channel
- [ ] Add Redis-backed rate limiting in the ingestion pipeline
- [ ] Expose Prometheus metrics for queue depth and normalization latency

## Query Service
- [ ] Replace stubbed search implementation with real OpenSearch queries
- [ ] Add cursor pagination and aggregation support
- [ ] Implement alert scheduling and notification delivery
- [ ] Connect dashboards to saved query definitions

## Storage Service
- [x] Scaffold OpenSearch client and index lifecycle management
- [x] Define OCSF-aware index templates and ILM policies
- [x] Implement bulk ingestion pipeline fed by normalized events

## Web
- [ ] Bootstrap UI shell with authentication
- [ ] Build search console backed by query API
- [ ] Add dashboard visualization components

## DevOps & Tooling
- [ ] Provide docker-compose to run full stack locally
- [ ] Add CI pipeline with linting, gofmt, and go test
- [ ] Publish OpenAPI docs automatically (query/core endpoints)

## Documentation
- [ ] Expand core pipeline docs with class mapping examples
- [ ] Document auth integration and token lifecycle
- [ ] Provide onboarding guide for adding new data sources
