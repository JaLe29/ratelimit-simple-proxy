#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROXY_PORT=$1

echo -e "${YELLOW}Test: WebSocket support${NC}"

# Install websocket-client if not available
if ! python3 -c "import websocket" 2>/dev/null; then
    echo "  Installing websocket-client..."
    pip3 install websocket-client
fi

# Test WebSocket connection
echo -n "  Testing WebSocket connection... "
python3 test/clients/websocket_test.py localhost $PROXY_PORT test.com

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ WebSocket test passed${NC}"
else
    echo -e "${RED}✗ WebSocket test failed${NC}"
    exit 1
fi