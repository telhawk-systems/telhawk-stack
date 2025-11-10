#!/bin/bash
set -e

OPENSEARCH_ADMIN_USER="${OPENSEARCH_ADMIN_USER:-admin}"
OPENSEARCH_PASSWORD="${OPENSEARCH_INITIAL_ADMIN_PASSWORD:-TelHawk123!}"

echo "TelHawk OpenSearch - Synchronous Startup"
echo "Setting up OpenSearch with user: ${OPENSEARCH_ADMIN_USER}"

# Determine which certificates to use
CERT_SOURCE=""
if [ -f "/certs/production/opensearch.pem" ]; then
    CERT_SOURCE="/certs/production"
    echo "Using production certificates"
elif [ -f "/certs/generated/opensearch.pem" ]; then
    CERT_SOURCE="/certs/generated"
    echo "Using generated self-signed certificates"
else
    echo "ERROR: No certificates found! cert-generator must run first."
    exit 1
fi

# Copy certificates to OpenSearch config
cp "${CERT_SOURCE}"/*.pem /usr/share/opensearch/config/
chmod 644 /usr/share/opensearch/config/*.pem
chmod 644 /usr/share/opensearch/config/*-key.pem

# Configure OpenSearch - single-node + SSL
# Remove any existing cluster.initial_cluster_manager_nodes to avoid duplicates
sed -i '/^cluster\.initial_cluster_manager_nodes:/d' /usr/share/opensearch/config/opensearch.yml

cat >> /usr/share/opensearch/config/opensearch.yml << 'EOF'

# TelHawk Single Node Configuration
cluster.initial_cluster_manager_nodes: ["opensearch"]
node.name: opensearch

# TelHawk SSL Configuration - NO DEMO CERTS
plugins.security.ssl.transport.pemcert_filepath: opensearch.pem
plugins.security.ssl.transport.pemkey_filepath: opensearch-key.pem
plugins.security.ssl.transport.pemtrustedcas_filepath: root-ca.pem
plugins.security.ssl.transport.enforce_hostname_verification: false
plugins.security.ssl.http.enabled: true
plugins.security.ssl.http.pemcert_filepath: opensearch.pem
plugins.security.ssl.http.pemkey_filepath: opensearch-key.pem
plugins.security.ssl.http.pemtrustedcas_filepath: root-ca.pem
plugins.security.allow_unsafe_democertificates: false
plugins.security.allow_default_init_securityindex: true
plugins.security.authcz.admin_dn:
  - CN=admin,OU=Security,O=TelHawk,L=City,ST=State,C=US
EOF

# Start OpenSearch in background
echo "Starting OpenSearch..."
/usr/share/opensearch/bin/opensearch &
OPENSEARCH_PID=$!

# Wait for OpenSearch to be ready (using admin cert - no password needed)
echo "Waiting for OpenSearch to be ready..."
MAX_WAIT=180
WAITED=0
until curl -fsk --cert /usr/share/opensearch/config/admin.pem \
    --key /usr/share/opensearch/config/admin-key.pem \
    https://localhost:9200/_cluster/health >/dev/null 2>&1; do

    if [ $WAITED -ge $MAX_WAIT ]; then
        echo "ERROR: OpenSearch failed to start within ${MAX_WAIT} seconds"
        kill $OPENSEARCH_PID 2>/dev/null || true
        exit 1
    fi

    sleep 2
    WAITED=$((WAITED + 2))
    echo "  ... waited ${WAITED}s"
done

echo "✓ OpenSearch is ready"

# Wait for security plugin to be fully initialized
echo "Waiting for security plugin to initialize..."
WAITED=0
until curl -sk --cert /usr/share/opensearch/config/admin.pem \
    --key /usr/share/opensearch/config/admin-key.pem \
    https://localhost:9200/_plugins/_security/api/internalusers 2>/dev/null | grep -q "admin"; do

    if [ $WAITED -ge 60 ]; then
        echo "ERROR: Security plugin failed to initialize within 60 seconds"
        kill $OPENSEARCH_PID 2>/dev/null || true
        exit 1
    fi

    sleep 2
    WAITED=$((WAITED + 2))
    echo "  ... waited ${WAITED}s"
done

echo "✓ Security plugin initialized"

# Now configure/update admin user credentials synchronously
echo "Configuring user: ${OPENSEARCH_ADMIN_USER}"

HTTP_CODE=$(curl -sk --cert /usr/share/opensearch/config/admin.pem \
    --key /usr/share/opensearch/config/admin-key.pem \
    -w "%{http_code}" \
    -o /tmp/curl_response.txt \
    -XPUT "https://localhost:9200/_plugins/_security/api/internalusers/${OPENSEARCH_ADMIN_USER}" \
    -H 'Content-Type: application/json' \
    -d "{
      \"password\": \"${OPENSEARCH_PASSWORD}\",
      \"backend_roles\": [\"admin\"],
      \"attributes\": {},
      \"description\": \"TelHawk admin user - NO DEMO CREDENTIALS\"
    }")

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    echo "✓ User ${OPENSEARCH_ADMIN_USER} configured successfully"
else
    echo "ERROR: Failed to configure user (HTTP $HTTP_CODE)"
    echo "Response: $(cat /tmp/curl_response.txt 2>/dev/null)"
    kill $OPENSEARCH_PID 2>/dev/null || true
    exit 1
fi

# Verify authentication works with the configured password
echo "Verifying authentication..."
if curl -fsk -u "${OPENSEARCH_ADMIN_USER}:${OPENSEARCH_PASSWORD}" \
    https://localhost:9200/_cluster/health >/dev/null 2>&1; then
    echo "✓ Authentication verified"
else
    echo "ERROR: Authentication verification failed"
    kill $OPENSEARCH_PID 2>/dev/null || true
    exit 1
fi

echo "✓ OpenSearch fully configured and ready for connections"
echo "  Credentials: ${OPENSEARCH_ADMIN_USER} / <configured>"

# Wait for OpenSearch process (this becomes the main process)
wait $OPENSEARCH_PID
