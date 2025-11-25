#!/bin/bash
# Run frontend in dev mode with hot reload
set -e

cd /app/web/frontend

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing frontend dependencies..."
    npm install
fi

# Run Vite dev server
# --host 0.0.0.0 allows external access from host machine
echo "Starting Vite dev server on http://localhost:5173"
npm run dev -- --host 0.0.0.0
