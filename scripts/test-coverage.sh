#!/bin/bash
# Test coverage report for all TelHawk services
#
# Excludes packages typically not covered by unit tests:
# - cmd/*       (main functions, application entry points)
# - */config    (configuration loading)
# - */server    (server initialization)

set -euo pipefail

# Change to project root
cd "$(dirname "$0")/.."

echo "TelHawk Stack - Test Coverage Report"
echo "===================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# List of services to test
services=(
    "auth"
    "ingest"
    "core"
    "storage"
    "query"
    "alerting"
    "rules"
    "web/backend"
    "cli"
    "common"
)

total_coverage=0
service_count=0

for service in "${services[@]}"; do
    # Check if service directory exists
    if [ ! -d "$service" ]; then
        continue
    fi

    service_name=$(basename "$service")
    printf "%-20s " "$service_name"

    # Exclude packages that are typically not tested (main, config, server setup)
    # Get list of packages excluding cmd/, config, and server packages
    packages=$(go list "./${service}/..." 2>/dev/null | grep -v -e '/cmd/' -e '/config$' -e '/server$' || echo "")

    # Run tests with coverage for this service
    if [ -z "$packages" ]; then
        # No packages to test after exclusions
        echo "${BLUE}  no tests${NC}"
        continue
    fi

    if output=$(echo "$packages" | xargs go test -cover 2>&1); then
        # Extract all coverage percentages from output
        # Look for lines like "coverage: 85.7% of statements"
        coverages=$(echo "$output" | grep -oP 'coverage: \K[0-9.]+(?=% of statements)')

        if [ -n "$coverages" ]; then
            # Calculate average coverage across all packages
            sum=0
            count=0
            while IFS= read -r cov; do
                sum=$(echo "$sum + $cov" | bc)
                count=$((count + 1))
            done <<< "$coverages"

            if [ $count -gt 0 ]; then
                coverage=$(echo "scale=1; $sum / $count" | bc)

                # Color code based on coverage
                if (( $(echo "$coverage >= 80" | bc -l) )); then
                    color=$GREEN
                elif (( $(echo "$coverage >= 50" | bc -l) )); then
                    color=$YELLOW
                else
                    color=$RED
                fi

                printf "${color}%6.1f%%${NC}  (%d packages)\n" "$coverage" "$count"
                total_coverage=$(echo "$total_coverage + $coverage" | bc)
                service_count=$((service_count + 1))
            else
                echo "${YELLOW}    0.0%${NC}"
            fi
        else
            # Check if there are any test files
            test_files=$(find "$service" -name "*_test.go" 2>/dev/null | wc -l)
            if [ "$test_files" -eq 0 ]; then
                echo "${BLUE}  no tests${NC}"
            else
                echo "${YELLOW}    0.0%${NC}"
            fi
        fi
    else
        # Check for test failures vs no tests
        if echo "$output" | grep -q "no test files"; then
            echo "${BLUE}  no tests${NC}"
        elif echo "$output" | grep -q "\[no test files\]"; then
            echo "${BLUE}  no tests${NC}"
        else
            echo "${RED}   FAILED${NC}"
            if [ -n "${VERBOSE:-}" ]; then
                echo "$output" | grep -E "(FAIL|Error)" | head -3 | sed 's/^/  /'
            fi
        fi
    fi
done

echo ""
echo "===================================="

if [ $service_count -gt 0 ]; then
    avg_coverage=$(echo "scale=1; $total_coverage / $service_count" | bc)
    echo -e "Average coverage: ${GREEN}${avg_coverage}%${NC} across $service_count service(s)"
else
    echo "No services with tests found"
fi

echo ""
echo "Excludes: cmd/*, */config, */server (infrastructure code)"
echo "Run with VERBOSE=1 to see error details"
