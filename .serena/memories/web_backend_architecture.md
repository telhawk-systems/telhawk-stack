# Web Backend Service Architecture

## Overview
The web backend is a Go HTTP server on port 3000 that serves the React frontend and provides API proxy/gateway functionality to backend services.

## Current HTTP Architecture

### Main.go Initialization Pattern
- **Environment-based configuration** (no config file support yet):
  - Port (default: 3000)
  - Service URLs: SearchServiceURL, CoreServiceURL, RulesServiceURL, AlertingServiceURL, AuthenticateServiceURL
  - NATS URL (optional, default: nats://nats:4222)
  - Cookie domain/secure flags
  - Dev mode flag
  
- **Proxy instances created** for each backend service:
  - `searchProxy` → Search service (8082)
  - `coreProxy` → Core service (8090)
  - `authenticateProxy` → Authenticate service (8080)
  - `rulesProxy` → Rules service (8084)
  - `alertingProxy` → Alerting service (8085)

- **NATS client initialization** (graceful degradation):
  - Attempts to connect at startup
  - Falls back to warnings if unavailable
  - Only enables AsyncQueryHandler if NATS succeeds
  - Proper cleanup with defer/Drain() on shutdown

## Proxy Architecture

### File: `internal/proxy/proxy.go`
- Simple HTTP reverse proxy implementation
- `NewProxy(targetURL, authClient)` pattern
- **Request handling**:
  - Copies method, path, query string
  - Forwards all headers
  - Injects Authorization header from `access_token` cookie if not present
  - Injects `X-User-ID` and `X-User-Roles` headers from context
  - Forwards entire response (status, headers, body)

- **30-second HTTP timeout**

### Routes: `internal/server/router.go`
```
Protected endpoints (behind auth middleware):
- /api/auth/* → authenticateProxy
- /api/search/* → searchProxy  
- /api/core/* → coreProxy
- /api/rules/* → rulesProxy
- /api/alerting/* → alertingProxy

Public endpoints:
- GET /api/auth/csrf-token (AuthHandler.GetCSRFToken)
- POST /api/auth/login (AuthHandler.Login)
- POST /api/auth/logout (AuthHandler.Logout)

Protected auth endpoints:
- GET /api/auth/me (AuthHandler.Me)
- GET /api/dashboard/metrics (DashboardHandler.GetMetrics) - with caching

Async Query endpoints (optional, only if NATS available):
- POST /api/async-query/submit (AsyncQueryHandler.SubmitQuery)
- GET /api/async-query/status/{id} (AsyncQueryHandler.GetQueryStatus)

- GET /api/health (simple health check)
- / (SPA static files)
```

## Authentication Flow

### Auth Client: `internal/auth/client.go`
- HTTP client to authenticate service
- 10-second timeout
- Methods:
  - `Login(username, password)` → POST `/api/v1/auth/login`
  - `ValidateToken(token)` → POST `/api/v1/auth/validate`
  - `RefreshToken(refreshToken)` → POST `/api/v1/auth/refresh`
  - `RevokeToken(token)` → POST `/api/v1/auth/revoke`

### Auth Middleware: `internal/auth/middleware.go`
- Protect handler that:
  1. Extracts token from Bearer header OR access_token cookie
  2. Validates token via auth client
  3. If invalid, attempts refresh with refresh_token cookie
  4. Sets new access token cookie if refreshed
  5. Injects UserID and Roles into request context
  6. Returns 401 if all fails
- Token in context available via `GetUserID()` and `GetRoles()` helpers

## Handlers

### Dashboard Handler: `internal/handlers/dashboard.go`
- GET /api/dashboard/metrics
- **5-minute cache** (configurable via DASHBOARD_CACHE_SECONDS env)
- **Direct HTTP call to search service**: POST `/api/v1/events/query`
- Sends JSON:API formatted query with aggregations
- Extracts meta.aggregations and meta.total, transforms to legacy shape
- Injects access_token from cookie into downstream request
- Returns cached results with X-Cache: HIT/MISS headers

### Auth Handler: `internal/handlers/auth.go`
- GetCSRFToken: Returns empty token (Go 1.25 uses header-based validation)
- Login: Calls auth client, sets HTTP-only cookies, returns different responses for CLI vs browser
- Logout: Revokes refresh token, clears cookies
- Me: Returns logged-in user info from context

### Async Query Handler: `internal/handlers/async_query.go`
- **NATS integration point**
- SubmitQuery (POST):
  - Accepts query JSON (query, time_range, limit)
  - Generates UUID query ID
  - Publishes to NATS subject: `search.jobs.query`
  - Stores pending result in in-memory cache
  - Returns immediately with 202 Accepted
  
- GetQueryStatus (GET):
  - Returns status from in-memory cache
  - Cache stores: status (pending/complete/failed), data, error, created_at
  - 5-minute TTL with background cleanup goroutine
  
- **Cache is in-memory map with RWMutex** (not persistent)
- Publishes QueryJobMessage to NATS with queryID, query, timeRange, limit

## Middleware Stack
1. **RequestID** (from common/middleware) - top level
2. **CORS** - allows http://localhost:5173 (Vite dev server)
3. **Security Headers**
4. **CSRF** - Go 1.25 header-based (Sec-Fetch-Site validation)

## NATS Usage

### Current Integration Points:
1. **Async Query Handler** (web backend):
   - Publishes to `search.jobs.query` subject
   - Expects responses via query results mechanism (not yet implemented for consumption)

2. **Respond Service** (for reference):
   - Has NATS Publisher and Handler infrastructure
   - Uses scheduler for correlation rules
   - Subscribes to events and alert jobs

### Missing/TODO:
- AsyncQueryHandler doesn't subscribe to results (one-way publish only)
- No mechanism to update in-memory cache when search service completes
- No persistent result storage between restarts
- Search service would need to publish results back to web for polling

## Configuration Pattern

### Environment Variables:
- `WEB_PORT=3000`
- `STATIC_DIR=./static`
- `AUTHENTICATE_SERVICE_URL=http://authenticate:8080`
- `SEARCH_SERVICE_URL=http://search:8082`
- `CORE_SERVICE_URL=http://core:8090`
- `RULES_SERVICE_URL=http://rules:8084`
- `ALERTING_SERVICE_URL=http://alerting:8085`
- `NATS_URL=nats://nats:4222`
- `COOKIE_DOMAIN=""` (optional)
- `COOKIE_SECURE=true`
- `DEV_MODE=false`
- `DASHBOARD_CACHE_SECONDS=300`

### Config Struct:
All fields in main.go `Config` struct, loaded via `loadConfig()` helper

## Integration Points for V2 Architecture

### HTTP Proxy-based (Current):
- ✓ Search service proxying
- ✓ Respond/Rules/Alerting service proxying  
- ✓ Core service proxying (placeholder)

### NATS-based (Partial):
- Async query submission (working)
- Need: Result consumption and cache update mechanism
- Need: Subscribe to search service results (likely new subject)
- Need: Handle result message format/schema

## Key Observations

1. **Proxy pattern is simple but effective**: No transformation, just headers injection
2. **Auth is centralized**: All auth decisions via authenticate service
3. **NATS integration is incomplete**: Publish-only, no subscription/result handling
4. **In-memory caching**: Dashboard and async query results cached in-memory
5. **No database**: Web backend is stateless except for volatile caches
6. **Graceful NATS degradation**: Service works without NATS, just disables async query
7. **No code generation**: Uses standard Go patterns, no special code gen
