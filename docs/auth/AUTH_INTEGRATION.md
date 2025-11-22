# Auth Integration and Token Lifecycle

## Overview

The TelHawk auth service provides centralized authentication and authorization for all services in the stack. It supports both **user authentication** (for web UI and API access) and **HEC token authentication** (for data ingestion).

## Authentication Flows

### 1. Web UI User Authentication

**Initial Login Flow:**
```
User Browser                  Web Service                Auth Service
     |                             |                           |
     |--- POST /login ------------>|                           |
     |    (username, password)     |                           |
     |                             |--- POST /api/v1/auth/login ->|
     |                             |                           |
     |                             |<-- JWT + Refresh Token ---|
     |<-- Set-Cookie: access_token |                           |
     |    Set-Cookie: refresh_token|                           |
     |                             |                           |
```

**Token Types:**
- **Access Token (JWT)**: Short-lived (15 minutes), contains user_id and roles
- **Refresh Token**: Long-lived (7 days), opaque random string for token renewal

**Authenticated Request Flow:**
```
User Browser                  Web Service                Auth Service
     |                             |                           |
     |--- GET /api/search -------->|                           |
     |    Cookie: access_token     |                           |
     |                             |--- POST /api/v1/auth/validate ->|
     |                             |    (access_token)         |
     |                             |<-- {valid: true, user_id, roles} |
     |                             |                           |
     |<-- Search Results ----------|                           |
     |                             |                           |
```

**Token Refresh Flow:**
```
User Browser                  Web Service                Auth Service
     |                             |                           |
     |--- GET /api/search -------->|                           |
     |    (expired access_token)   |                           |
     |                             |--- POST /api/v1/auth/refresh ->|
     |                             |    (refresh_token)        |
     |                             |<-- New JWT Access Token --|
     |<-- Set-Cookie: access_token |                           |
     |<-- Search Results ----------|                           |
     |                             |                           |
```

### 2. HEC Token Authentication (Data Ingestion)

**Token Creation (Admin Operation):**
```bash
POST /api/v1/auth/hec-tokens
Authorization: Bearer <admin-jwt>
Content-Type: application/json

{
  "name": "production-ingester",
  "expires_at": "2025-12-31T23:59:59Z"
}

# Response:
{
  "id": "uuid",
  "token": "base64-encoded-token",
  "name": "production-ingester",
  "user_id": "admin-user-id",
  "enabled": true,
  "created_at": "2024-11-05T00:00:00Z",
  "expires_at": "2025-12-31T23:59:59Z"
}
```

**HEC Token Usage in Ingestion:**
```
Data Source                   Ingest Service            Auth Service
     |                             |                           |
     |--- POST /services/collector ->|                          |
     |    Authorization: Telhawk token|                          |
     |    {event: "..."}            |                           |
     |                             |--- POST /api/v1/auth/hec-tokens/validate ->|
     |                             |    (token)                |
     |                             |<-- {valid: true, token_id, user_id} |
     |                             |                           |
     |                             |-- Forward to Core ------->|
     |<-- {ack: 0} ---------------|                           |
     |                             |                           |
```

## Token Details

### Access Token (JWT)

**Structure:**
```json
{
  "user_id": "uuid",
  "roles": ["analyst", "viewer"],
  "exp": 1730764800,
  "iat": 1730763900,
  "nbf": 1730763900,
  "iss": "telhawk-auth"
}
```

**Properties:**
- Algorithm: HS256 (HMAC-SHA256)
- Secret: Configurable via `ACCESS_SECRET` environment variable
- TTL: 15 minutes
- Claims: user_id, roles, standard JWT claims (exp, iat, nbf, iss)

### Refresh Token

**Properties:**
- Format: Base64-encoded 32-byte random value
- Storage: In-memory (production: PostgreSQL)
- TTL: 7 days
- Can be revoked explicitly via `/api/v1/auth/revoke`

### HEC Token

**Properties:**
- Format: Base64-encoded 32-byte random value
- Storage: In-memory (production: PostgreSQL)
- TTL: Configurable per-token (optional)
- Associated with a user account (typically an admin or service account)

## Roles and Permissions

| Role       | Permissions                                          |
|------------|------------------------------------------------------|
| `admin`    | Full system access, user management, HEC token creation |
| `analyst`  | Read/write queries, alerts, dashboards              |
| `viewer`   | Read-only access to searches and dashboards         |
| `ingester` | HEC token access only (no UI/API access)            |

## Web UI Integration Guide

### 1. Login Page

**Endpoint:** `POST /api/v1/auth/login`

**Request:**
```json
{
  "username": "analyst1",
  "password": "secure-password"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "random-base64-string",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

**Web Service Actions:**
1. Receive login credentials from browser
2. Forward to auth service
3. On success:
   - Store `access_token` in HTTP-only cookie (secure, SameSite=Strict)
   - Store `refresh_token` in HTTP-only cookie (secure, SameSite=Strict)
   - Redirect to dashboard

### 2. Authenticated Requests

**Middleware Pattern:**
```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract access token from cookie
        cookie, err := r.Cookie("access_token")
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Validate with auth service
        resp, err := validateToken(cookie.Value)
        if err != nil || !resp.Valid {
            // Try refresh flow
            refreshCookie, err := r.Cookie("refresh_token")
            if err != nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            newToken, err := refreshAccessToken(refreshCookie.Value)
            if err != nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            // Set new access token cookie
            http.SetCookie(w, &http.Cookie{
                Name:     "access_token",
                Value:    newToken,
                HttpOnly: true,
                Secure:   true,
                SameSite: http.SameSiteStrictMode,
                MaxAge:   900, // 15 minutes
            })
        }

        // Attach user context to request
        ctx := context.WithValue(r.Context(), "user_id", resp.UserID)
        ctx = context.WithValue(ctx, "roles", resp.Roles)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 3. Token Refresh

**Automatic Refresh Pattern:**
- Web service checks access token expiry before forwarding requests
- If expired but refresh token valid, automatically refresh
- Transparent to the user (no re-login required)

**Endpoint:** `POST /api/v1/auth/refresh`

**Request:**
```json
{
  "refresh_token": "refresh-token-from-cookie"
}
```

**Response:**
```json
{
  "access_token": "new-jwt-token",
  "refresh_token": "same-refresh-token",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

### 4. Logout

**Client-Side:** Clear cookies and redirect to login

**Optional Server-Side:** Revoke refresh token

**Endpoint:** `POST /api/v1/auth/revoke`

**Request:**
```json
{
  "token": "refresh-token-to-revoke"
}
```

**Response:** `204 No Content`

## Security Considerations

### 1. Token Storage (Web UI)

**Access Token:**
- Store in HTTP-only cookie (prevents XSS attacks)
- Set Secure flag (HTTPS only)
- Set SameSite=Strict (prevents CSRF)
- Short TTL (15 minutes) limits exposure window

**Refresh Token:**
- Same cookie security as access token
- Never expose to JavaScript
- Can be revoked server-side

**DO NOT:**
- Store tokens in localStorage (vulnerable to XSS)
- Store tokens in sessionStorage (vulnerable to XSS)
- Include tokens in URL parameters (logged everywhere)

### 2. Token Validation Caching

To reduce auth service load, web service can:
- Cache valid tokens for a short period (e.g., 1-2 minutes)
- Use Redis/in-memory cache keyed by token hash
- Invalidate cache on user logout or role change

**Example Cache Strategy:**
```go
// Cache valid tokens for 1 minute
type TokenCache struct {
    cache map[string]CachedToken
    mu    sync.RWMutex
}

type CachedToken struct {
    UserID    string
    Roles     []string
    ExpiresAt time.Time
}

func (tc *TokenCache) Get(token string) (*CachedToken, bool) {
    tc.mu.RLock()
    defer tc.mu.RUnlock()
    cached, ok := tc.cache[token]
    if !ok || time.Now().After(cached.ExpiresAt) {
        return nil, false
    }
    return &cached, true
}
```

### 3. Transport Security

- All auth endpoints MUST use HTTPS in production
- TLS 1.2 minimum
- Mutual TLS (mTLS) between services (optional but recommended)

### 4. Audit Logging

Auth service logs all authentication events:
- Login attempts (success/failure)
- Token validation requests
- Token refresh operations
- HEC token usage
- Session revocations

Logs include: timestamp, user_id, username, IP address, user agent, action, result, metadata

## Production Configuration

### Environment Variables

**Auth Service:**
```bash
AUTH_PORT=8080
ACCESS_SECRET=<strong-random-secret>  # Generate with: openssl rand -base64 32
REFRESH_SECRET=<strong-random-secret>
DB_CONNECTION=postgres://user:pass@postgres:5432/telhawk_auth
```

**Web Service:**
```bash
WEB_PORT=3000
AUTHENTICATE_SERVICE_URL=http://authenticate:8080
SEARCH_SERVICE_URL=http://search:8082
SESSION_COOKIE_DOMAIN=.telhawk.example.com
SESSION_COOKIE_SECURE=true
```

### Token TTLs (Recommended)

| Token Type       | TTL          | Rationale                                    |
|------------------|--------------|----------------------------------------------|
| Access Token     | 15 minutes   | Short window limits exposure if compromised  |
| Refresh Token    | 7 days       | Balance between security and UX              |
| HEC Token        | Configurable | Long-lived (months/years) for stable ingesters |

### Rate Limiting

Implement rate limiting on auth endpoints:
- Login: 5 attempts per IP per 15 minutes
- Token validation: 100 requests per token per minute
- Token refresh: 10 requests per token per minute

## Migration Path

**Current State:** In-memory storage (development only)

**Production State:**
1. Migrate to PostgreSQL for persistent storage
2. Add Redis for token validation cache
3. Implement distributed session store (if multi-region)
4. Add OAuth2/OIDC support for SSO integration (optional)

## Testing Checklist

- [ ] Login with valid credentials returns tokens
- [ ] Login with invalid credentials returns 401
- [ ] Expired access token triggers refresh flow
- [ ] Expired refresh token requires re-login
- [ ] Revoked token cannot be used
- [ ] Disabled user cannot login
- [ ] Role-based access control enforced
- [ ] HEC token validation works for ingestion
- [ ] Audit logs capture all auth events
- [ ] Cookies set with secure flags (httpOnly, secure, SameSite)

## Next Steps

1. Implement web service authentication middleware
2. Create login/logout UI components (React)
3. Add token refresh logic to web service
4. Integrate role-based UI rendering (hide features based on roles)
5. Add session timeout warnings in UI
6. Implement "remember me" functionality (optional)
