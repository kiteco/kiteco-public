package updates

import (
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

// NewMockManager returns a mocked manager implementation. It does not register with the visibility package
func NewMockManager() Manager {
	return &MockManager{
		readyChan: make(chan struct{}, 10),
	}
}

// MockManager implements Manager, it always returns that no updates are available
type MockManager struct {
	mu           sync.Mutex
	readyChan    chan (struct{})
	ready        bool
	checkedCount int
}

// Name implements interface Core
func (m *MockManager) Name() string {
	return "mock updates"
}

// RegisterHandlers implements interface Handlers
func (m *MockManager) RegisterHandlers(mux *mux.Router) {
	// These endpoints allow the sidebar to interact with the updater
	mux.HandleFunc("/clientapi/update/check", m.handleCheckForUpdates).Methods("GET")
	mux.HandleFunc("/clientapi/update/restartAndUpdate", m.handleRestartAndUpdate).Methods("GET")
	mux.HandleFunc("/clientapi/update/readyToRestart", m.handleReadyToRestart).Methods("GET")
}

// UpdateReady always returns that no update is available
func (m *MockManager) UpdateReady() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ready
}

// CheckForUpdates is a dummy implementation to implement interface Manaber
func (m *MockManager) CheckForUpdates(showModal bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkedCount++
}

// ReadyChan returns the update channel, no message will be written to it
func (m *MockManager) ReadyChan() chan struct{} {
	return m.readyChan
}

// additional functions for our mock manager

// GetCheckedCount returns how many times CheckForUpdates was called
func (m *MockManager) GetCheckedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.checkedCount
}

// SetUpdateReady overrides the value which will be returned by calls to UpdateReady()
func (m *MockManager) SetUpdateReady(ready bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ready = ready
}

// handleReadyToRestart is the handler for the endpoint that is called by the
// updater on platforms where the updater is external to kited
func (m *MockManager) handleReadyToRestart(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handleCheckForUpdates is the handler for /clientapi/update/check.
func (m *MockManager) handleCheckForUpdates(w http.ResponseWriter, r *http.Request) {
	m.CheckForUpdates(true)
	w.WriteHeader(http.StatusOK)
}

// handleRestartAndUpdate is the handler for /clientapi/update/restartAndUpdate.
func (m *MockManager) handleRestartAndUpdate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
