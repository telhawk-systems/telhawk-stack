# NATS Message Broker Connection Handling Analysis

## Summary
The TelHawk stack currently has **limited NATS integration** - only the Web service uses NATS for async query support. No production services are connected yet, though the architecture supports it via JetStream.

---

## 1. Services Using NATS

### Currently Connected:
- **Web Service** (`web/backend/cmd/web/main.go`):
  - Uses NATS for async query job submission
  - Connection is OPTIONAL (degrades gracefully if unavailable)
  - Default URL: `nats://nats:4222`
  - Environment variable: `NATS_URL`

### Not Using NATS:
- Respond service (no NATS integration)
- Search service (no NATS integration)
- Ingest service (no NATS integration)
- Authenticate service (no NATS integration)

---

## 2. NATS Connection Configuration

### Default Configuration (from `common/messaging/nats/client.go`):
```
URL:           nats.DefaultURL (localhost:4222)
Name:          "telhawk-client"
MaxReconnects: -1 (infinite reconnects)
ReconnectWait: 2 seconds
Timeout:       5 seconds
```

### Connection Options Applied:
- `nats.Name()` - Client name for connection identification
- `nats.MaxReconnects()` - Infinite retry attempts by default
- `nats.ReconnectWait()` - 2-second backoff between reconnects
- `nats.Timeout()` - 5-second connection timeout
- `nats.DisconnectErrHandler()` - Logs disconnect errors to stdout
- `nats.ReconnectHandler()` - Logs successful reconnects to stdout
- `nats.UserInfo()` - Username/password auth (optional)
- `nats.Token()` - Token-based auth (optional)

### Web Service Configuration:
```go
natsCfg := nats.DefaultConfig()
natsCfg.URL = cfg.NATSURL  // from NATS_URL env var
natsCfg.Name = "telhawk-web"
```

---

## 3. Reconnection Logic

### NATS Client-Level Reconnection:
- **MaxReconnects**: Set to -1 (infinite reconnection attempts)
- **ReconnectWait**: 2 seconds between attempts
- **Handled by**: nats-io/nats.go library (automatic)

### Web Service Connection Handling:
```go
if cfg.NATSURL != "" {
    natsCfg := nats.DefaultConfig()
    natsCfg.URL = cfg.NATSURL
    natsClient, err = nats.NewClient(natsCfg)
    if err != nil {
        log.Printf("Warning: Failed to connect to NATS at %s: %v", cfg.NATSURL, err)
        log.Printf("Async query support will be disabled")
        // Service continues without NATS
    }
}
```

### Connection Loss Handling:
- Web service publishes to NATS with 5-second context timeout
- If publish fails, HTTP 500 error returned to client
- No retry mechanism at application level - delegates to NATS library

### Reconnect Handlers:
```go
nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
    if err != nil {
        fmt.Printf("NATS disconnected: %v\n", err)
    }
})
nats.ReconnectHandler(func(_ *nats.Conn) {
    fmt.Println("NATS reconnected")
})
```

**Issue**: Both handlers use `fmt.Printf/Println` instead of service logger - loses context and severity.

---

## 4. Health Checks for NATS Connectivity

### Web Service:
```go
// On startup
if natsClient != nil {
    defer func() {
        if err := natsClient.Drain(); err != nil {
            log.Printf("Error draining NATS connection: %v", err)
        }
    }()
}
```

### Common Messaging Health Check Function:
Located in `common/messaging/health.go`

```go
func CheckClientHealth(ctx context.Context, client Client) HealthStatus {
    status := HealthStatus{}
    
    if client == nil {
        status.Error = "client is nil"
        return status
    }
    
    status.Connected = client.IsConnected()
    if !status.Connected {
        status.Error = "not connected to message broker"
        return status
    }
    
    // Measure latency with a request to internal subject
    start := time.Now()
    _, err := client.Request(ctx, "_HEALTH.ping", []byte("ping"), 2*time.Second)
    status.Latency = time.Since(start)
    
    // No responder is OK for health check
    if err != nil && status.Connected {
        status.Error = ""
    } else if err != nil {
        status.Error = fmt.Sprintf("health check failed: %v", err)
    }
    
    return status
}
```

**Status**: Available but NOT used by any service.

### NATS Container Health Check:
```yaml
healthcheck:
  test: ["CMD", "wget", "-q", "-O", "-", "http://localhost:8222/healthz"]
  interval: 10s
  timeout: 5s
  retries: 5
  start_period: 5s
```

**Note**: Tests NATS monitoring API, not client connectivity.

---

## 5. Message Publishing/Subscribing Failure Handling

### Publish Failures:
```go
// In AsyncQueryHandler.SubmitQuery()
if err := h.publisher.Publish(ctx, h.subject, data); err != nil {
    http.Error(w, fmt.Sprintf("failed to submit query: %v", err), http.StatusInternalServerError)
    return
}
```

**Strategy**: Fail-fast HTTP 500 error to client. No automatic retries.

### Subscribe Failures:
```go
// In nats.Client.Subscribe()
sub, err := c.conn.Subscribe(subject, func(msg *nats.Msg) {
    ctx := context.Background()
    if err := handler(ctx, natsToMessage(msg)); err != nil {
        fmt.Printf("Handler error for %s: %v\n", subject, err)
    }
})
if err != nil {
    return nil, err  // Subscription setup fails
}
```

**Strategy**: Subscription setup fails immediately. Handler errors logged but not retried.

### Missing Features:
- No application-level retry with exponential backoff
- No circuit breaker pattern
- No message buffering/queue persistence at application level
- No acknowledgment (ACK) mechanism for delivered messages
- No DLQ for failed publishes

---

## 6. JetStream Support (Unused)

Located in `common/messaging/nats/jetstream.go`

### Features:
- Persistent message storage
- Durable consumers with ACK/NAK
- Consumer redelivery with configurable attempts
- MaxDeliver: 3 (default), MaxAckPending: 100
- AckWait: 30 seconds

### Predefined Streams:
```go
SearchJobsStream = StreamConfig{
    Name:      "SEARCH_JOBS",
    Subjects:  []string{"search.jobs.>"},
    MaxAge:    1 * time.Hour,
    MaxBytes:  100 * 1024 * 1024,
    MaxMsgs:   10000,
    Retention: jetstream.WorkQueuePolicy,
    Storage:   jetstream.FileStorage,
}

SearchResultsStream = StreamConfig{
    Name:      "SEARCH_RESULTS",
    Subjects:  []string{"search.results.>"},
    MaxAge:    1 * time.Hour,
    MaxBytes:  500 * 1024 * 1024,
    MaxMsgs:   10000,
    Retention: jetstream.InterestPolicy,
    Storage:   jetstream.FileStorage,
}

RespondEventsStream = StreamConfig{
    Name:      "RESPOND_EVENTS",
    Subjects:  []string{"respond.>"},
    MaxAge:    24 * time.Hour,
    MaxBytes:  100 * 1024 * 1024,
    MaxMsgs:   100000,
    Retention: jetstream.InterestPolicy,
    Storage:   jetstream.FileStorage,
}
```

**Status**: Defined but not created or used by any service.

---

## 7. Message Subjects and Queue Groups

### Message Subjects (domain.action.resource pattern):

**Search Domain**:
- `search.jobs.query` - Ad-hoc search requests
- `search.jobs.correlate` - Correlation evaluation requests
- `search.results.query` - Ad-hoc query results (append .{id})
- `search.results.correlate` - Correlation match results

**Respond Domain**:
- `respond.alerts.created` - New alert created
- `respond.alerts.updated` - Alert status changed
- `respond.cases.created` - New case opened
- `respond.cases.updated` - Case status changed
- `respond.cases.assigned` - Case assigned to analyst

### Queue Groups (load-balanced consumers):
- `search-workers` - Pool of search/correlation workers
- `respond-workers` - Pool of alert/case processors
- `web-workers` - Pool of web notification handlers

---

## 8. Connection Resilience Issues

### Critical Issues:
1. **No automatic reconnection at application level** - relies on nats-io library
2. **Connection initialization is synchronous and blocking** - failure stops service startup (except Web which degrades gracefully)
3. **No persistent health monitoring** - no active health checks during runtime
4. **Handler logs use fmt.Printf** - loses service logger context and metrics
5. **No message acknowledgment/persistence** - messages can be lost if service dies
6. **No exponential backoff on publish failures** - immediate HTTP 500 errors
7. **Graceful shutdown doesn't wait for in-flight messages** - only drains connection

### Web Service Specific:
1. Connection is OPTIONAL but not monitored - no health check endpoint
2. AsyncQueryHandler has in-memory cache only - results lost on restart
3. No indication to client if NATS is unavailable vs. query failed
4. No circuit breaker for downstream search service

### Logging Issues:
- Disconnect/reconnect events printed to stdout without timestamp or severity
- Handler errors printed instead of logged with proper context
- No metrics collection for connection events or failures

---

## 9. Docker Compose Configuration

### NATS Service:
```yaml
nats:
  image: nats:latest
  container_name: telhawk-nats
  command: ["--jetstream", "--store_dir=/data"]
  expose:
    - "4222"
  ports:
    - "127.0.0.1:8222:8222"  # Management API
  volumes:
    - nats-data:/data
  healthcheck:
    test: ["CMD", "wget", "-q", "-O", "-", "http://localhost:8222/healthz"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 5s
  restart: unless-stopped
```

### Web Service Configuration:
```yaml
environment:
  - NATS_URL=nats://nats:4222
depends_on:
  - nats
```

**Note**: Web service depends on NATS being healthy, but handles graceful degradation.

---

## 10. Current Messaging Interface

### Publisher Interface:
```go
type Publisher interface {
    Publish(ctx context.Context, subject string, data []byte) error
    PublishMsg(ctx context.Context, msg *Message) error
    Request(ctx context.Context, subject string, data []byte, timeout time.Duration) (*Message, error)
    Close() error
}
```

### Subscriber Interface:
```go
type Subscriber interface {
    Subscribe(subject string, handler MessageHandler) (Subscription, error)
    QueueSubscribe(subject, queue string, handler MessageHandler) (Subscription, error)
    Close() error
}
```

### Client Interface (combines both):
```go
type Client interface {
    Publisher
    Subscriber
    Drain() error
    IsConnected() bool
}
```

---

## Recommendations for Production Readiness

### Immediate Fixes:
1. Replace fmt.Printf with service logger in disconnect/reconnect handlers
2. Add health check endpoints to services using NATS
3. Implement circuit breaker pattern for message publishing
4. Add observability/metrics for connection state and message operations

### Short-term Improvements:
1. Enable JetStream for critical message types
2. Add application-level retry logic with exponential backoff
3. Implement message acknowledgment patterns
4. Add persistent health monitoring during service runtime

### Long-term Architecture:
1. Extend all production services to use NATS (search, respond, ingest)
2. Implement cross-service communication patterns
3. Add message correlation and request/reply patterns
4. Consider CDC (Change Data Capture) patterns for state synchronization
