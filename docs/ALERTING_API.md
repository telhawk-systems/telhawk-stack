# TelHawk Alerting System API Reference

## Rules Service API (Port 8084)

Base URL: `http://rules:8084/api/v1`

### Detection Schemas

#### Create Detection Schema

Creates a new Detection Schema. Server generates `id` and `version_id`.

```http
POST /api/v1/schemas
Content-Type: application/json
Authorization: Bearer {token}
```

**Request Body** (users NEVER provide IDs):
```json
{
  "model": {
    "fields": ["src_endpoint.ip", "dst_endpoint.port", "actor.user.name"],
    "group_by": ["src_endpoint.ip"],
    "time_window": "5m",
    "threshold": 10,
    "aggregation": "count"
  },
  "view": {
    "title": "SSH Brute Force Attempt",
    "severity": "high",
    "priority": "P2",
    "fields_order": ["src_endpoint.ip", "count", "time_range"],
    "description_template": "{{count}} failed SSH attempts from {{src_endpoint.ip}}",
    "mitre_attack": {
      "tactics": ["TA0006"],
      "techniques": ["T1110.001"]
    }
  },
  "controller": {
    "query": "class_uid:3002 AND activity_id:1 AND dst_endpoint.port:22",
    "aggregation_field": "src_endpoint.ip",
    "condition": "count > 10",
    "lookback": "5m",
    "evaluation_interval": "1m"
  }
}
```

**Response** (201 Created):
```json
{
  "id": "018d3c3a-0000-7000-8000-rule00001",
  "version_id": "018d3c3a-7890-7000-8000-123456789abc",
  "version": 1,
  "model": {...},
  "view": {...},
  "controller": {...},
  "created_by": "018d3c3a-1111-7000-8000-admin000001",
  "created_at": "2025-01-09T12:00:00Z",
  "disabled_at": null,
  "hidden_at": null
}
```

**Important**: Server generates both `id` (stable) and `version_id` (version-specific). Users never provide these.

---

#### Update Detection Schema (Create New Version)

Creates a new version of an existing Detection Schema. Server reuses `id` from URL, generates new `version_id`.

```http
PUT /api/v1/schemas/:id
Content-Type: application/json
Authorization: Bearer {token}
```

**Path Parameters**:
- `id`: Stable rule UUID

**Request Body** (same as create, no IDs):
```json
{
  "model": {
    "fields": ["src_endpoint.ip", "dst_endpoint.port", "actor.user.name"],
    "group_by": ["src_endpoint.ip"],
    "time_window": "5m",
    "threshold": 15
  },
  "view": {
    "title": "SSH Brute Force Detection",
    "severity": "high",
    "priority": "P2",
    "fields_order": ["src_endpoint.ip", "count", "time_range"],
    "description_template": "{{count}} failed SSH attempts from {{src_endpoint.ip}}"
  },
  "controller": {
    "query": "class_uid:3002 AND activity_id:1 AND dst_endpoint.port:22",
    "aggregation_field": "src_endpoint.ip",
    "condition": "count > 15",
    "lookback": "5m"
  }
}
```

**Response** (200 OK):
```json
{
  "id": "018d3c3a-0000-7000-8000-rule00001",
  "version_id": "018d3c3a-9999-7000-8000-newversion2",
  "version": 2,
  "model": {...},
  "view": {...},
  "controller": {...},
  "created_by": "018d3c3a-1111-7000-8000-admin000001",
  "created_at": "2025-01-09T14:00:00Z",
  "disabled_at": null,
  "hidden_at": null
}
```

**Versioning**: Server reuses `id` from URL path, generates new `version_id`. Version number auto-increments.

---

#### List Detection Schemas

List all active (not disabled/hidden) Detection Schemas.

```http
GET /api/v1/schemas?page=1&limit=50&severity=high&title=brute
Authorization: Bearer {token}
```

**Query Parameters**:
- `page` (integer, default: 1): Page number
- `limit` (integer, default: 50, max: 100): Items per page
- `severity` (string, optional): Filter by severity (critical, high, medium, low)
- `title` (string, optional): Filter by view.title (substring match)
- `id` (UUID, optional): Filter by specific rule stable ID (shows only latest version)
- `include_disabled` (boolean, default: false): Include disabled schemas
- `include_hidden` (boolean, default: false): Include hidden schemas

**Response** (200 OK):
```json
{
  "schemas": [
    {
      "id": "018d3c3a-0000-7000-8000-rule00001",
      "version_id": "018d3c3a-7890-7000-8000-123456789abc",
      "version": 2,
      "model": {...},
      "view": {...},
      "controller": {...},
      "created_by": "018d3c3a-1111-7000-8000-admin000001",
      "created_at": "2025-01-09T14:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 25,
    "total_pages": 1
  }
}
```

---

#### Get Detection Schema

Get latest version of a Detection Schema by stable ID or get specific version by version_id.

```http
GET /api/v1/schemas/:id
Authorization: Bearer {token}
```

**Path Parameters**:
- `id`: Stable rule UUID (returns latest version) or `version_id` (returns specific version)

**Query Parameters**:
- `version` (integer, optional): Get specific version number (when using stable `id`)

**Response** (200 OK):
```json
{
  "id": "018d3c3a-0000-7000-8000-rule00001",
  "version_id": "018d3c3a-7890-7000-8000-123456789abc",
  "version": 2,
  "model": {...},
  "view": {...},
  "controller": {...},
  "created_by": "018d3c3a-1111-7000-8000-admin000001",
  "created_at": "2025-01-09T14:00:00Z",
  "disabled_at": null,
  "hidden_at": null
}
```

---

#### Get Schema Version History

Get all versions of a Detection Schema.

```http
GET /api/v1/schemas/:id/versions
Authorization: Bearer {token}
```

**Path Parameters**:
- `id`: Stable rule UUID

**Response** (200 OK):
```json
{
  "id": "018d3c3a-0000-7000-8000-rule00001",
  "title": "SSH Brute Force Detection",
  "versions": [
    {
      "version_id": "018d3c3a-7890-7000-8000-123456789abc",
      "version": 2,
      "title": "SSH Brute Force Detection",
      "created_by": "018d3c3a-1111-7000-8000-admin000001",
      "created_at": "2025-01-09T14:00:00Z",
      "disabled_at": null,
      "changes": "Increased threshold from 10 to 15, updated title"
    },
    {
      "version_id": "018d3c3a-7777-7000-8000-123456789abc",
      "version": 1,
      "title": "SSH Brute Force Attempt",
      "created_by": "018d3c3a-1111-7000-8000-admin000001",
      "created_at": "2025-01-09T12:00:00Z",
      "disabled_at": "2025-01-09T14:00:00Z",
      "changes": null
    }
  ]
}
```

---

#### Disable Detection Schema

Disable a Detection Schema (stops evaluation, remains visible).

```http
PUT /api/v1/schemas/:id/disable
Authorization: Bearer {token}
```

**Response** (200 OK):
```json
{
  "id": "018d3c3a-0000-7000-8000-rule00001",
  "version_id": "018d3c3a-7890-7000-8000-123456789abc",
  "version": 2,
  "disabled_at": "2025-01-09T15:00:00Z",
  "disabled_by": "018d3c3a-1111-7000-8000-admin000001"
}
```

---

#### Enable Detection Schema

Re-enable a disabled Detection Schema.

```http
PUT /api/v1/schemas/:id/enable
Authorization: Bearer {token}
```

**Response** (200 OK):
```json
{
  "id": "018d3c3a-0000-7000-8000-rule00001",
  "version_id": "018d3c3a-7890-7000-8000-123456789abc",
  "version": 2,
  "disabled_at": null,
  "disabled_by": null
}
```

---

#### Hide Detection Schema (Soft Delete)

Hide a Detection Schema from UI (soft delete).

```http
DELETE /api/v1/schemas/:id
Authorization: Bearer {token}
```

**Response** (200 OK):
```json
{
  "id": "018d3c3a-7890-7000-8000-123456789abc",
  "hidden_at": "2025-01-09T16:00:00Z",
  "hidden_by": "018d3c3a-1111-7000-8000-admin000001"
}
```

---

#### Test Detection Schema

Test a Detection Schema against historical data (replay).

```http
POST /api/v1/schemas/:id/test
Content-Type: application/json
Authorization: Bearer {token}
```

**Request Body**:
```json
{
  "time_range": {
    "from": "2025-01-09T00:00:00Z",
    "to": "2025-01-09T23:59:59Z"
  },
  "dry_run": true
}
```

**Response** (200 OK):
```json
{
  "schema_id": "018d3c3a-0000-7000-8000-rule00001",
  "version_id": "018d3c3a-7890-7000-8000-123456789abc",
  "schema_title": "SSH Brute Force Attempt",
  "time_range": {
    "from": "2025-01-09T00:00:00Z",
    "to": "2025-01-09T23:59:59Z"
  },
  "would_trigger": true,
  "trigger_count": 3,
  "triggers": [
    {
      "triggered_at": "2025-01-09T08:15:00Z",
      "aggregation_key": "192.168.1.100",
      "event_count": 15,
      "fields": {
        "src_endpoint.ip": "192.168.1.100",
        "count": 15
      }
    }
  ],
  "total_events_matched": 1247,
  "evaluation_duration_ms": 125
}
```

---

## Alerting Service API (Port 8085)

Base URL: `http://alerting:8085/api/v1`

### Alerts

#### List Alerts

List alerts with filtering and pagination.

```http
GET /api/v1/alerts?page=1&limit=50&severity=high&status=open
Authorization: Bearer {token}
```

**Query Parameters**:
- `page` (integer, default: 1)
- `limit` (integer, default: 50, max: 100)
- `severity` (string): critical, high, medium, low, informational
- `status` (string): open, investigating, resolved, false_positive
- `from` (ISO8601 timestamp): Filter by triggered_at >= from
- `to` (ISO8601 timestamp): Filter by triggered_at <= to
- `detection_schema_id` (UUID): Filter by stable rule ID (all versions)
- `detection_schema_version_id` (UUID): Filter by specific schema version
- `case_id` (UUID): Filter by case
- `priority` (string): P1, P2, P3, P4

**Response** (200 OK):
```json
{
  "alerts": [
    {
      "alert_id": "018d3c3a-7890-7000-8000-123456789abc",
      "detection_schema_id": "018d3c3a-0000-7000-8000-rule00001",
      "detection_schema_version_id": "018d3c3a-1234-7000-8000-abcdef123456",
      "detection_schema_title": "SSH Brute Force Attempt",
      "case_id": "018d3c3a-5678-7000-8000-fedcba654321",
      "title": "SSH Brute Force Attempt",
      "description": "15 failed SSH attempts from 192.168.1.100",
      "severity": "high",
      "priority": "P2",
      "status": "open",
      "triggered_at": "2025-01-09T12:00:00Z",
      "event_count": 15,
      "fields": {
        "src_endpoint.ip": "192.168.1.100",
        "count": 15
      }
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 150,
    "total_pages": 3
  }
}
```

---

#### Get Alert Details

Get detailed information about a specific alert.

```http
GET /api/v1/alerts/:id
Authorization: Bearer {token}
```

**Response** (200 OK):
```json
{
  "alert_id": "018d3c3a-7890-7000-8000-123456789abc",
  "detection_schema_id": "018d3c3a-0000-7000-8000-rule00001",
  "detection_schema_version_id": "018d3c3a-1234-7000-8000-abcdef123456",
  "detection_schema_title": "SSH Brute Force Attempt",
  "case_id": "018d3c3a-5678-7000-8000-fedcba654321",
  "title": "SSH Brute Force Attempt",
  "description": "15 failed SSH attempts from 192.168.1.100",
  "severity": "high",
  "priority": "P2",
  "status": "open",
  "triggered_at": "2025-01-09T12:00:00Z",
  "event_count": 15,
  "matched_events": [
    "018d3c3a-1111-7000-8000-event0001",
    "018d3c3a-1112-7000-8000-event0002"
  ],
  "fields": {
    "src_endpoint.ip": "192.168.1.100",
    "actor.user.name": "admin",
    "count": 15,
    "time_range": {
      "start": "2025-01-09T11:55:00Z",
      "end": "2025-01-09T12:00:00Z"
    }
  },
  "mitre_attack": {
    "tactics": ["TA0006"],
    "techniques": ["T1110.001"]
  },
  "detection_schema": {
    "model": {...},
    "view": {...},
    "controller": {...}
  },
  "metadata": {
    "evaluation_duration_ms": 45,
    "matched_query": "class_uid:3002 AND activity_id:1..."
  }
}
```

---

#### Update Alert Status

Update alert status or assign to analyst.

```http
PUT /api/v1/alerts/:id
Content-Type: application/json
Authorization: Bearer {token}
```

**Request Body**:
```json
{
  "status": "investigating",
  "assigned_to": "018d3c3a-2222-7000-8000-analyst001",
  "notes": "Investigating source IP, appears to be external scan"
}
```

**Response** (200 OK):
```json
{
  "alert_id": "018d3c3a-7890-7000-8000-123456789abc",
  "status": "investigating",
  "assigned_to": "018d3c3a-2222-7000-8000-analyst001",
  "updated_at": "2025-01-09T12:30:00Z"
}
```

---

### Cases

#### List Cases

List cases with filtering and pagination.

```http
GET /api/v1/cases?page=1&limit=20&status=open&severity=high
Authorization: Bearer {token}
```

**Query Parameters**:
- `page` (integer, default: 1)
- `limit` (integer, default: 20, max: 100)
- `status` (string): open, investigating, resolved, closed, false_positive
- `severity` (string): critical, high, medium, low
- `priority` (string): P1, P2, P3, P4
- `assigned_to` (UUID): Filter by assigned analyst
- `from` (ISO8601 timestamp): Created after
- `to` (ISO8601 timestamp): Created before

**Response** (200 OK):
```json
{
  "cases": [
    {
      "id": "018d3c3a-5678-7000-8000-fedcba654321",
      "title": "SSH Brute Force Campaign - Multiple IPs",
      "description": "Coordinated brute force attempts from multiple source IPs",
      "severity": "high",
      "priority": "P2",
      "status": "investigating",
      "created_by": "system",
      "created_at": "2025-01-09T12:00:00Z",
      "assigned_to": "018d3c3a-2222-7000-8000-analyst001",
      "alert_count": 5,
      "latest_alert_at": "2025-01-09T14:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "total_pages": 3
  }
}
```

---

#### Create Case

Create a new case manually.

```http
POST /api/v1/cases
Content-Type: application/json
Authorization: Bearer {token}
```

**Request Body**:
```json
{
  "title": "Suspicious Activity - Investigation Required",
  "description": "Multiple security alerts require investigation",
  "severity": "high",
  "priority": "P2",
  "assigned_to": "018d3c3a-2222-7000-8000-analyst001",
  "alert_ids": [
    "018d3c3a-7890-7000-8000-123456789abc",
    "018d3c3a-7891-7000-8000-123456789def"
  ]
}
```

**Response** (201 Created):
```json
{
  "id": "018d3c3a-5678-7000-8000-fedcba654321",
  "title": "Suspicious Activity - Investigation Required",
  "description": "Multiple security alerts require investigation",
  "severity": "high",
  "priority": "P2",
  "status": "open",
  "created_by": "018d3c3a-1111-7000-8000-admin000001",
  "created_at": "2025-01-09T15:00:00Z",
  "assigned_to": "018d3c3a-2222-7000-8000-analyst001",
  "alert_count": 2
}
```

---

#### Get Case Details

Get detailed information about a case including all linked alerts.

```http
GET /api/v1/cases/:id
Authorization: Bearer {token}
```

**Response** (200 OK):
```json
{
  "id": "018d3c3a-5678-7000-8000-fedcba654321",
  "title": "SSH Brute Force Campaign - Multiple IPs",
  "description": "Coordinated brute force attempts from multiple source IPs",
  "severity": "high",
  "priority": "P2",
  "status": "investigating",
  "created_by": "system",
  "created_at": "2025-01-09T12:00:00Z",
  "assigned_to": "018d3c3a-2222-7000-8000-analyst001",
  "resolved_at": null,
  "closed_at": null,
  "alerts": [
    {
      "alert_id": "018d3c3a-7890-7000-8000-123456789abc",
      "title": "SSH Brute Force Attempt",
      "severity": "high",
      "triggered_at": "2025-01-09T12:00:00Z",
      "added_at": "2025-01-09T12:00:05Z"
    },
    {
      "alert_id": "018d3c3a-7891-7000-8000-123456789def",
      "title": "SSH Brute Force Attempt",
      "severity": "high",
      "triggered_at": "2025-01-09T14:30:00Z",
      "added_at": "2025-01-09T14:30:03Z"
    }
  ],
  "timeline": [
    {
      "timestamp": "2025-01-09T12:00:00Z",
      "event": "case_created",
      "actor": "system",
      "details": "Auto-created from alert"
    },
    {
      "timestamp": "2025-01-09T12:15:00Z",
      "event": "case_assigned",
      "actor": "018d3c3a-1111-7000-8000-admin000001",
      "details": "Assigned to analyst001"
    },
    {
      "timestamp": "2025-01-09T14:30:00Z",
      "event": "alert_added",
      "actor": "system",
      "details": "Related alert linked to case"
    }
  ],
  "metadata": {
    "source_ips": ["192.168.1.100", "192.168.1.101"],
    "affected_users": ["admin", "root"],
    "total_events": 27
  }
}
```

---

#### Update Case

Update case status, assignment, or metadata.

```http
PUT /api/v1/cases/:id
Content-Type: application/json
Authorization: Bearer {token}
```

**Request Body**:
```json
{
  "status": "resolved",
  "resolution_notes": "False positive - automated testing from QA team",
  "assigned_to": null
}
```

**Response** (200 OK):
```json
{
  "id": "018d3c3a-5678-7000-8000-fedcba654321",
  "status": "resolved",
  "resolved_at": "2025-01-09T16:00:00Z",
  "resolved_by": "018d3c3a-2222-7000-8000-analyst001",
  "updated_at": "2025-01-09T16:00:00Z"
}
```

---

#### Link Alert to Case

Add an alert to an existing case.

```http
POST /api/v1/cases/:id/alerts
Content-Type: application/json
Authorization: Bearer {token}
```

**Request Body**:
```json
{
  "alert_id": "018d3c3a-7892-7000-8000-123456789ghi"
}
```

**Response** (200 OK):
```json
{
  "case_id": "018d3c3a-5678-7000-8000-fedcba654321",
  "alert_id": "018d3c3a-7892-7000-8000-123456789ghi",
  "added_at": "2025-01-09T16:30:00Z",
  "added_by": "018d3c3a-2222-7000-8000-analyst001"
}
```

---

#### Unlink Alert from Case

Remove an alert from a case.

```http
DELETE /api/v1/cases/:case_id/alerts/:alert_id
Authorization: Bearer {token}
```

**Response** (204 No Content)

---

## Error Responses

All endpoints may return the following error responses:

### 400 Bad Request
```json
{
  "error": "bad_request",
  "message": "Invalid request body",
  "details": {
    "field": "model.threshold",
    "issue": "must be a positive integer"
  }
}
```

### 401 Unauthorized
```json
{
  "error": "unauthorized",
  "message": "Missing or invalid authentication token"
}
```

### 403 Forbidden
```json
{
  "error": "forbidden",
  "message": "Insufficient permissions to perform this action"
}
```

### 404 Not Found
```json
{
  "error": "not_found",
  "message": "Detection schema not found",
  "resource_id": "018d3c3a-7890-7000-8000-123456789abc"
}
```

### 409 Conflict
```json
{
  "error": "conflict",
  "message": "A version of this rule was created concurrently. Please retry."
}
```

### 422 Unprocessable Entity
```json
{
  "error": "validation_failed",
  "message": "Schema validation failed",
  "errors": [
    {
      "field": "controller.query",
      "message": "Invalid query syntax: unexpected token 'AN'"
    },
    {
      "field": "model.time_window",
      "message": "Must be a valid duration (e.g., '5m', '1h', '24h')"
    }
  ]
}
```

### 500 Internal Server Error
```json
{
  "error": "internal_error",
  "message": "An unexpected error occurred",
  "request_id": "018d3c3a-9999-7000-8000-request0001"
}
```

---

## Authentication

All API endpoints require authentication using JWT tokens obtained from the auth service.

**Header**:
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Required Roles**:
- Detection Schema CRUD: `admin` or `analyst`
- Alert viewing: `admin`, `analyst`, or `viewer`
- Alert updates: `admin` or `analyst`
- Case management: `admin` or `analyst`

---

## Rate Limiting

API endpoints are rate-limited per user/token:

- **Rules Service**: 100 requests/minute
- **Alerting Service**: 200 requests/minute
- **Test endpoint**: 10 requests/minute (expensive operation)

**Rate Limit Headers**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704812400
```

**429 Too Many Requests**:
```json
{
  "error": "rate_limit_exceeded",
  "message": "Rate limit exceeded. Retry after 30 seconds.",
  "retry_after": 30
}
```
