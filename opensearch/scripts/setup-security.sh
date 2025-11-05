#!/bin/bash
set -e

OPENSEARCH_ADMIN_USER="${OPENSEARCH_ADMIN_USER:-admin}"
OPENSEARCH_PASSWORD="${OPENSEARCH_INITIAL_ADMIN_PASSWORD:-TelHawk123!}"

echo "Configuring OpenSearch with custom credentials..."
echo "Admin user: ${OPENSEARCH_ADMIN_USER}"

# If security config doesn't exist or certs don't exist, run demo installer
if [ ! -f "/usr/share/opensearch/config/esnode.pem" ]; then
    echo "Generating demo SSL certificates..."
    /usr/share/opensearch/plugins/opensearch-security/tools/install_demo_configuration.sh -y -i -s
fi

# Wait for OpenSearch to start
echo "Waiting for OpenSearch to start..."
until curl -fk -u admin:admin https://localhost:9200 >/dev/null 2>&1; do
    sleep 2
done

echo "OpenSearch is up. Updating admin credentials..."

# Update or create admin user with the configured username and password
curl -fk -XPUT "https://localhost:9200/_plugins/_security/api/internalusers/${OPENSEARCH_ADMIN_USER}" \
    -u "admin:admin" \
    -H 'Content-Type: application/json' \
    -d "{
      \"password\": \"${OPENSEARCH_PASSWORD}\",
      \"backend_roles\": [\"admin\"],
      \"attributes\": {}
    }" && echo "User ${OPENSEARCH_ADMIN_USER} configured successfully" || echo "Warning: Failed to configure user"

# If username is not 'admin', try to delete the default admin user
if [ "${OPENSEARCH_ADMIN_USER}" != "admin" ]; then
    echo "Removing default admin user since custom username is specified..."
    curl -fk -XDELETE "https://localhost:9200/_plugins/_security/api/internalusers/admin" \
        -u "admin:admin" || echo "Default admin user already removed or cannot be deleted"
fi

echo "OpenSearch security configuration complete"
