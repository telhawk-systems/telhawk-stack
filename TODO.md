# TelHawk Stack TODO

**Last Updated:** 2025-11-07

## Recent Accomplishments (Nov 6, 2025)
- ✅ Fixed OpenSearch integration (removed duplicate write indices)
- ✅ Enabled auth event forwarding to HEC endpoint
- ✅ Verified OCSF 1.1.0 compliance for all events
- ✅ Built professional search console with Tailwind CSS
- ✅ Added event table with color-coded severity display
- ✅ Created event detail modal for OCSF field inspection
- ✅ Configured query service with correct index pattern

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
- [x] Configure HEC event ingestion and OCSF normalization ✅
  * Fixed ingest service URLs to use Docker service names
  * HEC endpoint fully operational at /services/collector/event
  * Events normalized to OCSF format with fallback HECNormalizer
  * Auth events forwarded to HEC endpoint with OCSF Authentication class (3002)
  * 12+ events successfully stored and queryable in OpenSearch
  * Fixed query service index pattern (telhawk-events-*)
- [x] Implement HEC ack channel ✅
  * In-memory ack manager with configurable TTL
  * Per-event tracking with status (Pending, Success, Failed)
  * Automatic cleanup of expired acks
  * Query endpoint at /services/collector/ack
  * Prometheus metrics: acks_pending, acks_completed_total
  * Documentation: ingest/FEATURES.md
- [x] Add Redis-backed rate limiting in the ingestion pipeline ✅
  * Two-tier rate limiting: IP-based (pre-auth) and token-based (post-auth)
  * Redis sliding window algorithm with Lua scripts
  * Configurable limits per time window
  * Graceful degradation if Redis unavailable
  * Returns HTTP 429 when rate limit exceeded
  * Prometheus metrics: rate_limit_hits_total
  * Documentation: ingest/FEATURES.md
- [x] Expose Prometheus metrics for queue depth and normalization latency ✅
  * Metrics endpoint at /metrics
  * Queue metrics: queue_depth, queue_capacity
  * Normalization: duration histogram, error counter
  * Storage: duration histogram, error counter
  * Event tracking: events_total by endpoint and status
  * Rate limiting and ack metrics included
  * Documentation: ingest/FEATURES.md

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
- [x] Implement alert scheduling and notification delivery ✅
  * Alert scheduler with configurable check intervals
  * Multiple notification channels: webhook, Slack, log
  * Alert execution with time-based lookback windows
  * Alert state tracking (last triggered time)
  * Metrics for alert executions, triggers, and notifications
  * Graceful shutdown with timer cleanup
  * Documentation: docs/ALERT_SCHEDULING.md
- [ ] Connect dashboards to saved query definitions

## Storage Service
- [x] Scaffold OpenSearch client and index lifecycle management
- [x] Define OCSF-aware index templates and ILM policies
- [x] Implement bulk ingestion pipeline fed by normalized events

## Web
- [x] Bootstrap UI shell with authentication (React, NOT NextJS - avoid magic) ✅
  * Go backend: Auth middleware, session management, API proxy
  * React frontend: Vite + TypeScript, protected routes, auth context
  * JWT authentication with HTTP-only cookies
  * Distroless Docker image
  * Integration with auth, query, and core services
  * Documentation: web/README.md
- [x] Build search console backed by query API ✅
  * SearchConsole component with time range picker (15m, 1h, 24h, 7d, custom)
  * EventsTable component with color-coded severity display
  * EventDetailModal for full OCSF event inspection
  * Tailwind CSS v3 for modern, responsive styling
  * Query performance metrics display
  * Time-based event filtering integrated with query API
- [x] Add dashboard visualization components ✅
  * DashboardOverview with time range selector and auto-refresh
  * MetricCard components for key statistics (total events, critical events, unique users/IPs)
  * SeverityChart (pie chart) for event distribution by severity
  * EventClassChart (bar chart) for top event classes
  * TimelineChart (line chart) for events over time
  * Integration with OpenSearch aggregation API
  * Tab navigation between Overview and Search
  * Documentation: docs/DASHBOARD_VISUALIZATION.md
- [ ] Create event type-specific views for major OCSF classes
  * Authentication events (3002): Login attempts, user context, session details
  * Network Activity (4001): Source/dest IPs, ports, protocols, connection status
  * Process Activity (1007): Command lines, parent/child processes, user context
  * File Activity (4006): File paths, operations, hashes, permissions
  * DNS Activity (4003): Query/response, domain names, record types
  * HTTP Activity (4002): URLs, methods, status codes, user agents
  * Detection Finding (2004): Rule names, severity, tactics/techniques
  * Custom table columns and detail views per event type
  * Type-aware filtering and search
- [ ] Establish testing strategy for UI rendering validation

## Authentication & Authorization
- [x] Basic authentication (register, login, validate, refresh) ✅
  * JWT-based authentication with access/refresh tokens
  * HTTP-only secure cookies
  * PostgreSQL-backed user storage
  * Audit logging for auth events
- [x] HEC token management (backend) ✅
  * Create, validate, revoke HEC tokens
  * Token association with users
  * Token validation caching (5m TTL)
- [ ] User management UI
  * List/view users
  * Create/edit/delete users
  * Role assignment (admin, analyst, viewer)
  * Password reset functionality
- [ ] HEC token management UI
  * View HEC tokens for current user
  * Create new HEC tokens with names
  * Revoke existing tokens
  * Token usage statistics
- [ ] Advanced authorization
  * Role-based access control (RBAC) enforcement
  * Per-index access controls
  * API rate limiting per user
  * Session management dashboard

## DevOps & Tooling
- [x] Provide docker-compose to run full stack locally ✅
  * Full stack deployment with auth, ingest, core, storage, query services
  * OpenSearch integration with health checks
  * CLI tool (thawk) container for administration
  * Documentation: DOCKER.md
- [x] **SECURITY: Enable TLS for all internal service communication** ✅
  * ✅ OpenSearch: HTTPS with self-signed certificates (implemented)
  * ✅ PostgreSQL: SSL/TLS enabled with self-signed certificates (implemented)
  * ✅ Auth service (port 8080): TLS support with feature flag
  * ✅ Ingest service (port 8088): TLS support with feature flag
  * ✅ Core service (port 8090): TLS support with feature flag
  * ✅ Storage service (port 8083): TLS support with feature flag
  * ✅ Query service (port 8082): TLS support with feature flag
  * ✅ Web service (port 3000): TLS support with feature flag
  * ✅ Certificate generator for all Go services (telhawk-certs container)
  * ✅ TLS_SKIP_VERIFY flags for self-signed certificate support
  * ✅ Production certificate support via /certs/production/ mount
  * **Documentation:** docs/TLS_CONFIGURATION.md, .env.example
  * **Note:** TLS disabled by default - enable via environment variables
- [x] Build OCSF event seeder for development and testing ✅
  * Generate realistic fake events for 7 major OCSF classes
  * Authentication (3002): login attempts (85% success), logout/MFA/password change (98% success) ✓
  * Network Activity (4001): TCP/UDP/ICMP connections, firewall events ✓
  * Process Activity (1007): process starts with command lines and parent processes ✓
  * File Activity (4006): file create/read/update/delete/rename operations ✓
  * DNS Activity (4003): DNS queries and responses with various record types ✓
  * HTTP Activity (4002): HTTP requests with realistic status codes and paths ✓
  * Detection Finding (2004): security alerts with MITRE ATT&CK tactics ✓
  * Configurable event volume, timing, time-spread, and batch size
  * Direct ingestion via HEC endpoint with proper token authentication
  * Comprehensive test suite with 10 test cases covering all event types
  * **Tool:** tools/event-seeder/
  * **Documentation:** tools/event-seeder/README.md
  * **Fixes Applied:**
    - Updated OpenSearch mappings for complex nested OCSF objects
    - Created OCSF passthrough normalizer for pre-formatted events
    - Fixed authentication event logic: realistic failure rates per action type
    - All 7 event types verified and working
- [x] Fix HEC token creation in CLI ✅
  * Implemented real API calls to auth service endpoints
  * HEC token management: create, list, revoke via /api/v1/hec/tokens
  * JWT token parsing to extract user_id for authorization
  * Audit logging for all HEC token operations
  * CLI commands: thawk token create/list/revoke
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

## Data Management & Retention
- [ ] Index lifecycle management (ILM)
  * Automated index rollover
  * Hot/warm/cold tier policies
  * Data retention policies by index pattern
  * Index size and age-based triggers
- [ ] Backup and restore
  * OpenSearch snapshot configuration
  * Automated backup schedules
  * Point-in-time recovery
  * Disaster recovery procedures

## Detection & Analytics
- [ ] Saved searches and query library
  * Save/load searches with names
  * Query templates for common patterns
  * Share searches between users
  * Search history per user
- [ ] Alerting and correlation
  * Alert rule creation UI
  * Schedule-based alert evaluation
  * Threshold-based alerting
  * Notification channels (email, webhook, Slack)
  * Alert history and status tracking
- [ ] Threat intelligence
  * IOC (Indicators of Compromise) management
  * STIX/TAXII feed integration
  * Automatic enrichment of events with threat intel
  * IOC matching and flagging

## Investigation & Response
- [ ] Case management
  * Create cases from events/alerts
  * Investigation workflow
  * Evidence collection and tagging
  * Case timeline reconstruction
  * Team collaboration (notes, assignments)
- [ ] Event correlation
  * Link related events
  * Investigation graph visualization
  * Pattern detection across events

## Integrations & Enrichment
- [ ] GeoIP enrichment
  * Automatic IP geolocation
  * ASN and organization lookup
  * Country/city/coordinates in events
- [ ] Asset inventory integration
  * Asset database
  * Automatic asset correlation
  * Vulnerability context
- [ ] External integrations
  * Webhook notifications
  * SOAR platform connectors
  * Ticketing system integration (Jira, ServiceNow)

## Data Collection
- [ ] Additional ingestion protocols
  * Syslog server (RFC 5424, RFC 3164)
  * Beats protocol (for Elastic agents)
  * Fluentd/Fluent Bit compatible endpoint
  * Cloud log collection (AWS CloudWatch, Azure Monitor)
- [ ] Log forwarder agent
  * Lightweight agent for file tailing
  * Multi-platform support (Linux, Windows, macOS)
  * Buffering and retry logic
  * TLS mutual authentication

## Reporting & Export
- [ ] Report generation
  * Scheduled reports
  * PDF/CSV export
  * Email delivery
  * Custom report templates
- [ ] Compliance reporting
  * Pre-built compliance templates (PCI-DSS, HIPAA, SOC 2)
  * Evidence collection for audits
  * Automated compliance checks

## Operational Monitoring
- [ ] System health dashboard
  * Service status indicators
  * Ingestion rate graphs
  * Storage capacity monitoring
  * Query performance metrics
- [ ] DLQ management UI
  * View failed events
  * Retry/replay failed events
  * Delete or archive DLQ entries
  * DLQ statistics and trends

## Performance & Scalability
- [ ] Query optimization
  * Query result caching
  * Connection pooling
  * Query plan optimization
- [ ] Horizontal scaling guide
  * Multi-node OpenSearch configuration
  * Load balancer setup
  * Service replication strategies
  * Performance tuning documentation

## Security Hardening
- [ ] Enhanced security features
  * Security headers (CSP, HSTS, X-Frame-Options)
  * Input validation and sanitization
  * SQL injection prevention
  * XSS protection
- [ ] Secrets management
  * HashiCorp Vault integration
  * Encrypted configuration values
  * Certificate auto-rotation
  * Secure credential storage

## Documentation
