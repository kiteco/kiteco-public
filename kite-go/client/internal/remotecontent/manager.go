package remotecontent

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/conversion/remotecontent"
	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

var rcKiteTarget = "https://" + domains.RemoteConfig

var fetchRemoteContentRequestTimeout = 15 * time.Second

// Manager retrieves and serves remote content for use by the copilot
type Manager struct {
	m        sync.Mutex
	settings component.SettingsManager
	filepath string

	remoteContent  remotecontent.RemoteContent
	loadedFromFile bool
}

// NewManager creates a new copilot remote content manager
func NewManager(path string) *Manager {
	m := &Manager{
		filepath:       path,
		loadedFromFile: false,
	}

	err := m.load()
	if err != nil {
		log.Printf("error loading remotecontent from %s: %v, using remote only", path, err)
	}

	return m
}

// Name implements component.Core
func (m *Manager) Name() string {
	return "remotecontent"
}

// Initialize implements component.Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.settings = opts.Settings
}

// RegisterHandlers implements component.Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {

	// fetch updates the kited cache (m.remoteContent) and responds when that operation is complete
	mux.HandleFunc("/clientapi/remotecontent/fetch", m.handleFetchRemoteContent).Methods("POST")

	// get returns the contents of the kited cache (m.remoteContent)
	mux.HandleFunc("/clientapi/remotecontent/get", m.handleGetRemoteContent).Methods("GET")
}

func (m *Manager) handleFetchRemoteContent(w http.ResponseWriter, r *http.Request) {
	err := m.determineRemoteContent()
	if err != nil {
		errors.New("Unable to determine remote content", err.Error())
	}
}

func (m *Manager) handleGetRemoteContent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(m.remoteContent)
}

// RemoteContent implements component.RemoteContentManager
func (m *Manager) RemoteContent() remotecontent.RemoteContent {
	return m.remoteContent
}

// determineRemoteContent updates the content stored by the manager
func (m *Manager) determineRemoteContent() error {
	err := m.load()
	if err != nil {
		// Log the error, but try to load from remote
		err = errors.New("could not load remotecontent from disk", m.filepath, err.Error())
		rollbar.Error(err)
	}

	if m.loadedFromFile {
		// Contents in the file take priority over remote contents
		return nil
	}

	pl, err := m.fetchRemoteContent()
	if err != nil {
		return err
	}

	return m.setRemoteContentFromPayload(pl, false)
}

func (m *Manager) fetchRemoteContent() (json.RawMessage, error) {
	dest := fmt.Sprintf("%s/convcohort/remotecontent/", rcKiteTarget)
	ctx, cancel := context.WithTimeout(context.Background(), fetchRemoteContentRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", dest, nil)
	if err != nil {
		return json.RawMessage{}, err
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return json.RawMessage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return json.RawMessage{}, errors.Errorf("returned status %d", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return json.RawMessage{}, err
	}
	return b, nil
}

func (m *Manager) setRemoteContentFromPayload(payload json.RawMessage, fromFile bool) error {
	m.m.Lock()
	defer m.m.Unlock()

	var rc remotecontent.RemoteContent
	if err := json.Unmarshal(payload, &rc); err != nil {
		err = errors.New("could not unmarshal remote content payload", payload, err.Error())
		rollbar.Error(err)
		return err
	}

	m.remoteContent = rc
	m.loadedFromFile = fromFile
	return nil
}

// load loads settings from disk
func (m *Manager) load() error {
	if m.filepath == "" {
		return nil
	}

	f, err := os.Open(m.filepath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	defer f.Close()

	payload, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	return m.setRemoteContentFromPayload(payload, true)
}
