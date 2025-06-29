#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROXY_IMAGE="ratelimit-proxy:test"
PROXY_PORT=8088
BACKEND_PORT=8081

echo -e "${YELLOW}🚀 Starting integration tests for ratelimit-simple-proxy${NC}"

# Check if running on Linux
if ! grep -qi 'linux' /proc/version; then
    echo -e "${RED}This test runner requires Linux for --network host mode!${NC}"
    exit 1
fi

# Create test configuration first
echo -e "${YELLOW}📝 Creating test configuration...${NC}"
mkdir -p test/config
rm -f test/config/test-config.yaml
cat <<EOF > test/config/test-config.yaml
ipHeader:
  headers:
    - "X-Forwarded-For"
    - "X-Real-IP"

googleAuth:
  enabled: false
  authDomain: "auth.test.com"

rateLimits:
  test.com:
    destination: "http://localhost:$BACKEND_PORT"
    perSecond: 3
    requests: 5
EOF

# Cleanup function
cleanup() {
    echo -e "${YELLOW}🧹 Cleaning up...${NC}"
    docker stop proxy-test backend-test 2>/dev/null || true
    docker rm proxy-test backend-test 2>/dev/null || true
    docker rmi $PROXY_IMAGE 2>/dev/null || true
    rm -f test/config/test-config.yaml
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Build Docker image
echo -e "${YELLOW}🔨 Building Docker image...${NC}"
docker build -t $PROXY_IMAGE ..

# Start test backend server
echo -e "${YELLOW}🌐 Starting test backend server...${NC}"
docker run -d --name backend-test --network host \
    -v $(pwd)/backend:/app \
    python:3.9-alpine sh -c "
        pip install flask flask-socketio &&
        cd /app &&
        python app.py
    "

# Wait for backend to start
sleep 5

# Health check for backend
BACKEND_HEALTHY=0
for i in {1..10}; do
    if curl -s http://localhost:$BACKEND_PORT/hello | grep -q "Hello from backend"; then
        BACKEND_HEALTHY=1
        break
    fi
    sleep 1
done
if [ $BACKEND_HEALTHY -ne 1 ]; then
    echo -e "${RED}Backend is not responding on http://localhost:$BACKEND_PORT/hello${NC}"
    docker logs backend-test || true
    exit 1
fi

# Start proxy
echo -e "${YELLOW}🔄 Starting proxy...${NC}"
docker run -d --name proxy-test --network host \
    -v $(pwd)/test/config/test-config.yaml:/config.yaml \
    -e PROXY_PORT=$PROXY_PORT \
    $PROXY_IMAGE

# Wait for proxy to start
sleep 3

# Health check for proxy
PROXY_HEALTHY=0
for i in {1..10}; do
    if curl -s -H "Host: test.com" http://localhost:$PROXY_PORT/hello | grep -q "Hello from backend"; then
        PROXY_HEALTHY=1
        break
    fi
    sleep 1
done
if [ $PROXY_HEALTHY -ne 1 ]; then
    echo -e "${RED}Proxy is not responding on http://localhost:$PROXY_PORT/hello (Host: test.com)${NC}"
    docker logs proxy-test || true
    exit 1
fi

# Run all test suites
echo -e "${YELLOW}🧪 Running test suites...${NC}"

failed_tests=0

# Run each test suite
for test_file in test/suites/*.sh; do
    if [ -f "$test_file" ]; then
        echo -e "${YELLOW}Running $(basename "$test_file" .sh)...${NC}"
        if bash "$test_file" $PROXY_PORT; then
            echo -e "${GREEN}✓ $(basename "$test_file" .sh) passed${NC}"
        else
            echo -e "${RED}✗ $(basename "$test_file" .sh) failed${NC}"
            ((failed_tests++))
        fi
        echo
    fi
done

# Final results
echo -e "${YELLOW}📊 Test Results:${NC}"
if [ $failed_tests -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    echo -e "${GREEN}✓ Domain normalization (www/non-www) works${NC}"
    echo -e "${GREEN}✓ WebSocket support works${NC}"
    echo -e "${GREEN}✓ Rate limiting works${NC}"
    echo -e "${GREEN}✓ IP detection works${NC}"
    echo -e "${GREEN}✓ Unknown host handling works${NC}"
    echo -e "${GREEN}✓ Concurrent requests work${NC}"
    exit 0
else
    echo -e "${RED}❌ $failed_tests test suite(s) failed!${NC}"
    exit 1
fi