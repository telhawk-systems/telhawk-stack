# Auth Service

Centralized authentication and authorization service for TelHawk Stack.

## Features

- **JWT-based authentication** with access and refresh tokens
- **User management** with role-based access control (RBAC)
- **HEC token management** for Splunk-compatible ingestion authentication
- **Session management** with token refresh and revocation
- **Password hashing** using bcrypt
- **RESTful API** for easy integration

## Roles

- `admin` - Full system access, user management
- `analyst` - Read/write access to security data, create queries and alerts
- `viewer` - Read-only access to dashboards and searches
- `ingester` - HEC token access for data ingestion only

## API Endpoints

### Authentication

#### Register User
```bash
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "analyst1",
  "email": "analyst1@example.com",
  "password": "secure-password",
  "roles": ["analyst"]
}
```

#### Login
```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "analyst1",
  "password": "secure-password"
}

# Response:
{
  "access_token": "eyJhbGc...",
  "refresh_token": "random-token",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

#### Refresh Token
```bash
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "random-token"
}
```

#### Validate Token
```bash
POST /api/v1/auth/validate
Content-Type: application/json

{
  "token": "eyJhbGc..."
}

# Response:
{
  "valid": true,
  "user_id": "uuid",
  "roles": ["analyst"]
}
```

#### Revoke Token
```bash
POST /api/v1/auth/revoke
Content-Type: application/json

{
  "token": "refresh-token-to-revoke"
}
```

## Usage in Other Services

Other services validate tokens by calling the auth service:

```go
// In any service (ingest, query, web)
func validateRequest(token string) error {
    resp, err := http.Post(
        "http://auth:8080/api/v1/auth/validate",
        "application/json",
        bytes.NewBuffer([]byte(fmt.Sprintf(`{"token":"%s"}`, token))),
    )
    // ... handle response
}
```

## Building

```bash
cd auth
go mod tidy
go build -o ../bin/auth ./cmd/auth
```

## Running

```bash
./bin/auth
# Listens on :8080
```

## Configuration

Environment variables:
- `AUTH_PORT` - Server port (default: 8080)
- `ACCESS_SECRET` - JWT access token secret
- `REFRESH_SECRET` - JWT refresh token secret
- `DB_CONNECTION` - PostgreSQL connection string (future)

## Storage

Currently uses in-memory storage for development. Production will use PostgreSQL.

## HEC Token Management

For ingestion services, create HEC tokens:

```bash
POST /api/v1/auth/hec-tokens
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "name": "production-ingester",
  "expires_at": "2025-12-31T23:59:59Z"
}
```

These tokens are used with the `Authorization: Splunk <hec-token>` header in ingestion requests.
