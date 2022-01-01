package network

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

var (
	networkCheckInterval      = 60 * time.Second //should be less than the connectionTimeout in authClient
	networkCheckErrorInterval = 5 * time.Second  //should be less than the connectionTimeout in authClient
	checkOnlineTimeout        = 2 * time.Second
)

// NewManager returns a new network manager component
func NewManager(components *component.Manager) *Manager {
	return &Manager{
		components:   components,
		online:       1,
		forceOffline: false,
		client:       &http.Client{},
	}
}

// Manager is a component to monitor the network network of kited
type Manager struct {
	client       *http.Client
	components   *component.Manager
	online       int64
	kitedOnline  int64
	forceOffline bool

	ctx    context.Context
	cancel context.CancelFunc
}

// Name implements interface Core
func (m *Manager) Name() string {
	return "network"
}

// Initialize implements the interface Initializers
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	go m.checkLoop(m.ctx)
}

// Terminate impelements component.Terminater
func (m *Manager) Terminate() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// RegisterHandlers implements interface Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/online", m.returnTrue)
	mux.HandleFunc("/clientapi/checkonline", m.returnTrue)
	mux.HandleFunc("/clientapi/kited_online", m.returnTrue)
}

func (m *Manager) doOnlineCheck(ctx context.Context) time.Duration {
	if m.forceOffline {
		atomic.StoreInt64(&m.online, 0)
		m.components.NetworkOffline()
		return networkCheckInterval
	}

	defer func() {
		if r := recover(); r != nil {
			rollbar.PanicRecovery(r)
		}
	}()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://clients3.google.com/generate_204", nil)
	resp, err := m.client.Do(req)
	if err != nil {
		if m.Online() {
			atomic.StoreInt64(&m.online, 0)
			m.components.NetworkOffline()
		}
		return networkCheckErrorInterval
	}

	io.Copy(ioutil.Discard, resp.Body)
	defer resp.Body.Close()

	if !m.Online() {
		atomic.StoreInt64(&m.online, 1)
		m.components.NetworkOnline()
	}

	return networkCheckInterval
}

func (m *Manager) checkLoop(ctx context.Context) {
	timer := time.NewTimer(networkCheckInterval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			func() {
				checkCtx, cancel := context.WithTimeout(ctx, checkOnlineTimeout)
				defer cancel()
				nextTick := m.doOnlineCheck(checkCtx)
				timer.Reset(nextTick)
			}()
		}
	}
}

// KitedInitialized implements interface KitedEventer
// it sets the value of kitedOnline to true when called
func (m *Manager) KitedInitialized() {
	atomic.StoreInt64(&m.kitedOnline, 1)
}

// KitedUninitialized implements interface KitedEventer
// it sets the value of kitedOnline to false when called
func (m *Manager) KitedUninitialized() {
	atomic.StoreInt64(&m.kitedOnline, 0)
}

// Online implements interface component.NetworkManager
func (m *Manager) Online() bool {
	return atomic.LoadInt64(&m.online) == int64(1)
}

// KitedOnline implements interface component.KitedOnline
func (m *Manager) KitedOnline() bool {
	return atomic.LoadInt64(&m.kitedOnline) == int64(1)
}

// CheckOnline implements interface component.NetworkManager
func (m *Manager) CheckOnline(ctx context.Context) bool {
	m.doOnlineCheck(ctx)
	return m.Online()
}

// SetOffline implements interface component.NetworkManager
func (m *Manager) SetOffline(offline bool) {
	atomic.StoreInt64(&m.online, 0)
	m.forceOffline = offline
}

type checkOnlineResponse struct {
	Online bool `json:"online"`
}

func (m *Manager) sendStatus(online bool, w http.ResponseWriter, r *http.Request) {
	buf, err := json.Marshal(checkOnlineResponse{
		Online: online,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (m *Manager) returnTrue(w http.ResponseWriter, r *http.Request) {
	m.sendStatus(true, w, r)
}
