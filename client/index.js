onst WebSocket = require('ws');

const ws = new WebSocket('ws://localhost:8080/ws');

ws.on('open', function open() {
  console.log('Connected to server');
  ws.send('Hello from Node.js client');
});

ws.on('message', function message(data) {
  console.log('Received from server:', data.toString());
});
