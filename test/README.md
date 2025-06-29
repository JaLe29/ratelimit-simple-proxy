# Integration Tests

This directory contains comprehensive integration tests for the ratelimit-simple-proxy.

## Structure

```
test/
├── README.md                 # This file
├── run.sh                    # Main test runner
├── config/                   # Test configurations
│   └── test-config.yaml     # Generated test config
├── suites/                   # Individual test suites
│   ├── domain_normalization.sh  # Tests www/non-www domains
│   ├── websocket.sh             # Tests WebSocket support
│   ├── rate_limiting.sh         # Tests rate limiting
│   ├── ip_detection.sh          # Tests IP detection
│   └── error_handling.sh        # Tests error scenarios
├── clients/                  # Test clients
│   └── websocket_test.py     # WebSocket test client
└── backend/                  # Test backend server
    └── app.py                # Flask + WebSocket server
```

## Running Tests

### Run all tests
```bash
./test/run.sh
```

### Run individual test suite
```bash
# First start the infrastructure
docker build -t ratelimit-proxy:test .
docker run -d --name backend-test -p 8081:8080 -v $(pwd)/test/backend:/app python:3.9-alpine sh -c "pip install flask flask-socketio && python /app/app.py"
docker run -d --name proxy-test -p 8088:8080 -v $(pwd)/test/config/test-config.yaml:/app/config.yaml ratelimit-proxy:test

# Then run specific test
./test/suites/domain_normalization.sh 8088
./test/suites/websocket.sh 8088
./test/suites/rate_limiting.sh 8088
```

## What's Tested

### 1. Domain Normalization
- ✅ `test.com` works
- ✅ `www.test.com` works (normalized to `test.com`)
- ✅ Both use same rate limiter and configuration

### 2. WebSocket Support
- ✅ WebSocket upgrade through proxy
- ✅ Message echo functionality
- ✅ Connection handling

### 3. Rate Limiting
- ✅ Requests within limit (5 total, 3 per second)
- ✅ Rate limit exceeded (429 status)
- ✅ Per-IP rate limiting

### 4. IP Detection
- ✅ X-Forwarded-For header
- ✅ X-Real-IP header
- ✅ Fallback to "empty" when no headers

### 5. Error Handling
- ✅ Unknown host returns 502
- ✅ Concurrent requests work correctly
- ✅ Graceful error responses

## Docker Image

The test uses a tagged Docker image `ratelimit-proxy:test` to clearly identify test builds from production builds.

## Requirements

- Docker
- curl
- Python 3 (for WebSocket tests)
- bash

## Cleanup

Tests automatically clean up Docker containers and images on completion or failure.

## Docker Networking (Linux)

Pokud testy selhávají na připojení k backendu, je to pravděpodobně kvůli tomu, že Docker na Linuxu nepropojuje `localhost` mezi kontejnery a hostitelem. Tento test runner používá `host.docker.internal` a parametr `--add-host=host.docker.internal:host-gateway` pro správné propojení.

Pokud používáte starší Docker, může být potřeba aktualizovat Docker Engine.