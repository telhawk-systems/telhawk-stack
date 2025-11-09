#!/bin/bash
# Pre-commit/pre-push checks for TelHawk Stack
# Runs the same checks that CI runs to catch issues before committing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# List of Go modules
MODULES=(
  "auth"
  "cli"
  "common"
  "core"
  "ingest"
  "query"
  "storage"
  "web/backend"
  "tools/event-seeder"
  "tools/normalizer-generator"
  "tools/ocsf-generator"
)

echo "=== Running pre-push checks for TelHawk Stack ==="
echo ""

# 1. Format check
echo -e "${YELLOW}1. Checking code formatting (gofmt)...${NC}"
unformatted=$(gofmt -l $(find . -name "*.go" -not -path "*/node_modules/*" -not -path "*/ocsf-schema/*" -not -path "*/vendor/*") 2>/dev/null || true)
if [ -n "$unformatted" ]; then
  echo -e "${RED}✗ ERROR: The following files are not properly formatted:${NC}"
  echo "$unformatted"
  echo ""
  echo "Run 'gofmt -w .' to fix formatting"
  exit 1
fi
echo -e "${GREEN}✓ All files properly formatted${NC}"
echo ""

# 2. Go mod tidy
echo -e "${YELLOW}2. Verifying go.mod files are tidy...${NC}"
for dir in "${MODULES[@]}"; do
  echo "  Checking $dir"
  (cd "$dir" && go mod tidy)
done
if ! git diff --exit-code --quiet go.mod go.sum 2>/dev/null; then
  echo -e "${RED}✗ ERROR: go.mod or go.sum files are not tidy${NC}"
  echo "Some modules had changes after 'go mod tidy'"
  echo "Run 'go mod tidy' in the affected directories and commit the changes"
  exit 1
fi
echo -e "${GREEN}✓ All go.mod files are tidy${NC}"
echo ""

# 3. Go vet
echo -e "${YELLOW}3. Running go vet...${NC}"
for dir in "${MODULES[@]}"; do
  echo "  Vetting $dir"
  (cd "$dir" && go vet ./...)
done
echo -e "${GREEN}✓ go vet passed${NC}"
echo ""

# 4. Tests
echo -e "${YELLOW}4. Running tests with race detector...${NC}"
failed_tests=()
for dir in "${MODULES[@]}"; do
  echo "  Testing $dir"
  if ! (cd "$dir" && go test -race ./... 2>&1); then
    failed_tests+=("$dir")
  fi
done
if [ ${#failed_tests[@]} -ne 0 ]; then
  echo -e "${RED}✗ ERROR: Tests failed in the following modules:${NC}"
  printf '  - %s\n' "${failed_tests[@]}"
  exit 1
fi
echo -e "${GREEN}✓ All tests passed${NC}"
echo ""

# 5. Linting (optional, can be slow)
if command -v golangci-lint &> /dev/null; then
  echo -e "${YELLOW}5. Running golangci-lint (this may take a minute)...${NC}"
  failed_lint=()
  for dir in "${MODULES[@]}"; do
    echo "  Linting $dir"
    if ! (cd "$dir" && golangci-lint run --timeout=5m 2>&1); then
      failed_lint+=("$dir")
    fi
  done
  if [ ${#failed_lint[@]} -ne 0 ]; then
    echo -e "${RED}✗ ERROR: Linting failed in the following modules:${NC}"
    printf '  - %s\n' "${failed_lint[@]}"
    exit 1
  fi
  echo -e "${GREEN}✓ Linting passed${NC}"
else
  echo -e "${YELLOW}⚠ golangci-lint not found, skipping lint checks${NC}"
  echo "  Install from: https://golangci-lint.run/welcome/install/"
fi
echo ""

echo -e "${GREEN}=== All checks passed! Safe to push. ===${NC}"
