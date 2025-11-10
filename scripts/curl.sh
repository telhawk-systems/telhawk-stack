#!/bin/bash
# Wrapper around curl that runs inside the TelHawk Docker network
# This allows you to make API calls to internal services (auth, rules, query, etc.)
# that are not exposed externally.
#
# Usage: ./scripts/curl.sh [curl options and arguments]
#
# Examples:
#   ./scripts/curl.sh http://rules:8084/api/v1/schemas
#   ./scripts/curl.sh -X POST http://auth:8080/api/v1/users -H "Content-Type: application/json" -d '{"username":"test"}'
#   ./scripts/curl.sh -s http://query:8082/health | jq '.'

set -e

# Determine the Docker network name
NETWORK="telhawk-stack_telhawk"

# Check if network exists
if ! docker network inspect "$NETWORK" > /dev/null 2>&1; then
    echo "Error: Docker network '$NETWORK' not found" >&2
    echo "Make sure the TelHawk stack is running with 'docker-compose up -d'" >&2
    exit 1
fi

# Run curl in a container on the TelHawk network
# --rm: Remove container after execution
# --network: Connect to TelHawk network
# telhawk-stack-devtools: Alpine-based image with curl, bash, jq, wget
docker run --rm --network "$NETWORK" telhawk-stack-devtools curl "$@"
