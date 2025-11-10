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
# If script is in ./tmp, it's at /tmp in container
# If script is in ./scripts, it's at /scripts in container
if [[ "$SCRIPT_PATH" == tmp/* ]]; then
    CONTAINER_PATH="/${SCRIPT_PATH}"
elif [[ "$SCRIPT_PATH" == scripts/* ]]; then
    CONTAINER_PATH="/${SCRIPT_PATH}"
else
    # For absolute paths or other locations, try to map them
    CONTAINER_PATH="/$SCRIPT_PATH"
fi

# Execute the script in the devtools container
docker exec -i telhawk-devtools bash "$CONTAINER_PATH"
