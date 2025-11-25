#!/bin/bash
# Stop all TelHawk services

echo "Stopping all TelHawk services..."

for service in authenticate ingest search respond web; do
    if [ -f "/tmp/${service}.pid" ]; then
        pid=$(cat "/tmp/${service}.pid")
        if kill -0 "$pid" 2>/dev/null; then
            echo "Stopping $service (PID: $pid)..."
            kill "$pid"
            rm "/tmp/${service}.pid"
        else
            echo "$service not running (stale PID file)"
            rm "/tmp/${service}.pid"
        fi
    else
        echo "$service: no PID file found"
    fi
done

echo ""
echo "✓ All services stopped"
