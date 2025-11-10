# Services

Purpose: Orient engineers to each microservice and how they fit together.

## Architecture (High‑Level)

```
Sources → ingest → core → storage ↔ OpenSearch → query → web
```

## auth (Authentication & RBAC)
- Purpose: central authN/Z for all services; token and session management.
- Highlights: JWT access/refresh, roles (admin/analyst/viewer/ingester), HEC token validation API.
- Related docs: `docs/CONFIGURATION.md`, `docs/HELPER_SCRIPTS.md`.

## ingest (Splunk HEC‑compatible Ingestion)
- Purpose: receive events over HEC and other inputs; authenticate, validate, forward to core.
- Endpoints: `/services/collector/event`, `/services/collector/raw`, health/ready as standard.
- Compatibility: see `docs/SPLUNK_COMPATIBILITY.md` (ACK supported).
- Metrics: see `docs/PROMETHEUS_METRICS.md`.

## core (OCSF Normalization)
- Purpose: transform raw events to OCSF, enrich, classify, and route.
- Notes: schema coverage managed in `docs/OCSF_COVERAGE.md` and related normalization docs.
- Metrics: normalization counts, errors, latencies.

## storage (OpenSearch Client & Lifecycle)
- Purpose: index management, templates/mappings, bulk ingest, retention/rollover.
- Notes: deployment/production concerns in `docs/PRODUCTION.md`.

## query (Query API)
- Purpose: programmatic search/aggregation over OpenSearch; saved searches and alerts.
- Endpoints: `POST /api/v1/search`, selected GETs for saved artifacts (see repo docs under `query/`).

## web (Frontend UI)
- Purpose: dashboards, search, alerting, and investigations.
- Notes: UX principles in `docs/UX_DESIGN_PHILOSOPHY.md`.

## Cross‑Cutting
- Configuration: `docs/CONFIGURATION.md`
- Production: `docs/PRODUCTION.md`
- Metrics: `docs/PROMETHEUS_METRICS.md`
- Splunk compatibility: `docs/SPLUNK_COMPATIBILITY.md`
- Helper scripts: `docs/HELPER_SCRIPTS.md`
