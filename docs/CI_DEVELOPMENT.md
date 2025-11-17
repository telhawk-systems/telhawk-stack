# CI Development Guide

This document explains how to run the same checks locally that are run in CI, so you can catch issues before pushing.

## Overview

The TelHawk Stack uses GitHub Actions for continuous integration. The workflow runs on every push to `main` and on all pull requests.

**Workflow File:** `.github/workflows/go-build-test-lint.yml`

The CI pipeline consists of three parallel jobs:

1. **Test** - Formatting, vetting, and unit tests
2. **Lint** - Static analysis with golangci-lint
3. **Build** - Compile all services

## Git Hooks (Automatic Formatting)

Git hooks can automatically format your Go code before each commit, ensuring you never commit unformatted code.

### Install Git Hooks

Run once after cloning the repository:

```bash
./scripts/install-hooks.sh
```

This installs:
- **pre-commit hook**: Automatically runs `gofmt -w` on all staged Go files before committing

### What the Hook Does

When you run `git commit`, the pre-commit hook:
1. Finds all staged `.go` files
2. Runs `gofmt -w` on each file
3. Re-stages the formatted files
4. Proceeds with the commit

You'll see output like:
```
Formatting auth/internal/handlers/auth.go
Formatting core/internal/pipeline/pipeline.go

✓ Go files have been automatically formatted with gofmt
```

### Bypassing the Hook

To commit without running hooks (not recommended):

```bash
git commit --no-verify
```

## Running Checks Locally

### 1. Format Check (gofmt)

Check if all Go files are properly formatted:

```bash
# Check formatting (returns list of unformatted files)
gofmt -l $(find . -name "*.go" -not -path "*/node_modules/*" -not -path "*/ocsf-schema/*" -not -path "*/vendor/*")

# Fix formatting automatically
gofmt -w .
```

**What CI checks:** All `.go` files must be formatted with `gofmt`. No configuration needed.

### 2. Go Vet

Run Go's built-in static analyzer:

```bash
# Run on all modules
for dir in auth cli common core ingest query storage web/backend tools/normalizer-generator tools/ocsf-generator; do
  echo "Vetting $dir"
  (cd "$dir" && go vet ./...)
done
```

**What CI checks:** Common Go mistakes, shadowed variables, nil pointer issues, etc.

### 3. Go Mod Tidy

Ensure `go.mod` and `go.sum` are clean:

```bash
# Check all modules
for dir in auth cli common core ingest query storage web/backend tools/normalizer-generator tools/ocsf-generator; do
  echo "Checking $dir"
  (cd "$dir" && go mod tidy)
done

# Check if any files changed
git diff go.mod go.sum
```

**What CI checks:** Verifies that `go.mod` and `go.sum` are tidy. If they're not, CI will fail.

### 4. Run Tests

Run all tests with race detection:

```bash
# Run tests in all modules
for dir in auth cli common core ingest query storage web/backend tools/normalizer-generator tools/ocsf-generator; do
  echo "Testing $dir"
  (cd "$dir" && go test -v -race -coverprofile=coverage.out ./...)
done
```

**What CI checks:** All tests must pass, and the race detector must not find any data races.

### 5. Linting (golangci-lint)

Install golangci-lint:

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Or download from https://github.com/golangci/golangci-lint/releases
```

Run the linter:

```bash
# Run on all modules
for dir in auth cli common core ingest query storage web/backend tools/normalizer-generator tools/ocsf-generator; do
  echo "Linting $dir"
  (cd "$dir" && golangci-lint run --timeout=5m)
done
```

**What CI checks:** 15+ linters including errcheck, gosec, govet, staticcheck, and more. Configuration in `.golangci.yml`.

### 6. Build All Services

Verify everything compiles:

```bash
# Build all services
cd auth && go build -v -o ../bin/auth ./cmd/auth && cd ..
cd ingest && go build -v -o ../bin/ingest ./cmd/ingest && cd ..
cd core && go build -v -o ../bin/core ./cmd/core && cd ..
cd storage && go build -v -o ../bin/storage ./cmd/storage && cd ..
cd query && go build -v -o ../bin/query ./cmd/query && cd ..
cd web/backend && go build -v -o ../../bin/web ./cmd/web && cd ../..
cd cli && go build -v -o ../bin/thawk . && cd ..
cd && go build -v -o ../../bin/event-seeder . && cd ../..
```

**What CI checks:** All services must compile successfully.

## Quick Pre-Push Script

Create a script to run all checks before pushing:

```bash
#!/bin/bash
# Save as scripts/pre-push.sh

set -e

echo "=== Running pre-push checks ==="

echo ""
echo "1. Checking formatting..."
unformatted=$(gofmt -l $(find . -name "*.go" -not -path "*/node_modules/*" -not -path "*/ocsf-schema/*" -not -path "*/vendor/*"))
if [ -n "$unformatted" ]; then
  echo "ERROR: The following files are not properly formatted:"
  echo "$unformatted"
  echo "Run 'gofmt -w .' to fix"
  exit 1
fi
echo "✓ All files properly formatted"

echo ""
echo "2. Running go mod tidy..."
for dir in auth cli common core ingest query storage web/backend tools/normalizer-generator tools/ocsf-generator; do
  (cd "$dir" && go mod tidy)
done
echo "✓ All go.mod files tidy"

echo ""
echo "3. Running go vet..."
for dir in auth cli common core ingest query storage web/backend tools/normalizer-generator tools/ocsf-generator; do
  echo "  Vetting $dir"
  (cd "$dir" && go vet ./...)
done
echo "✓ go vet passed"

echo ""
echo "4. Running tests..."
for dir in auth cli common core ingest query storage web/backend tools/normalizer-generator tools/ocsf-generator; do
  echo "  Testing $dir"
  (cd "$dir" && go test -race ./...)
done
echo "✓ All tests passed"

echo ""
echo "5. Running linter (this may take a minute)..."
for dir in auth cli common core ingest query storage web/backend tools/normalizer-generator tools/ocsf-generator; do
  echo "  Linting $dir"
  (cd "$dir" && golangci-lint run --timeout=5m)
done
echo "✓ Linting passed"

echo ""
echo "=== All checks passed! ==="
```

Make it executable:

```bash
chmod +x scripts/pre-push.sh
```

Run before pushing:

```bash
./scripts/pre-push.sh
```

## CI Configuration

### Workflow Configuration

The workflow is defined in `.github/workflows/go-build-test-lint.yml`:

- **Triggers:** Push to `main`, internal pull requests
- **Go Version:** 1.23 (matches project requirement)
- **Test Timeout:** 5 minutes per module
- **Artifact Retention:** Binaries kept for 7 days

### Linter Configuration

Linting configuration is in `.golangci.yml`:

- **Enabled Linters:** 15+ including security (gosec), style (gocritic, revive), and correctness (errcheck, staticcheck)
- **Excluded Paths:** Tests have relaxed rules, generated code and migrations are excluded
- **Timeout:** 5 minutes
- **Common Initialisms:** Custom list includes OCSF, HEC, JWT, etc.

To modify linter behavior, edit `.golangci.yml`.

### Coverage Reports

Coverage reports are generated locally for each module during test runs:

```bash
# Run tests with coverage
cd <module> && go test -coverprofile=coverage.out ./...

# View coverage report in browser
go tool cover -html=coverage.out
```

Coverage files are created as `coverage.out` in each module directory and are excluded from git (via `.gitignore`).

## Troubleshooting

### Linter Timeout

If golangci-lint times out, increase the timeout:

```yaml
# In .github/workflows/go-build-test-lint.yml
args: --timeout=10m  # Increase from 5m
```

### False Positives

To disable specific linter checks:

```go
// nolint:errcheck // Explanation of why this is safe
someFunction()
```

Or add exclusions to `.golangci.yml`:

```yaml
issues:
  exclude-rules:
    - text: "specific error message"
      linters:
        - linter-name
```

### Module-Specific Issues

If a specific module has unique requirements, create a `.golangci.yml` in that module's directory to override the root configuration.

## See Also

- [golangci-lint Documentation](https://golangci-lint.run/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Go Testing Documentation](https://golang.org/pkg/testing/)
