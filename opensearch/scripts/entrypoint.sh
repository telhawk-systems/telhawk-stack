#!/bin/bash
set -e

OPENSEARCH_ADMIN_USER="${OPENSEARCH_ADMIN_USER:-admin}"
OPENSEARCH_PASSWORD="${OPENSEARCH_INITIAL_ADMIN_PASSWORD:-TelHawk123!}"

echo "Setting up OpenSearch security..."

# MUST run demo installer before starting OpenSearch
if [ ! -f "/usr/share/opensearch/config/esnode.pem" ]; then
    echo "Generating demo SSL certificates..."
    /usr/share/opensearch/plugins/opensearch-security/tools/install_demo_configuration.sh -y -i -s
fi

# Start OpenSearch in background
echo "Starting OpenSearch..."
/usr/share/opensearch/bin/opensearch &
OPENSEARCH_PID=$!

# Wait for OpenSearch to be ready, then update credentials
(
    echo "Waiting for OpenSearch to start..."
    sleep 60
    until curl -fk -u admin:admin https://localhost:9200 >/dev/null 2>&1; do
        sleep 2
    done
    
    echo "OpenSearch is up. Updating credentials to user=${OPENSEARCH_ADMIN_USER}..."
    
    # Update or create admin user
    curl -fk -XPUT "https://localhost:9200/_plugins/_security/api/internalusers/${OPENSEARCH_ADMIN_USER}" \
        -u "admin:admin" \
        -H 'Content-Type: application/json' \
        -d "{
          \"password\": \"${OPENSEARCH_PASSWORD}\",
          \"backend_roles\": [\"admin\"],
          \"attributes\": {}
        }" && echo "User ${OPENSEARCH_ADMIN_USER} configured successfully" || echo "Warning: Failed to configure user"
    
    # If custom username, optionally delete default admin
    if [ "${OPENSEARCH_ADMIN_USER}" != "admin" ]; then
        echo "Note: Default 'admin' user still exists. Custom user '${OPENSEARCH_ADMIN_USER}' has been created."
    fi
    
    echo "OpenSearch security configuration complete"
) &

# Wait for OpenSearch process
wait $OPENSEARCH_PID
