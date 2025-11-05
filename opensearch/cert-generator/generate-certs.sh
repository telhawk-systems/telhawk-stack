#!/bin/sh
set -e

CERT_DIR="/certs/generated"
PROD_CERT_DIR="/certs/production"

echo "TelHawk Certificate Generator - NO DEMO CREDENTIALS"

# Check for production certificates first
if [ -f "${PROD_CERT_DIR}/opensearch.pem" ] && [ -f "${PROD_CERT_DIR}/opensearch-key.pem" ] && [ -f "${PROD_CERT_DIR}/root-ca.pem" ]; then
    echo "✓ Production certificates found, using those"
    exit 0
fi

# Check if certificates already exist
if [ -f "${CERT_DIR}/opensearch.pem" ] && [ -f "${CERT_DIR}/opensearch-key.pem" ]; then
    echo "✓ Certificates already exist, skipping generation"
    exit 0
fi

echo "Generating self-signed certificates..."
mkdir -p "${CERT_DIR}"

# Generate CA certificate
openssl genrsa -out "${CERT_DIR}/root-ca-key.pem" 2048
openssl req -new -x509 -sha256 -days 3650 \
    -key "${CERT_DIR}/root-ca-key.pem" \
    -out "${CERT_DIR}/root-ca.pem" \
    -subj "/C=US/ST=State/L=City/O=TelHawk/OU=Security/CN=TelHawk-CA"

# Generate node certificate
openssl genrsa -out "${CERT_DIR}/opensearch-key.pem" 2048
openssl req -new -key "${CERT_DIR}/opensearch-key.pem" \
    -out "${CERT_DIR}/opensearch.csr" \
    -subj "/C=US/ST=State/L=City/O=TelHawk/OU=Security/CN=opensearch"

# Create SAN config
cat > "${CERT_DIR}/san.cnf" << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = opensearch
DNS.2 = localhost
DNS.3 = telhawk-opensearch
IP.1 = 127.0.0.1
EOF

# Sign node certificate with CA
openssl x509 -req -in "${CERT_DIR}/opensearch.csr" \
    -CA "${CERT_DIR}/root-ca.pem" \
    -CAkey "${CERT_DIR}/root-ca-key.pem" \
    -CAcreateserial \
    -out "${CERT_DIR}/opensearch.pem" \
    -days 365 \
    -sha256 \
    -extensions v3_req \
    -extfile "${CERT_DIR}/san.cnf"

# Generate admin certificate
openssl genrsa -out "${CERT_DIR}/admin-key.pem" 2048
openssl req -new -key "${CERT_DIR}/admin-key.pem" \
    -out "${CERT_DIR}/admin.csr" \
    -subj "/C=US/ST=State/L=City/O=TelHawk/OU=Security/CN=admin"

openssl x509 -req -in "${CERT_DIR}/admin.csr" \
    -CA "${CERT_DIR}/root-ca.pem" \
    -CAkey "${CERT_DIR}/root-ca-key.pem" \
    -CAcreateserial \
    -out "${CERT_DIR}/admin.pem" \
    -days 365 \
    -sha256

# Set permissions - make readable by all (opensearch user needs to read)
chmod 755 "${CERT_DIR}"
chmod 644 "${CERT_DIR}"/*.pem
chmod 644 "${CERT_DIR}"/*-key.pem  # Keys need to be readable by opensearch user
rm -f "${CERT_DIR}"/*.csr "${CERT_DIR}"/*.srl "${CERT_DIR}/san.cnf"

echo "✓ Self-signed certificates generated successfully"
echo "✓ Certificates stored in: ${CERT_DIR}"
echo "✓ NO DEMO CREDENTIALS USED"
