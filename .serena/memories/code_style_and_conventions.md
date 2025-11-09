# TelHawk Stack - Code Style and Conventions

## Go Language Standards

### Go Version
- **Version**: Go 1.24.2
- All services use the same Go version

### Standard Go Conventions
This project follows standard Go conventions and idioms:
- **Formatting**: Standard `go fmt` formatting
- **Naming**: CamelCase for exported names, camelCase for unexported
- **Package names**: Short, lowercase, no underscores
- **Error handling**: Explicit error returns and checks
- **Comments**: Exported symbols have doc comments starting with the symbol name

### No Explicit Linting Configuration
- No `.golangci.yml` or similar configuration found
- Follows standard Go best practices
- Use `go vet` and `go fmt` for code quality

## Project Structure Patterns

### Service Directory Layout
Each service follows the standard Go project layout:
```
service-name/
├── cmd/
│   └── service-name/
│       └── main.go          # Entry point
├── internal/                 # Private application code
│   ├── config/              # Configuration handling
│   ├── handlers/            # HTTP handlers
│   ├── service/             # Business logic
│   ├── repository/          # Data access layer
│   └── ...
├── pkg/                     # Public library code (if any)
├── migrations/              # Database migrations (for auth)
├── config.yaml              # Default configuration
├── Dockerfile               # Container definition
└── go.mod                   # Go module definition
```

### Key Directories
- **cmd/**: Application entry points (main.go files)
- **internal/**: Private application code, not importable by other projects
- **pkg/**: Public library code, importable by other projects
- **migrations/**: SQL migration files (numbered: `NNN_description.up.sql`, `NNN_description.down.sql`)

## Configuration Conventions

### Configuration Files
- **Location**: `/etc/telhawk/<service>/config.yaml` in containers
- **Format**: YAML
- **Library**: Viper for loading and parsing
- **Overrides**: Environment variables take precedence over YAML

### Environment Variable Naming
Pattern: `<SERVICE>_<SECTION>_<KEY>`

Examples:
```bash
AUTH_SERVER_PORT=8080
AUTH_JWT_SECRET=your-secret-key
INGEST_AUTH_URL=http://auth:8080
QUERY_OPENSEARCH_PASSWORD=secret
```

### No CLI Arguments for Configuration
Configuration is managed entirely through:
1. YAML config files (defaults)
2. Environment variables (overrides)

This follows the 12-factor app methodology.

## Database Conventions

### PostgreSQL (auth service)
- **Primary Keys**: UUID type
- **Timestamps**: `created_at`, `updated_at` columns (TIMESTAMPTZ)
- **Triggers**: Automatic `updated_at` updates via triggers
- **Foreign Keys**: CASCADE delete behavior
- **Indexes**: Created on lookup fields
- **JSON Storage**: JSONB for flexible metadata
- **Migrations**: Numbered files with up/down pairs

Example table structure:
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    roles TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### OpenSearch
- **Index Pattern**: `telhawk-events-YYYY.MM.DD`
- **Query Pattern**: `telhawk-events-*` for searches
- **Field Mappings**: OCSF-optimized with nested objects

## Testing Conventions

### Test Files
- **Location**: Alongside source files
- **Naming**: `*_test.go`
- **Integration Tests**: `*_integration_test.go`
- **Test Data**: In `testdata/` directories

### Test Organization
```go
// Unit test example
func TestFunctionName(t *testing.T) {
    // Arrange
    input := setupInput()
    
    // Act
    result := FunctionName(input)
    
    // Assert
    if result != expected {
        t.Errorf("expected %v, got %v", expected, result)
    }
}
```

### Key Test Files
- `core/internal/pipeline/integration_test.go`: End-to-end normalization
- `ingest/internal/handlers/hec_handler_test.go`: HEC endpoint tests
- `query/internal/service/service_test.go`: Query API tests

## Error Handling Patterns

### Error Returns
```go
// Always return errors as last value
func DoSomething() (Result, error) {
    if err := validate(); err != nil {
        return Result{}, fmt.Errorf("validation failed: %w", err)
    }
    return result, nil
}
```

### Error Wrapping
Use `fmt.Errorf` with `%w` verb to wrap errors for better stack traces.

### Retry Pattern
- **Strategy**: Exponential backoff
- **Attempts**: 3 attempts typical
- **Retry On**: 5xx, 429, network errors
- **No Retry**: 4xx client errors (except 429)

## HTTP Handler Patterns

### Standard Handler Structure
```go
func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    var req RequestType
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    
    // 2. Validate
    if err := req.Validate(); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // 3. Process
    result, err := h.service.Process(r.Context(), req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // 4. Respond
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

## Logging Conventions

Services use structured logging (implementation varies by service).
Log important events:
- Service startup/shutdown
- Authentication events
- Error conditions
- Event processing metrics

## Security Best Practices

### Credentials
- **Never hardcode** secrets or passwords
- Use environment variables for sensitive config
- Default passwords in docker-compose.yml are for **development only**

### JWT Handling
- Access tokens for short-term authentication
- Refresh tokens stored in database with revocation
- Token validation on every protected endpoint

### Database Security
- Use prepared statements (pgx driver does this automatically)
- Validate all user input
- Hash passwords with bcrypt (work factor 10+)

### OWASP Top 10 Awareness
- Prevent SQL injection (use prepared statements)
- Prevent XSS (escape output)
- Prevent CSRF (use tokens for state-changing operations)
- Validate and sanitize all inputs

## API Conventions

### REST Endpoints
- **Path Pattern**: `/api/v1/<resource>/<action>`
- **Methods**: GET, POST, PUT, DELETE (RESTful)
- **Content-Type**: `application/json`
- **Authentication**: `Authorization: Bearer <token>` header

### Response Format
```json
{
  "data": {},
  "error": "error message if any",
  "metadata": {}
}
```

## Common Imports and Libraries

### Standard Libraries
- `context`: Context management
- `encoding/json`: JSON encoding/decoding
- `net/http`: HTTP server and client
- `fmt`, `errors`: Error handling

### Third-Party Libraries
- `github.com/spf13/viper`: Configuration
- `github.com/spf13/cobra`: CLI (in cli service)
- `github.com/golang-jwt/jwt/v5`: JWT tokens
- `github.com/google/uuid`: UUID generation
- `github.com/jackc/pgx/v5`: PostgreSQL driver
- `golang.org/x/crypto/bcrypt`: Password hashing