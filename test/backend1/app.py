#!/usr/bin/env python3

from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/hello')
def hello():
    return "Hello from backend1"

@app.route('/ip')
def get_ip():
    # Return the client IP that was forwarded by the proxy
    client_ip = request.headers.get('X-Forwarded-For', 'unknown')
    return f"Client IP: {client_ip}"

@app.route('/headers')
def get_headers():
    # Return forwarded headers for verification
    headers = {
        'X-Forwarded-Host': request.headers.get('X-Forwarded-Host'),
        'X-Forwarded-Proto': request.headers.get('X-Forwarded-Proto'),
        'X-Forwarded-For': request.headers.get('X-Forwarded-For'),
        'Host': request.headers.get('Host'),
    }
    return jsonify(headers)

@app.route('/status')
def status():
    return jsonify({"status": "ok", "service": "backend1"})

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080, debug=True)