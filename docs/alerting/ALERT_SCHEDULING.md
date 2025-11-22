# Alert Scheduling and Notification Delivery

The search service implements a comprehensive alert scheduling and notification system that executes saved queries on a schedule and delivers notifications when results are found.

## Architecture

### Components

1. **Scheduler** (`internal/scheduler`) - Manages periodic execution of active alerts
2. **Notification Channels** (`internal/notification`) - Delivers notifications via multiple channels
3. **Alert Store** - Persists alert definitions and execution state
4. **Alert Executor** - Executes search queries for alerts

### How It Works

```
┌──────────────┐
│   Scheduler  │───┐
│  (30s check) │   │
└──────────────┘   │
                   ↓
       ┌────────────────────────┐
       │  Active Alerts Sync    │
       │  - Load active alerts  │
       │  - Start/stop timers   │
       └────────────────────────┘
                   │
                   ↓
       ┌────────────────────────┐
       │  Alert Timer (per ID)  │
       │  - Fires at interval   │
       │  - Executes query      │
       │  - Checks for results  │
       └────────────────────────┘
                   │
                   ↓ (results > 0)
       ┌────────────────────────┐
       │  Notification Delivery │
       │  - Webhook             │
       │  - Slack               │
       │  - Log                 │
       └────────────────────────┘
```

## Configuration

### Environment Variables

```bash
# Enable alert scheduling (default: false)
QUERY_ALERTING_ENABLED=true

# Check interval for syncing active alerts (default: 30 seconds)
QUERY_ALERTING_CHECK_INTERVAL_SECONDS=30

# Webhook URL for generic webhook notifications
QUERY_ALERTING_WEBHOOK_URL=https://your-webhook.example.com/alerts

# Slack webhook URL for Slack notifications
QUERY_ALERTING_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK

# Timeout for notification delivery (default: 10 seconds)
QUERY_ALERTING_NOTIFICATION_TIMEOUT_SECONDS=10
```

### YAML Configuration

```yaml
alerting:
  enabled: true
  check_interval_seconds: 30
  webhook_url: "https://your-webhook.example.com/alerts"
  slack_webhook_url: "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
  notification_timeout_seconds: 10
```

## Alert Definition

### Alert Structure

```json
{
  "id": "a1b6e360-3c35-4d63-87fd-03b27ef77d1f",
  "name": "Suspicious admin logins",
  "description": "Detects admin logins from unusual geographies",
  "query": "severity:high AND class_name:\"Authentication\"",
  "severity": "high",
  "schedule": {
    "interval_minutes": 5,
    "lookback_minutes": 15
  },
  "status": "active",
  "owner": "soc@telhawk.local",
  "last_triggered_at": "2025-11-07T09:00:00Z"
}
```

### Alert Fields

- **id**: Unique identifier (UUID)
- **name**: Human-readable alert name
- **description**: Optional description
- **query**: OpenSearch query string (supports full query_string syntax)
- **severity**: Alert severity (`critical`, `high`, `medium`, `low`, `info`)
- **schedule**:
  - `interval_minutes`: How often to run the alert (minimum 1 minute)
  - `lookback_minutes`: Time window to search (defaults to interval if not set)
- **status**: `active` (scheduled) or `paused` (not scheduled)
- **owner**: Optional owner identifier
- **last_triggered_at**: Last time the alert found results (auto-updated)

## API Endpoints

### List Alerts

```bash
GET /api/v1/alerts
```

**Response:**
```json
{
  "alerts": [
    {
      "id": "...",
      "name": "Suspicious admin logins",
      "status": "active",
      ...
    }
  ]
}
```

### Create Alert

```bash
POST /api/v1/alerts
Content-Type: application/json

{
  "name": "Failed SSH Logins",
  "description": "Detects multiple failed SSH authentication attempts",
  "query": "class_name:\"Authentication\" AND activity_name:\"Logon\" AND status:\"Failure\" AND metadata.product.name:\"SSH\"",
  "severity": "medium",
  "schedule": {
    "interval_minutes": 10,
    "lookback_minutes": 30
  },
  "status": "active",
  "owner": "security-team@example.com"
}
```

**Response:** HTTP 201 with created alert

### Update Alert

```bash
POST /api/v1/alerts
Content-Type: application/json

{
  "id": "a1b6e360-3c35-4d63-87fd-03b27ef77d1f",
  "name": "Updated Alert Name",
  ...
}
```

**Response:** HTTP 200 with updated alert

### Get Alert

```bash
GET /api/v1/alerts/{alertId}
```

**Response:** HTTP 200 with alert details

### Patch Alert Status

```bash
PATCH /api/v1/alerts/{alertId}
Content-Type: application/json

{
  "status": "paused"
}
```

**Response:** HTTP 200 with updated alert

## Notification Channels

### Webhook Notifications

Generic webhook notifications are sent as HTTP POST with JSON payload:

```json
{
  "alert_id": "a1b6e360-3c35-4d63-87fd-03b27ef77d1f",
  "alert_name": "Suspicious admin logins",
  "severity": "high",
  "description": "Detects admin logins from unusual geographies",
  "query": "severity:high AND class_name:\"Authentication\"",
  "result_count": 5,
  "results": [
    {
      "time": 1699360800,
      "class_name": "Authentication",
      "severity_id": 4,
      ...
    }
  ],
  "timestamp": "2025-11-07T09:15:00Z"
}
```

### Slack Notifications

Slack notifications use the Incoming Webhooks feature:

1. Go to https://api.slack.com/messaging/webhooks
2. Create an incoming webhook for your workspace
3. Set `QUERY_ALERTING_SLACK_WEBHOOK_URL` to the webhook URL

Slack messages include:
- Alert name and severity
- Result count
- Query text
- Color-coded by severity:
  - Critical: Dark red (#8B0000)
  - High: Red (#FF0000)
  - Medium: Orange (#FFA500)
  - Low: Yellow (#FFFF00)
  - Info: Blue (#0000FF)

### Log Notifications

All alerts also log to the service's standard output for debugging:

```
2025/11/07 09:15:00 ALERT TRIGGERED: Suspicious admin logins (id=a1b6e360..., severity=high, results=5)
```

### Multi-Channel Delivery

The system supports sending to multiple channels simultaneously. If any channel succeeds, the alert is considered delivered. Individual channel failures are logged but don't block other channels.

## Metrics

Alert metrics are exposed via the `/healthz` endpoint:

```bash
GET /healthz
```

**Response:**
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime_seconds": 3600,
  "scheduler": {
    "alerts_triggered": 42,
    "alert_executions": 120,
    "alert_errors": 2,
    "notifications_sent": 42,
    "notification_errors": 0,
    "last_check_time": "2025-11-07T09:15:00Z",
    "active_alert_count": 8
  }
}
```

### Metric Definitions

- **alerts_triggered**: Number of times alerts found results
- **alert_executions**: Total number of alert query executions
- **alert_errors**: Number of failed alert executions
- **notifications_sent**: Number of successful notifications delivered
- **notification_errors**: Number of failed notification deliveries
- **last_check_time**: Last time scheduler checked for active alerts
- **active_alert_count**: Number of currently scheduled alerts

## Scheduler Behavior

### Alert Activation

1. Scheduler checks for active alerts every `check_interval_seconds` (default: 30s)
2. For each alert with `status: "active"`:
   - Creates a timer that fires every `interval_minutes`
   - Timer executes the alert query with the specified lookback window
3. Alerts with `status: "paused"` are ignored

### Alert Execution

When an alert timer fires:
1. Execute search query against OpenSearch with time range `[now - lookback, now]`
2. If results > 0:
   - Send notifications to all configured channels
   - Update `last_triggered_at` timestamp
   - Increment metrics counters
3. If results = 0:
   - No action taken (silent)

### Error Handling

- **Query failures**: Logged and counted in `alert_errors` metric
- **Notification failures**: Logged and counted in `notification_errors` metric
- **Network errors**: Automatic retry on next interval
- **Alert not found**: Timer is stopped and removed

### Graceful Shutdown

On service shutdown:
1. Stop accepting new alert checks
2. Stop all active alert timers
3. Wait for in-flight executions to complete (with timeout)
4. Log final metrics

## Example Queries

### Failed Authentication Attempts

```json
{
  "query": "class_name:\"Authentication\" AND activity_name:\"Logon\" AND status:\"Failure\"",
  "schedule": {
    "interval_minutes": 5,
    "lookback_minutes": 10
  }
}
```

### Suspicious Network Activity

```json
{
  "query": "class_name:\"Network Activity\" AND severity_id:>=4 AND dst.port:(22 OR 3389 OR 445)",
  "schedule": {
    "interval_minutes": 10,
    "lookback_minutes": 30
  }
}
```

### Detection Findings

```json
{
  "query": "class_name:\"Detection Finding\" AND confidence_id:>=3",
  "schedule": {
    "interval_minutes": 1,
    "lookback_minutes": 5
  }
}
```

### High Severity Events

```json
{
  "query": "severity_id:>=4",
  "schedule": {
    "interval_minutes": 3,
    "lookback_minutes": 10
  }
}
```

## Testing

Run the scheduler tests:

```bash
cd query
go test -v ./internal/scheduler/...
```

Test with a local alert:

```bash
# Enable alerting
export QUERY_ALERTING_ENABLED=true
export QUERY_ALERTING_CHECK_INTERVAL_SECONDS=30

# Start search service
go run cmd/query/main.go

# Create test alert
curl -X POST http://localhost:8082/api/v1/alerts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Alert",
    "query": "*",
    "severity": "info",
    "schedule": {
      "interval_minutes": 1,
      "lookback_minutes": 5
    },
    "status": "active"
  }'
```

## Best Practices

1. **Alert Naming**: Use descriptive names that indicate what is being detected
2. **Lookback Windows**: Set lookback >= interval to avoid missing events
3. **Query Optimization**: Use specific field queries rather than wildcard searches
4. **Status Management**: Use `paused` status during maintenance rather than deleting alerts
5. **Notification Testing**: Test webhook endpoints before enabling production alerts
6. **Severity Levels**: Use consistent severity assignments across alerts
7. **Owner Assignment**: Assign alerts to teams/individuals for accountability

## Troubleshooting

### Alerts Not Firing

1. Check `/healthz` to see if scheduler is running
2. Verify alert status is `active`
3. Check `alert_executions` metric is incrementing
4. Manually run the alert query via `/api/v1/search` to verify results
5. Review logs for error messages

### Notifications Not Delivered

1. Check `notification_errors` metric
2. Verify webhook URLs are accessible
3. Test webhook endpoints independently
4. Check service logs for delivery failures
5. Verify notification timeout is sufficient

### High Alert Volume

1. Review `alert_executions` and adjust intervals
2. Add more specific filters to queries
3. Increase lookback windows to reduce execution frequency
4. Consider aggregating similar alerts

## Future Enhancements

- Persistent alert storage (PostgreSQL)
- Email notification channel
- Alert templates and macros
- Alert correlation and suppression
- Notification rate limiting
- Alert escalation policies
- Web UI for alert management
