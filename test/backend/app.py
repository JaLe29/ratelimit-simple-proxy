#!/usr/bin/env python3

from flask import Flask, request, jsonify
from flask_socketio import SocketIO, emit
import json

app = Flask(__name__)
app.config['SECRET_KEY'] = 'test-secret'
socketio = SocketIO(app, cors_allowed_origins="*", async_mode="threading")

@app.route('/hello')
def hello():
    return "Hello from backend"

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
    return jsonify({"status": "ok", "service": "backend"})

@socketio.on('connect')
def handle_connect():
    print('Client connected')
    emit('connected', {'data': 'Connected to backend'})

@socketio.on('message')
def handle_message(data):
    print(f'Received message: {data}')
    # Echo the message back
    emit('message', data)

@socketio.on('disconnect')
def handle_disconnect():
    print('Client disconnected')

if __name__ == '__main__':
    socketio.run(app, host='0.0.0.0', port=8081, debug=True, allow_unsafe_werkzeug=True)