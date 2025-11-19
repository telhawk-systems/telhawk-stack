#!/bin/bash
# Wrapper to execute bash scripts inside the devtools container
# This allows scripts to access internal TelHawk services
#
# Usage: ./scripts/script_exec.sh <script-path>
#
# Examples:
#   ./scripts/script_exec.sh tmp/test_api.sh
#   ./scripts/script_exec.sh scripts/create_rules.sh
#
# The script will be executed with bash and has access to:
#   - curl, jq, wget
#   - All internal services (auth, rules, query, core, storage, opensearch)
#   - /tmp directory (mounted from host ./tmp)
#   - /scripts directory (mounted read-only from host ./scripts)

set -e

SCRIPT_PATH="$1"

if [ -z "$SCRIPT_PATH" ]; then
    echo "Usage: $0 <script-path>" >&2
    echo "" >&2
    echo "Examples:" >&2
    echo "  $0 tmp/my_script.sh" >&2
    echo "  $0 scripts/my_script.sh" >&2
    exit 1
fi

if [ ! -f "$SCRIPT_PATH" ]; then
    echo "Error: Script not found: $SCRIPT_PATH" >&2
    exit 1
fi

# Check if devtools container is running
if ! docker ps --format '{{.Names}}' | grep -q '^telhawk-devtools$'; then
    echo "Error: devtools container is not running" >&2
    echo "Start it with: docker-compose --profile devtools up -d devtools" >&2
    exit 1
fi

# Determine the path inside the container
# With the new setup, everything is at /workspace
CONTAINER_PATH="/workspace/${SCRIPT_PATH}"

# Execute the script in the devtools container
docker exec -i telhawk-devtools bash "$CONTAINER_PATH"
