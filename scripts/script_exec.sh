#!/bin/bash
# Wrapper to execute bash scripts inside the dev container
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
#   - All internal services (authenticate, search, respond, ingest)
#   - /app directory (bind-mounted from host)

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

# Check if dev container is running
if ! docker ps --format '{{.Names}}' | grep -q '^telhawk-dev$'; then
    echo "Error: telhawk-dev container not running" >&2
    echo "Start it with: docker compose -f docker-compose.dev.yml up -d" >&2
    exit 1
fi

# Path inside the container (everything is at /app)
CONTAINER_PATH="/app/${SCRIPT_PATH}"

# Execute the script in the dev container
docker exec -i telhawk-dev bash "$CONTAINER_PATH"
