package health

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// NewManager returns a new health manager component
func NewManager() *Manager {
	return &Manager{}
}

// Manager is a component to monitor the health of kited
type Manager struct {
}

// Name implements interface Core
func (m *Manager) Name() string {
	return "health"
}

// RegisterHandlers implements interface Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	// This endpoint checks whether the cocoa event loop is responsive
	mux.HandleFunc("/clientapi/health", m.handleHealth)

	//this endpoints is used by plugin to test the connection availability
	mux.HandleFunc("/clientapi/ping", m.handlePing)
}

// HandleHealth handles the /clientapi/health endpoint and responds with a boolean
func (m *Manager) handleHealth(w http.ResponseWriter, r *http.Request) {
	if IsResponsive() {
		fmt.Fprintln(w, "Application is healthy.")
	} else {
		http.Error(w, "The Cocoa UI thread is unresponsive.", http.StatusServiceUnavailable)
	}
}

func (m *Manager) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}
