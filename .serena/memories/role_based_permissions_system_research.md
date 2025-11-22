# TelHawk Role-Based Permissions System - Research & Architecture

## Current State Analysis

### 1. User Model & Roles Storage

**User Struct** (`authenticate/internal/models/user.go`):
```go
type User struct {
    ID           string     // UUID v7
    Username     string     // Unique
    Email        string     // Unique
    PasswordHash string     // Bcrypt
    Roles        []string   // Array of role strings
    CreatedAt    time.Time
    DisabledAt   *time.Time // Lifecycle timestamp (soft disable)
    DisabledBy   *string    // Who disabled the user
    DeletedAt    *time.Time // Lifecycle timestamp (soft delete)
    DeletedBy    *string    // Who deleted the user
}
```

**Current Roles** (defined in `authenticate/internal/models/user.go`):
- `admin` - Full system access
- `analyst` - Investigation capabilities
- `viewer` - Read-only access
- `ingester` - Data ingestion only

**Role Storage**:
- Stored in PostgreSQL `users` table as `TEXT[]` (PostgreSQL array type)
- Roles array: `ARRAY['admin']` or `ARRAY['analyst', 'viewer']`
- Immutable pattern: roles only change via UPDATE (never append-only like audit)

### 2. Database Schema

**PostgreSQL `users` table** (`authenticate/migrations/001_init.up.sql`):
- `roles TEXT[] NOT NULL DEFAULT '{}'` - Array column
- Lifecycle timestamps: `disabled_at`, `deleted_at` (no boolean flags)
- User can't disable/delete themselves (CHECK constraint)
- Indexes on: `username`, `email`, `active` (WHERE clause)
- Audit log table captures all auth/user events

**Sessions & HEC Tokens**: Have their own lifecycle timestamps but not roles

### 3. JWT Token Structure

**JWT Claims** (`authenticate/pkg/tokens/jwt.go`):
```go
type Claims struct {
    UserID string   // In token
    Roles  []string // In token
    jwt.RegisteredClaims
}
```

**Key Points**:
- Roles embedded in JWT access token (15-minute TTL)
- Token includes `user_id` and `roles` claims
- No individual permissions in token (only roles)
- Refresh token is random string (not JWT)

### 4. Authentication Flow

**Web Frontend → Web Backend → Authenticate Service**:

1. **Login** (`POST /api/auth/login`):
   - Web Frontend sends credentials to Web Backend
   - Web Backend calls Authenticate Service (`/api/v1/auth/login`)
   - Returns JWT access token + refresh token
   - Tokens set as HTTP-only cookies (browser) or returned in body (CLI)

2. **Token Validation** (Web Backend Auth Middleware):
   - Middleware calls Authenticate Service (`/api/v1/auth/validate`)
   - Service validates JWT and returns: `{valid: bool, user_id: string, roles: []string}`
   - Roles extracted from JWT claims, validated
   - Context enriched: `UserIDKey`, `RolesKey`

3. **Service-to-Service Headers**:
   - Web Backend → Respond/Search/Authenticate services via proxy
   - Headers injected: `X-User-ID` (user ID), `X-User-Roles` (comma-separated)
   - These headers used for authorization checks in backend services

### 5. Current Permission Checks

**Authenticate Service** (`authenticate/internal/handlers/auth_handler.go`):
- **CreateUser**: Requires `X-User-Roles` header to contain "admin" (line 36)
  ```go
  if !strings.Contains(roles, "admin") {
      http.Error(w, "Forbidden: admin role required", http.StatusForbidden)
  }
  ```
- **ListHECTokens**: Admin users see all tokens with usernames; regular users see only their own (line 441-495)

**Respond Service** (`respond/internal/handlers/handlers.go`):
- **TODO comment on line 197**: "Get user ID from auth context" - not implemented yet
- No permission checks currently implemented
- All endpoints accessible to any authenticated user

**Search Service**: Similar - no role-based checks implemented

### 6. Web Backend Router Setup

**Router** (`web/backend/internal/server/router.go`):
- All endpoints (except login/CSRF) protected by `AuthMiddleware.Protect()`
- Middleware validates token and enriches context with `UserIDKey`, `RolesKey`
- Helper functions: `auth.GetUserID(ctx)`, `auth.GetRoles(ctx)` available in handlers
- Proxy layer (`proxy/proxy.go`) injects `X-User-ID` and `X-User-Roles` headers

**Web Backend Auth Middleware** (`web/backend/internal/auth/middleware.go`):
- Validates token via `authClient.ValidateToken(token)`
- Sets context values: `UserIDKey`, `RolesKey`
- Handles token refresh automatically
- Can get roles via: `auth.GetRoles(r.Context())` → `[]string`

### 7. Proxy Layer

**Proxy** (`web/backend/internal/proxy/proxy.go`):
- Forwards all headers from original request
- Injects `X-User-ID` header (from context)
- Injects `X-User-Roles` header (comma-separated from context)
- Forward all Authorization header if present
- Used for: authenticate, search, core, rules, alerting services

### 8. Existing Audit Trail

**Audit Table** (`authenticate/migrations/001_init.up.sql`):
- Captures: `actor_type`, `actor_id`, `actor_name`, `action`, `resource_type`, `resource_id`, `result`
- Actions: login, logout, user_create, token_operations, password_reset, etc.
- Metadata field (JSONB) for additional context
- All operations logged with IP, User-Agent, timestamp
- Signature field for tamper-proofing (HMAC)

**Audit Constants** (`authenticate/internal/models/audit.go`):
- Actions like: `ActionLogin`, `ActionUserCreate`, `ActionUpdate`, `ActionDelete`, `ActionPasswordReset`
- Results: `ResultSuccess`, `ResultFailure`
- Actor types: `ActorTypeUser`, `ActorTypeService`, `ActorTypeSystem`

## Architecture Observations

### Current Limitations

1. **No Resource-Level Permissions**: Only role-based checks exist
   - Can't say "analyst can view events but not alerts"
   - All role members have identical capabilities

2. **No Permission Caching**: Every request calls validate endpoint
   - 5-minute caching mentioned in CLAUDE.md but not visible in code
   - Could be in ingest service for HEC tokens

3. **Header-Based Auth in Services**: 
   - Backend services rely on `X-User-ID` and `X-User-Roles` headers
   - No cryptographic verification (trusts web backend)
   - Potential for header spoofing if network compromised

4. **Hardcoded String Checks**:
   - Line 36 of auth_handler.go: `strings.Contains(roles, "admin")`
   - Not reusable, spreads across handlers
   - Should centralize to permission checking utility

### Design Patterns in Use

1. **Immutable Pattern**: Lifecycle timestamps, no in-place updates for user status
2. **JSON:API**: Some endpoints return JSON:API format (CreateUser, CreateHECToken)
3. **Context-Based Authorization**: User info passed via context in web backend
4. **Header-Based Authorization**: Backend services read headers injected by proxy

### Integration Points

- **Web → Authenticate**: Direct HTTP calls for login/validate
- **Web → Services**: Proxy passes X-User-ID, X-User-Roles headers
- **Services → Service**: Direct HTTP with headers (or would need auth middleware)
- **Ingest → Authenticate**: Caches HEC token validation for 5 minutes
- **Audit Trail**: Service layer logs all auth events

## API Endpoints Needing Permission Gates

### Authenticate Service
- `POST /api/v1/users/create` - Create user [NEEDS: admin]
- `GET /api/v1/users` - List users [NEEDS: admin?]
- `GET /api/v1/users/{id}` - Get user [NEEDS: admin or own user?]
- `PUT /api/v1/users/{id}` - Update user [NEEDS: admin or own?]
- `DELETE /api/v1/users/{id}` - Delete user [NEEDS: admin]
- `POST /api/v1/users/{id}/reset-password` - Reset password [NEEDS: admin or self?]
- `POST /api/v1/hec/tokens/create` - Create HEC token [NEEDS: authenticated]
- `GET /api/v1/hec/tokens` - List HEC tokens [NEEDS: authenticated or admin]
- `DELETE /api/v1/hec/tokens/{id}/revoke` - Revoke HEC token [NEEDS: authenticated or admin]

### Respond Service
- `GET /schemas` - List detection rules [NEEDS: analyst+ or all users?]
- `POST /schemas` - Create rule [NEEDS: admin?]
- `GET /schemas/{id}` - Get rule [NEEDS: analyst+?]
- `PUT /schemas/{id}` - Update rule [NEEDS: admin?]
- `DELETE /schemas/{id}` - Hide rule [NEEDS: admin?]
- `PUT /schemas/{id}/disable` - Disable rule [NEEDS: admin?]
- `POST /schemas/{id}/enable` - Enable rule [NEEDS: admin?]
- All alert/case endpoints - Need checks

### Search Service
- All search endpoints - Need checks based on role

### Web Service
- Dashboard, saved searches, etc. - Need checks

## Recommendation for Implementation

### Phase 1: Foundation
1. Create permission check utilities and middleware
2. Define permission matrix (which roles can do what)
3. Implement in Authenticate service first
4. Add tests for permission enforcement

### Phase 2: Integration
1. Roll out to Respond service
2. Roll out to Search service
3. Add permission checks to Web service handlers

### Phase 3: Advanced Features
1. Resource-level permissions (if needed)
2. Permission caching
3. Audit trail queries for permission denied events
