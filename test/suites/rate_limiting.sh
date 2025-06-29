#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROXY_PORT=$1

echo -e "${YELLOW}Test: Rate limiting${NC}"

# Make requests within rate limit (5 total requests allowed)
echo -n "  Testing requests within rate limit... "
success_count=0
for i in {1..5}; do
    response=$(curl -s -w "%{http_code}" -H "Host: test.com" -H "X-Forwarded-For: 192.168.1.$i" \
        http://localhost:$PROXY_PORT/hello)

    http_code="${response: -3}"
    if [ "$http_code" = "200" ]; then
        ((success_count++))
    fi
done

if [ $success_count -eq 5 ]; then
    echo -e "${GREEN}✓ (5/5 successful)${NC}"
else
    echo -e "${RED}✗ (Only $success_count/5 successful)${NC}"
    exit 1
fi

# Test rate limit exceeded
echo -n "  Testing rate limit exceeded... "
response=$(curl -s -w "%{http_code}" -H "Host: test.com" -H "X-Forwarded-For: 192.168.1.6" \
    http://localhost:$PROXY_PORT/hello)

http_code="${response: -3}"
if [ "$http_code" = "429" ]; then
    echo -e "${GREEN}✓ (Correctly blocked with 429)${NC}"
else
    echo -e "${RED}✗ (Expected 429, got $http_code)${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Rate limiting test passed${NC}"