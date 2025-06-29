#!/usr/bin/env python3

from flask import Flask, request, jsonify
from flask_socketio import SocketIO, emit
import json

app = Flask(__name__)
app.config['SECRET_KEY'] = 'test-secret'
socketio = SocketIO(app, cors_allowed_origins="*")

@app.route('/api/status')
def status():
    return jsonify({"status": "ok", "service": "backend2"})

@app.route('/api/echo')
def echo():
    message = request.args.get('message', 'Hello')
    return jsonify({"echo": message})

@socketio.on('connect')
def handle_connect():
    print('Client connected')
    emit('connected', {'data': 'Connected to backend2'})

@socketio.on('message')
def handle_message(data):
    print(f'Received message: {data}')
    # Echo the message back
    emit('message', data)

@socketio.on('disconnect')
def handle_disconnect():
    print('Client disconnected')

if __name__ == '__main__':
    socketio.run(app, host='0.0.0.0', port=8080, debug=True)