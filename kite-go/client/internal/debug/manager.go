package debug

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/community"
)

type userMachine struct {
	Machine string
	User    *community.User
}

// Manager provides debug information over HTTP
type Manager struct {
	mu      sync.Mutex
	machine string
	auth    component.AuthClient
}

// NewManager returns a new manager component to handle debug requests
func NewManager() *Manager {
	return &Manager{}
}

// Name implements component Core
func (m *Manager) Name() string {
	return "debug"
}

// Initialize implements component Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.auth = opts.AuthClient
	m.machine = opts.Platform.MachineID
}

// RegisterHandlers implements component Handler
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/debug/user-machine", m.handleUserMachine).Methods("GET")

	// Register default http.ServeMux where net/http/pprof registers its handlers.
	mux.PathPrefix("/debug/").Handler(http.DefaultServeMux)
}

// handleUserMachine returns the currently logged in user and
func (m *Manager) handleUserMachine(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	//we ignore the error to return a nil user in the response
	user, _ := m.auth.GetUser()

	buf, err := json.Marshal(userMachine{
		Machine: m.machine,
		User:    user,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("error encoding: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(buf)
}
