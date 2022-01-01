package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"golang.org/x/net/websocket"
)

// Manager manages websocket connections
type Manager struct {
	mutex             sync.Mutex
	active            bool
	activeConns       map[*websocket.Conn]bool
	addConnectionChan chan string
	broadcastChan     chan string
	ctxCancel         func()
}

// NewManager returns a new Manager.
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		active:            true,
		activeConns:       make(map[*websocket.Conn]bool),
		addConnectionChan: make(chan string),
		broadcastChan:     make(chan string),
		ctxCancel:         cancel,
	}

	// launch broadcasting go routine and return when it's ready
	onReady := make(chan struct{})
	go m.broadcaster(ctx, onReady)
	<-onReady

	return m
}

// HandleEventsWS is the websocket endpoint used by UI's to subscribe to
// responses and events from the backend.
func (m *Manager) HandleEventsWS(conn *websocket.Conn) {
	defer conn.Close()

	if !m.IsActive() {
		log.Println("ignoring connection")
		return
	}

	// Add this connection to the map of connections that receive responses from the backend.
	m.addConnection(conn)
	defer m.removeConnection(conn)

	// Listen for events from UI component
	for {
		var uiEvent event.Event
		err := websocket.JSON.Receive(conn, &uiEvent)
		if err != nil {
			log.Printf("error reading event on `%s`: %v\n", getSocketPath(conn), err)
			return
		}
	}
}

// AcceptConnections sets whether the Manager will accept new websocket connections
func (m *Manager) AcceptConnections(val bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.active = val
}

// CloseConnections will close all active connections
func (m *Manager) CloseConnections() {
	m.ctxCancel()

	m.mutex.Lock()
	defer m.mutex.Unlock()
	for conn := range m.activeConns {
		err := conn.Close()
		if err != nil {
			log.Printf("error closing connection `%s`: %v\n", getSocketPath(conn), err)
		}
	}
	m.activeConns = make(map[*websocket.Conn]bool)
}

// IsActive returns if the Manager is active.
func (m *Manager) IsActive() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.active
}

// BroadcastJSON serializes the provided data and sends it to all websocket connections.
func (m *Manager) BroadcastJSON(data interface{}) {
	buf, err := json.Marshal(data)
	if err != nil {
		log.Println("cannot marshal broadcast data:", err)
	}
	m.Broadcast(string(buf))
}

// Broadcast sends the provided string to all websocket connections.
func (m *Manager) Broadcast(data string) {
	// Note that broadcastChan is intentionally not a buffered channel. This is so that
	// any blocking when writing responses will immediately cause subsequent events to drop.
	// This means that when the writing is unblocked, the most recent event will be sent (instead
	// of N old, buffered events)
	select {
	case m.broadcastChan <- data:
	default:
		log.Println("dropping response")
	}
}

// ActiveConnections returns all active connections.
func (m *Manager) ActiveConnections() []*websocket.Conn {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	var conns []*websocket.Conn
	for conn := range m.activeConns {
		conns = append(conns, conn)
	}
	return conns
}

// ConnectionAdded returns a channel that communicates if a connection is added to the manager
func (m *Manager) ConnectionAdded() chan string {
	return m.addConnectionChan
}

// --

func (m *Manager) addConnection(conn *websocket.Conn) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.activeConns[conn] = true
	log.Printf("got connection `%s` (total=%d)\n", getSocketPath(conn), len(m.activeConns))
	select {
	case m.addConnectionChan <- getSocketPath(conn):
	default:
		// no-op
	}
}

func (m *Manager) removeConnection(conn *websocket.Conn) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.activeConns, conn)
	log.Printf("lost connection `%s` (total=%d)\n", getSocketPath(conn), len(m.activeConns))
}

func (m *Manager) broadcaster(ctx context.Context, onReady chan<- struct{}) {
	defer func() {
		if ex := recover(); ex != nil {
			rollbar.PanicRecovery(ex)
		}
	}()

	// tell caller that this go routine is up and running
	close(onReady)

	for {
		select {
		case <-ctx.Done():
			return
		case data := <-m.broadcastChan:
			activeConns := m.ActiveConnections()
			for _, conn := range activeConns {
				err := websocket.Message.Send(conn, data)
				if err != nil {
					log.Printf("error sending message to connection `%s`: %v\n", getSocketPath(conn), err)
				}
			}
		}
	}
}

func getSocketPath(conn *websocket.Conn) string {
	if conn == nil {
		return ""
	}

	if req := conn.Request(); req != nil && req.URL != nil {
		return req.URL.Path
	}
	return ""
}
