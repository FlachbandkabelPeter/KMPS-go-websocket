package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "sync/atomic"

    "github.com/gorilla/websocket"
)

type Ticket struct {
    ID         int    `json:"id"`
    AssignedTo string `json:"assigned_to,omitempty"` // leer, wenn nicht zugewiesen
}

type ClientMessage struct {
    MessageType string `json:"message_type"` // z.B. "init", "assign_ticket", ...
    ClientID    string `json:"client_id,omitempty"`
    TicketID    int    `json:"ticket_id,omitempty"`
}

type ServerMessage struct {
    MessageType string    `json:"message_type"` // "ticket_list"
    Tickets     []Ticket  `json:"tickets"`
}

// Action repräsentiert eine Aktion, die an den TicketManager gesendet wird.
type Action struct {
    ActionType string 
    TicketID   int
    ClientID   string
}

// TicketManager verwaltet Tickets und hält die Channels für Aktionen und Client-Verbindungen
type TicketManager struct {
    tickets       []Ticket
    nextTicketID  int64 // wird atomar hochgezählt
    actionChan    chan Action

    // Clients verwalten
    registerChan   chan *websocket.Conn
    unregisterChan chan *websocket.Conn
    clients        map[*websocket.Conn]bool
}

// Neuer TicketManager
func NewTicketManager() *TicketManager {
    return &TicketManager{
        tickets:       make([]Ticket, 0),
        nextTicketID:  0,
        actionChan:    make(chan Action),
        registerChan:   make(chan *websocket.Conn),
        unregisterChan: make(chan *websocket.Conn),
        clients:        make(map[*websocket.Conn]bool),
    }
}

// Hauptschleife des Managers
func (tm *TicketManager) Run() {
    for {
        select {
        case action := <-tm.actionChan:
            switch action.ActionType {

            case "create_ticket":
                newID := int(atomic.AddInt64(&tm.nextTicketID, 1))
                newTicket := Ticket{
                    ID:         newID,
                    AssignedTo: "",
                }
                tm.tickets = append(tm.tickets, newTicket)
                log.Printf("Neues Ticket erstellt: %d\n", newID)
                tm.broadcastTickets()

            case "assign_ticket":
                // Ticket-Zuweisung aktualisieren
                for i := range tm.tickets {
                    if tm.tickets[i].ID == action.TicketID {
                        tm.tickets[i].AssignedTo = action.ClientID
                        log.Printf("Ticket %d zugewiesen an Client '%s'\n", action.TicketID, action.ClientID)
                        tm.broadcastTickets()
                        break
                    }
                }
            case "abandon_ticket":
                // Zuordnung aufheben, wenn das Ticket von diesem Client gehalten wird
                for i := range tm.tickets {
                    if tm.tickets[i].ID == action.TicketID {
                        if tm.tickets[i].AssignedTo == action.ClientID {
                            tm.tickets[i].AssignedTo = ""
                            log.Printf("Ticket %d freigegeben von Client '%s'\n", action.TicketID, action.ClientID)
                            tm.broadcastTickets()
                        } else {
                            log.Printf("Ticket %d war nicht dem Client '%s' zugewiesen – keine Aktion.\n", 
                                action.TicketID, action.ClientID)
                        }
                        break
                    }
                }

            case "delete_ticket":
                // Ticket wirklich aus der Liste entfernen
                for i := range tm.tickets {
                    if tm.tickets[i].ID == action.TicketID {
                        // Ticket aus dem Slice entfernen
                        tm.tickets = append(tm.tickets[:i], tm.tickets[i+1:]...)
                        log.Printf("Ticket %d wurde gelöscht.\n", action.TicketID)
                        tm.broadcastTickets()
                        break
                    }
                }
            }
        

        case conn := <-tm.registerChan:
            // Neuen Client registrieren
            tm.clients[conn] = true
            // Schicke ihm direkt einmal den aktuellen Zustand
            tm.sendTickets(conn)

        case conn := <-tm.unregisterChan:
            // Client entfernen
            delete(tm.clients, conn)
            conn.Close()
        }
    }
}

// broadcastTickets schickt allen Clients die aktuelle Ticket-Liste
func (tm *TicketManager) broadcastTickets() {
    msg := ServerMessage{
        MessageType: "ticket_list",
        Tickets:     tm.tickets,
    }

    data, err := json.Marshal(msg)
    if err != nil {
        log.Println("Fehler beim Marshallen der Ticketliste:", err)
        return
    }

    for conn := range tm.clients {
        if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
            log.Println("Fehler beim Broadcast:", err)
        }
    }
}

// sendTickets schickt die aktuelle Ticketliste an einen einzelnen Client
func (tm *TicketManager) sendTickets(conn *websocket.Conn) {
    msg := ServerMessage{
        MessageType: "ticket_list",
        Tickets:     tm.tickets,
    }

    data, _ := json.Marshal(msg)
    conn.WriteMessage(websocket.TextMessage, data)
}

// Startet eine Goroutine, die die Tastatureingaben des Servers überwacht
func (tm *TicketManager) listenForConsole() {
    scanner := bufio.NewScanner(os.Stdin)
    go func() {
        for scanner.Scan() {
            text := scanner.Text()
            if text == "n" {
                // Neues Ticket anlegen
                tm.actionChan <- Action{
                    ActionType: "create_ticket",
                }
            }
        }
    }()
}

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocket-Handler
func wsHandler(tm *TicketManager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            log.Println("Fehler beim Upgrade auf WebSocket:", err)
            return
        }

        // Client registrieren
        tm.registerChan <- conn

        // Routine für Nachrichten vom Client empfangen
        go func() {
            defer func() {
                // Bei Verlassen -> abmelden
                tm.unregisterChan <- conn
            }()
            for {
                _, msg, err := conn.ReadMessage()
                if err != nil {
                    log.Println("Fehler beim Lesen von Client-Message:", err)
                    return
                }

                // JSON parsen
                var clientMsg ClientMessage
                if err := json.Unmarshal(msg, &clientMsg); err != nil {
                    log.Println("Fehler beim Unmarshallen:", err)
                    continue
                }

                switch clientMsg.MessageType {
                case "init":
                case "assign_ticket":
                    tm.actionChan <- Action{
                        ActionType: "assign_ticket",
                        TicketID:   clientMsg.TicketID,
                        ClientID:   clientMsg.ClientID,
                    }
                case "abandon_ticket":
                    tm.actionChan <- Action{
                        ActionType: "abandon_ticket",
                        TicketID:   clientMsg.TicketID,
                        ClientID:   clientMsg.ClientID,
                    }
        
                case "delete_ticket":
                    tm.actionChan <- Action{
                        ActionType: "delete_ticket",
                        TicketID:   clientMsg.TicketID,
                    }

                default:
                    log.Println("Unbekannter message_type:", clientMsg.MessageType)
                }
            }
        }()
    }
}

func main() {
    // TicketManager starten
    tm := NewTicketManager()
    go tm.Run()

    // Server liest Tastatureingaben (Taste "n" => neues Ticket)
    tm.listenForConsole()

    // HTTP / WebSocket-Endpoints
    http.HandleFunc("/ws", wsHandler(tm))



    serverAddr := ":8080"
    fmt.Printf("Server startet auf %s ...\n", serverAddr)
    if err := http.ListenAndServe(serverAddr, nil); err != nil {
        log.Fatal("ListenAndServe Fehler:", err)
    }
}
