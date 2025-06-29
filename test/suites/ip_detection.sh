#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROXY_PORT=$1

echo -e "${YELLOW}Test: IP detection${NC}"

# Test different IP headers
for header in "X-Forwarded-For" "X-Real-IP"; do
    echo -n "  Testing $header... "
    response=$(curl -s -w "%{http_code}" -H "Host: test.com" -H "$header: 10.0.0.1" \
        http://localhost:$PROXY_PORT/ip)

    http_code="${response: -3}"
    body="${response%???}"

    if [ "$http_code" = "200" ] && echo "$body" | grep -q "10.0.0.1"; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗ (Status: $http_code)${NC}"
        exit 1
    fi
done

echo -e "${GREEN}✓ IP detection test passed${NC}"