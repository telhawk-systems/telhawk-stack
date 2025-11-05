#!/bin/bash
set -e

OPENSEARCH_ADMIN_USER="${OPENSEARCH_ADMIN_USER:-admin}"
OPENSEARCH_PASSWORD="${OPENSEARCH_INITIAL_ADMIN_PASSWORD:-TelHawk123!}"

echo "Setting up OpenSearch security..."

# Check if we have proper certificates, if not generate self-signed ones
if [ ! -f "/usr/share/opensearch/config/esnode.pem" ]; then
    if [ -f "/usr/share/opensearch/config/certs/opensearch.pem" ] && [ -f "/usr/share/opensearch/config/certs/opensearch-key.pem" ]; then
        echo "Using provided certificates from /usr/share/opensearch/config/certs/"
        cp /usr/share/opensearch/config/certs/* /usr/share/opensearch/config/
    else
        echo "No certificates found. Generating self-signed certificates..."
        
        # Generate self-signed certificate
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout /usr/share/opensearch/config/opensearch-key.pem \
            -out /usr/share/opensearch/config/opensearch.pem \
            -subj "/C=US/ST=State/L=City/O=TelHawk/OU=Security/CN=opensearch" \
            -addext "subjectAltName=DNS:opensearch,DNS:localhost,IP:127.0.0.1"
        
        # Create admin cert (same as node cert for simplicity)
        cp /usr/share/opensearch/config/opensearch.pem /usr/share/opensearch/config/admin.pem
        cp /usr/share/opensearch/config/opensearch-key.pem /usr/share/opensearch/config/admin-key.pem
        
        # Set proper permissions
        chmod 600 /usr/share/opensearch/config/*-key.pem
        chmod 644 /usr/share/opensearch/config/*.pem
        
        # Create CA (use same cert as CA for self-signed)
        cp /usr/share/opensearch/config/opensearch.pem /usr/share/opensearch/config/root-ca.pem
        
        # Update opensearch.yml to use our certificates
        cat >> /usr/share/opensearch/config/opensearch.yml << EOF

# Custom SSL Configuration
plugins.security.ssl.transport.pemcert_filepath: opensearch.pem
plugins.security.ssl.transport.pemkey_filepath: opensearch-key.pem
plugins.security.ssl.transport.pemtrustedcas_filepath: root-ca.pem
plugins.security.ssl.transport.enforce_hostname_verification: false
plugins.security.ssl.http.enabled: true
plugins.security.ssl.http.pemcert_filepath: opensearch.pem
plugins.security.ssl.http.pemkey_filepath: opensearch-key.pem
plugins.security.ssl.http.pemtrustedcas_filepath: root-ca.pem
plugins.security.allow_unsafe_democertificates: false
plugins.security.authcz.admin_dn:
  - CN=opensearch,OU=Security,O=TelHawk,L=City,ST=State,C=US
EOF
        
        echo "Self-signed certificates generated successfully"
    fi
fi

# Start OpenSearch in background
echo "Starting OpenSearch..."
/usr/share/opensearch/bin/opensearch &
OPENSEARCH_PID=$!

# Wait for OpenSearch to be ready, then update credentials
(
    echo "Waiting for OpenSearch to start..."
    sleep 60
    
    # Try with default creds first
    until curl -fk -u admin:admin https://localhost:9200 >/dev/null 2>&1; do
        sleep 2
    done
    
    echo "OpenSearch is up. Updating credentials to user=${OPENSEARCH_ADMIN_USER}..."
    
    # Update or create admin user with custom password
    curl -fk -XPUT "https://localhost:9200/_plugins/_security/api/internalusers/${OPENSEARCH_ADMIN_USER}" \
        -u "admin:admin" \
        -H 'Content-Type: application/json' \
        -d "{
          \"password\": \"${OPENSEARCH_PASSWORD}\",
          \"backend_roles\": [\"admin\"],
          \"attributes\": {}
        }" && echo "User ${OPENSEARCH_ADMIN_USER} configured successfully" || echo "Warning: Failed to configure user"
    
    if [ "${OPENSEARCH_ADMIN_USER}" != "admin" ]; then
        echo "Note: Default 'admin' user still exists. Custom user '${OPENSEARCH_ADMIN_USER}' has been created."
    fi
    
    echo "OpenSearch security configuration complete"
) &

# Wait for OpenSearch process
wait $OPENSEARCH_PID
