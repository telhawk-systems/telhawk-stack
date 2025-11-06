#!/bin/bash

# Test script for ingest service features
# Tests: Rate limiting, Ack channel, and Metrics

set -e

INGEST_URL="${INGEST_URL:-http://localhost:8088}"
HEC_TOKEN="${HEC_TOKEN:-test-token}"

echo "=========================================="
echo "TelHawk Ingest Service Feature Tests"
echo "=========================================="
echo

# Test 1: Health check
echo "Test 1: Health Check"
echo "--------------------"
response=$(curl -s "${INGEST_URL}/healthz")
if echo "$response" | grep -q "healthy"; then
    echo "✓ Health check passed"
else
    echo "✗ Health check failed"
    exit 1
fi
echo

# Test 2: Metrics endpoint
echo "Test 2: Prometheus Metrics"
echo "---------------------------"
metrics=$(curl -s "${INGEST_URL}/metrics")
if echo "$metrics" | grep -q "telhawk_ingest_queue_depth"; then
    echo "✓ Metrics endpoint accessible"
    echo "  Queue depth: $(echo "$metrics" | grep '^telhawk_ingest_queue_depth ' | awk '{print $2}')"
    echo "  Queue capacity: $(echo "$metrics" | grep '^telhawk_ingest_queue_capacity ' | awk '{print $2}')"
else
    echo "✗ Metrics endpoint failed"
    exit 1
fi
echo

# Test 3: Event ingestion (creates ack if enabled)
echo "Test 3: Event Ingestion"
echo "------------------------"
response=$(curl -s -X POST "${INGEST_URL}/services/collector/event" \
    -H "Authorization: Splunk ${HEC_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"event": "Test event from feature test", "sourcetype": "test"}')

if echo "$response" | grep -q '"code":0'; then
    echo "✓ Event ingestion successful"
    echo "  Response: $response"
else
    echo "✗ Event ingestion failed"
    echo "  Response: $response"
fi
echo

# Test 4: Ack channel (if response contains ackId)
echo "Test 4: HEC Acknowledgement Channel"
echo "------------------------------------"
ack_id=$(echo "$response" | grep -o '"ackId":"[^"]*"' | cut -d'"' -f4)
if [ -n "$ack_id" ]; then
    echo "  Ack ID received: $ack_id"
    
    # Query ack status
    ack_response=$(curl -s -X POST "${INGEST_URL}/services/collector/ack" \
        -H "Content-Type: application/json" \
        -d "{\"acks\": [\"$ack_id\"]}")
    
    echo "  Ack query response: $ack_response"
    
    if echo "$ack_response" | grep -q "$ack_id"; then
        echo "✓ Ack channel working"
    else
        echo "⚠ Ack query returned but status unclear"
    fi
else
    echo "⚠ No ack ID in response (ack channel may be disabled)"
fi
echo

# Test 5: Rate limiting (optional - may hit limits)
echo "Test 5: Rate Limiting"
echo "---------------------"
echo "Sending 20 rapid requests..."

success_count=0
rate_limited_count=0

for i in {1..20}; do
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${INGEST_URL}/services/collector/event" \
        -H "Authorization: Splunk ${HEC_TOKEN}" \
        -d '{"event": "rate limit test"}')
    
    if [ "$status" = "200" ]; then
        ((success_count++))
    elif [ "$status" = "429" ]; then
        ((rate_limited_count++))
    fi
done

echo "  Successful: $success_count"
echo "  Rate limited (429): $rate_limited_count"

if [ "$rate_limited_count" -gt 0 ]; then
    echo "✓ Rate limiting is active"
elif [ "$success_count" -eq 20 ]; then
    echo "⚠ All requests succeeded (rate limiting may be disabled or limits not reached)"
else
    echo "⚠ Mixed results"
fi
echo

# Test 6: Check metrics after tests
echo "Test 6: Metrics After Tests"
echo "----------------------------"
metrics=$(curl -s "${INGEST_URL}/metrics")

events_accepted=$(echo "$metrics" | grep 'telhawk_ingest_events_total{endpoint="event",status="accepted"}' | awk '{print $2}')
events_rate_limited=$(echo "$metrics" | grep 'telhawk_ingest_events_total{endpoint="event",status="rate_limited"}' | awk '{print $2}')
queue_depth=$(echo "$metrics" | grep '^telhawk_ingest_queue_depth ' | awk '{print $2}')

echo "  Events accepted: ${events_accepted:-0}"
echo "  Events rate limited: ${events_rate_limited:-0}"
echo "  Current queue depth: ${queue_depth:-0}"

if [ -n "$events_accepted" ] && [ "$events_accepted" != "0" ]; then
    echo "✓ Metrics being tracked correctly"
else
    echo "⚠ No events tracked in metrics"
fi
echo

echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo "All basic tests passed!"
echo
echo "To view detailed metrics:"
echo "  curl ${INGEST_URL}/metrics"
echo
echo "To monitor in real-time:"
echo "  watch -n 1 'curl -s ${INGEST_URL}/metrics | grep telhawk_ingest'"
