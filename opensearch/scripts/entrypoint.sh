#!/bin/bash
set -e

OPENSEARCH_PASSWORD="${OPENSEARCH_INITIAL_ADMIN_PASSWORD:-TelHawk123!}"

# Start OpenSearch in background
/usr/share/opensearch/bin/opensearch &
OPENSEARCH_PID=$!

# Run security setup in background after delay
(
    sleep 60
    /usr/local/bin/setup-security.sh
) &

# Wait for OpenSearch process
wait $OPENSEARCH_PID
