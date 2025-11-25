#!/bin/bash
# Wrapper around curl that runs inside the TelHawk dev container
# This allows you to make API calls to internal services
#
# Usage: ./scripts/curl.sh [curl options and arguments]
#
# Examples:
#   ./scripts/curl.sh http://respond:8085/api/v1/schemas
#   ./scripts/curl.sh -X POST http://authenticate:8080/api/v1/users -H "Content-Type: application/json" -d '{"username":"test"}'
#   ./scripts/curl.sh -s http://search:8082/health | jq '.'

set -e

# Check if dev container is running
if ! docker ps --format '{{.Names}}' | grep -q '^telhawk-dev$'; then
    echo "Error: telhawk-dev container not running" >&2
    echo "Start it with: docker compose -f docker-compose.dev.yml up -d" >&2
    exit 1
fi

# Run curl inside the dev container
docker exec telhawk-dev curl "$@"
