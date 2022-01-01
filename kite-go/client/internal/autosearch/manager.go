package autosearch

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
	return "autosearch"
}

// Terminate implements component Terminater. It closes all open WS connections.
func (c *Manager) Terminate() {
	c.ws.CloseConnections()
}

// RegisterHandlers implements component Handlers. It adds a handler for the Websocket connection.
func (c *Manager) RegisterHandlers(mux *mux.Router) {
	mux.Handle("/autosearch", websocket.Handler(c.ws.HandleEventsWS))
}

type autosearchID struct {
	ID string `json:"id"`
}

// EventResponse implements component.EventResponse
func (c *Manager) EventResponse(r *response.Root) {
	// TODO(tarak): This is a bit of hack to send locator id's to the new sidebar's autosearch
	// functionality. It extracts the "locator" field from the response.PythonRoot response.
	// This is only temporary until we fully depricate the unified response
	var locator string
	for _, result := range r.Results {
		switch t := result.(type) {
		case map[string]interface{}:
			if loc, ok := t["autosearch_id"]; ok {
				locator, ok = loc.(string)
			}
		case *response.Autosearch:
			locator = t.AutosearchID
		}
	}
	if locator != "" {
		c.ws.BroadcastJSON(autosearchID{
			ID: locator,
		})
	}
}

// ActiveConnectionCount returns the number of currently active connections
func (c *Manager) ActiveConnectionCount() int {
	active := c.ws.ActiveConnections()
	return len(active)
}
