package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kiteserver"
)

const flagUpdateInterval = 1 * time.Hour

// Collection of Key values that are useful to other components
const (
	ServerKey                    = "server"
	StatusIconKey                = "show_status_icon"
	CompletionsDisabledKey       = "completions_disabled"
	MetricsDisabledKey           = "metrics_disabled"
	HasDoneOnboardingKey         = "has_done_onboarding"
	AutosearchEnabledKey         = "autosearch_default"
	AutoInstallPluginsEnabledKey = "auto_install_new_editor_plugins"
	AutoInstalledPluginIDsKey    = "plugins_autoinstalled"
	NotifyUninstalledPluginsKey  = "notify_uninstalled_plugins"
	SetupCompletedKey            = "setup_completed"
	HaveShownWelcome             = "have_shown_welcome"
	InstallTimeKey               = "install_time"
	TFThreadsKey                 = "tf_threads"
	AutostartDisabledKey         = "autostart_disabled"
	MaxFileSizeKey               = "max_file_size_kb"
	PredictiveNavMaxFilesKey     = "pred_nav_max_files"

	// one of "direct", "environment", "manual". See below for the constants
	proxyModeKey = "proxy_mode"
	// an URL with all our proxy settings
	// schemes "http" or "socks5" are supported
	// scheme, host, port are required
	// username and password are optional
	proxyURLKey = "proxy_url"

	ChooseEngineKey = "test_choose_engine"
	// SelectedEngineKey defines the code engine selected by the user during onboarding
	SelectedEngineKey = "selected_engine"
	EmailRequiredKey  = "email_required"
	CountryISOKey     = "country_iso"

	// NoProxySentinel indicates that we should use no proxy
	NoProxySentinel = "direct"
	// EnvironmentProxySentinel indicates that the system-wide proxy configuration will be used for outgoing HTTP connections
	EnvironmentProxySentinel = "environment"

	manualProxyModeSentinel = "manual"

	ProLaunchNotificationDismissed = "pro_launch_notification_dismissed"

	ShowCompletionsCTA       = "show_completions_cta"
	ShowCompletionsCTANotif  = "show_completions_cta_notif"
	CompletionsCTALastShown  = "completions_cta_last_shown"
	RCDisabledCompletionsCTA = "rc_disabled_completions_cta"
	RCDisabledLexicalPython  = "rc_disabled_lexical_python"

	ConversionCohort            = "conversion_cohort"
	TrialDuration               = "trial_duration"
	PaywallLastUpdated          = "paywall_last_updated"
	PaywallCompletionsLimit     = "paywall_completions_limit"
	PaywallCompletionsRemaining = "paywall_completions_remaining"
	ShowPaywallExhaustedNotif   = "show_paywall_exhausted_notif"

	AllFeaturesPro = "all_features_pro"

	KiteServer = "kite_enterprise_server"
)

// Manager provides an HTTP interface for managing settings.
type Manager struct {
	filepath            string
	settings            *sync.Map
	notificationTargets []component.SettingsNotifier
	cohort              component.CohortManager
}

// NewManager creates a new Manager for handle settings of the kited client
// exposes a simple REST api, see https://kite.quip.com/ZdxEAGXI75IC for details.
func NewManager(path string) *Manager {
	m := &Manager{
		filepath:            path,
		settings:            &sync.Map{},
		notificationTargets: []component.SettingsNotifier{},
	}

	err := m.load()
	if err != nil {
		log.Printf("error loading settings from %s: %v, using defaults", path, err)
	}

	// tracking existing users as of just before Kite Pro launch
	if err := m.save(); err != nil {
		log.Printf("error saving initial settings: %v\n", err)
		// setup default settings
		m.Set(ServerKey, *specs[ServerKey].defaultValue)
		m.Set(StatusIconKey, *specs[StatusIconKey].defaultValue)
		m.Set(CompletionsDisabledKey, *specs[CompletionsDisabledKey].defaultValue)
		m.Set(MetricsDisabledKey, *specs[MetricsDisabledKey].defaultValue)
		m.Set(HasDoneOnboardingKey, *specs[HasDoneOnboardingKey].defaultValue)
		m.Set(proxyModeKey, *specs[proxyModeKey].defaultValue)
		m.Set(InstallTimeKey, *specs[InstallTimeKey].defaultValue)
		m.Set(MaxFileSizeKey, *specs[MaxFileSizeKey].defaultValue)
		m.Set(ProLaunchNotificationDismissed, *specs[ProLaunchNotificationDismissed].defaultValue)
		m.Set(ShowCompletionsCTA, *specs[ShowCompletionsCTA].defaultValue)
		m.Set(CompletionsCTALastShown, *specs[CompletionsCTALastShown].defaultValue)
		m.Set(ShowCompletionsCTANotif, *specs[ShowCompletionsCTANotif].defaultValue)
		m.Set(KiteServer, *specs[KiteServer].defaultValue)
		m.Set(AutoInstallPluginsEnabledKey, *specs[AutoInstallPluginsEnabledKey].defaultValue)
		m.Set(NotifyUninstalledPluginsKey, *specs[NotifyUninstalledPluginsKey].defaultValue)
	}

	return m
}

// NewTestManager returns a Manager that does not persist values to disk
func NewTestManager() *Manager {
	return NewManager("")
}

// Name implements component.Core
func (m *Manager) Name() string {
	return "settings"
}

// Initialize implements component.Initialize
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.cohort = opts.Cohort
}

// RegisterHandlers implements component.Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/settings/"+KiteServer+"/status", m.handleKiteServerStatus).Methods("GET")
	mux.HandleFunc("/clientapi/settings/"+ConversionCohort, m.handleConversionCohort).Methods("GET")
	mux.HandleFunc("/clientapi/settings/{key}", m.handleGet).Methods("GET")
	mux.HandleFunc("/clientapi/settings/"+SetupCompletedKey, m.handleSetSetupCompleted).Methods("PUT", "POST")
	mux.HandleFunc("/clientapi/settings/{key}", m.handleSet).Methods("PUT", "POST")
	mux.HandleFunc("/clientapi/settings/{key}", m.handleDelete).Methods("DELETE")
}

// AddNotificationTarget adds a new settings notifier, it will be called after settings were changed or were removed
func (m *Manager) AddNotificationTarget(target component.SettingsNotifier) {
	m.notificationTargets = append(m.notificationTargets, target)
}

// AddNotificationTargetKey adds a new settings notifier, it will be called after settings were changed or were removed
func (m *Manager) AddNotificationTargetKey(key string, callback func(value string)) {
	m.notificationTargets = append(m.notificationTargets, &stringKeyListener{watchedKey: key, callback: callback})
}

// NotifyProxyValue sets a notifier on the proxy value string
func (m *Manager) NotifyProxyValue(callback func(value string)) {
	cb := func(value string) { callback(m.GetProxyValue()) }
	m.notificationTargets = append(m.notificationTargets, &stringKeyListener{watchedKey: proxyModeKey, callback: cb})
	m.notificationTargets = append(m.notificationTargets, &stringKeyListener{watchedKey: proxyURLKey, callback: cb})
}

// Get returns the value associated with the key, and true/false depending
// on whether the key was found.
func (m *Manager) Get(key string) (string, bool) {
	v, ok := m.settings.Load(key)
	if !ok {
		spec, ok := specs[key]
		if !ok {
			return "", false
		}
		if spec.defaultValue != nil {
			return *spec.defaultValue, true
		}

		return "", false
	}

	return v.(string), true
}

// GetMaxFileSizeBytes returns max_file_size_kb in bytes as an int
func (m *Manager) GetMaxFileSizeBytes() int {
	kb, _ := m.GetInt(MaxFileSizeKey)
	return kb << 10
}

// GetProxyValue gets the new format proxy settings
func (m *Manager) GetProxyValue() (retVal string) {
	mode, _ := m.Get(proxyModeKey)
	switch mode {
	case "":
		return EnvironmentProxySentinel
	case manualProxyModeSentinel:
		url, _ := m.Get(proxyURLKey)
		if url == "" {
			// if no URL is set yet, use the environment
			return EnvironmentProxySentinel
		}
		return url
	default:
		return mode
	}
}

// GetObj JSON decodes the value of key into obj.
func (m *Manager) GetObj(key string, obj interface{}) error {
	val, ok := m.Get(key)
	if !ok {
		return fmt.Errorf("no value for key \"%s\" found", key)
	}

	return json.Unmarshal([]byte(val), obj)
}

// GetTime returns the parsed time for the provided key.
func (m *Manager) GetTime(key string) (time.Time, error) {
	val, ok := m.Get(key)
	if !ok {
		return time.Time{}, errors.Errorf("no value for key \"%s\" found", key)
	}
	return time.Parse(time.RFC3339, val)
}

// GetDuration returns the parsed duration for the provided key.
func (m *Manager) GetDuration(key string) (time.Duration, error) {
	val, ok := m.Get(key)
	if !ok {
		return 0, errors.Errorf("no value for key \"%s\" found", key)
	}
	return time.ParseDuration(val)
}

// GetInt returns the value of the key converted to an int.
func (m *Manager) GetInt(key string) (int, error) {
	val, ok := m.Get(key)
	if !ok {
		return -1, errors.Errorf("no value for key \"%s\" found", key)
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		return -1, err
	}
	return intVal, nil
}

// GetBool returns the value associated with key, parsed as boolean and true/false depending
// on whether the key was found.
func (m *Manager) GetBool(key string) (bool, bool) {
	v, ok := m.Get(key)

	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, false
	}

	return b, ok
}

// BoolGetter returns an accessor to a boolean setting.
func (m *Manager) BoolGetter(key string) func() bool {
	return func() bool {
		v, ok := m.GetBool(key)
		return v && ok
	}
}

// Set will set the value for the provided key
// it returns an error and a value suitable as http response code
func (m *Manager) Set(key, value string) error {
	spec, ok := specs[key]
	if ok {
		if spec.validate != nil {
			if err := spec.validate(value); err != nil {
				return err
			}
		}
	}

	oldVal, _ := m.Get(key)

	m.settings.Store(key, value)
	if err := m.save(); err != nil {
		// revert
		m.settings.Store(key, oldVal)
		return err
	}

	if oldVal != value {
		for _, target := range m.notificationTargets {
			target.Updated(key, value)
		}
	}

	return nil
}

// SetObj json serializes the provided object and associates it with the key
func (m *Manager) SetObj(key string, obj interface{}) error {
	buf, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	return m.Set(key, string(buf))
}

// SetBool returns the value associated with key, parsed as boolean and true/false depending
// on whether the key was found.
func (m *Manager) SetBool(key string, value bool) error {
	return m.Set(key, strconv.FormatBool(value))
}

// Delete will remove the provided key from settings
// It returns an error and a value suitable as http status code
func (m *Manager) Delete(key string) error {
	spec, ok := specs[key]
	if ok && spec.preventDeletion {
		return fmt.Errorf("cannot delete key: %s", key)
	}

	m.settings.Delete(key)
	if err := m.save(); err != nil {
		return err
	}

	for _, target := range m.notificationTargets {
		target.Deleted(key)
	}

	return nil
}

// HandleGet gets the value of a single setting
func (m *Manager) handleGet(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if key == "" {
		http.Error(w, "no key specified", http.StatusBadRequest)
		return
	}

	v, ok := m.Get(key)
	if !ok {
		http.Error(w, fmt.Sprintf("key not found: %s", key), http.StatusNotFound)
		return
	}

	w.Write([]byte(v))
}

// handleSet sets the value of a single setting
func (m *Manager) handleSet(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if key == "" {
		http.Error(w, "no key specified", http.StatusBadRequest)
		return
	}

	val, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = m.Set(key, string(val))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleDelete handles deleting a setting.
func (m *Manager) handleDelete(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if key == "" {
		http.Error(w, "no key specified", http.StatusBadRequest)
		return
	}

	err := m.Delete(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleSetSetupCompleted is a custom handler that checks if it should also record an install time
func (m *Manager) handleSetSetupCompleted(w http.ResponseWriter, r *http.Request) {
	newv, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	new, err := strconv.ParseBool(string(newv))
	if err != nil {
		http.Error(w, "invalid value for setting", http.StatusInternalServerError)
		return
	}

	if err := m.cohort.SetSetupCompleted(new); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetDeploymentID returns the current server deployment ID if it exists
func (m *Manager) GetDeploymentID() string {
	hostPort, _ := m.Get(KiteServer)
	id, _, _ := kiteserver.GetHealth(hostPort)
	return id
}

func (m *Manager) handleKiteServerStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	out := map[string]interface{}{}
	defer func() {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(out)
	}()

	hostPort, _ := m.Get(KiteServer)
	_, ping, err := kiteserver.GetHealth(hostPort)
	out["available"] = err == nil
	out["ping"] = ping
}

// --

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

	settings := make(map[string]string)
	err = json.NewDecoder(f).Decode(&settings)

	for key, value := range settings {
		m.Set(key, value)
	}

	return nil
}

// save saves settings to disk
func (m *Manager) save() error {
	if m.filepath == "" {
		return nil
	}

	f, err := os.Create(m.filepath)
	if err != nil {
		return fmt.Errorf("error creating settings file at %s: %s", m.filepath, err.Error())
	}
	defer f.Close()

	settings := make(map[string]string)
	m.settings.Range(func(key, value interface{}) bool {
		settings[key.(string)] = value.(string)
		return true
	})

	if err := json.NewEncoder(f).Encode(settings); err != nil {
		return fmt.Errorf("error encoding settings: %v", err)
	}
	return nil
}
