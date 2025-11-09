# TelHawk Stack - Testing Guide

## Test Organization

### Test File Naming
- **Unit tests**: `*_test.go` (alongside source files)
- **Integration tests**: `*_integration_test.go`
- **Test data**: `testdata/` directories
- **Examples**: `examples_test.go`

### Test File Locations
Tests are located in the same package as the code they test:
```
service/internal/handlers/
├── hec.go
├── hec_handler_test.go          # Unit tests
└── hec_integration_test.go      # Integration tests
```

## Running Tests

### Basic Test Commands

#### Run All Tests
```bash
go test ./...
```

#### Run Tests for Specific Module
```bash
cd auth && go test ./...
cd core && go test ./...
cd query && go test ./...
```

#### Run Specific Test File
```bash
go test ./core/internal/pipeline
```

#### Run Specific Test Function
```bash
go test -v ./core/internal/pipeline -run TestNormalization
go test -v ./ingest/internal/handlers -run TestHECHandler
```

### Verbose Output
```bash
# Show test names and results
go test -v ./...

# Show only failed tests
go test ./...
```

### Test Coverage

#### Generate Coverage Report
```bash
# Coverage for all packages
go test -cover ./...

# Coverage for specific package
go test -cover ./core/internal/pipeline
```

#### HTML Coverage Report
```bash
# Generate coverage data
go test -coverprofile=coverage.out ./...

# View in browser
go tool cover -html=coverage.out

# View coverage by function
go tool cover -func=coverage.out
```

#### Coverage by Package
```bash
# See coverage percentage per package
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

### Advanced Testing Options

#### Race Detection
```bash
# Detect race conditions (important for concurrent code)
go test -race ./...

# Race detection for specific package
go test -race ./ingest/internal/handlers
```

#### Short Mode (Skip Long Tests)
```bash
# Skip tests marked with t.Skip() or checking testing.Short()
go test -short ./...
```

#### Parallel Execution
```bash
# Run tests in parallel (default)
go test -parallel 4 ./...
```

#### Timeout
```bash
# Set timeout for test execution
go test -timeout 30s ./...
go test -timeout 5m ./...
```

#### Verbose Logging
```bash
# Show all logs, even from passing tests
go test -v -args -test.v
```

## Key Test Files

### Core Service Tests
- `core/internal/pipeline/integration_test.go`: End-to-end normalization pipeline
- `core/internal/normalizer/ocsf_passthrough_test.go`: OCSF passthrough normalizer
- `core/internal/service/storage_test.go`: Storage service tests
- `core/pkg/ocsf/examples_test.go`: OCSF event examples

### Ingest Service Tests
- `ingest/internal/handlers/hec_handler_test.go`: HEC endpoint handler tests

### Query Service Tests
- `query/internal/service/service_test.go`: Query API service tests
- `query/internal/scheduler/scheduler_test.go`: Query scheduler tests
- `query/internal/notification/notification_test.go`: Notification tests

### Tools Tests
- `tools/event-seeder/main_test.go`: Event seeder utility tests

## Test Patterns

### Table-Driven Tests
Common pattern in Go for testing multiple scenarios:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "TEST",
            wantErr:  false,
        },
        {
            name:     "empty input",
            input:    "",
            expected: "",
            wantErr:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
            }
            if result != tt.expected {
                t.Errorf("expected: %v, got: %v", tt.expected, result)
            }
        })
    }
}
```

### Setup and Teardown
```go
func TestMain(m *testing.M) {
    // Setup
    setup()
    
    // Run tests
    code := m.Run()
    
    // Teardown
    teardown()
    
    os.Exit(code)
}
```

### Helper Functions
```go
func TestFunction(t *testing.T) {
    t.Helper() // Mark as helper for better error reporting
    // ... test logic
}
```

### Subtests
```go
func TestFeature(t *testing.T) {
    t.Run("scenario1", func(t *testing.T) {
        // Test scenario 1
    })
    
    t.Run("scenario2", func(t *testing.T) {
        // Test scenario 2
    })
}
```

## Integration Testing

### Running Integration Tests
Integration tests typically require external dependencies (database, OpenSearch, Redis).

```bash
# Start dependencies with Docker
docker-compose up -d auth-db opensearch redis

# Run integration tests
go test -v ./core/internal/pipeline -run Integration

# Stop dependencies
docker-compose down
```

### Integration Test Pattern
```go
// +build integration

func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    
    // Setup external dependencies
    db := setupDatabase(t)
    defer db.Close()
    
    // Run test
    // ...
}
```

## Benchmarking

### Run Benchmarks
```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkNormalization ./core/internal/pipeline

# Show memory allocations
go test -bench=. -benchmem ./...

# Run N times for statistical significance
go test -bench=. -benchtime=10s ./...
go test -bench=. -count=5 ./...
```

### Benchmark Pattern
```go
func BenchmarkFunction(b *testing.B) {
    // Setup
    input := setupInput()
    
    // Reset timer to exclude setup time
    b.ResetTimer()
    
    // Run function b.N times
    for i := 0; i < b.N; i++ {
        Function(input)
    }
}
```

## Mocking and Test Doubles

### Interface-Based Mocking
```go
// Define interface
type Storage interface {
    Store(event *ocsf.Event) error
}

// Mock implementation for tests
type mockStorage struct {
    storeFunc func(*ocsf.Event) error
}

func (m *mockStorage) Store(event *ocsf.Event) error {
    return m.storeFunc(event)
}

// Use in test
func TestWithMock(t *testing.T) {
    mock := &mockStorage{
        storeFunc: func(e *ocsf.Event) error {
            return nil
        },
    }
    // ... use mock in test
}
```

## Test Data Management

### Using testdata/ Directory
```
service/
├── internal/
│   ├── normalizer/
│   │   ├── normalizer.go
│   │   ├── normalizer_test.go
│   │   └── testdata/
│   │       ├── valid_event.json
│   │       ├── invalid_event.json
│   │       └── expected_output.json
```

### Loading Test Data
```go
func loadTestData(t *testing.T, filename string) []byte {
    t.Helper()
    data, err := os.ReadFile(filepath.Join("testdata", filename))
    if err != nil {
        t.Fatalf("failed to load test data: %v", err)
    }
    return data
}
```

## Continuous Integration

Tests should be run in CI/CD pipeline:
```bash
# In CI environment
go test -v -race -cover ./...
```

## Testing Best Practices

1. **Test in Isolation**: Each test should be independent
2. **Use Table-Driven Tests**: For testing multiple scenarios
3. **Test Edge Cases**: Empty inputs, nil values, boundary conditions
4. **Test Error Paths**: Not just happy paths
5. **Use Meaningful Names**: Test names should describe what they test
6. **Keep Tests Fast**: Unit tests should run quickly
7. **Use Mocks**: For external dependencies
8. **Clean Up**: Use defer for cleanup or t.Cleanup()
9. **Avoid Flaky Tests**: Tests should be deterministic
10. **Test Public APIs**: Focus on exported functions/methods

## Quick Reference

| Task | Command |
|------|---------|
| Run all tests | `go test ./...` |
| Run with coverage | `go test -cover ./...` |
| Run specific test | `go test -v ./<package> -run <TestName>` |
| Race detection | `go test -race ./...` |
| Benchmarks | `go test -bench=. ./...` |
| HTML coverage | `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out` |
| Integration tests | `go test -v ./<package> -run Integration` |
| Skip long tests | `go test -short ./...` |