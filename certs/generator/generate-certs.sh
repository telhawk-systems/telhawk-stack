#!/bin/sh
# TelHawk Stack Certificate Generator
# Generates self-signed certificates for all Go services
set -e

CERTS_DIR=${CERTS_DIR:-/certs}
GENERATED_DIR="$CERTS_DIR/generated"
PRODUCTION_DIR="$CERTS_DIR/production"

# Service names for certificate generation
SERVICES="auth ingest core storage query web"

echo "TelHawk Certificate Generator"
echo "=============================="

# Check if production certificates exist
if [ -d "$PRODUCTION_DIR" ] && [ "$(ls -A $PRODUCTION_DIR 2>/dev/null)" ]; then
    echo "✓ Production certificates found in $PRODUCTION_DIR"
    echo "  Skipping certificate generation (using provided certificates)"
    exit 0
fi

# Check if generated certificates already exist
if [ -d "$GENERATED_DIR" ] && [ -f "$GENERATED_DIR/auth.pem" ]; then
    echo "✓ Self-signed certificates already exist in $GENERATED_DIR"
    echo "  Skipping certificate generation"
    exit 0
fi

echo "Generating self-signed certificates..."
mkdir -p "$GENERATED_DIR"

# Generate CA certificate
echo "  → Creating Certificate Authority (CA)..."
openssl req -new -x509 -days 3650 -nodes \
    -subj "/CN=TelHawk Stack CA/O=TelHawk Systems/C=US" \
    -keyout "$GENERATED_DIR/ca-key.pem" \
    -out "$GENERATED_DIR/ca.pem" 2>/dev/null

# Generate certificate for each service
for SERVICE in $SERVICES; do
    echo "  → Generating certificate for $SERVICE..."
    
    # Create Subject Alternative Names configuration
    cat > "$GENERATED_DIR/$SERVICE.cnf" <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = $SERVICE.telhawk.local
O = TelHawk Systems
C = US

[v3_req]
keyUsage = keyEncipherment, dataEncipherment, digitalSignature
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = $SERVICE
DNS.2 = $SERVICE.telhawk.local
DNS.3 = telhawk-$SERVICE
DNS.4 = localhost
IP.1 = 127.0.0.1
EOF

    # Generate private key
    openssl genrsa -out "$GENERATED_DIR/$SERVICE-key.pem" 2048 2>/dev/null
    
    # Generate certificate signing request
    openssl req -new \
        -key "$GENERATED_DIR/$SERVICE-key.pem" \
        -out "$GENERATED_DIR/$SERVICE.csr" \
        -config "$GENERATED_DIR/$SERVICE.cnf" 2>/dev/null
    
    # Sign certificate with CA
    openssl x509 -req -days 3650 \
        -in "$GENERATED_DIR/$SERVICE.csr" \
        -CA "$GENERATED_DIR/ca.pem" \
        -CAkey "$GENERATED_DIR/ca-key.pem" \
        -CAcreateserial \
        -out "$GENERATED_DIR/$SERVICE.pem" \
        -extensions v3_req \
        -extfile "$GENERATED_DIR/$SERVICE.cnf" 2>/dev/null
    
    # Clean up CSR and config
    rm "$GENERATED_DIR/$SERVICE.csr" "$GENERATED_DIR/$SERVICE.cnf"
done

# Set appropriate permissions
chmod 644 "$GENERATED_DIR"/*.pem
chmod 600 "$GENERATED_DIR"/*-key.pem

echo ""
echo "✓ Certificate generation complete!"
echo "  Certificates stored in: $GENERATED_DIR"
echo "  CA certificate: ca.pem"
echo "  Service certificates: {service}.pem, {service}-key.pem"
echo ""
echo "To use production certificates, mount them to: $PRODUCTION_DIR"
