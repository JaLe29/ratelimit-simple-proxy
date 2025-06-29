#!/usr/bin/env python3

import sys
import time
import websocket
import json

def test_websocket(host, port, domain):
    """Test WebSocket connection through proxy"""

    # Create WebSocket URL
    ws_url = f"ws://{host}:{port}"

    # Create WebSocket connection
    ws = websocket.create_connection(
        ws_url,
        header=[f"Host: {domain}"]
    )

    try:
        # Send test message
        test_message = {"type": "test", "message": "Hello WebSocket!"}
        ws.send(json.dumps(test_message))

        # Wait for response
        response = ws.recv()
        response_data = json.loads(response)

        # Verify response
        if response_data.get("type") == "test" and response_data.get("message") == "Hello WebSocket!":
            print("WebSocket echo test passed")
            return True
        else:
            print(f"WebSocket test failed - unexpected response: {response_data}")
            return False

    except Exception as e:
        print(f"WebSocket test failed with error: {e}")
        return False
    finally:
        ws.close()

if __name__ == "__main__":
    if len(sys.argv) != 4:
        print("Usage: python3 websocket_test.py <host> <port> <domain>")
        sys.exit(1)

    host = sys.argv[1]
    port = sys.argv[2]
    domain = sys.argv[3]

    success = test_websocket(host, port, domain)
    sys.exit(0 if success else 1)