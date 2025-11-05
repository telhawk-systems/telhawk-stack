#!/bin/bash
set -e

OPENSEARCH_PASSWORD="${OPENSEARCH_INITIAL_ADMIN_PASSWORD:-admin}"

echo "Configuring OpenSearch with custom password..."

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

echo "OpenSearch is up. Updating admin password..."

# Update admin password using the Security API
curl -fk -XPUT "https://localhost:9200/_plugins/_security/api/internalusers/admin" \
    -u "admin:admin" \
    -H 'Content-Type: application/json' \
    -d "{
      \"password\": \"${OPENSEARCH_PASSWORD}\",
      \"backend_roles\": [\"admin\"],
      \"attributes\": {}
    }" || echo "Password already set or API not ready"

echo "OpenSearch configuration complete"
