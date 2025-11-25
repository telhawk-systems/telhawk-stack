#!/bin/bash
# Start all TelHawk services in dev mode
# Run this inside the dev container or via: docker exec -it telhawk-dev bash scripts/dev/start-all.sh

set -e

echo "Starting TelHawk Stack in development mode..."
echo ""

# Build all services first
echo "==> Building services..."
/app/scripts/dev/build.sh

echo ""
echo "==> Services built. Starting in background with 'nohup'..."
echo ""

# Start each service in background
cd /app

echo "Starting authenticate on :8080..."
nohup /app/bin/authenticate > /var/log/telhawk/authenticate.log 2>&1 &
echo $! > /tmp/authenticate.pid

sleep 2

echo "Starting ingest on :8088..."
nohup /app/bin/ingest > /var/log/telhawk/ingest.log 2>&1 &
echo $! > /tmp/ingest.pid

echo "Starting search on :8082..."
nohup /app/bin/search > /var/log/telhawk/search.log 2>&1 &
echo $! > /tmp/search.pid

echo "Starting respond on :8085..."
nohup /app/bin/respond > /var/log/telhawk/respond.log 2>&1 &
echo $! > /tmp/respond.pid

echo "Starting web backend on :3000..."
nohup /app/bin/web > /var/log/telhawk/web.log 2>&1 &
echo $! > /tmp/web.pid

echo ""
echo "✓ All services started!"
echo ""
echo "Service logs:"
echo "  tail -f /var/log/telhawk/authenticate.log"
echo "  tail -f /var/log/telhawk/ingest.log"
echo "  tail -f /var/log/telhawk/search.log"
echo "  tail -f /var/log/telhawk/respond.log"
echo "  tail -f /var/log/telhawk/web.log"
echo ""
echo "To start frontend dev server (with hot reload):"
echo "  /app/scripts/dev/run-frontend.sh"
echo ""
echo "To stop all services:"
echo "  /app/scripts/dev/stop-all.sh"
