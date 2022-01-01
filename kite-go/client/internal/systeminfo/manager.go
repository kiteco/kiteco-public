package systeminfo

import (
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
)

// Manager provides information about kited
type Manager struct {
	clientVersion string
}

// Initialize implements component Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.clientVersion = opts.Platform.ClientVersion
}

// Name implements component Core
func (m *Manager) Name() string {
	return "systeminfo"
}

// NewManager returns a new manager
func NewManager() *Manager {
	return &Manager{}
}

// RegisterHandlers implements component Handler
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/systeminfo", m.handleGetSystemInfo).Methods("GET")
	mux.HandleFunc("/clientapi/version", m.handleVersion).Methods("GET")
}
