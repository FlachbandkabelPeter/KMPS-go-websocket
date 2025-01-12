const WebSocket = require('ws');
const readline = require('readline');

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout
});

let clientId = "";
let ws;

// Schritt 1: Benutzer nach Client-ID fragen
rl.question('Bitte geben Sie eine Client-ID ein: ', (id) => {
  clientId = id;
  startWebSocket();
});

function startWebSocket() {
  ws = new WebSocket('ws://localhost:8080/ws');
  
  ws.on('open', () => {
    console.log('Verbunden mit Server.');

    // Senden einer "init"-Nachricht
    const initMsg = {
      message_type: 'init',
      client_id: clientId
    };
    ws.send(JSON.stringify(initMsg));

    // Nachdem wir verbunden sind, können wir z.B. Eingaben für Ticket-Zuweisungen erfassen
    promptForTicketAssignment();
  });

  ws.on('message', (data) => {
    const msg = JSON.parse(data);
    if (msg.message_type === 'ticket_list') {
      console.log('Aktuelle Tickets:', msg.tickets);
    } else {
      console.log('Unbekannte Servernachricht:', msg);
    }
  });

  ws.on('close', () => {
    console.log('Verbindung geschlossen.');
    process.exit(0);
  });

  ws.on('error', (err) => {
    console.error('WebSocket-Fehler:', err);
  });
}

function promptForTicketAssignment() {
  rl.question('Ticketnummer eingeben zum Zuweisen (oder Enter, um erneut anzuzeigen): ', (ticketNum) => {
    if (ticketNum.trim() !== '') {
      const assignMsg = {
        message_type: 'assign_ticket',
        client_id: clientId,
        ticket_id: parseInt(ticketNum, 10)
      };
      ws.send(JSON.stringify(assignMsg));
    }
    // Nochmal abfragen
    promptForTicketAssignment();
  });
}
