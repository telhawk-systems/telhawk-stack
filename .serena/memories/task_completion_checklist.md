# TelHawk Stack - Task Completion Checklist

## When a Coding Task is Completed

Follow these steps to ensure code quality and proper integration:

### 1. Run Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection (for concurrency changes)
go test -race ./...
```

**Required**: All tests must pass before considering the task complete.

### 2. Format Code
```bash
# Format all Go code
go fmt ./...
```

**Required**: Code must be properly formatted using standard Go formatting.

### 3. Tidy Dependencies
```bash
# Clean up go.mod and go.sum
go mod tidy
```

**Required**: If you added or removed dependencies, run this to clean up module files.

### 4. Build the Service
```bash
# Build the specific service you modified
cd <service-name> && go build -o ../bin/<service-name> ./cmd/<service-name>

# Example for auth service:
cd auth && go build -o ../bin/auth ./cmd/auth
```

**Required**: Ensure the service builds without errors.

### 5. Test with Docker (if applicable)
```bash
# Rebuild the specific service container
docker-compose build <service-name>

# Restart the service
docker-compose up -d <service-name>

# Check logs for errors
docker-compose logs -f <service-name>

# Verify service health
docker-compose ps
```

**Recommended**: For significant changes, test in the Docker environment to ensure proper integration.

### 6. Database Migrations (if schema changed)
If you modified the database schema:

```bash
# Create new migration files in auth/migrations/
# Format: NNN_description.up.sql and NNN_description.down.sql

# Test migration up
migrate -database "postgres://telhawk:password@localhost:5432/telhawk_auth?sslmode=disable" -path migrations up

# Test migration down (rollback)
migrate -database "postgres://telhawk:password@localhost:5432/telhawk_auth?sslmode=disable" -path migrations down 1

# Re-apply
migrate -database "postgres://telhawk:password@localhost:5432/telhawk_auth?sslmode=disable" -path migrations up
```

**Required**: If database schema was modified, create and test migrations.

### 7. Update Documentation
Update relevant documentation if you:
- Changed API endpoints → Update service README
- Modified configuration → Update `docs/CONFIGURATION.md`
- Changed architecture → Update `README.md` and `CLAUDE.md`
- Added new features → Update service-specific documentation

**Recommended**: Keep documentation in sync with code changes.

### 8. Code Review Checklist
Review your own code for:
- [ ] No hardcoded credentials or secrets
- [ ] Proper error handling (all errors checked and handled)
- [ ] Input validation (prevent injection attacks)
- [ ] Proper logging (info for important events, errors for failures)
- [ ] Context usage (for cancellation and timeouts)
- [ ] Thread safety (if using concurrency)
- [ ] No TODO or FIXME comments left unresolved
- [ ] Follows project conventions (see code_style_and_conventions.md)

### 9. Integration Testing (for major changes)
For significant features or bug fixes:

```bash
# Start the full stack
docker-compose up -d

# Run integration tests (if available)
cd core && go test -v ./internal/pipeline -run Integration

# Manual testing with CLI tool
docker-compose run --rm thawk auth login -u admin -p admin123
docker-compose run --rm thawk token create --name test-token
# ... perform end-to-end testing
```

**Recommended**: For features that span multiple services.

### 10. Performance Check (for performance-sensitive code)
If you modified performance-critical paths:

```bash
# Run benchmarks
go test -bench=. -benchmem ./...

# Check for memory leaks with pprof
# (requires additional setup)
```

**Optional**: Only needed for performance-critical changes.

## For OCSF Normalizer Changes

If you modified OCSF normalizers or the generator:

### 1. Regenerate Normalizers
```bash
cd tools/normalizer-generator
go run main.go
```

### 2. Test Normalization
```bash
cd core
go test ./internal/pipeline -run TestNormalization
go test ./internal/normalizer -v
```

### 3. Verify OCSF Compliance
- Check that `category_uid`, `class_uid`, `activity_id`, `type_uid` are set correctly
- Verify required OCSF fields are populated
- Test with sample events

## For Frontend Changes

If you modified the React frontend:

### 1. Build Frontend
```bash
cd web/frontend
npm run build
```

### 2. Test in Development
```bash
cd web/frontend
npm start
# Test in browser at http://localhost:3000
```

### 3. Test Production Build
```bash
docker-compose build web
docker-compose up -d web
# Test at http://localhost:3000
```

## Git Workflow

### Before Committing
1. Ensure all tests pass
2. Format code
3. Tidy dependencies
4. Review changes: `git diff`

### Commit Message Format
```
<type>: <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Example:
```
feat: add rate limiting to HEC endpoint

Implements Redis-backed sliding window rate limiting for the HEC
ingestion endpoint. Limits based on IP address and HEC token.

Closes #123
```

## Summary: Minimum Required Steps

For most tasks, you MUST complete at least:

1. ✅ **Run tests**: `go test ./...`
2. ✅ **Format code**: `go fmt ./...`
3. ✅ **Build service**: `cd <service> && go build ./...`
4. ✅ **Verify it works**: Test the specific functionality you changed

For tasks involving dependencies:
5. ✅ **Tidy modules**: `go mod tidy`

For tasks involving database changes:
6. ✅ **Create and test migrations**

For tasks involving Docker deployment:
7. ✅ **Test with Docker**: `docker-compose up -d --build`