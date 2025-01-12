const WebSocket = require('ws');
const readline = require('readline');

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout
});

let clientId = "";
let ws;
let ticketList = []; 

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

    promptForTicketAction();
  });

  ws.on('message', (data) => {
    const msg = JSON.parse(data);
    if (msg.message_type === 'ticket_list') {
      // Server schickt die aktuelle Ticketliste
      ticketList = msg.tickets || [];
      console.log('\n--- Neue Ticketliste vom Server ---');
      console.log(ticketList);
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

function promptForTicketAction() {
  console.log(`
----------------------------------------
Wählen Sie eine Aktion:
  [1] Ticket zuweisen
  [2] Ticket abgeben
  [3] Ticket löschen
  [4] Aktuelle Tickets anzeigen
  [q] Beenden
----------------------------------------
  `);

  rl.question('?', (choice) => {
    switch (choice.trim().toLowerCase()) {
      case '1':
        promptForTicketNumber('assign_ticket');
        break;

      case '2':
        promptForTicketNumber('abandon_ticket');
        break;

      case '3':
        promptForTicketNumber('delete_ticket');
        break;

      case '4':
        console.log('\n--- Aktuelle Tickets ---');
        console.log(ticketList);
        // Zurück zum Menü
        promptForTicketAction();
        break;

      case 'q':
      case 'quit':
        console.log('Beende Programm...');
        process.exit(0);

      default:
        console.log('Ungültige Eingabe!');
        promptForTicketAction();
        break;
    }
  });
}

// Fragt nach einer Ticketnummer und sendet die passende Nachricht ans WebSocket
function promptForTicketNumber(actionType) {
  rl.question('Bitte Ticketnummer eingeben: ', (ticketNum) => {
    const trimmed = ticketNum.trim();
    if (trimmed !== '') {
      const msg = {
        message_type: actionType,
        client_id: clientId,
        ticket_id: parseInt(trimmed, 10)
      };
      ws.send(JSON.stringify(msg));

      console.log(`Gesendet: ${JSON.stringify(msg)}`);
    } else {
      console.log("Keine Ticketnummer eingegeben. Aktion abgebrochen.");
    }
    promptForTicketAction();
  });
}