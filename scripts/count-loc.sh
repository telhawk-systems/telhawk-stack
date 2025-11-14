#!/bin/bash
# Count lines of code for all TelHawk services
#
# Excludes:
# - Test files (*_test.go)
# - Generated code (*_generated.go, generated/*)
# - Vendor dependencies
# - Third-party code

set -euo pipefail

# Change to project root
cd "$(dirname "$0")/.."

echo "TelHawk Stack - Lines of Code Report"
echo "====================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# List of services to analyze
services=(
    "auth"
    "ingest"
    "core"
    "storage"
    "query"
    "alerting"
    "rules"
    "web/backend"
    "web/frontend"
    "cli"
    "common"
    "tools"
)

total_loc=0
total_files=0

# Function to count lines in a directory
count_lines() {
    local dir=$1
    local service_name=$2

    if [ ! -d "$dir" ]; then
        return
    fi

    # Count Go files (excluding tests and generated)
    go_files=$(find "$dir" -name "*.go" \
        ! -name "*_test.go" \
        ! -name "*_generated.go" \
        ! -path "*/vendor/*" \
        ! -path "*/node_modules/*" \
        ! -path "*/generated/*" \
        2>/dev/null || echo "")

    # Count TypeScript/JavaScript files (excluding tests and node_modules)
    ts_files=$(find "$dir" \( -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx" \) \
        ! -name "*.test.ts" \
        ! -name "*.test.tsx" \
        ! -name "*.test.js" \
        ! -name "*.test.jsx" \
        ! -name "*.spec.ts" \
        ! -name "*.spec.tsx" \
        ! -name "*.spec.js" \
        ! -name "*.spec.jsx" \
        ! -path "*/node_modules/*" \
        ! -path "*/dist/*" \
        ! -path "*/build/*" \
        2>/dev/null || echo "")

    # Count Python files (excluding tests)
    py_files=$(find "$dir" -name "*.py" \
        ! -name "*_test.py" \
        ! -name "test_*.py" \
        ! -path "*/venv/*" \
        ! -path "*/__pycache__/*" \
        2>/dev/null || echo "")

    # Count shell scripts
    sh_files=$(find "$dir" -name "*.sh" \
        2>/dev/null || echo "")

    # Combine all files into a single list
    all_files=$(
        echo "$go_files"
        echo "$ts_files"
        echo "$py_files"
        echo "$sh_files"
    )

    # Remove empty lines
    all_files=$(echo "$all_files" | grep -v '^[[:space:]]*$' || echo "")

    if [ -z "$all_files" ]; then
        echo "0:0"
        return
    fi

    # Count lines in all files
    file_count=$(echo "$all_files" | wc -l)
    loc=$(echo "$all_files" | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")

    # If loc is empty or non-numeric, set to 0
    if ! [[ "$loc" =~ ^[0-9]+$ ]]; then
        loc="0"
    fi

    echo "$loc:$file_count"
}

# Function to get language breakdown
get_language_breakdown() {
    local dir=$1

    if [ ! -d "$dir" ]; then
        return
    fi

    # Count Go lines
    go_loc=$(find "$dir" -name "*.go" \
        ! -name "*_test.go" \
        ! -name "*_generated.go" \
        ! -path "*/vendor/*" \
        ! -path "*/generated/*" \
        2>/dev/null | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")

    # Count TypeScript/JavaScript lines
    ts_loc=$(find "$dir" \( -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx" \) \
        ! -name "*.test.*" \
        ! -name "*.spec.*" \
        ! -path "*/node_modules/*" \
        ! -path "*/dist/*" \
        ! -path "*/build/*" \
        2>/dev/null | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")

    # Count Python lines
    py_loc=$(find "$dir" -name "*.py" \
        ! -name "*_test.py" \
        ! -name "test_*.py" \
        ! -path "*/venv/*" \
        2>/dev/null | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")

    echo "$go_loc:$ts_loc:$py_loc"
}

printf "%-20s %12s %10s %s\n" "Service" "Lines" "Files" "Languages"
printf "%-20s %12s %10s %s\n" "-------" "-----" "-----" "---------"

for service in "${services[@]}"; do
    # Check if service directory exists
    if [ ! -d "$service" ]; then
        continue
    fi

    service_name=$(basename "$service")

    # Count lines
    result=$(count_lines "$service" "$service_name")
    loc=$(echo "$result" | cut -d: -f1)
    files=$(echo "$result" | cut -d: -f2)

    if [ "$loc" = "0" ]; then
        printf "%-20s ${BLUE}%12s${NC} %10s\n" "$service_name" "no code" "-"
        continue
    fi

    # Get language breakdown
    lang_breakdown=$(get_language_breakdown "$service")
    go_loc=$(echo "$lang_breakdown" | cut -d: -f1)
    ts_loc=$(echo "$lang_breakdown" | cut -d: -f2)
    py_loc=$(echo "$lang_breakdown" | cut -d: -f3)

    # Build language string
    langs=""
    [ "$go_loc" != "0" ] && langs="${langs}Go "
    [ "$ts_loc" != "0" ] && langs="${langs}TS/JS "
    [ "$py_loc" != "0" ] && langs="${langs}Python "
    [ -z "$langs" ] && langs="Other"

    # Color code based on size
    if [ "$loc" -gt 5000 ]; then
        color=$RED
    elif [ "$loc" -gt 2000 ]; then
        color=$YELLOW
    else
        color=$GREEN
    fi

    printf "%-20s ${color}%12s${NC} %10s %s\n" "$service_name" "$(printf "%'d" $loc)" "$files" "$langs"

    total_loc=$((total_loc + loc))
    total_files=$((total_files + files))
done

echo ""
echo "====================================="
printf "%-20s ${CYAN}%12s${NC} %10s\n" "TOTAL" "$(printf "%'d" $total_loc)" "$total_files"
echo ""

# Calculate some stats
if [ $total_files -gt 0 ]; then
    avg_lines_per_file=$((total_loc / total_files))
    echo "Average lines per file: $avg_lines_per_file"
fi

echo ""
echo "Excludes: test files, generated code, vendor/, node_modules/"
