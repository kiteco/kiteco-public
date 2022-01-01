// Package liveupdates current implementation is a bit complicated and needs a refactor:
// -macOS does idle checking and the forced timeout check
// -windows uses the last event timestamp and checks visibility. It doesn't do the idle check.
// -linux uses the last event and has a mock implementation for visibility. It doesn't do the idle check.
package liveupdates

import (
	"context"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/updates"
	"github.com/kiteco/kiteco/kite-go/client/sidebar"
	"github.com/kiteco/kiteco/kite-go/client/sysidle"
	"github.com/kiteco/kiteco/kite-go/client/visibility"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

const idleUpdateTimeout = 10 * time.Minute

var forceUpdateThreshold = 1 * 24 * 60 * 60 // 1 day in seconds

// NewManager initializes the updater, and starts a goroutine that will
// apply updates when the app is idle.
func NewManager(bundlePath string) updates.Manager {
	ctx, cancel := context.WithCancel(context.Background())
	mgr := LiveManager{
		ready:     make(chan struct{}, 10),
		ctxCancel: cancel,
	}
	start(ctx, bundlePath, mgr.onUpdateReady, mgr.getLastEvent)
	visibility.Listen("updates", mgr.onVisibilityCheck)
	sysidle.Listen("updates", mgr.onIdleCheck)
	return &mgr
}

// LiveManager downloads and applies updates
type LiveManager struct {
	ready           chan struct{} // receives events when an update is ready
	visibleRecently bool
	ctxCancel       func()

	m         sync.Mutex
	lastEvent time.Time
}

// Name implements interface Core
func (m *LiveManager) Name() string {
	return "updater"
}

// Terminate implements interface Terminater
func (m *LiveManager) Terminate() {
	if m.ctxCancel != nil {
		m.ctxCancel()
	}
}

// RegisterHandlers implements interface Handlers
func (m *LiveManager) RegisterHandlers(mux *mux.Router) {
	// These endpoints allow the sidebar to interact with the updater
	mux.HandleFunc("/clientapi/update/check", m.handleCheckForUpdates).Methods("GET")
	mux.HandleFunc("/clientapi/update/restartAndUpdate", m.handleRestartAndUpdate).Methods("GET")
	mux.HandleFunc("/clientapi/update/readyToRestart", m.handleReadyToRestart).Methods("GET")
	mux.HandleFunc("/clientapi/update/restart", m.handleRestart).Methods("POST")
}

// ProcessedEvent implements core.ProcessedEventer
func (m *LiveManager) ProcessedEvent(evt *event.Event, editorEvt *component.EditorEvent) {
	m.setLastEvent(time.Now())
}

// ReadyChan returns the channel which delivers update messages
func (m *LiveManager) ReadyChan() chan struct{} {
	return m.ready
}

// onUpdateReady is called when Sparkle has finished downloading an update
func (m *LiveManager) onUpdateReady() {
	m.ready <- struct{}{}
}

// onVisibilityCheck is called each time the sidebar visibility is checked. If the
// sidebar has not been visible in the past 10-15 seconds then we automatically apply
// pending updates.
func (m *LiveManager) onVisibilityCheck(visibleNow, visibleRecently bool) {
	if runtime.GOOS == "darwin" {
		// macOS updates based on system idle status, not visibility
		return
	}
	m.visibleRecently = visibleRecently
	if !visibleNow && !visibleRecently && updateReady() {
		m.RestartAndUpdate()
	}
}

// onIdleCheck is called each time system idle is checked.
// If the user has been idle for the past 30 seconds then we automatically apply
// pending updates.
func (m *LiveManager) onIdleCheck(isIdle bool) {
	if updateReady() {
		if isIdle || secondsSinceUpdateReady() >= forceUpdateThreshold {
			m.RestartAndUpdate()
		}
	}
}

// CheckForUpdates is called when the user explicitly checks for updates via the menubar
func (m *LiveManager) CheckForUpdates(showModal bool) {
	if err := checkForUpdates(showModal); err != nil {
		log.Println(err)
		rollbar.Error(err)
	}
}

// UpdateReady returns true if an update has been downloaded and is waiting to
// be installed.
func (m *LiveManager) UpdateReady() bool {
	return updateReady()
}

// RestartAndUpdate installs the pending update
func (m *LiveManager) RestartAndUpdate() {
	if err := restartAndUpdate(); err != nil {
		log.Println(err)
	}
}

// Restart restarts the application
func (m *LiveManager) Restart() {
	if err := restart(); err != nil {
		log.Println(err)
	}
}

//--

func (m *LiveManager) getLastEvent() time.Time {
	m.m.Lock()
	defer m.m.Unlock()
	return m.lastEvent
}

func (m *LiveManager) setLastEvent(ts time.Time) {
	m.m.Lock()
	defer m.m.Unlock()
	m.lastEvent = ts
}

// handleReadyToRestart is the handler for the endpoint that is called by the
// updater on platforms where the updater is external to kited (i.e. windows and linux).
// If it returns 200 then the updater will restart and update kited.
// NOTE: This is used by the windows updater and the linux updater!
func (m *LiveManager) handleReadyToRestart(w http.ResponseWriter, r *http.Request) {
	// If there has been an event in the last 10 minutes, do not update yet
	if time.Since(m.getLastEvent()) < idleUpdateTimeout {
		w.WriteHeader(http.StatusConflict)
		return
	}

	// If the sidebar has been visible recently, do not update yet
	if m.visibleRecently {
		// Set RestartIfPreviouslyVisible so that if the update happens anyways, we
		// bring the sidebar back up
		sidebar.SetRestartIfPreviouslyVisible(true)

		w.WriteHeader(http.StatusConflict)
		return
	}

	// Set RestartIfPreviouslyVisible back to false because we are now updating because
	// the sidebar is not visible
	sidebar.SetRestartIfPreviouslyVisible(false)
	w.WriteHeader(http.StatusOK)
}

// handleCheckForUpdates is the handler for /clientapi/update/check. It can trigger UI
// elements to be displayed within the menubar application (but the endpoint never
// blocks on the UI). It is invoked from the sidebar when the user explicitly checks
// for updates.
func (m *LiveManager) handleCheckForUpdates(w http.ResponseWriter, r *http.Request) {
	m.CheckForUpdates(true)
}

// handleRestartAndUpdate is the handler for /clientapi/update/restartAndUpdate. It
// restarts the app, applying any waiting updates.
func (m *LiveManager) handleRestartAndUpdate(w http.ResponseWriter, r *http.Request) {
	if !updateReady() {
		http.Error(w, "there is no update to install", http.StatusBadRequest)
		return
	}
	m.RestartAndUpdate()
}

// handleRestart is the handler for /clientapi/update/restart. It
// restarts the app without applying updates.
func (m *LiveManager) handleRestart(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	// Give handler some time to return successfully before restarting
	go func() {
		time.Sleep(time.Second * 5)
		m.Restart()
	}()
}
