# Documentation Index

Overview: see project `README.md` for a high-level intro.

## Core Guides
- [Local Development](LOCAL_DEVELOPMENT.md)
- [Configuration](CONFIGURATION.md)
- [Services & Architecture](SERVICES.md) - authenticate, ingest, search, respond, web
- [Security Architecture](SECURITY_ARCHITECTURE.md)
- [TLS Configuration](TLS_CONFIGURATION.md)
- [Production Deployment](PRODUCTION.md)
- [Logging](LOGGING.md)
- [Handler Conventions](HANDLER_CONVENTIONS.md)
- [JSON:API Conventions](API_JSONAPI.md)
- [Prometheus Metrics](PROMETHEUS_METRICS.md)
- [Helper Scripts](HELPER_SCRIPTS.md)
- [CI Development](CI_DEVELOPMENT.md)

## Feature Documentation

### Alerting & Detection (`alerting/`)
- [Alerting Architecture](alerting/ALERTING_ARCHITECTURE.md)
- [Alerting API](alerting/ALERTING_API.md)
- [Alerts API (JSON:API)](alerting/ALERTS_API_JSON.md)
- [Alert Scheduling](alerting/ALERT_SCHEDULING.md)
- [Detection Rules Roadmap](alerting/DETECTION_RULES_ROADMAP.md)

### Authentication (`auth/`)
- [Auth Integration](auth/AUTH_INTEGRATION.md)
- [Auth PostgreSQL Integration](auth/AUTH_POSTGRES_INTEGRATION.md)
- [Auth Event Forwarding](auth/AUTH_EVENT_FORWARDING.md)
- [User Management (CLI)](auth/CLI_USER_MANAGEMENT.md)
- [Privileges Feature Spec](auth/PRIVILEGES_FEATURE_SPEC.md)

### Ingestion (`ingest/`)
- [Ingest Features](ingest/INGEST_FEATURES_IMPLEMENTATION.md)
- [HEC Token Validation](ingest/HEC_TOKEN_VALIDATION.md)
- [DLQ & Backpressure](ingest/DLQ_AND_BACKPRESSURE.md)
- [Splunk Compatibility](ingest/SPLUNK_COMPATIBILITY.md)
- [Splunk Attack Data Import](ingest/SPLUNK_ATTACK_DATA_IMPORT.md)

### OCSF & Normalization (`ocsf/`)
- [OCSF Coverage](ocsf/OCSF_COVERAGE.md)
- [OCSF Coverage Analysis](ocsf/OCSF_COVERAGE_ANALYSIS.md)
- [Complete OCSF Coverage](ocsf/COMPLETE_OCSF_COVERAGE.md)
- [Normalization Integration](ocsf/NORMALIZATION_INTEGRATION.md)
- [Normalization Status](ocsf/NORMALIZATION_STATUS.md)
- [Normalizer Generation](ocsf/NORMALIZER_GENERATION.md)

### Search & Query (`search/`)
- [Query API](search/QUERY_API.md)
- [Query Language Design](search/QUERY_LANGUAGE_DESIGN.md)
- [Query Pagination & Aggregations](search/QUERY_PAGINATION_AGGREGATIONS.md)
- [Saved Searches](search/SAVED_SEARCHES.md)
- [Events API](search/EVENTS_API.md)

### Dashboards (`dashboards/`)
- [Dashboards API](dashboards/DASHBOARDS_API.md)
- [Dashboard Visualization](dashboards/DASHBOARD_VISUALIZATION.md)

### CLI (`cli/`)
- [CLI Configuration](cli/CLI_CONFIGURATION.md)

## Other
- [UX Design Philosophy](UX_DESIGN_PHILOSOPHY.md)
