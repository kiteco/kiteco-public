package activefile

import (
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/internal/ws"
	"github.com/kiteco/kiteco/kite-go/response"
	"golang.org/x/net/websocket"
)

// Manager manages websocket connections
type Manager struct {
	ws *ws.Manager
}

// NewManager returns a new Manager.
func NewManager() *Manager {
	return &Manager{
		ws: ws.NewManager(),
	}
}

// Name implements component Core
func (c *Manager) Name() string {
	return "activefile"
}

// Terminate implements component Terminater. It closes all open WS connections.
func (c *Manager) Terminate() {
	c.ws.CloseConnections()
}

// RegisterHandlers implements component Handlers. It adds a handler for the Websocket connection.
func (c *Manager) RegisterHandlers(mux *mux.Router) {
	mux.Handle("/active-file", websocket.Handler(c.ws.HandleEventsWS))
}

type activeFile struct {
	Filename string `json:"filename"`
}

// EventResponse implements component.EventResponse
func (c *Manager) EventResponse(r *response.Root) {
	c.ws.BroadcastJSON(activeFile{
		Filename: r.Filename,
	})
}
