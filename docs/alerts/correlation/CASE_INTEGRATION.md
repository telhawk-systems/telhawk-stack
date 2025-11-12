# Case Management Integration

**Status**: Design Phase
**Version**: 1.0
**Last Updated**: 2025-11-11

## Overview

This document specifies how correlation alerts integrate with the TelHawk case management system for investigation workflows.

## Background

The alerting service generates correlation alerts which are stored in OpenSearch (`telhawk-alerts-*` indices). The case management system (in the `alerting` service) allows analysts to:
- Create cases from alerts
- Group related alerts into cases
- Track investigation lifecycle
- Link alerts across correlation rules

## Design Goals

1. **Automatic Case Creation**: High-severity correlation alerts auto-create cases
2. **Smart Grouping**: Related alerts grouped into same case
3. **Correlation-Aware**: Cases understand multi-event context
4. **Analyst Efficiency**: Reduce context switching between alerts

---

## Alert → Case Linking

### Database Schema

```sql
-- Existing tables (from alerting/migrations/001_init.up.sql)
CREATE TABLE cases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    severity VARCHAR(20) NOT NULL,
    status VARCHAR(50) NOT NULL, -- open, in_progress, resolved, closed
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMP,
    closed_by UUID,
    assigned_to UUID
);

CREATE TABLE case_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    case_id UUID NOT NULL REFERENCES cases(id),
    alert_id VARCHAR(255) NOT NULL, -- OpenSearch document ID
    alert_index VARCHAR(255) NOT NULL, -- telhawk-alerts-YYYY.MM.DD
    detection_schema_id UUID, -- Stable rule ID
    detection_schema_version_id UUID, -- Version-specific ID
    added_at TIMESTAMP NOT NULL DEFAULT NOW(),
    added_by UUID
);
```

### Alert Structure (OpenSearch)

```json
{
  "_index": "telhawk-alerts-2024.01.15",
  "_id": "alert-1699564800-abc123",
  "_source": {
    "id": "alert-1699564800-abc123",
    "detection_schema_id": "rule-uuid",
    "detection_schema_version_id": "version-uuid",
    "correlation_type": "join",
    "severity": "critical",
    "title": "Post-Auth-Failure Privilege Action",
    "description": "User admin performed privileged action after failed auth",
    "matched_events": [
      {"id": "event-1", "index": "telhawk-events-2024.01.15"},
      {"id": "event-2", "index": "telhawk-events-2024.01.15"}
    ],
    "metadata": {
      "user.name": "admin",
      "src_endpoint.ip": "10.0.1.100",
      "correlation_context": {
        "left_event": "event-1",
        "right_event": "event-2",
        "join_field": "user.name",
        "time_delta": "2m15s"
      }
    },
    "@timestamp": "2024-01-15T10:30:00Z"
  }
}
```

---

## Auto-Case Creation

### Trigger Conditions

**Create case automatically if**:
1. Alert severity is `critical`
2. Correlation type is high-risk (`join`, `temporal_ordered`, `baseline_deviation`)
3. No similar case exists in last 24 hours

### Implementation

```go
func (as *AlertingService) processAlert(ctx context.Context, alert *Alert) error {
    // Store alert in OpenSearch
    if err := as.storageClient.StoreAlert(ctx, alert); err != nil {
        return err
    }

    // Check if auto-case creation criteria met
    if shouldCreateCase(alert) {
        // Check for existing similar case
        existingCase, err := as.findSimilarCase(ctx, alert)
        if err != nil {
            return err
        }

        if existingCase != nil {
            // Add alert to existing case
            if err := as.addAlertToCase(ctx, existingCase.ID, alert); err != nil {
                return err
            }
        } else {
            // Create new case
            caseID, err := as.createCaseFromAlert(ctx, alert)
            if err != nil {
                return err
            }

            // Link alert to case
            if err := as.addAlertToCase(ctx, caseID, alert); err != nil {
                return err
            }
        }
    }

    return nil
}

func shouldCreateCase(alert *Alert) bool {
    // Critical severity
    if alert.Severity == "critical" {
        return true
    }

    // High-risk correlation types
    highRiskTypes := []string{"join", "temporal_ordered", "baseline_deviation", "missing_event"}
    for _, t := range highRiskTypes {
        if alert.CorrelationType == t {
            return true
        }
    }

    return false
}
```

---

## Smart Alert Grouping

### Grouping Strategies

#### 1. By Entity

Group alerts affecting same entity (user, host, IP):

```go
func (as *AlertingService) findSimilarCase(ctx context.Context, alert *Alert) (*Case, error) {
    // Extract entity keys from alert
    entities := extractEntities(alert)

    // Search for open cases with same entities in last 24h
    query := `
        SELECT c.* FROM cases c
        JOIN case_alerts ca ON c.id = ca.case_id
        JOIN alerts a ON ca.alert_id = a.id
        WHERE c.status IN ('open', 'in_progress')
          AND c.created_at > NOW() - INTERVAL '24 hours'
          AND (
              a.metadata->>'user.name' = $1
              OR a.metadata->>'src_endpoint.ip' = $2
              OR a.metadata->>'dst_endpoint.hostname' = $3
          )
        ORDER BY c.created_at DESC
        LIMIT 1
    `

    var existingCase Case
    err := as.db.QueryRow(ctx, query, entities.User, entities.SrcIP, entities.DstHost).Scan(&existingCase)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return &existingCase, err
}
```

#### 2. By Technique (MITRE ATT&CK)

Group alerts with same technique ID:

```go
// Alert metadata includes MITRE technique
alert.Metadata["mitre_technique"] = "T1110.001" // Brute Force: Password Guessing

// Group by technique
query := `
    SELECT c.* FROM cases c
    JOIN case_alerts ca ON c.id = ca.case_id
    JOIN alerts a ON ca.alert_id = a.id
    WHERE c.status IN ('open', 'in_progress')
      AND a.metadata->>'mitre_technique' = $1
      AND c.created_at > NOW() - INTERVAL '6 hours'
    ORDER BY c.created_at DESC
    LIMIT 1
`
```

#### 3. By Correlation Chain

Group alerts that are part of same attack sequence:

```go
// For temporal_ordered or join alerts, check if events overlap
func (as *AlertingService) isPartOfChain(ctx context.Context, alert *Alert, existingCase *Case) bool {
    // Get alerts in case
    caseAlerts := as.getCaseAlerts(ctx, existingCase.ID)

    // Check if alert's matched events overlap with case alerts
    for _, caseAlert := range caseAlerts {
        if hasOverlappingEvents(alert.MatchedEvents, caseAlert.MatchedEvents) {
            return true
        }
    }

    return false
}
```

---

## Case Enrichment

### Correlation Context

When creating case from correlation alert, include rich context:

```go
func (as *AlertingService) createCaseFromAlert(ctx context.Context, alert *Alert) (string, error) {
    caseTitle := generateCaseTitle(alert)
    caseDescription := generateCaseDescription(alert)

    case := &Case{
        Title:       caseTitle,
        Description: caseDescription,
        Severity:    alert.Severity,
        Status:      "open",
        Metadata: map[string]interface{}{
            "initial_alert_id":        alert.ID,
            "correlation_type":        alert.CorrelationType,
            "detection_schema_id":     alert.DetectionSchemaID,
            "entities":                extractEntities(alert),
            "timeline":                buildTimeline(alert),
            "investigation_hints":     generateInvestigationHints(alert),
        },
    }

    return as.repo.CreateCase(ctx, case)
}

func generateCaseDescription(alert *Alert) string {
    var desc strings.Builder

    desc.WriteString(alert.Description)
    desc.WriteString("\n\n## Correlation Details\n\n")

    switch alert.CorrelationType {
    case "join":
        desc.WriteString(fmt.Sprintf("- **Type**: Multi-event correlation (join)\n"))
        desc.WriteString(fmt.Sprintf("- **Events**: %d matched events\n", len(alert.MatchedEvents)))
        desc.WriteString(fmt.Sprintf("- **Time span**: %s\n", alert.Metadata["time_span"]))
        desc.WriteString(fmt.Sprintf("- **Join field**: %s\n", alert.Metadata["join_field"]))

    case "baseline_deviation":
        desc.WriteString(fmt.Sprintf("- **Type**: Behavioral anomaly (baseline deviation)\n"))
        desc.WriteString(fmt.Sprintf("- **Current value**: %v\n", alert.Metadata["current_value"]))
        desc.WriteString(fmt.Sprintf("- **Baseline**: %v (±%v)\n", alert.Metadata["baseline_mean"], alert.Metadata["baseline_stddev"]))
        desc.WriteString(fmt.Sprintf("- **Deviation**: %.2fσ\n", alert.Metadata["deviation"]))

    case "temporal_ordered":
        desc.WriteString(fmt.Sprintf("- **Type**: Attack sequence (temporal ordered)\n"))
        desc.WriteString(fmt.Sprintf("- **Sequence**: %s\n", alert.Metadata["sequence"]))
    }

    desc.WriteString("\n## Entities Involved\n\n")
    entities := extractEntities(alert)
    if entities.User != "" {
        desc.WriteString(fmt.Sprintf("- **User**: %s\n", entities.User))
    }
    if entities.SrcIP != "" {
        desc.WriteString(fmt.Sprintf("- **Source IP**: %s\n", entities.SrcIP))
    }
    if entities.DstHost != "" {
        desc.WriteString(fmt.Sprintf("- **Destination**: %s\n", entities.DstHost))
    }

    return desc.String()
}
```

### Investigation Hints

Provide analysts with next steps:

```go
func generateInvestigationHints(alert *Alert) []string {
    hints := []string{}

    switch alert.CorrelationType {
    case "join":
        hints = append(hints,
            "Review timeline of all events for user "+alert.Metadata["user.name"].(string),
            "Check if source IP "+alert.Metadata["src_endpoint.ip"].(string)+" is known malicious",
            "Investigate what data was accessed after failed authentication",
        )

    case "baseline_deviation":
        hints = append(hints,
            "Compare current activity to historical baseline",
            "Check if user account was recently compromised",
            "Review recent access to sensitive resources",
        )

    case "event_count":
        hints = append(hints,
            "Check if source IP is conducting distributed attack",
            "Review authentication logs for pattern",
            "Consider temporary account lockout",
        )
    }

    return hints
}
```

---

## API Endpoints

### Create Case from Alert

```
POST /api/v1/cases/from-alert
Authorization: Bearer <token>
Content-Type: application/json

{
  "alert_id": "alert-1699564800-abc123",
  "alert_index": "telhawk-alerts-2024.01.15"
}
```

Response:
```json
{
  "case": {
    "id": "case-uuid",
    "title": "Compromised Account: admin",
    "severity": "critical",
    "status": "open",
    "created_at": "2024-01-15T10:30:05Z"
  }
}
```

### Add Alert to Existing Case

```
POST /api/v1/cases/{case_id}/alerts
Authorization: Bearer <token>
Content-Type: application/json

{
  "alert_id": "alert-1699564900-def456",
  "alert_index": "telhawk-alerts-2024.01.15"
}
```

### Get Case with Alerts

```
GET /api/v1/cases/{case_id}?include=alerts
Authorization: Bearer <token>
```

Response:
```json
{
  "case": {
    "id": "case-uuid",
    "title": "Compromised Account: admin",
    "alerts": [
      {
        "id": "alert-1",
        "title": "Brute Force Login Attempts",
        "correlation_type": "event_count",
        "timestamp": "2024-01-15T10:15:00Z"
      },
      {
        "id": "alert-2",
        "title": "Post-Auth-Failure Privilege Action",
        "correlation_type": "join",
        "timestamp": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
```

---

## UI/UX Considerations

### Case Timeline View

Show alerts in chronological order with correlation context:

```
┌─────────────────────────────────────────────────────┐
│ Case: Compromised Account - admin                  │
│ Status: In Progress  │  Severity: Critical         │
└─────────────────────────────────────────────────────┘

Timeline:
10:15 AM  [event_count] Brute Force Login Attempts
          ↓ 15 failed login attempts from 10.0.1.100
          
10:30 AM  [join] Post-Auth-Failure Privilege Action
          ↓ User admin performed privileged action
          ↓ Correlated with previous failed auth
          
10:45 AM  [baseline_deviation] Abnormal File Access
          ↓ 500 files accessed (10x baseline)
```

### Alert Correlation Graph

Visualize relationships between alerts:

```
        [Failed Auth]
              ↓
        [File Delete] ──→ [External Connection]
              ↓
     [Data Exfiltration]
```

---

## Configuration

```yaml
alerting:
  case_management:
    auto_create_enabled: true
    auto_create_severities:
      - critical
    auto_create_correlation_types:
      - join
      - temporal_ordered
      - baseline_deviation
    grouping_strategy: "entity" # entity, technique, chain
    grouping_window: "24h"
    max_alerts_per_case: 100
```

---

## References

- [CORE_TYPES.md](CORE_TYPES.md) - Correlation types
- [OCSF_INTEGRATION.md](OCSF_INTEGRATION.md) - Event correlation
- `alerting/internal/models/case.go` - Case data model
- `docs/ALERT_SCHEDULING.md` - Alerting service overview
