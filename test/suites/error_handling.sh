#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROXY_PORT=$1

echo -e "${YELLOW}Test: Error handling${NC}"

# Test unknown host
echo -n "  Testing unknown host... "
response=$(curl -s -w "%{http_code}" -H "Host: unknown.com" \
    http://localhost:$PROXY_PORT/hello)

http_code="${response: -3}"

if [ "$http_code" = "502" ]; then
    echo -e "${GREEN}✓ (Correctly returned 502)${NC}"
else
    echo -e "${RED}✗ (Expected 502, got $http_code)${NC}"
    exit 1
fi

# Test concurrent requests
echo -n "  Testing 10 concurrent requests... "

# Make 10 concurrent requests
success_count=0
for i in {1..10}; do
    (
        response=$(curl -s -w "%{http_code}" -H "Host: test.com" -H "X-Forwarded-For: 192.168.1.$i" \
            http://localhost:$PROXY_PORT/hello)
        http_code="${response: -3}"
        if [ "$http_code" = "200" ]; then
            echo "success" >> /tmp/concurrent_test_results
        else
            echo "failed" >> /tmp/concurrent_test_results
        fi
    ) &
done

wait

success_count=$(grep -c "success" /tmp/concurrent_test_results 2>/dev/null || echo "0")
rm -f /tmp/concurrent_test_results

if [ $success_count -eq 10 ]; then
    echo -e "${GREEN}✓ (10/10 successful)${NC}"
else
    echo -e "${RED}✗ (Only $success_count/10 successful)${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Error handling test passed${NC}"