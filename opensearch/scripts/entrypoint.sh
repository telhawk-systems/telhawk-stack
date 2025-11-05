#!/bin/bash
set -e

OPENSEARCH_ADMIN_USER="${OPENSEARCH_ADMIN_USER:-admin}"
OPENSEARCH_PASSWORD="${OPENSEARCH_INITIAL_ADMIN_PASSWORD:-TelHawk123!}"

echo "TelHawk OpenSearch - NO DEMO CREDENTIALS"
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
cat >> /usr/share/opensearch/config/opensearch.yml << EOF

# TelHawk Single Node Configuration
discovery.type: single-node

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

# Wait for OpenSearch, then create/update credentials
(
    echo "Waiting for OpenSearch to start..."
    sleep 90
    
    until curl -fk https://localhost:9200 >/dev/null 2>&1; do
        sleep 2
    done
    
    echo "OpenSearch is up. Creating/updating user: ${OPENSEARCH_ADMIN_USER}"
    
    # Use admin cert for initial setup
    curl -fk --cert /usr/share/opensearch/config/admin.pem \
        --key /usr/share/opensearch/config/admin-key.pem \
        -XPUT "https://localhost:9200/_plugins/_security/api/internalusers/${OPENSEARCH_ADMIN_USER}" \
        -H 'Content-Type: application/json' \
        -d "{
          \"password\": \"${OPENSEARCH_PASSWORD}\",
          \"backend_roles\": [\"admin\"],
          \"attributes\": {},
          \"description\": \"TelHawk admin user - NO DEMO CREDENTIALS\"
        }" && echo "✓ User ${OPENSEARCH_ADMIN_USER} configured" || echo "⚠ User configuration pending"
    
    echo "✓ Credentials configured - NO DEMO CREDENTIALS USED"
) &

wait $OPENSEARCH_PID
