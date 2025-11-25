# TelHawk Feature Gaps & TODO

A brutally honest assessment of what's missing. Prioritized by pain level.

---

## Authentication & Identity Management

### Password Management
- [ ] **Password reset flow** - No self-service password reset
- [ ] **Password reset via email** - No email integration at all
- [ ] **Password complexity requirements** - Configurable policies
- [ ] **Password expiration** - Force rotation after N days
- [ ] **Password history** - Prevent reuse of last N passwords

### Multi-Factor Authentication
- [ ] **TOTP (2FA)** - Google Authenticator, Authy, etc.
- [ ] **WebAuthn/FIDO2** - Yubikey, hardware keys
- [ ] **Backup codes** - Recovery codes for lost 2FA devices
- [ ] **Per-user MFA enforcement** - Admin can require MFA for specific users/roles
- [ ] **MFA bypass for service accounts** - With audit trail

### Enterprise Identity
- [ ] **SAML 2.0 SSO** - Okta, OneLogin, Azure AD
- [ ] **OIDC/OAuth2** - Generic OIDC provider support
- [ ] **Active Directory/LDAP** - On-prem directory integration
- [ ] **SCIM provisioning** - Automated user provisioning/deprovisioning
- [ ] **Group sync** - Map AD/LDAP groups to TelHawk roles

---

## HEC Token Management

### Token Lifecycle
- [ ] **Bulk token creation** - Create 500 tokens without carpal tunnel
- [ ] **Token templates** - Predefined configs for common agent types
- [ ] **Token expiration** - Auto-expire tokens after N days
- [ ] **Token rotation** - Generate new token, grace period, revoke old
- [ ] **Token naming conventions** - Enforce naming patterns

### Token Observability
- [ ] **Last used timestamp** - When was this token last seen?
- [ ] **Usage frequency** - Events per hour/day/week per token
- [ ] **Source IP tracking** - Which IPs used this token?
- [ ] **Event count per token** - Total events ingested via this token
- [ ] **Token health dashboard** - At-a-glance token status

### Token-to-Event Linkage (Nonrepudiation)
- [ ] **Store HEC token ID in event metadata** - Which token ingested this?
- [ ] **Query events by token** - "Show me all events from token X"
- [ ] **Token audit trail** - Full history of token usage
- [ ] **Forensic token analysis** - "What happened with this token?"

---

## Endpoint Management

### Agent Onboarding
- [ ] **Endpoint agent documentation** - What agents work? How to configure?
- [ ] **Agent compatibility matrix** - Splunk UF, Elastic Agent, Fluentd, etc.
- [ ] **Configuration templates** - Ready-to-use agent configs
- [ ] **Onboarding wizard** - Step-by-step endpoint setup
- [ ] **Bulk endpoint registration** - CSV import of 500 endpoints

### Endpoint Tracking
- [ ] **Endpoint registry** - Database of known endpoints
- [ ] **Endpoint-to-token mapping** - Which endpoint uses which token?
- [ ] **Endpoint health monitoring** - Is this endpoint sending data?
- [ ] **Last seen per endpoint** - When did we last hear from endpoint X?
- [ ] **Endpoint inventory export** - CSV/JSON dump of all endpoints

### Endpoint Checklists
- [ ] **Windows endpoint checklist** - Step-by-step setup guide
- [ ] **Linux endpoint checklist** - Step-by-step setup guide
- [ ] **macOS endpoint checklist** - Step-by-step setup guide
- [ ] **Network device checklist** - Firewalls, switches, routers
- [ ] **Cloud service checklist** - AWS, Azure, GCP log forwarding

---

## Data Enrichment

### Asset Context
- [ ] **IP-to-owner mapping** - Who owns 10.1.2.3?
- [ ] **Asset database** - Hostname, owner, department, criticality
- [ ] **CMDB integration** - ServiceNow, etc.
- [ ] **Dynamic asset discovery** - Auto-populate from event data
- [ ] **Asset criticality scoring** - Business impact levels

### User Context
- [ ] **User profile enrichment** - Department, manager, location
- [ ] **User-to-IP correlation** - wizardofoz69's usual login locations
- [ ] **User behavior baseline** - What's normal for this user?
- [ ] **Geographic login tracking** - Where does this user log in from?
- [ ] **Impossible travel detection** - Login from NYC then Tokyo in 5 minutes

### Active Directory Enrichment
- [ ] **AD user attributes** - Pull displayName, department, title
- [ ] **AD group membership** - Is user in "Domain Admins"?
- [ ] **AD computer objects** - Enrich endpoints with AD data
- [ ] **AD event correlation** - Link Windows events to AD objects

### Threat Intelligence
- [ ] **IP reputation lookup** - Is this IP known bad?
- [ ] **Domain reputation** - Is this domain malicious?
- [ ] **File hash lookup** - VirusTotal, etc.
- [ ] **MITRE ATT&CK mapping** - Tag events with techniques

---

## Alerting & Notifications

### Email Notifications
- [ ] **SMTP configuration** - Basic email sending
- [ ] **Alert email notifications** - Email when alert fires
- [ ] **Digest emails** - Daily/weekly summary
- [ ] **Escalation emails** - Notify manager if no response
- [ ] **Email templates** - Customizable alert emails

### Other Notification Channels
- [ ] **Slack integration** - Post alerts to channels
- [ ] **Microsoft Teams** - Teams webhook support
- [ ] **PagerDuty** - On-call alerting
- [ ] **Webhook generic** - POST to arbitrary URLs
- [ ] **SMS/text alerts** - Twilio integration

---

## Audit & Compliance

### Audit Trail
- [ ] **Admin action logging** - Who changed what, when?
- [ ] **Configuration change history** - Track rule/config changes
- [ ] **User login history** - All auth attempts with details
- [ ] **Token usage audit** - Complete token activity log
- [ ] **Data access logging** - Who searched for what?

### Compliance Reporting
- [ ] **User access review** - Who has access to what?
- [ ] **Privileged access report** - List all admin users
- [ ] **Failed login report** - Brute force detection
- [ ] **Dormant account report** - Users who haven't logged in
- [ ] **Token inventory report** - All tokens with status

---

## Multi-Tenancy & Resource Management

### Rate Limiting
- [ ] **Per-tenant rate limiting** - Prevent noisy tenants from overwhelming the system
- [ ] **Per-token burst limits** - Allow short bursts, then throttle
- [ ] **Tenant quota dashboard** - Visualize usage vs limits per tenant
- [ ] **Graceful degradation** - Slow down vs hard reject when limits hit
- [ ] **Rate limit alerts** - Notify when tenant approaches/exceeds limits
- [ ] **Dynamic rate adjustment** - Auto-adjust limits based on system load

### Resource Isolation
- [ ] **Tenant storage quotas** - Max events per day/month per tenant
- [ ] **Query resource limits** - Prevent expensive queries from impacting others
- [ ] **Index isolation options** - Separate indices per tenant for large customers
- [ ] **Priority queues** - Premium tenants get priority processing

### Data Lifecycle & Cold Storage
- [ ] **Document OpenSearch ISM policy** - Current hot/warm/cold/delete lifecycle
- [ ] **Configurable retention per tenant** - Some tenants need 90 days, some need 2 years
- [ ] **Archive to S3/blob storage** - Long-term cold storage beyond OpenSearch
- [ ] **Restore from archive UI** - Self-service retrieval of archived data
- [ ] **Retention policy dashboard** - Visualize data aging across indices

> **Note:** OpenSearch ISM (Index State Management) handles the actual lifecycle transitions. This section tracks documentation and tenant-facing controls.

---

## AI Integration

Target: End of January 2025

### Natural Language Query
- [ ] **NL-to-SPL translation** - "Show me failed logins from yesterday" → query
- [ ] **Query suggestions** - AI-powered autocomplete based on context
- [ ] **Query explanation** - Explain what a complex query does in plain English
- [ ] **Query optimization** - Suggest more efficient query patterns

### Investigation Assistance
- [ ] **Alert summarization** - AI-generated summary of alert context
- [ ] **Incident timeline generation** - Auto-build timeline from related events
- [ ] **Recommended next steps** - "You might want to check these related events"
- [ ] **Similar incident lookup** - "This looks like an incident from last month"

### Threat Analysis
- [ ] **IOC extraction** - Automatically identify indicators in events
- [ ] **MITRE ATT&CK mapping suggestions** - "This behavior matches T1078"
- [ ] **Risk scoring assistance** - AI-informed risk prioritization
- [ ] **Report generation** - Draft incident reports from case data

---

## White Label & MSP Support

### Branding Customization
- [ ] **Custom logo upload** - Replace TelHawk logo with tenant/MSP branding
- [ ] **Custom color scheme** - Primary/secondary colors per tenant
- [ ] **Custom favicon** - Tenant-specific browser icon
- [ ] **White label email templates** - Branded notification emails
- [ ] **Custom login page** - Tenant-specific login branding

### MSP Features
- [ ] **MSP admin portal** - Manage multiple tenant instances
- [ ] **Cross-tenant reporting** - Aggregated metrics across all tenants
- [ ] **Tenant provisioning API** - Automated tenant creation/teardown
- [ ] **Billing integration hooks** - Usage data for MSP billing systems
- [ ] **Tenant impersonation** - MSP admin can view as specific tenant (audited)

---

## API & Integration

### API Improvements
- [ ] **API documentation** - OpenAPI/Swagger specs
- [ ] **API versioning** - v1, v2 with deprecation policy
- [ ] **API rate limiting dashboard** - Who's hitting limits?
- [ ] **API key management** - Separate from HEC tokens
- [ ] **GraphQL endpoint** - Alternative to REST

### Integrations
- [ ] **SOAR integration** - Phantom, XSOAR, Swimlane
- [ ] **Ticketing integration** - Jira, ServiceNow
- [ ] **BI tools** - Grafana, Tableau connectors
- [ ] **Syslog output** - Forward alerts to other SIEMs
- [ ] **CEF/LEEF output** - Standard formats

---

## Priority Matrix

| Gap | Pain Level | Effort | Priority |
|-----|------------|--------|----------|
| Password reset | HIGH | LOW | P1 |
| TOTP 2FA | HIGH | MEDIUM | P1 |
| Token last-used tracking | HIGH | LOW | P1 |
| Token-to-event linkage | HIGH | MEDIUM | P1 |
| Endpoint documentation | HIGH | LOW | P1 |
| Email notifications | HIGH | MEDIUM | P1 |
| Per-tenant rate limiting | HIGH | MEDIUM | P1 |
| AI: NL-to-query translation | HIGH | HIGH | P1 |
| Document ISM policy | MEDIUM | LOW | P2 |
| IP-to-owner mapping | MEDIUM | MEDIUM | P2 |
| User geographic tracking | MEDIUM | MEDIUM | P2 |
| AD/LDAP integration | MEDIUM | HIGH | P2 |
| SAML/OIDC SSO | MEDIUM | HIGH | P2 |
| White label branding | MEDIUM | MEDIUM | P2 |
| Yubikey/WebAuthn | MEDIUM | HIGH | P3 |
| SCIM provisioning | LOW | HIGH | P3 |
| SOAR integration | LOW | MEDIUM | P3 |
| MSP admin portal | LOW | HIGH | P3 |

---

## Quick Wins (Can Do This Week)

1. ~~**Token last-used timestamp** - Add `last_used_at` column, update on each use~~ **DONE** (Redis via `common/hecstats`)
2. ~~**Store token_id in events** - Add to event metadata during ingestion~~ **DONE** (see below)
3. **Endpoint agent docs** - Markdown files for common agents
4. **Password reset (basic)** - Admin-initiated password reset
5. **API docs skeleton** - Basic OpenAPI spec
6. ~~**Integrate hecstats into ingest** - Wire up the collector in HEC handler~~ **DONE** (see below)
7. ~~**Expose token stats in web UI** - Read from Redis, display in token management page~~ **DONE** (see below)
8. ~~**Populate audit context fields** - Update Go handlers to set `*_from_ip` and `*_source_type`~~ **DONE** (see below)

---

## Completed

### Migration 002: Request Context Audit Trail

Added `*_from_ip` (INET) and `*_source_type` (SMALLINT) columns to all mutable tables:

| Table | Columns Added |
|-------|---------------|
| `organizations` | created_from_ip/source_type, disabled_from_ip/source_type, deleted_from_ip/source_type |
| `clients` | created_from_ip/source_type, disabled_from_ip/source_type, deleted_from_ip/source_type |
| `users` | created_from_ip/source_type, disabled_from_ip/source_type, deleted_from_ip/source_type |
| `roles` | created_from_ip/source_type, deleted_from_ip/source_type |
| `sessions` | ip_address, user_agent, source_type, revoked_from_ip/source_type |
| `hec_tokens` | created_from_ip/source_type, disabled_from_ip/source_type, revoked_from_ip/source_type |
| `user_roles` | granted_from_ip/source_type, revoked_from_ip/source_type |
| `role_permissions` | granted_from_ip/source_type |
| `audit_log` | source_type (already had ip_address) |

**Source types:** 0=unknown, 1=web, 2=cli, 3=api, 4=system/internal

### HEC Token Usage Stats (Redis)

Real-time usage tracking stored in Redis (not PostgreSQL) to handle high write volumes from multiple ingest instances.

**Package:** `common/hecstats`

**Redis key structure:**
| Key Pattern | Type | TTL | Purpose |
|-------------|------|-----|---------|
| `hec:stats:{token_id}` | Hash | - | last_used_at, last_used_ip, total_events |
| `hec:hourly:{token_id}:{YYYYMMDDHH}` | Counter | 48h | Events per hour (rolling window) |
| `hec:daily:{token_id}:{YYYYMMDD}` | Counter | 7d | Events per day |
| `hec:ips:{token_id}:{YYYYMMDD}` | Set | 7d | Unique source IPs per day |
| `hec:instances:{token_id}` | Hash | 24h | Which ingest instances handle this token |

**Usage:**
```go
// Ingest service - accumulate stats, flush every 30s
collector := hecstats.NewCollector(client, 30*time.Second, logger)
collector.Record(tokenID, eventCount, clientIP)

// Web/API - read stats
stats, _ := client.GetStats(ctx, tokenID)
// stats.TotalEvents, stats.EventsLastHour, stats.UniqueIPsToday, etc.
```

### Event Ingestion Context (Nonrepudiation)

Every event stored in OpenSearch now includes:

| Field | Type | Purpose |
|-------|------|---------|
| `client_id` | keyword | Multi-tenant data isolation |
| `hec_token_id` | keyword | Which HEC token ingested this event |
| `ingest_source_ip` | ip | IP address of the ingestion request |

**Files modified:**
- `ingest/internal/service/ingest_service.go` - Add fields to both pipeline and non-pipeline paths
- `ingest/internal/storage/opensearch.go` - Add fields to index mapping

**Example queries:**
```
# Find all events ingested by a specific token
GET telhawk-events-*/_search
{ "query": { "term": { "hec_token_id": "abc-123-def" } } }

# Find all events from a specific source IP
GET telhawk-events-*/_search
{ "query": { "term": { "ingest_source_ip": "192.168.1.100" } } }
```

### HEC Stats Collector Integration

The ingest service now tracks HEC token usage in real-time via the `hecstats.Collector`.

**Implementation:**
- `HECHandler` now accepts an optional `*hecstats.Collector`
- After successful ingestion, `collector.Record(tokenID, eventCount, clientIP)` is called
- Stats are buffered in memory and flushed to Redis every 30 seconds
- Collector is initialized in `main.go` when Redis is enabled

**Files modified:**
- `ingest/internal/handlers/hec_handler.go` - Added collector to struct and Record() calls
- `ingest/cmd/ingest/main.go` - Initialize collector and pass to handler

### Audit Context Population

The authenticate service now populates `*_from_ip` and `*_source_type` audit fields.

**New Types:**
- `common/httputil.RequestContext` - Holds IP, source type, and user agent
- `common/httputil.SourceType` - Enum: Unknown(0), Web(1), CLI(2), API(3), System(4)

**Model Updates:**
- `models.Session` - Added `IPAddress`, `UserAgent`, `SourceType`, revocation audit fields
- `models.HECToken` - Added `CreatedFromIP`, `CreatedSourceType`, disable/revoke audit fields
- `models.User` - Added `CreatedFromIP`, `CreatedSourceType`, disable/delete audit fields

**Repository Updates:**
- `CreateSession` - Now inserts `ip_address`, `user_agent`, `source_type`
- `CreateHECToken` - Now inserts `created_from_ip`, `created_source_type`
- `CreateUser` - Now inserts `created_from_ip`, `created_source_type`

**Service Updates:**
- `Login()` - Populates session audit fields from ipAddress/userAgent params
- `CreateUser()` - Populates user audit fields
- `CreateHECToken()` - Populates token audit fields
- Added `inferSourceType()` helper to detect CLI vs Web from User-Agent

### HEC Token Stats in Web UI

The web frontend now displays real-time HEC token usage statistics from Redis.

**New Files:**
- `web/backend/internal/handlers/hec_stats.go` - Handler for stats endpoints

**API Endpoints:**
- `GET /api/hec/stats/{id}` - Get stats for a single token
- `POST /api/hec/stats/batch` - Get stats for multiple tokens (JSON body: `{"token_ids": [...]}`)

**Response Format:**
```json
{
  "token_id": "019ab35c-bfa7-7321-8bc3-4b7fe9d06ab2",
  "last_used_at": "2025-11-24T14:04:51Z",
  "last_used_ip": "10.0.0.1",
  "total_events": 1445,
  "events_last_hour": 200,
  "events_last_24h": 1445,
  "unique_ips_today": 3,
  "ingest_instances": {"ingest-1": "2025-11-24T14:04:51Z"},
  "stats_retrieved_at": "2025-11-24T14:06:19Z"
}
```

**Frontend Changes:**
- `web/frontend/src/pages/TokensPage.tsx` - Added "Last Used" and "Events (24h)" columns
- Click a row to expand and see detailed stats (total events, last hour, unique IPs, last IP, ingest instances)
- `web/frontend/src/services/api.ts` - Added `getHECTokenStats()` and `getHECTokenStatsBatch()` methods
- `web/frontend/src/types/index.ts` - Added `HECTokenStats` interface

**Configuration:**
- Web backend reads from Redis via `REDIS_URL` environment variable (default: `redis://redis:6379`)

---

## Notes

This document exists because a frustrated human rightfully pointed out that a SIEM without these features is like a car without wheels. Sure, it's technically a car, but good luck getting anywhere.

The Almighty may know which IP address wizardofoz69 logs in from, but we should probably know too.
