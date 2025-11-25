#!/bin/bash
# Run a single TelHawk service
# Usage: ./scripts/dev/run.sh <service>
# Example: ./scripts/dev/run.sh ingest

SERVICE=$1

if [ -z "$SERVICE" ]; then
    echo "Usage: $0 <service>"
    echo "Available services: authenticate, ingest, search, respond, web"
    exit 1
fi

cd /app

case $SERVICE in
    authenticate)
        cd authenticate
        CONFIG_FILE=config.yaml go run ./cmd/authenticate
        ;;
    ingest)
        cd ingest
        CONFIG_FILE=config.yaml go run ./cmd/ingest
        ;;
    search)
        cd search
        CONFIG_FILE=config.yaml go run ./cmd/search
        ;;
    respond)
        cd respond
        CONFIG_FILE=config.yaml go run ./cmd/respond
        ;;
    web)
        cd web/backend
        CONFIG_FILE=config.yaml go run ./cmd/web
        ;;
    *)
        echo "Unknown service: $SERVICE"
        echo "Available services: authenticate, ingest, search, respond, web"
        exit 1
        ;;
esac
