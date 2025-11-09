# TelHawk Stack - Security and Credentials Reference

## Default Credentials (Development Only)

### Database Credentials

#### PostgreSQL (auth-db)
- **Host**: `auth-db:5432` (Docker) or `localhost:5432` (external)
- **Database**: `telhawk_auth`
- **Username**: `telhawk`
- **Password**: `telhawk-auth-dev` (development) or `password` (varies by config)
- **Connection String**: `postgres://telhawk:password@auth-db:5432/telhawk_auth?sslmode=disable`

#### OpenSearch
- **URL**: `https://opensearch:9200` (internal) or `https://localhost:9200` (external)
- **Username**: `admin`
- **Password**: `TelHawk123!`
- **Admin Password (initial)**: Set in `OPENSEARCH_INITIAL_ADMIN_PASSWORD` env var

#### Redis
- **Host**: `redis:6379`
- **Password**: None (default)
- **Database**: 0

### Application Users

#### Default Admin User
Created by database migration `001_init.up.sql`:
- **Username**: `admin`
- **Password**: `admin123`
- **Email**: `admin@telhawk.local`
- **Roles**: `[admin]`

**CRITICAL**: Change this password immediately after first deployment!

## Security Configuration

### JWT Authentication

#### JWT Secret
**Environment Variable**: `AUTH_JWT_SECRET`

**CRITICAL**: MUST be set to a secure random value in production!

```bash
# Generate secure secret (example)
openssl rand -base64 32

# Set in environment
export AUTH_JWT_SECRET="your-generated-secret-here"
```

#### Token Expiration
- **Access Token**: Configured in auth service (default: 15 minutes)
- **Refresh Token**: Configured in auth service (default: 7 days)

### HEC Token Management

HEC tokens are used for event ingestion authentication.

#### Token Format
- UUID v4 format
- Stored hashed in database
- Associated with a user account

#### Token Operations
```bash
# Create token
thawk token create --name "my-ingestion-token"

# List tokens
thawk token list

# Revoke token
thawk token revoke <token-id>
```

### Password Security

#### Password Hashing
- **Algorithm**: bcrypt
- **Work Factor**: 10+ (configurable)
- **Implementation**: `golang.org/x/crypto/bcrypt`

#### Password Requirements
Enforce in your application:
- Minimum length: 8 characters
- Complexity requirements (recommended)
- No common passwords

### TLS/SSL Configuration

#### Development Mode (Default)
```bash
# TLS disabled for easier development
AUTH_TLS_ENABLED=false
AUTH_TLS_SKIP_VERIFY=true
```

#### Production Mode
```bash
# TLS enabled for all services
AUTH_TLS_ENABLED=true
INGEST_TLS_ENABLED=true
CORE_TLS_ENABLED=true
STORAGE_TLS_ENABLED=true
QUERY_TLS_ENABLED=true
WEB_TLS_ENABLED=true

# Enforce certificate validation
AUTH_TLS_SKIP_VERIFY=false
INGEST_TLS_SKIP_VERIFY=false
CORE_TLS_SKIP_VERIFY=false
STORAGE_TLS_SKIP_VERIFY=false
QUERY_TLS_SKIP_VERIFY=false
WEB_TLS_SKIP_VERIFY=false
```

#### PostgreSQL SSL
```bash
# Development (no SSL)
postgres://user:pass@host:5432/db?sslmode=disable

# Production (require SSL)
postgres://user:pass@host:5432/db?sslmode=require

# With certificate verification
postgres://user:pass@host:5432/db?sslmode=verify-full&sslrootcert=/path/to/ca.crt
```

### Certificate Management

#### Certificate Locations
- **Go Services**: `/certs/` from `telhawk-certs` volume
- **OpenSearch**: `/certs/` from `opensearch-certs` volume

#### Certificate Files
```
/certs/
├── ca.crt           # Certificate Authority
├── server.crt       # Server certificate
├── server.key       # Server private key
├── client.crt       # Client certificate (mTLS)
└── client.key       # Client private key (mTLS)
```

#### Regenerate Certificates
```bash
# Remove old certificates
docker volume rm telhawk-stack_telhawk-certs
docker volume rm telhawk-stack_opensearch-certs

# Certificates regenerated on next startup
docker-compose up -d
```

## Rate Limiting

### Configuration
Rate limiting uses Redis for tracking:

```bash
# IP-based rate limiting (pre-authentication)
INGEST_RATE_LIMIT_IP_ENABLED=true
INGEST_RATE_LIMIT_IP_REQUESTS=100
INGEST_RATE_LIMIT_IP_WINDOW=60s

# Token-based rate limiting (post-authentication)
INGEST_RATE_LIMIT_TOKEN_ENABLED=true
INGEST_RATE_LIMIT_TOKEN_REQUESTS=1000
INGEST_RATE_LIMIT_TOKEN_WINDOW=60s
```

### Rate Limit Response
When rate limit exceeded:
- **HTTP Status**: 429 Too Many Requests
- **Header**: `Retry-After: <seconds>`
- **Body**: Error message with retry information

## RBAC (Role-Based Access Control)

### Roles
Stored in PostgreSQL `users.roles` column (TEXT array):

```sql
-- Example roles
['admin']           -- Full system access
['user']            -- Basic access
['analyst']         -- Read-only access
['ingest']          -- Ingestion-only access
```

### Role Enforcement
Implemented in auth service middleware and endpoint handlers.

## Audit Logging

### Audit Log Table
All authentication and authorization events logged to `audit_log` table:

```sql
CREATE TABLE audit_log (
    id UUID PRIMARY KEY,
    user_id UUID,              -- User who performed action
    action VARCHAR(255),       -- Action performed
    resource VARCHAR(255),     -- Resource affected
    metadata JSONB,            -- Additional context
    ip_address INET,           -- Source IP
    user_agent TEXT,           -- Client user agent
    status VARCHAR(50),        -- success/failure
    created_at TIMESTAMPTZ
);
```

### Audit Events Forwarded to SIEM
Auth events also forwarded to ingest service as OCSF Authentication events:
- **Class**: Authentication (class_uid: 3002)
- **Categories**: Login, logout, token creation, token revocation
- **Indexed**: In OpenSearch for searching/alerting

## Security Best Practices

### Production Deployment

#### 1. Change All Default Passwords
```bash
# PostgreSQL
ALTER USER telhawk WITH PASSWORD 'new-secure-password';

# OpenSearch
# Use OpenSearch security plugin to change admin password

# Default application user
# Login and change via API or CLI
```

#### 2. Set Secure JWT Secret
```bash
# Generate random secret
openssl rand -base64 32

# Set in environment
export AUTH_JWT_SECRET="<generated-secret>"
```

#### 3. Enable TLS Everywhere
```bash
# All service TLS
*_TLS_ENABLED=true
*_TLS_SKIP_VERIFY=false

# Database SSL
# Use sslmode=require or sslmode=verify-full
```

#### 4. Use Secrets Management
Don't use plain environment variables in production:
- Docker Secrets
- Kubernetes Secrets
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault

Example with Docker Secrets:
```yaml
services:
  auth:
    secrets:
      - jwt_secret
secrets:
  jwt_secret:
    external: true
```

#### 5. Network Security
```bash
# Restrict service exposure
# Only expose necessary ports (web, ingest)
# Keep internal services (auth, core, storage) on private network
```

#### 6. Regular Security Updates
```bash
# Keep dependencies updated
go get -u ./...
go mod tidy

# Rebuild Docker images
docker-compose build --no-cache
```

### OWASP Top 10 Protection

#### A01: Broken Access Control
- ✅ RBAC implemented
- ✅ JWT token validation on all protected endpoints
- ✅ Session management with revocation

#### A02: Cryptographic Failures
- ✅ Passwords hashed with bcrypt
- ✅ TLS for data in transit
- ✅ Secure random token generation (UUID v4)

#### A03: Injection
- ✅ Prepared statements (pgx driver automatic)
- ✅ Input validation on all endpoints
- ✅ Output encoding

#### A04: Insecure Design
- ✅ Defense in depth (multiple security layers)
- ✅ Rate limiting
- ✅ Dead letter queue for failed events

#### A05: Security Misconfiguration
- ✅ Secure defaults
- ⚠️ MUST change default passwords in production
- ⚠️ MUST enable TLS in production

#### A06: Vulnerable Components
- ✅ Regular dependency updates
- ✅ Go module vendoring

#### A07: Authentication Failures
- ✅ JWT-based authentication
- ✅ Secure session management
- ✅ Password hashing
- ✅ Token revocation support

#### A08: Software and Data Integrity
- ✅ Code signing (Docker images)
- ✅ Audit logging
- ✅ Event validation (OCSF compliance)

#### A09: Logging Failures
- ✅ Comprehensive audit logging
- ✅ Auth events forwarded to SIEM
- ✅ Structured logging

#### A10: Server-Side Request Forgery
- ✅ Input validation
- ✅ Allowlist for external requests

## Secure Configuration Checklist

Before production deployment:

### Credentials
- [ ] Change PostgreSQL password
- [ ] Change OpenSearch admin password
- [ ] Change default admin user password
- [ ] Set secure `AUTH_JWT_SECRET`
- [ ] Use secrets management (not plain env vars)

### TLS/SSL
- [ ] Enable TLS for all services (`*_TLS_ENABLED=true`)
- [ ] Disable certificate skip (`*_TLS_SKIP_VERIFY=false`)
- [ ] Enable PostgreSQL SSL (`sslmode=require`)
- [ ] Verify OpenSearch TLS configuration

### Network
- [ ] Restrict exposed ports (only web and ingest public)
- [ ] Configure firewall rules
- [ ] Use private network for internal services
- [ ] Enable rate limiting

### Monitoring
- [ ] Enable audit logging
- [ ] Configure log aggregation
- [ ] Set up security alerts
- [ ] Monitor authentication failures

### Access Control
- [ ] Review user roles and permissions
- [ ] Implement principle of least privilege
- [ ] Disable or remove unused accounts
- [ ] Configure session timeout

### Data Protection
- [ ] Enable encryption at rest (database, OpenSearch)
- [ ] Configure backup encryption
- [ ] Implement data retention policies
- [ ] Set up disaster recovery

## Common Security Commands

### Check Service Security

```bash
# Test TLS connection
openssl s_client -connect localhost:8080 -tls1_2

# Verify certificate
openssl x509 -in /certs/server.crt -text -noout

# Test authentication
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

### Security Auditing

```bash
# Check for vulnerabilities in dependencies
go list -m all | nancy sleuth

# Check for known vulnerabilities
docker scan telhawk/auth:latest

# Audit database access
docker-compose exec auth-db psql -U telhawk -d telhawk_auth -c "SELECT * FROM audit_log ORDER BY created_at DESC LIMIT 10;"
```