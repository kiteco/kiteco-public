package notifications

import (
	"fmt"
	"hash/maphash"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/client/sidebar"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/conversion"
	cohorts "github.com/kiteco/kiteco/kite-golib/conversion"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

var _ component.Handlers = &Manager{}
var _ component.Initializer = &Manager{}
var _ component.ProcessedEventer = &Manager{}
var _ component.NotificationsManager = &Manager{}

// NewManager returns a new sidebar component
func NewManager(dismissed bool) *Manager {
	m := &Manager{}
	m.showProNotif.Store(!dismissed)
	return m
}

// Manager is the component to handle sidebar requests
type Manager struct {
	showProNotif atomic.Value // bool

	payloads sync.Map
	notifdir atomic.Value // string

	uid     func() string
	license interface {
		licensing.TrialAvailableGetter
		licensing.ProductGetter
	}
	settings component.SettingsManager
	cohort   component.CohortManager

	notifyFn func(string) error
}

// Name implements component Core
func (m *Manager) Name() string {
	return "notifications"
}

// Initialize implements component.Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.license = opts.License
	m.settings = opts.Settings
	m.cohort = opts.Cohort
	m.uid = opts.UserIDs.ForgetfulMetricsID
	fp := filepath.Join(opts.Platform.KiteRoot, "notifications")
	err := os.MkdirAll(fp, os.ModePerm)
	if err != nil {
		log.Println("failed to make notifications directory", err)
	} else {
		m.notifdir.Store(fp)
	}
}

// RegisterHandlers implements component Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/notifications", m.listNotifs).Methods("GET")
	mux.HandleFunc("/clientapi/notifications/", m.listNotifs).Methods("GET")
	mux.HandleFunc("/clientapi/notifications/{id}", m.serveNotif).Methods("GET")
	mux.HandleFunc("/clientapi/notifications/{id}/notify", m.showNotif).Methods("GET")
}

func (m *Manager) listNotifs(w http.ResponseWriter, r *http.Request) {
	writeFname := func(fname string) {
		ext := ".html"
		if !strings.HasSuffix(fname, ext) {
			return
		}
		w.Write([]byte(fname[:len(fname)-len(ext)]))
		w.Write([]byte("\n"))
	}

	fs, _ := ioutil.ReadDir(m.notifdir.Load().(string))
	for _, f := range fs {
		writeFname(f.Name())
	}

	statics, _ := AssetDir("static")
	for _, name := range statics {
		writeFname(name)
	}

	m.payloads.Range(func(key, _ interface{}) bool {
		w.Write([]byte(key.(string)))
		w.Write([]byte("\n"))
		return true
	})
}

func (m *Manager) serveNotif(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	_, err := w.Write(m.getNotifByID(id))
	if err != nil {
		rollbar.Error(errors.New("error serving notification"), err.Error())
	}
}

func (m *Manager) getNotifByID(id string) []byte {
	fp := filepath.Join(m.notifdir.Load().(string), fmt.Sprintf("%s.html", id))
	if b, err := ioutil.ReadFile(fp); err == nil {
		return b
	}

	// use statically cohorted IDs when checking statically bundled payloads
	if b, err := Asset(fmt.Sprintf("static/%s.html", m.staticID(id))); err == nil {
		return b
	}

	payload, ok := m.payloads.Load(id)
	if ok {
		return payload.([]byte)
	}

	rollbar.Error(errors.New("no payload for requested notification"), id)
	return MustAsset("static/not_found.html")
}

func (m *Manager) showNotif(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := m.notify(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *Manager) showPayload(payload string) error {
	var h maphash.Hash
	_, err := h.WriteString(payload)
	if err != nil {
		return errors.Errorf("failed to write payload to maphash.Hash", err)
	}

	// technically a memory leak, but in practice these payloads are tiny,
	// so it won't be an issue in practice.
	name := fmt.Sprintf("%d.html", h.Sum64())
	m.payloads.Store(name, *(*[]byte)(unsafe.Pointer(&payload)))
	return m.notify(name)
}

// ProcessedEvent implements component.ProcessedEventer
func (m *Manager) ProcessedEvent(event *event.Event, editorEvent *component.EditorEvent) {
	if !m.showProNotif.Load().(bool) {
		return
	}

	// ConversionCohort is not yet determined. Unclear if pro_launch notification is correct to show.
	if setupCompleted, _ := m.settings.GetBool(settings.SetupCompletedKey); !setupCompleted {
		return
	}

	// This method handles showing the Pro notification, which only applies to opt-in trials.
	if cohort := m.cohort.ConversionCohort(); cohort != cohorts.OptIn {
		return
	}

	if !m.license.TrialAvailable() || m.license.GetProduct() == licensing.Pro {
		return
	}

	action := event.GetAction()
	if action == "skip" {
		action = event.GetInitialAction()
	}

	switch action {
	case "edit", "select":
	default:
		return
	}

	l := lang.FromFilename(event.GetFilename())
	if l != lang.Python {
		return
	}

	// don't show again until Kite is restarted
	m.showProNotif.Store(false)

	m.ShowNotificationByID(conversion.ProLaunch)
	clienttelemetry.Event("cta_shown", map[string]interface{}{
		// see full list of sources in sidebar/src/store/license.tsx
		"cta_source": "desktop_notif",
	})
}

func (m *Manager) notify(id string) error {
	if m.notifyFn != nil {
		return m.notifyFn(id)
	}
	return sidebar.Notify(id)
}
