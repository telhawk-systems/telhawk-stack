#!/bin/bash
# Build all TelHawk services
set -e

cd /app

echo "Building authenticate..."
cd authenticate && go build -o ../bin/authenticate ./cmd/authenticate && cd ..

echo "Building ingest..."
cd ingest && go build -o ../bin/ingest ./cmd/ingest && cd ..

echo "Building search..."
cd search && go build -o ../bin/search ./cmd/search && cd ..

echo "Building respond..."
cd respond && go build -o ../bin/respond ./cmd/respond && cd ..

echo "Building web backend..."
cd web/backend && go build -o ../../bin/web ./cmd/web && cd ../..

echo "Building CLI..."
cd cli && go build -o ../bin/thawk ./cmd/thawk && cd ..

echo ""
echo "All services built in /app/bin/"
ls -la bin/
