#!/bin/bash
set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TOKEN_FILE=".hec-token"
SEEDER_NAME="First Time Seeder"
HEC_URL="http://localhost:8088"
AUTH_URL="http://auth:8080"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}TelHawk Stack - First Time Event Seeding${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Step 1: Check if Docker services are running
echo -e "${YELLOW}[1/6]${NC} Checking Docker services..."

if ! docker-compose ps | grep -q "telhawk-auth.*Up.*healthy"; then
    echo -e "${YELLOW}→${NC} Starting required services (auth, ingest, core, storage, opensearch)..."
    docker-compose up -d auth ingest core storage opensearch web

    echo -e "${YELLOW}→${NC} Waiting for services to become healthy (this may take 30-60 seconds)..."
    sleep 10

    # Wait for services to be healthy
    MAX_WAIT=60
    ELAPSED=0
    while [ $ELAPSED -lt $MAX_WAIT ]; do
        if docker-compose ps | grep -q "telhawk-auth.*Up.*healthy" && \
           docker-compose ps | grep -q "telhawk-ingest.*Up.*healthy"; then
            echo -e "${GREEN}✓${NC} Services are healthy"
            break
        fi
        sleep 5
        ELAPSED=$((ELAPSED + 5))
        echo -e "${YELLOW}→${NC} Still waiting... (${ELAPSED}s/${MAX_WAIT}s)"
    done

    if [ $ELAPSED -ge $MAX_WAIT ]; then
        echo -e "${RED}✗${NC} Timeout waiting for services to become healthy"
        echo -e "${YELLOW}→${NC} Check service logs with: docker-compose logs auth ingest"
        exit 1
    fi
else
    echo -e "${GREEN}✓${NC} Services are already running"
fi

# Step 2: Build CLI if needed
echo -e "\n${YELLOW}[2/6]${NC} Checking CLI build..."

if [ ! -f "cli/bin/thawk" ]; then
    echo -e "${YELLOW}→${NC} Building CLI tool..."
    cd cli && go build -o bin/thawk . && cd ..
    echo -e "${GREEN}✓${NC} CLI built successfully"
else
    echo -e "${GREEN}✓${NC} CLI already built"
fi

# Step 3: Login to auth service
echo -e "\n${YELLOW}[3/6]${NC} Authenticating with auth service..."

docker-compose run --rm thawk auth login -u admin -p admin123 --auth-url "$AUTH_URL" > /dev/null 2>&1
echo -e "${GREEN}✓${NC} Authenticated as admin"

# Step 4: Check for existing HEC token or create new one
echo -e "\n${YELLOW}[4/6]${NC} Managing HEC token..."

# Function to validate token
validate_token() {
    local token=$1
    # Try to use the token with a simple request
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Splunk $token" \
        "$HEC_URL/services/collector/event" \
        -X POST -d '{"event":"test"}' 2>/dev/null || echo "000")

    # 200 = success, 401 = unauthorized, 400 = bad request (but token is valid)
    if [ "$response" = "200" ] || [ "$response" = "400" ]; then
        return 0
    else
        return 1
    fi
}

TOKEN=""
TOKEN_VALID=false

# Check if token file exists and is valid
if [ -f "$TOKEN_FILE" ]; then
    TOKEN=$(cat "$TOKEN_FILE" | tr -d '\n')
    echo -e "${YELLOW}→${NC} Found existing token file, validating..."

    if validate_token "$TOKEN"; then
        echo -e "${GREEN}✓${NC} Existing token is valid"
        TOKEN_VALID=true
    else
        echo -e "${YELLOW}→${NC} Existing token is invalid, will create new one"
    fi
fi

# Create new token if needed
if [ "$TOKEN_VALID" = false ]; then
    echo -e "${YELLOW}→${NC} Creating new HEC token..."

    # Login via web API to get session cookie
    curl -s -X POST http://localhost:3000/api/auth/login \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' \
        -c /tmp/seed-cookies.txt > /dev/null

    # Create HEC token via web API
    TOKEN_RESPONSE=$(curl -s -b /tmp/seed-cookies.txt \
        -X POST http://localhost:3000/api/auth/api/v1/hec/tokens \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"$SEEDER_NAME\"}")

    TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.data.attributes.token')

    if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
        echo -e "${RED}✗${NC} Failed to create HEC token"
        echo "Response: $TOKEN_RESPONSE"
        exit 1
    fi

    # Save token to file
    echo "$TOKEN" > "$TOKEN_FILE"
    echo -e "${GREEN}✓${NC} Created and saved new HEC token: $TOKEN"

    # Clean up cookie file
    rm -f /tmp/seed-cookies.txt
fi

# Step 5: Run seeder for all detection rules
echo -e "\n${YELLOW}[5/6]${NC} Generating events from detection rules..."
echo -e "${YELLOW}→${NC} This will generate events matching all supported detection rules\n"

cd cli
./bin/thawk seeder run \
    --from-rules ../alerting/rules/ \
    --token "$TOKEN" \
    --hec-url "$HEC_URL" 2>&1 | grep -E "Loading rules|Generating events|✓ Sent|⚠ WARN|Seeding complete|Events sent"

cd ..

# Step 6: Verify events in OpenSearch
echo -e "\n${YELLOW}[6/6]${NC} Verifying events in OpenSearch..."

sleep 2  # Give OpenSearch a moment to index

EVENT_COUNT=$(curl -s -k -u admin:TelHawk123! \
    "https://localhost:9200/telhawk-events-*/_count" | jq -r '.count')

if [ -z "$EVENT_COUNT" ] || [ "$EVENT_COUNT" = "null" ]; then
    echo -e "${YELLOW}⚠${NC} Could not verify event count in OpenSearch"
else
    echo -e "${GREEN}✓${NC} Total events in OpenSearch: $EVENT_COUNT"
fi

# Final summary
echo -e "\n${BLUE}========================================${NC}"
echo -e "${GREEN}✓ Seeding Complete!${NC}"
echo -e "${BLUE}========================================${NC}\n"

echo -e "HEC Token: ${GREEN}$TOKEN${NC}"
echo -e "Token saved to: ${YELLOW}$TOKEN_FILE${NC} (git-ignored)\n"

echo -e "${BLUE}Next Steps:${NC}"
echo -e "  1. View events in web UI: ${YELLOW}http://localhost:3000${NC}"
echo -e "  2. Query OpenSearch directly:"
echo -e "     ${YELLOW}curl -k -u admin:TelHawk123! 'https://localhost:9200/telhawk-events-*/_search?size=5'${NC}"
echo -e "  3. Re-run seeder anytime:"
echo -e "     ${YELLOW}./scripts/first-time-seed.sh${NC}\n"
