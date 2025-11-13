# Structured Logging

TelHawk Stack uses Go's standard library `log/slog` for structured, context-aware logging across all services.

## Overview

All services implement structured logging with:
- **JSON output** by default (configurable to text format)
- **Request ID correlation** via middleware
- **Service tagging** for multi-service deployments
- **Log levels**: debug, info, warn, error
- **Context propagation** for distributed tracing

## Configuration

All services support logging configuration via YAML config or environment variables:

```yaml
logging:
  level: info    # debug, info, warn, error
  format: json   # json or text
```

Environment variables:
```bash
# Per-service configuration
AUTH_LOGGING_LEVEL=debug
AUTH_LOGGING_FORMAT=json

QUERY_LOGGING_LEVEL=info
QUERY_LOGGING_FORMAT=text

INGEST_LOGGING_LEVEL=warn
INGEST_LOGGING_FORMAT=json
```

## Using the Logger

### In Service Code

```go
import (
    "log/slog"
    "github.com/telhawk-systems/telhawk-stack/common/logging"
)

// Initialize in main.go
logger := logging.New(
    logging.ParseLevel(cfg.Logging.Level),
    cfg.Logging.Format,
).With(logging.Service("auth"))
logging.SetDefault(logger)

// Use structured logging
slog.Info("User created",
    slog.String("user_id", user.ID.String()),
    slog.String("username", user.Username),
)
```

### Context-Aware Logging

The logger automatically extracts request IDs from context:

```go
import (
    "log/slog"
    "github.com/telhawk-systems/telhawk-stack/common/logging"
)

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    logger := logging.Default()

    // Automatically includes request_id from r.Context()
    logger.InfoContext(r.Context(), "Creating user",
        slog.String("username", req.Username),
        slog.String("ip", ipAddress),
    )
}
```

### Standard Fields

Use the common field helpers for consistency:

```go
import "github.com/telhawk-systems/telhawk-stack/common/logging"

slog.Info("Request processed",
    logging.UserID(userID),
    logging.IP(ipAddress),
    logging.Method(r.Method),
    logging.Path(r.URL.Path),
    logging.Status(http.StatusOK),
    logging.Duration(latencyMS),
)
```

### Available Field Helpers

| Helper | Field Name | Description |
|--------|------------|-------------|
| `Service(name)` | `service` | Service identifier |
| `UserID(id)` | `user_id` | User UUID |
| `Username(name)` | `username` | Username |
| `IP(ip)` | `ip` | IP address |
| `Method(method)` | `method` | HTTP method |
| `Path(path)` | `path` | HTTP path |
| `Status(code)` | `status` | HTTP status code |
| `Duration(ms)` | `duration_ms` | Duration in milliseconds |
| `Error(err)` | `error` | Error message |
| `TokenID(id)` | `token_id` | HEC token ID |
| `EventID(id)` | `event_id` | Event ID |
| `Query(q)` | `query` | Query string |

## Log Levels

Use appropriate log levels for different scenarios:

### Debug
Low-level debugging information, verbose output.
```go
slog.Debug("Query translation details",
    slog.Any("original_query", q),
    slog.Any("opensearch_dsl", osQuery),
)
```

### Info
Normal operational messages.
```go
slog.Info("Service started",
    slog.Int("port", cfg.Server.Port),
    slog.String("version", version),
)
```

### Warn
Warning conditions that should be investigated.
```go
slog.Warn("Rate limit approaching",
    slog.Int("current_rate", currentRate),
    slog.Int("limit", maxRate),
)
```

### Error
Error conditions that need attention.
```go
slog.Error("Database connection failed",
    slog.String("error", err.Error()),
    slog.String("host", dbHost),
)
```

## Request ID Middleware

Request IDs are automatically handled by the existing middleware in `common/middleware/requestid.go`:

1. Checks for existing `X-Request-ID` header
2. Generates new UUID if not present
3. Adds to response header
4. Stores in request context
5. Logger automatically extracts for correlation

No additional code needed - just use `logger.InfoContext(r.Context(), ...)` in handlers.

## Output Format

### JSON (Default)

Production-friendly, parseable format:

```json
{
  "time": "2025-01-13T10:05:35Z",
  "level": "INFO",
  "msg": "User login successful",
  "service": "auth",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "username": "admin",
  "ip": "192.168.1.100"
}
```

### Text (Development)

Human-readable format for local development:

```
2025-01-13T10:05:35Z INFO User login successful service=auth request_id=550e8400-e29b-41d4-a716-446655440000 user_id=123e4567-e89b-12d3-a456-426614174000 username=admin ip=192.168.1.100
```

## Best Practices

### DO

✅ Use structured fields instead of string interpolation:
```go
// Good
slog.Info("User created", slog.String("user_id", userID))

// Bad
slog.Info(fmt.Sprintf("User created: %s", userID))
```

✅ Use context-aware logging in HTTP handlers:
```go
logger.InfoContext(r.Context(), "Processing request")
```

✅ Include relevant context with error logs:
```go
slog.Error("Database query failed",
    slog.String("error", err.Error()),
    slog.String("query", queryStr),
    slog.Int("retry_count", retries),
)
```

✅ Use standard field names for consistency:
```go
slog.Info("Request complete",
    logging.Method(r.Method),
    logging.Path(r.URL.Path),
    logging.Status(statusCode),
    logging.Duration(latencyMS),
)
```

### DON'T

❌ Don't use `log.Printf()` or `fmt.Println()`:
```go
// Bad
log.Printf("User created: %s", userID)
fmt.Println("Debug info")

// Good
slog.Info("User created", slog.String("user_id", userID))
slog.Debug("Debug info")
```

❌ Don't log sensitive data:
```go
// Bad - logs password
slog.Info("Login attempt", slog.String("password", password))

// Good - no sensitive data
slog.Info("Login attempt", slog.String("username", username))
```

❌ Don't use inconsistent field names:
```go
// Bad - multiple names for same concept
slog.Info("Event 1", slog.String("userId", id))
slog.Info("Event 2", slog.String("user", id))
slog.Info("Event 3", slog.String("uid", id))

// Good - consistent field name
slog.Info("Event 1", logging.UserID(id))
slog.Info("Event 2", logging.UserID(id))
slog.Info("Event 3", logging.UserID(id))
```

## Observability

Structured logs enable:

1. **Log aggregation** - Parse JSON logs with ELK, Splunk, or similar
2. **Request tracing** - Follow requests across services via request_id
3. **Performance monitoring** - Track duration_ms fields
4. **Error analysis** - Filter by level=ERROR and analyze error patterns
5. **User activity** - Track actions by user_id

## Migration from Old Logging

Replace unstructured logging:

```go
// Before
log.Printf("User %s logged in from %s", username, ip)

// After
slog.Info("User login",
    logging.Username(username),
    logging.IP(ip),
)
```

Replace debug print statements:

```go
// Before
fmt.Printf("DEBUG: Query: %+v\n", query)

// After
slog.Debug("Query details", slog.Any("query", query))
```

## Testing

In tests, you can use a test logger or disable logging:

```go
import (
    "log/slog"
    "io"
)

func TestSomething(t *testing.T) {
    // Disable logging in tests
    slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

    // Or use a test logger
    var buf bytes.Buffer
    logger := slog.New(slog.NewJSONHandler(&buf, nil))

    // Run tests...
}
```

## See Also

- [Go slog documentation](https://pkg.go.dev/log/slog)
- `common/logging/` - Logging package source code
- `common/middleware/requestid.go` - Request ID middleware
- [CONFIGURATION.md](./CONFIGURATION.md) - Service configuration
