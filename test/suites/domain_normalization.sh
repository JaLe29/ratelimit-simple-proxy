#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROXY_PORT=$1

echo -e "${YELLOW}Test: Domain normalization (www and non-www)${NC}"

# Test both www and non-www versions
for host in "test.com" "www.test.com"; do
    echo -n "  Testing $host... "
    response=$(curl -s -w "%{http_code}" -H "Host: $host" \
        http://localhost:$PROXY_PORT/hello)

    http_code="${response: -3}"
    body="${response%???}"

    if [ "$http_code" = "200" ] && echo "$body" | grep -q "Hello from backend"; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗ (Status: $http_code)${NC}"
        exit 1
    fi
done

echo -e "${GREEN}✓ Domain normalization test passed${NC}"