package sidebar

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/sidebar"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

var _ component.Handlers = &Manager{}

// NewManager returns a new sidebar component
func NewManager() *Manager {
	return &Manager{}
}

// Manager is the component to handle sidebar requests
type Manager struct{}

// Name implements component Core
func (m *Manager) Name() string {
	return "sidebar"
}

// RegisterHandlers implements component Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	// This endpoint allows plugins to cause the sidebar to be opened to a specific flyout
	mux.HandleFunc("/clientapi/sidebar/open", m.handleOpenSidebar).Methods("GET")
	mux.HandleFunc("/clientapi/sidebar/focus", m.handleFocusSidebar).Methods("GET")
}

func (m *Manager) handleOpenSidebar(w http.ResponseWriter, r *http.Request) {
	if err := sidebar.Start(); err != nil {
		rollbar.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (m *Manager) handleFocusSidebar(w http.ResponseWriter, r *http.Request) {
	if err := sidebar.Focus(); err != nil {
		rollbar.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
