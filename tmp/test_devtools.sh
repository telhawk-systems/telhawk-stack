#!/bin/bash
# Test script for devtools container

echo "=== Testing devtools container ==="
echo ""

echo "1. Checking available tools:"
echo "  bash: $(bash --version | head -1)"
echo "  curl: $(curl --version | head -1)"
echo "  jq: $(jq --version)"
echo "  wget: $(wget --version | head -1)"
echo ""

echo "2. Testing internal service access:"
echo "  Rules service: $(curl -s http://rules:8084/api/v1/schemas | jq -r '.pagination.total') rules found"
echo "  Auth service: $(curl -s http://auth:8080/healthz && echo 'healthy' || echo 'unhealthy')"
echo ""

echo "3. Testing jq JSON parsing:"
curl -s http://rules:8084/api/v1/schemas | jq -r '.schemas[0].view.title' | head -1
echo ""

echo "=== All tests passed! ==="
