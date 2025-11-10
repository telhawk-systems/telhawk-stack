# TelHawk Stack TODO

**Last Updated:** 2025-11-08

## Recent Accomplishments (Nov 8, 2025)
- ✅ Upgraded all services to Go 1.25
- ✅ Migrated from gorilla/csrf to Go 1.25 net/http.CrossOriginProtection
- ✅ Implemented complete user management UI with CRUD operations
- ✅ Implemented HEC token management UI with create/list/revoke
- ✅ Added server-side dashboard metrics caching (5-minute TTL)
- ✅ Fixed CSRF middleware to properly exempt authenticated endpoints
- ✅ Added pagination support to web UI search results
- ✅ Implemented event type-specific views for all 7 major OCSF classes
- ✅ Added dynamic table columns that adapt based on event type (class_uid)
- ✅ Created type-specific detail views with organized, relevant fields
- ✅ Added event type icons and color coding to UI components

## Previous Accomplishments (Nov 6, 2025)
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

## Query Language
- [ ] Phase 1: JSON Query Foundation
  * Define canonical JSON query structure (select, filter, timeRange, aggregations, sort, pagination)
  * Implement JSON → OpenSearch DSL translator
  * JSON schema validation with comprehensive error messages
  * Update query service API to accept JSON queries: POST /api/v1/query
  * Field path syntax: jq-style OCSF paths (`.time`, `.actor.user.name`, `.src_endpoint.ip`)
  * Filter operators: eq, ne, gt, gte, lt, lte, in, contains, startsWith, endsWith, regex, exists, cidr
  * Compound filters: and, or, not with nesting support
  * OCSF-aware field defaults per event class (auto-inject select clause)
  * Unit tests for translator covering all operators and edge cases
  * **Documentation:** docs/QUERY_LANGUAGE_DESIGN.md ✅
- [ ] Phase 2: Filter Chip Integration
  * Update web UI to generate JSON queries from filter chips
  * Implement event class → default fields mapping
  * Filter chip state → JSON query translation
  * Multiple values for same field → OR logic optimization
  * Replace OpenSearch query_string with JSON queries in UI
  * Update SearchConsole.tsx to use JSON query builder
  * Frontend utility: `web/frontend/src/utils/queryBuilder.ts`
- [ ] Phase 3: Text Syntax Parser (Future)
  * Design text syntax grammar (field:value, operators, boolean logic)
  * Implement parser using participle (Go parser library)
  * Text syntax → JSON query AST translation
  * Field name aliases for user convenience (user → .actor.user.name, src_ip → .src_endpoint.ip)
  * Support wildcards, CIDR notation, comparison operators
  * Grouping and precedence with parentheses
  * API accepts both text and JSON input formats
  * UI "Advanced Query" mode with text input and syntax highlighting
  * Parser error messages with helpful suggestions
- [ ] Phase 4: Saved Searches
  * Database schema for saved searches (PostgreSQL JSONB storage)
  * API endpoints: POST/GET/PUT/DELETE /api/v1/searches
  * Store queries as canonical JSON (version-safe, portable)
  * Save search metadata: name, description, owner, sharing permissions
  * UI: Save/Load buttons in search console
  * Share searches with other users (view/edit/admin permissions)
  * Search templates library (pre-built queries for common use cases)
  * Version history for saved searches (track changes over time)
- [ ] Phase 5: S3 Cold Storage Integration (Long-term)
  * JSON query → S3/Parquet predicate translation
  * Time range → partition pruning logic (year/month/day partitions)
  * Select clause → Parquet column projection
  * Filter conditions → Parquet row group filtering
  * Query router: time-based tier selection (hot/warm/cold)
  * Integration with DuckDB or AWS Athena for S3 queries
  * Result merging across multiple tiers
  * Performance optimization: parallel partition scans

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
  * Server-side caching for dashboard metrics (configurable TTL, default 5 minutes)
  * Dedicated /api/dashboard/metrics GET endpoint with cache headers
  * Fixed OpenSearch aggregation errors with .keyword subfields
  * Tab navigation between Overview and Search
  * Pagination support for search results
  * Documentation: docs/DASHBOARD_VISUALIZATION.md
- [x] Create event type-specific views for major OCSF classes ✅
  * Authentication events (3002): Username, source IP, status with color coding
  * Network Activity (4001): Source/dest endpoints with ports, protocol
  * Process Activity (1007): Process name/command line, PID, user context
  * File Activity (4006): File paths, operations, user context
  * DNS Activity (4003): Query hostname, query type, answers
  * HTTP Activity (4002): Request method/URL, status code with color, destination
  * Detection Finding (2004): Finding title, MITRE ATT&CK tactic/technique
  * Event type detection utility based on class_uid
  * Dynamic table columns that adapt to event type
  * Type-specific detail views with relevant fields organized by section
  * Event type icons and color coding in modal headers
  * Components: TypeSpecificColumns, TypeSpecificDetails, eventTypes utility
  * Files: web/frontend/src/utils/eventTypes.ts, web/frontend/src/components/events/
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
- [x] User management UI ✅
  * List/view users with real-time status
  * Create new users with username, email, password, role selection
  * Edit user roles (admin, analyst, viewer)
  * Enable/disable user accounts
  * Reset user passwords
  * Delete users
  * Complete CRUD operations via /api/v1/auth/register and user endpoints
- [x] HEC token management UI ✅
  * View HEC tokens for current user
  * Create new HEC tokens with names
  * Revoke existing tokens
  * Token display with masked values (full value shown once on creation)
  * Copy-to-clipboard functionality
  * Status indicators and creation dates
  * Admin-only access via /tokens route
- [ ] Advanced authorization
  * Role-based access control (RBAC) enforcement
  * Per-index access controls
  * API rate limiting per user
  * Session management dashboard

## Multi-Tenancy & Organization Management
- [ ] Multi-tenant architecture
  * Organization/tenant data model
  * Data isolation per tenant (events, users, configs)
  * Tenant-specific OpenSearch indices/aliases
  * Cross-tenant query prevention
- [ ] Organization management
  * Create/update/delete organizations
  * Organization admin role with limited scope
  * Per-org branding (logos, colors, custom domains)
  * Organization switching in UI for multi-org users
- [ ] Tenant quotas & limits
  * Per-tenant ingestion rate limits
  * Per-tenant storage quotas
  * Per-tenant user/token limits
  * Quota enforcement and alerting
  * Billing/usage tracking per tenant

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
- [x] Add CI pipeline with linting, gofmt, and go test ✅
  * GitHub Actions workflow: .github/workflows/go-build-test-lint.yml
  * Three parallel jobs: Test, Lint, Build
  * Test job: gofmt check, go vet, go mod tidy verification, go test with race detector
  * Lint job: golangci-lint across all 11 Go modules
  * Build job: compile all services and upload artifacts
  * Configuration: .golangci.yml with 15+ linters enabled
  * Runs on push to main and internal pull requests
  * Coverage reports generated locally per module
  * Pre-push script: scripts/pre-push.sh for local validation
  * Documentation: docs/CI_DEVELOPMENT.md
- [ ] Publish OpenAPI docs automatically (query/core endpoints)

## Documentation
- [ ] Expand core pipeline docs with class mapping examples
- [x] Document auth integration and token lifecycle ✅
  * Documentation: docs/AUTH_INTEGRATION.md
  * Covers: JWT/refresh tokens, HEC tokens, web UI integration
  * Security patterns: cookie storage, token refresh, rate limiting
  * PostgreSQL schema with audit logging
- [x] Document alerting and detection rules API ✅
  * Documentation: docs/ALERTING_API.md
  * Covers: Rules service (detection schemas), Alerting service (alerts, cases)
  * Detection schema CRUD with MVC pattern and immutable versioning
  * Case management and alert-to-case associations
  * Complete API reference with error codes
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
- [ ] Data governance workflows
  * Legal hold management (per-tenant, per-event)
  * Right-to-be-forgotten/deletion workflows
  * Per-tenant retention policies
  * Data classification and tagging
  * Rehydration/restore workflows for archived data
  * Data residency controls (region/zone restrictions)
  * Compliance certification tracking

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
  * **API Documentation:** See [docs/ALERTING_API.md](docs/ALERTING_API.md) for Rules and Alerting service endpoints
- [ ] Threat intelligence
  * IOC (Indicators of Compromise) management
  * STIX/TAXII feed integration
  * Automatic enrichment of events with threat intel
  * IOC matching and flagging

## Investigation & Response
- [ ] **B** Entity modeling (MVP: clickable entities + timeline)
  * Make user/IP/hostname clickable in event table
  * Entity detail page showing all events for that entity
  * Real-time query across all event types (no entity DB needed)
  * Entity timeline sorted by time
  * Basic entity stats (event count, first/last seen)
  * Support entity types: user, IP, hostname, process, file
  * Entity filter using existing query language
  * Related entities view (this user → these hosts)
  * **Est: 2-3 days implementation**
- [ ] Entity profiling & advanced modeling
  * Entity database for historical profiling
  * Entity baselining and anomaly detection
  * Session/identity stitching across event types
  * Relationship graph (entity-to-entity connections)
  * Graph visualization for lateral movement
  * Kill chain progression tracking
  * Blast radius analysis
  * Risk scoring per entity
  * Machine learning-based entity behavior
- [ ] Incident/Case management (full lifecycle)
  * Create cases from events/alerts
  * Case state tracking (open, acknowledged, in-progress, resolved, closed)
  * Case ownership and assignment
  * Due dates and SLA tracking
  * Investigation workflow with status updates
  * Evidence collection and tagging
  * Case timeline reconstruction
  * Team collaboration (notes, comments, assignments)
  * Analyst notes with rich text/markdown
  * Case templates for common scenarios
  * Incident metrics (MTTA, MTTR, volume by type)
  * Case reporting and export
- [ ] Event correlation & analysis
  * Link related events across classes
  * Investigation graph visualization
  * Pattern detection across events
  * Automatic event grouping/clustering
  * Cross-event threat scoring
- [ ] Guided investigation workflows
  * Investigation playbooks/templates
  * Automated triage checklists
  * Default investigation paths per event type
  * Contextual investigation suggestions
  * Investigation progress tracking

## Integrations & Enrichment
- [ ] GeoIP enrichment
  * Automatic IP geolocation
  * ASN and organization lookup
  * Country/city/coordinates in events
- [ ] Asset inventory integration
  * Asset database
  * Automatic asset correlation
  * Vulnerability context
- [ ] SOAR & automation framework
  * Generic outbound action framework ("when X, do Y")
  * Action types: webhook, API call, script execution
  * Ticketing system integration (Jira, ServiceNow, PagerDuty)
  * ChatOps integration (Slack, Teams, Discord)
  * Network actions (firewall block, route null)
  * Identity provider actions (disable user, revoke tokens, MFA reset)
  * Endpoint actions (isolate host, kill process, quarantine file)
  * Playbook engine (multi-step automated response)
  * Runbook integration (human + automated steps)
  * Action audit trail
  * Dry-run/approval workflows for sensitive actions
  * Action templates library
- [ ] Detection content packs
  * Shipped detection rules mapped to MITRE ATT&CK
  * OCSF-native detection content
  * Use case packages (ransomware, data exfil, insider threat)
  * Community detection sharing
  * Detection effectiveness tracking

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
- [ ] Compliance reporting & audit trail
  * Pre-built compliance templates (PCI-DSS, HIPAA, SOC 2, GDPR)
  * Evidence collection for audits
  * Automated compliance checks
  * Exportable audit packages
  * Policy violation reporting
  * Access reviews and certifications
- [ ] Comprehensive audit logging
  * End-to-end audit log of all user actions
  * Query audit trail (who ran which queries, when)
  * Alert/rule modification audit trail
  * User/token management audit trail
  * Configuration change audit trail
  * Schema change tracking
  * Queryable audit domain (audit logs as OCSF events)
  * Audit log retention and immutability
  * HMAC/signature verification for audit logs
  * Audit log export for compliance

## Operational Monitoring
- [ ] Central ops & admin control plane
  * Unified admin console for all services
  * Service health dashboard (all services in one view)
  * Real-time metrics aggregation
  * Cross-service dependency visualization
  * Service topology map
  * Centralized configuration management
  * DLQ monitoring across all services
  * Rate limit monitoring and adjustment
  * Ingestion failure tracking and alerting
  * OpenSearch cluster health integration
  * Resource utilization tracking (CPU, memory, disk)
  * Ops console as first-class TelHawk component
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
- [ ] Secrets management & supply chain security
  * HashiCorp Vault integration for secret storage
  * AWS Secrets Manager / Azure Key Vault integration
  * KMS integration for encryption keys
  * Secure HEC token storage (encrypted at rest)
  * Secure database credential management
  * TLS certificate auto-rotation
  * Secure credential storage patterns
  * SBOM (Software Bill of Materials) generation
  * Container image signing and verification
  * Release attestation (SLSA compliance)
  * Dependency vulnerability scanning
  * Supply chain security documentation
- [ ] Container hardening
  * Opinionated seccomp profiles
  * AppArmor/SELinux policies
  * BPF-based security policies
  * Minimal base images (distroless)
  * Non-root user execution
  * Read-only filesystems
  * Capability dropping
  * Network policy definitions

## Documentation
