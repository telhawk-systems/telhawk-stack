# Rules Service Tests

Comprehensive test suite for the rules microservice, bringing coverage from 0% to **90%+**.

## Test Coverage

| Component | Coverage | Type |
|-----------|----------|------|
| **Service Layer** | 98.4% | Unit Tests |
| **Handlers** | 84.2% | Unit Tests |
| **Repository** | Integration Tests | Requires PostgreSQL |

## Test Files

### Unit Tests (No Database Required)

#### `internal/service/service_test.go`
Tests for business logic layer:
- Schema creation and validation
- Schema updates with version management
- Builtin rule protection
- Schema retrieval (by ID and version)
- Listing with pagination and filtering
- Version history tracking
- Enable/disable/hide operations
- Active parameter set management

**Run tests:**
```bash
go test ./internal/service
go test -v ./internal/service  # Verbose output
go test -cover ./internal/service  # With coverage
```

#### `internal/handlers/handlers_test.go`
Tests for HTTP handlers and JSON:API responses:
- Health check endpoint
- Correlation types endpoint
- CRUD operations for detection schemas
- HTTP method validation
- Request/response format validation
- Error handling (400, 403, 404, 500)
- JSON:API serialization

**Run tests:**
```bash
go test ./internal/handlers
go test -v ./internal/handlers  # Verbose output
go test -cover ./internal/handlers  # With coverage
```

### Integration Tests (Require Database)

#### `internal/repository/postgres_test.go`
Tests for database operations:
- Connection management
- Schema creation with UUID v7 generation
- Version tracking and history
- Filtering and pagination
- Lifecycle management (disable/enable/hide)
- Transaction handling
- Index usage verification

**Setup test database:**

The tests expect a PostgreSQL database at:
```
postgres://telhawk:telhawk-rules-dev@localhost:5433/telhawk_rules?sslmode=disable
```

You can override this with the `RULES_DB_TEST_URL` environment variable.

**Using Docker:**
```bash
# Start test database
docker run -d \
  --name telhawk-rules-test \
  -e POSTGRES_USER=telhawk \
  -e POSTGRES_PASSWORD=telhawk-rules-dev \
  -e POSTGRES_DB=telhawk_rules \
  -p 5433:5432 \
  postgres:15

# Run migrations
cd rules
export RULES_DB_TEST_URL="postgres://telhawk:telhawk-rules-dev@localhost:5433/telhawk_rules?sslmode=disable"
# Apply migrations from rules/migrations/

# Run tests
go test ./internal/repository

# Clean up
docker stop telhawk-rules-test
docker rm telhawk-rules-test
```

**Skip integration tests:**
```bash
go test -short ./internal/repository  # Skips tests marked with testing.Short()
```

## Running All Tests

```bash
# Run all unit tests (service + handlers)
go test ./internal/service ./internal/handlers

# Run with coverage
go test -cover ./internal/service ./internal/handlers

# Run all tests including integration (requires database)
go test ./internal/...

# Run with verbose output and coverage
go test -v -cover ./internal/...
```

## Coverage Report

Generate HTML coverage report:
```bash
# Generate coverage for unit tests
go test -coverprofile=coverage.out ./internal/service ./internal/handlers

# View in browser
go tool cover -html=coverage.out

# View summary by function
go tool cover -func=coverage.out
```

## Test Patterns Used

### 1. Table-Driven Tests
Used extensively for testing multiple scenarios:
```go
tests := []struct {
    name     string
    input    string
    expected string
    wantErr  bool
}{
    {name: "valid case", input: "test", expected: "result", wantErr: false},
    {name: "error case", input: "", expected: "", wantErr: true},
}
```

### 2. Mocking
- Repository layer uses `testify/mock` for mocking repository interface
- Handlers use real service with mocked repository

### 3. HTTP Testing
Handlers use `httptest` for testing HTTP endpoints:
```go
req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
w := httptest.NewRecorder()
handler.ListSchemas(w, req)
assert.Equal(t, http.StatusOK, w.Code)
```

### 4. Test Helpers
- `createTestSchema()` - Creates test detection schemas
- `setupTestDB()` - Sets up test database connection
- `createBuiltinSchema()` - Creates builtin test schemas

## Key Test Scenarios Covered

### Service Layer
- ✅ Create schema with auto-generated UUIDs
- ✅ Update schema creates new version
- ✅ Builtin rule protection prevents modification
- ✅ Get schema by version ID or stable ID
- ✅ List with pagination (page, limit validation)
- ✅ Filter by severity, title, ID
- ✅ Include/exclude disabled and hidden schemas
- ✅ Version history retrieval
- ✅ Enable/disable/hide operations
- ✅ Active parameter set management

### Handlers
- ✅ All HTTP methods validated (405 for wrong method)
- ✅ JSON request body parsing
- ✅ JSON:API response format
- ✅ Query parameter parsing
- ✅ Error responses (400, 403, 404, 500)
- ✅ Content-Type headers
- ✅ Builtin rule protection at HTTP layer

### Repository (Integration)
- ✅ Database connection and pooling
- ✅ CRUD operations
- ✅ UUID v7 generation
- ✅ Immutable versioning pattern
- ✅ Lifecycle timestamp management
- ✅ Window function (ROW_NUMBER) for version calculation
- ✅ Pagination and filtering
- ✅ Transaction handling
- ✅ Error handling (not found, constraints)

## Test Maintenance

### Adding New Tests
1. Follow existing patterns (table-driven, subtests)
2. Use descriptive test names
3. Include both success and error cases
4. Mock external dependencies
5. Clean up resources in defer statements

### Updating Tests
When modifying code:
1. Update corresponding tests
2. Verify coverage hasn't decreased
3. Run `go test -cover` to check
4. Add tests for new edge cases

## CI/CD Integration

Example GitHub Actions workflow:
```yaml
name: Tests
on: [push, pull_request]
jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run unit tests
        run: go test -v -cover ./internal/service ./internal/handlers
        working-directory: rules

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_USER: telhawk
          POSTGRES_PASSWORD: telhawk-rules-dev
          POSTGRES_DB: telhawk_rules
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5433:5432
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run integration tests
        env:
          RULES_DB_TEST_URL: postgres://telhawk:telhawk-rules-dev@localhost:5433/telhawk_rules?sslmode=disable
        run: go test -v ./internal/repository
        working-directory: rules
```

## Common Issues

### Repository tests fail with "database not found"
- Ensure test database is running on port 5433
- Check credentials match test configuration
- Verify migrations have been applied

### Import errors
- Run `go mod tidy` to resolve dependencies
- Ensure all test dependencies are in go.mod

### Coverage seems low
- Ensure all test files are named `*_test.go`
- Check that test functions start with `Test`
- Verify `go test` is finding all test files

## Further Improvements

Potential areas for additional testing:
- [ ] Benchmark tests for performance-critical paths
- [ ] Fuzz tests for input validation
- [ ] Race condition detection (`go test -race`)
- [ ] More edge cases for concurrent operations
- [ ] Mock tests for external service calls (when added)
