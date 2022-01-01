package livemetrics

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/distatus/battery"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	tele "github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/cpuinfo"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics/completions"
	"github.com/kiteco/kiteco/kite-go/client/internal/performance"
	plugins "github.com/kiteco/kiteco/kite-go/client/internal/plugins_new"
	"github.com/kiteco/kiteco/kite-go/client/internal/proxy"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/client/visibility"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/kitestatus"
	"github.com/kiteco/kiteco/kite-go/lang"
	navmetrics "github.com/kiteco/kiteco/kite-go/navigation/metrics"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/kiteserver"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/tfserving"
	"github.com/kiteco/kiteco/kite-golib/macaddr"
	"github.com/kiteco/kiteco/kite-golib/mixpanel"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/kite-golib/telemetry"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// compile-time check that we implement the intended components
var (
	_ = component.UserAuth((*Manager)(nil))
)

const (
	statusMetricPeriod = time.Minute * 10 // period for sending batched status metrics
	inactivityInterval = 30 * time.Minute // period of inactivity before a "coding session" considered to start
)

// SidebarStatus is used to unmarshall the JSON that is sent from the sidebar to kited.
type SidebarStatus struct {
	Summations map[string]int `json:"summations"`
	MostRecent map[string]int `json:"mostRecent"`
}

// Manager is responsible for collating all status metrics
type Manager struct {
	settings component.SettingsManager

	visibility              *visibilityMetrics
	sidebarSumStatus        *sidebarSumMetrics
	sidebarMostRecentStatus *sidebarMostRecentMetrics
	langs                   *languageMetrics
	editors                 *editorMetrics
	versions                *versionMetrics
	cpu                     *cpuMetrics
	indel                   *indelMetricsByLang
	completions             *completions.MetricsByLang
	proselected             *metrics.SmartSelectedMetrics
	signatures              *metrics.SignaturesMetric
	index                   *metrics.IndexMetric
	watcher                 *metrics.WatcherMetric
	tfservingMetrics        *tfserving.Metrics

	kitedURL *url.URL
	proxy    component.AuthClient
	userIds  userids.IDs

	ggnnSubtokenEnabled bool

	telemetry          telemetry.Client
	mixpanelToken      string
	lastEditTimeByLang map[lang.Language]time.Time

	isNewInstall     bool
	showChooseEngine bool

	mu                      sync.Mutex
	machineAddress          string
	machineStartTime        time.Time
	loginTime               time.Time
	startTime               time.Time
	lastSampleTime          time.Time
	menubarVisible          bool
	sidebarUpdates          int
	region                  string
	clientVersion           string
	lastIdentifyID          string
	getDeploymentID         func() string
	windowsDomainMembership bool
	ctxCancel               func()

	countersLock sync.Mutex
	counters     map[string]int

	loggedin bool
	disabled bool

	golangRequests uint64
	jsRequests     uint64
	jsxRequests    uint64
	vueRequests    uint64
}

// NewManager creates a new metrics manager object
func NewManager(sigs *metrics.SignaturesMetric, completions *completions.MetricsByLang, watcher *metrics.WatcherMetric, proselected *metrics.SmartSelectedMetrics, tfservingMetrics *tfserving.Metrics) *Manager {
	now := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		sidebarUpdates:          0,
		sidebarSumStatus:        newSidebarSumMetrics(),
		sidebarMostRecentStatus: newSidebarMostRecentMetrics(),
		visibility:              newVisibilityMetrics(),
		langs:                   newLanguageMetrics(),
		editors:                 newEditorMetrics(),
		versions:                newVersionsMetrics(),
		cpu:                     newCPUMetrics(),
		indel:                   newIndelMetricsByLang(),
		proselected:             proselected,
		signatures:              sigs,
		completions:             completions,
		watcher:                 watcher,
		index:                   &metrics.IndexMetric{},
		tfservingMetrics:        tfservingMetrics,
		machineAddress:          getDefaultMACAddr(),
		startTime:               now,
		lastSampleTime:          now,
		counters:                make(map[string]int),
		ctxCancel:               cancel,
		lastEditTimeByLang:      make(map[lang.Language]time.Time),
	}
	visibility.Listen("metrics", m.visibility.Track)
	m.startStatusTracking(ctx)
	return m
}

// Name implements component Core
func (m *Manager) Name() string {
	return "metrics"
}

// Initialize implements component Initializer.
// The metrics manager needs to know about the current port of Kited and access to the backend server
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.kitedURL = opts.KitedURL
	m.clientVersion = opts.Platform.ClientVersion
	m.mixpanelToken = opts.Configuration.MixpanelToken
	m.telemetry = telemetry.NewCommonClient(telemetry.StreamKiteStatus)
	m.proxy = opts.AuthClient
	m.userIds = opts.UserIDs

	m.getDeploymentID = opts.Settings.GetDeploymentID

	m.ggnnSubtokenEnabled = opts.Platform.GGNNSubtokenEnabled

	m.isNewInstall = opts.Platform.IsNewInstall
	m.showChooseEngine, _ = opts.Settings.GetBool(settings.ChooseEngineKey)

	XXXXXXX

	// Metrics opt-out
	m.disabled, _ = opts.Settings.GetBool(settings.MetricsDisabledKey)
	if m.disabled {
		tele.Disable()
		rollbar.Disable()
	}

	m.settings = opts.Settings

	// communicate to other processes (e.g. KiteService) whether or not the user has disabled metrics
	m.saveDisabledStatus(m.disabled, opts.Platform.KiteRoot)
}

// Terminate implements component Terminater
func (m *Manager) Terminate() {
	m.cpu.close()
	if m.ctxCancel != nil {
		m.ctxCancel()
		m.ctxCancel = nil
	}
	m.completions.Flush()
	m.telemetry.Close()
}

// GitFound ...
func (m *Manager) GitFound() bool {
	return m.gitRepos.gitFound()
}

// RegisterHandlers implements interface to setup the HTTP endpoint handlers.
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	// This endpoint allows the sidebar to send status updates to incorporate into kite_status
	mux.HandleFunc("/clientapi/metrics/sidebar", m.handleSidebarStatus).Methods("POST")

	// This endpoint allows the editor plugins to increment generic counters
	mux.HandleFunc("/clientapi/metrics/counters", m.handleAdd).Methods("POST")

	// This endpoint returns the current metrics status
	mux.HandleFunc("/clientapi/metrics", m.handleStatus).Methods("GET")

	// This endpoint returns the current metrics ID
	mux.HandleFunc("/clientapi/metrics/id", m.handleID).Methods("GET")

	// This endpoint returns the install ID
	mux.HandleFunc("/clientapi/metrics/install_id", m.handleInstallID).Methods("GET")

	// This endpoint allows sending of any cio metrics.
	mux.HandleFunc("/clientapi/metrics/cio", m.handleCustomCIOEvent).Methods("POST")

	// This endpoint allows sending of any mixpanel metrics.
	mux.HandleFunc("/clientapi/metrics/mixpanel", m.handleCustomMixpanelEvent).Methods("POST")

	// This endpoint checks for unintsalled plugins for editors currently running
	mux.HandleFunc("clientapi/metrics/detect_uninstalled_plugins", m.handleDetectUninstalledPlugin).Methods("POST")
}

// LoggedIn implements component UserAuth to initialize the metrics when the user logged in
func (m *Manager) LoggedIn() {
	// Don't hold lock while Identify is being called
	m.Identify()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loggedin = true
}

// LoggedOut implements component UserAuth
func (m *Manager) LoggedOut() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loggedin = false
}

// ProcessedEvent implements component ProcessedEventer
// this exposes the old metrics manager as component aware of events. It can be removed as soon as the new
// manager is moved from new_kited
func (m *Manager) ProcessedEvent(event *event.Event, editorEvent *component.EditorEvent) {
	m.TrackEvent(event)
	m.trackEditorEvent(editorEvent)
	m.cpu.recordActive()

	l := lang.FromFilename(event.GetFilename())
	if event.GetAction() == "edit" {
		// this is so that we can track the number of completions used
		if sels := event.GetSelections(); len(sels) == 1 {
			if sel := sels[0]; sel.Start != nil && sel.End != nil && (sel.GetStart() == sel.GetEnd()) {
				// TODO(juan): need to use the text from the original editor event here incase some diffing happened
				// in the processor and we do not include the full text in the event.
				m.completions.Get(l).BufferEdited(editorEvent.Text, int(sel.GetEnd()))
			}
		}

		// Track coding speed via edit insertions/deletions
		m.indel.get(l).update(event.GetDiffs())

		if !m.disabled {
			// Send coding session start events to cio on edit.
			m.trackCodingSessionStart(event, editorEvent)
		}
	}
}

// EventResponse implements component EventResponser. It is currently used to
// track the number of events processed with/without an index loaded.
func (m *Manager) EventResponse(resp *response.Root) {
	m.index.EventHandled(resp.LocalIndexPresent)
}

// Completions returns the completion metrics
func (m *Manager) Completions() *completions.MetricsByLang {
	return m.completions
}

// TrackEvent tracks the given event with any registered listeners
func (m *Manager) TrackEvent(evt *event.Event) {
	// track coding and python coding
	m.visibility.TrackEvent(evt)

	// track which language the user is coding in
	m.langs.TrackEvent(evt)

	// track which editor the user is coding in
	m.editors.TrackEvent(evt)

	// track which git repo the user is coding in
	m.gitRepos.TrackEvent(evt)
}

// trackEditorEvent tracks the given editor event with any registered listeners
func (m *Manager) trackEditorEvent(evt *component.EditorEvent) {
	m.versions.TrackEvent(evt)
}

// HandleAdd adds to a user defined counter
func (m *Manager) handleAdd(w http.ResponseWriter, r *http.Request) {
	m.countersLock.Lock()
	defer m.countersLock.Unlock()

	var req struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading req body: %v", err), http.StatusInternalServerError)
		return
	}

	if err := json.Unmarshal(buf, &req); err != nil {
		http.Error(w, fmt.Sprintf("error unmarshalling request: %v", err), http.StatusBadRequest)
		return
	}

	m.counters[req.Name] += req.Value
}

// getDefaultMACAddr gets a network address to use as a machine identifier.
func getDefaultMACAddr() string {
	addr, err := macaddr.Primary()
	if err != nil {
		log.Println("no network interfaces available. Using <default> as mac address.")
		return "<default>"
	}
	return addr.String()
}

// Identify sets the userid/installid for metrics sent to mixpanel
func (m *Manager) Identify() {
	m.identify(nil)
}

// UpdateUser implements MetricsManager. It re-identifies the current user if additional
// traits are passed
func (m *Manager) UpdateUser(traits map[string]interface{}) {
	m.identify(traits)
}

// identify identifies the user at all servers which are used for metrics
// an already identified user is only re-identified if the user id changed
// or additional traits were passed
func (m *Manager) identify(additionalTraits map[string]interface{}) {
	// Check if we really need to re-identify
	metricsID := m.userIds.MetricsID()
	if m.getLastIdentifyID() == metricsID && len(additionalTraits) == 0 {
		return
	}

	log.Println("identifying with id:", metricsID)

	// Setup a temporary mixpanel client to identify
	mixpanelToken := m.mixpanelToken
	if !m.proxy.HasProductionTarget() {
		mixpanelToken = ""
	}
	mixpanel := mixpanel.NewMetrics(mixpanelToken)

	// Make sure to reset everything
	defer func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		// Remove data that was collected before registering
		m.visibility.lockedZero()
		m.sidebarSumStatus.zero()
		m.sidebarMostRecentStatus.zero()
		m.langs.zero()
		m.editors.zero()
		m.versions.zero()
		m.cpu.zero()
		m.gitRepos.zero()
		m.lastIdentifyID = metricsID
	}()

	// If metrics are disabled, we still want to identify on mixpanel and create a telemetry
	// client because we want to send an anonymized status event.
	if m.disabled {
		_ = mixpanel.Identify(metricsID, m.mergeProps(map[string]interface{}{
			"$username":        m.userIds.UserID(),
			"metrics_disabled": m.disabled,
		}, additionalTraits))
		tele.Update(m.mergeProps(map[string]interface{}{
			"email":            m.userIds.Email(),
			"metrics_disabled": m.disabled,
		}, additionalTraits))
		return
	}

	cpuData := cpuinfo.Get()
	cpuProps := map[string]interface{}{
		"cpu":          cpuData,
		"cpu_vendor":   cpuData.VendorID,
		"cpu_cores":    cpuData.PhysicalCores,
		"cpu_threads":  cpuData.LogicalCores,
		"cpu_features": cpuData.Features,
	}

	// the choose engine flag must only be set for new installs
	// it defaults to false for existing users, but is not send in that case
	var engineProps map[string]interface{}
	if m.isNewInstall {
		engineProps = map[string]interface{}{
			"test_onboarding_choose_engine_v2": m.showChooseEngine,
		}
	}

	teamServerUserHash := m.teamServerUserHash()

	emailRequired, _ := m.settings.Get(settings.EmailRequiredKey)
	countryISO, _ := m.settings.Get(settings.CountryISOKey)

	err := mixpanel.Identify(metricsID, m.mergeProps(map[string]interface{}{
		"$username":                 m.userIds.UserID(),
		"$email":                    m.userIds.Email(),
		"$last_login":               time.Now().String(),
		"machine":                   m.userIds.MachineID(),
		"install_id":                m.userIds.InstallID(),
		"client_version":            m.clientVersion,
		"os":                        runtime.GOOS,
		"kite_local":                true,
		"metrics_disabled":          m.disabled,
		"windows_domain_membership": m.windowsDomainMembership,
		"server_user_hash":          teamServerUserHash,
		"onboarding_email_required": emailRequired,
		"country_iso":               countryISO,
	}, additionalTraits, cpuProps, engineProps))
	if err != nil {
		log.Printf("error identifying user %s in mixpanel", m.userIds.String())
	}

	// Forward events to Customer.io
	// TODO(naman) this also updates Mixpanel, so duplicates the above
	tele.Update(m.mergeProps(map[string]interface{}{
		"user_id":                   m.userIds.UserID(),
		"email":                     m.userIds.Email(),
		"machine":                   m.userIds.MachineID(),
		"install_id":                m.userIds.InstallID(),
		"last_login":                time.Now().Unix(),
		"client_version":            m.clientVersion,
		"os":                        runtime.GOOS,
		"kite_local":                true,
		"metrics_disabled":          m.disabled,
		"windows_domain_membership": m.windowsDomainMembership,
		"server_user_hash":          teamServerUserHash,
		"onboarding_email_required": emailRequired,
		"country_iso":               countryISO,
	}, additionalTraits, cpuProps, engineProps))
}

// SetMenubarVisible sets whether or not the menubar is visible
func (m *Manager) SetMenubarVisible(v bool) {
	m.menubarVisible = v
}

// IsMenubarVisible returns the current visibility status of the icon in the menubar
func (m *Manager) IsMenubarVisible() bool {
	return m.menubarVisible
}

// HandleSidebarStatus handles POST request from the sidebar
func (m *Manager) handleSidebarStatus(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var data SidebarStatus
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	m.sidebarSumStatus.update(data.Summations)
	m.sidebarMostRecentStatus.update(data.MostRecent)
	m.sidebarUpdates++
}

// HandleStatus handles a GET request for the current status
func (m *Manager) handleStatus(w http.ResponseWriter, r *http.Request) {
	s, _ := m.statusAt(time.Now(), false)
	buf, err := json.Marshal(s)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshalling json: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

// handleID handles a GET request for the current metrics ID
func (m *Manager) handleID(w http.ResponseWriter, r *http.Request) {
	mid := m.userIds.MetricsID()
	fmid := m.userIds.ForgetfulMetricsID()
	buf, err := json.Marshal(
		struct {
			MetricsID          string `json:"metrics_id"`
			ForgetfulMetricsID string `json:"forgetful_metrics_id"`
		}{
			mid,
			fmid,
		},
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshalling json: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (m *Manager) handleInstallID(w http.ResponseWriter, r *http.Request) {
	id := m.userIds.InstallID()
	buf, err := json.Marshal(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshalling json: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

// handleCustomCIOEvent handles the POST request for /clientapi/metrics/cio
func (m *Manager) handleCustomCIOEvent(w http.ResponseWriter, r *http.Request) {
	ce, status, err := m.getCustomEvent(r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
	tele.Default.CIOOnly().Event(ce.Event, ce.Props)
}

// handleCustomMixpanelEvent handles the POST request for /clientapi/metrics/mixpanel
func (m *Manager) handleCustomMixpanelEvent(w http.ResponseWriter, r *http.Request) {
	ce, status, err := m.getCustomEvent(r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
	tele.Default.MPOnly().Event(ce.Event, ce.Props)
}

func (m *Manager) getCustomEvent(r *http.Request) (telemetry.CustomEvent, int, error) {
	var ce telemetry.CustomEvent

	if err := json.NewDecoder(r.Body).Decode(&ce); err != nil {
		return ce, http.StatusBadRequest, errors.Errorf("error unmarshalling request: %v", err)
	}
	if ce.Key != "XXXXXXX" {
		return ce, http.StatusBadRequest, errors.Errorf("incorrect key")
	}

	return ce, 0, nil
}

func (m *Manager) handleDetectUninstalledPlugin(w http.ResponseWriter, r *http.Request) {
	_, pluginsStatus := m.statusAt(time.Now(), false)
	notifyUninstalledPlugins, _ := m.settings.GetBool(settings.NotifyUninstalledPluginsKey)
	if notifyUninstalledPlugins {
		m.detectUninstalledPlugins(pluginsStatus)
	}
}

func (m *Manager) statusAt(now time.Time, clear bool) (map[string]interface{}, *plugins.PluginResponse) {
	expire, planEnd, plan, product := m.proxy.LicenseStatus()
	s := map[string]interface{}{
		"region":                m.region,
		"client_version":        m.clientVersion,
		"mac_address":           m.machineAddress,
		"os_version":            performance.OsVersion(),
		"memory_usage":          performance.MemoryUsage(),
		"uptime":                int64(now.Sub(m.lastSampleTime).Seconds()),
		"menubar_visible":       m.menubarVisible,
		"logged_in":             m.loggedin,
		"ggnn_subtoken_enabled": m.ggnnSubtokenEnabled,
		"license_expire":        expire.Unix(),
		"plan_end":              planEnd.Unix(),
		"plan":                  plan,
		"product":               product,
	}

	disabled, _ := m.settings.GetBool(settings.CompletionsDisabledKey)
	s["completions_disabled"] = disabled

	d := m.getDeploymentID()
	if d != "" {
		s["server_deployment_id"] = d
	}
	teamServerUserHash := m.teamServerUserHash()
	if teamServerUserHash != "" {
		s["server_user_hash"] = teamServerUserHash
	}

	// Add values from the kitestatus package
	ksValues := kitestatus.Get()
	for k, v := range ksValues {
		if _, ok := kitestatusAllowed[k]; ok {
			s[k] = v
		}
	}

	hasProxy, _ := proxy.Global.IsProxied("https://" + domains.Alpha)
	s["has_proxy"] = hasProxy

	pluginsStatus := m.editorStatusesResponse()
	if ei := m.editorStatuses(pluginsStatus); ei != nil {
		for k, v := range ei {
			s[k] = v
		}
	}

	// add memory metrics
	if memInfo, err := mem.VirtualMemory(); err == nil {
		s["memory_total"] = memInfo.Total
		s["memory_available"] = memInfo.Available
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	s["num_gc_cycles"] = ms.NumGC

	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) >= 1 {
		s["cpu_mhz"] = cpuInfo[0].Mhz

		var threads int32
		for _, c := range cpuInfo {
			threads += c.Cores
		}
		s["cpu_threads"] = threads
	}

	// discharging iff at least one battery is discharging and no batteries are charging
	batteries, _ := battery.GetAll()
	var discharging bool
	for _, bat := range batteries {
		if bat == nil {
			continue
		}
		if bat.State == battery.Charging {
			discharging = false
			break
		}
		if bat.State == battery.Discharging {
			discharging = true
		}
	}
	s["battery_discharging"] = discharging

	m.countersLock.Lock()
	for k, v := range m.counters {
		s[k] = v
	}
	m.countersLock.Unlock()

	var (
		sigs       metrics.SignaturesSnapshot
		index      metrics.IndexSnapshot
		watchCount int64
	)
	m.completions.ReadAndFlatten(clear, s)
	m.indel.readAndFlatten(clear, s)
	m.proselected.ReadAndFlatten(clear, s)
	watchCount = m.watcher.Read(clear)
	if !clear {
		sigs = m.signatures.Read()
		index = m.index.Read()

		for k, v := range m.langs.get() {
			s[k] = v
		}
		for k, v := range m.editors.get() {
			s[k] = v
		}
		for k, v := range m.versions.get() {
			s[k] = v
		}
		for k, v := range m.visibility.get() {
			s[k] = v
		}
		for k, v := range m.gitRepos.get() {
			s[k] = v
		}
		cpuMetrics := m.cpu.get()

		s["cpu_samples_list"] = cpuMetrics.samples
		s["active_cpu_samples_list"] = cpuMetrics.activeSamples
		s["temperatures"] = cpuMetrics.temperatures
		s["fan_speeds"] = cpuMetrics.fanSpeeds
		s["load_avg"] = cpuMetrics.loadAvg
	} else {
		sigs = m.signatures.ReadAndClear()
		index = m.index.ReadAndClear()

		for k, v := range m.langs.dump() {
			s[k] = v
		}
		for k, v := range m.editors.dump() {
			s[k] = v
		}
		for k, v := range m.versions.dump() {
			s[k] = v
		}
		for k, v := range m.visibility.dump() {
			s[k] = v
		}
		for k, v := range m.gitRepos.dump() {
			s[k] = v
		}

		cpuMetrics := m.cpu.dump()

		s["cpu_samples_list"] = cpuMetrics.samples
		s["active_cpu_samples_list"] = cpuMetrics.activeSamples
		s["temperatures"] = cpuMetrics.temperatures
		s["fan_speeds"] = cpuMetrics.fanSpeeds
		s["load_avg"] = cpuMetrics.loadAvg

		m.sidebarSumStatus.zero()
		m.sidebarMostRecentStatus.zero()
		m.sidebarUpdates = 0
		m.counters = make(map[string]int)
		m.lastSampleTime = now

		kitestatus.Reset()
	}

	s["signatures_triggered"] = sigs.Triggered
	s["signatures_shown"] = sigs.Shown

	s["events_with_index"] = index.EventsWithIndex
	s["events_without_index"] = index.EventsWithoutIndex

	s["watch_count"] = watchCount

	// TODO(naman, ed) generalize this via the "is file supported" endpoint
	s["golang_requests"] = atomic.LoadUint64(&m.golangRequests)
	if clear {
		atomic.StoreUint64(&m.golangRequests, 0)
	}

	// server timeout metrics
	tfmetrics := m.tfservingMetrics.Read(clear)
	s["tfserving_requests"] = tfmetrics.Requests
	s["tfserving_timeouts"] = tfmetrics.Timeouts
	s["tfserving_othererrs"] = tfmetrics.OtherErrs
	s["tfserving_success"] = tfmetrics.Success
	s["tfserving_queuefull"] = tfmetrics.QueueFull
	s["tfserving_connectionclosed"] = tfmetrics.ConnectionClosed
	s["tfserving_connectionrefused"] = tfmetrics.ConnectionRefused
	s["tfserving_otherconnerr"] = tfmetrics.OtherConnErr
	s["tfserving_otherunavailable"] = tfmetrics.OtherUnavailable

	// navigation metrics
	for k, v := range navmetrics.Read(clear) {
		s[k] = v
	}

	removeZerosFromMap(reflect.ValueOf(s))
	return s, pluginsStatus
}

func removeZerosFromMap(m reflect.Value) {
	for _, k := range m.MapKeys() {
		v := m.MapIndex(k)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if v.Int() == 0 {
				m.SetMapIndex(k, reflect.Value{})
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if v.Uint() == 0 {
				m.SetMapIndex(k, reflect.Value{})
			}
		case reflect.Map:
			removeZerosFromMap(v)
		}
	}
}

// PermissionsRequest increments the permissions request counter(s).
// Currently only track golang
func (m *Manager) PermissionsRequest(fn string) {
	switch lang.FromFilename(fn) {
	case lang.Golang:
		atomic.AddUint64(&m.golangRequests, 1)
	case lang.JavaScript:
		atomic.AddUint64(&m.jsRequests, 1)
	case lang.JSX:
		atomic.AddUint64(&m.jsxRequests, 1)
	case lang.Vue:
		atomic.AddUint64(&m.vueRequests, 1)
	}
}

// SetRegion sets the region we are connected to
func (m *Manager) SetRegion(region string) {
	m.region = region
}

// GetRegion returns the currently set region value, this is useful for test cases
func (m *Manager) GetRegion() string {
	return m.region
}

func (m *Manager) editorStatusesResponse() *plugins.PluginResponse {
	targetURL, err := m.kitedURL.Parse("/clientapi/plugins")
	if err != nil {
		return nil
	}

	resp, err := http.Get(targetURL.String())
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var status plugins.PluginResponse
	if err := json.Unmarshal(body, &status); err != nil {
		return nil
	}

	return &status
}

// editorStatuses returns information about each editor
func (m *Manager) editorStatuses(status *plugins.PluginResponse) map[string]bool {
	statuses := make(map[string]bool)

	if status == nil {
		return statuses
	}

	for _, ed := range status.Plugins {
		var editorInstalled, pluginInstalled bool
		if len(ed.Editors) != 0 {
			editorInstalled = true
			for _, e := range ed.Editors {
				if e.PluginInstalled {
					pluginInstalled = true
					break
				}
			}
		}
		statuses[ed.ID+"_installed"] = editorInstalled
		statuses[ed.ID+"_running"] = ed.Running
		statuses[ed.ID+"_plugin_installed"] = pluginInstalled
	}

	return statuses
}

// Send current status to mixpanel and zero out data
func (m *Manager) sendStatusMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.telemetry == nil {
		return
	}

	now := time.Now()
	status, pluginsStatus := m.statusAt(now, true)

	if !m.disabled {
		m.postTelemetryEvent(status)
		notifyUninstalledPlugins, _ := m.settings.GetBool(settings.NotifyUninstalledPluginsKey)
		if notifyUninstalledPlugins {
			m.detectUninstalledPlugins(pluginsStatus)
		}
	}

	// Check for any edit events in supported files
	bashEdit, _ := status["bash_edit"].(uint64)
	cEdit, _ := status["c_edit"].(uint64)
	cppEdit, _ := status["cpp_edit"].(uint64)
	csharpEdit, _ := status["csharp_edit"].(uint64)
	cssEdit, _ := status["css_edit"].(uint64)
	goEdit, _ := status["go_edit"].(uint64)
	htmlEdit, _ := status["html_edit"].(uint64)
	javaEdit, _ := status["java_edit"].(uint64)
	jsEdit, _ := status["javascript_edit"].(uint64)
	jsxEdit, _ := status["jsx_edit"].(uint64)
	kotlinEdit, _ := status["kotlin_edit"].(uint64)
	lessEdit, _ := status["less_edit"].(uint64)
	objcEdit, _ := status["objectivec_edit"].(uint64)
	perlEdit, _ := status["perl_edit"].(uint64)
	phpEdit, _ := status["php_edit"].(uint64)
	pythonEdit, _ := status["python_edit"].(uint64)
	rubyEdit, _ := status["ruby_edit"].(uint64)
	scalaEdit, _ := status["scala_edit"].(uint64)
	tsxEdit, _ := status["tsx_edit"].(uint64)
	tsEdit, _ := status["typescript_edit"].(uint64)
	vueEdit, _ := status["vue_edit"].(uint64)
	if bashEdit > 0 ||
		cEdit > 0 ||
		cppEdit > 0 ||
		csharpEdit > 0 ||
		cssEdit > 0 ||
		goEdit > 0 ||
		htmlEdit > 0 ||
		javaEdit > 0 ||
		jsEdit > 0 ||
		jsxEdit > 0 ||
		kotlinEdit > 0 ||
		lessEdit > 0 ||
		objcEdit > 0 ||
		perlEdit > 0 ||
		phpEdit > 0 ||
		pythonEdit > 0 ||
		rubyEdit > 0 ||
		scalaEdit > 0 ||
		tsxEdit > 0 ||
		tsEdit > 0 ||
		vueEdit > 0 {
		m.postAnonymousTelemetryEvent("anon_supported_file_edited")
	}
}

// startStatusTracking starts a goroutine to send status metric every statusMetricPeriod
func (m *Manager) startStatusTracking(ctx context.Context) {
	go func() {
		defer func() {
			if ex := recover(); ex != nil {
				rollbar.PanicRecovery(ex)
			}
		}()

		ticker := time.NewTicker(statusMetricPeriod)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				m.sendStatusMetrics()
			}
		}
	}()
}

// saveDisabledStatus touches a file in the kite root directory to indicate that the user disabled metrics, so that
// related processes (e.g. KiteService) know not to log metrics.
func (m *Manager) saveDisabledStatus(disabled bool, kiteRoot string) {
	file := filepath.Join(kiteRoot, "metrics-disabled")

	exists := true
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		exists = false
	} else if err != nil {
		log.Printf("could not stat %s: %v", file, err)
		return
	}

	if exists == disabled {
		// nothing to do
		return
	}

	if disabled {
		err = ioutil.WriteFile(file, nil, 0600)
		if err != nil {
			log.Printf("error writing %s: %v", file, err)
		}
		return
	}

	err = os.Remove(file)
	if err != nil {
		rollbar.Error(err)
	}
}

func (m *Manager) getLastIdentifyID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastIdentifyID
}

// concat returns a generic map which contains the data of all maps passed as parameter
func (m *Manager) mergeProps(into map[string]interface{}, from ...map[string]interface{}) map[string]interface{} {
	for _, d := range from {
		if d != nil {
			for k, v := range d {
				into[k] = v
			}
		}
	}
	return into
}

// postAnonymousTelemetryEvent posts a new event to t.kite.com which uses anonymized data
// and only a subset of the properties
func (m *Manager) postAnonymousTelemetryEvent(eventName string) {
	// generate anonymous id in the same way the old segment code did
	h := sha256.New()
	h.Write([]byte(m.userIds.MetricsID()))
	anonymousID := base64.StdEncoding.EncodeToString(h.Sum(nil))

	_ = m.telemetry.Track(context.Background(), m.userIds.MetricsID(), eventName, map[string]interface{}{
		"anonymous_id": anonymousID,
		"sent_at":      time.Now().Unix(),
		"source":       "kited",
		"os":           runtime.GOOS,
	})
}

// postTelemetryEvent posts a new event to t.kite.com using the metrics id and the full set of status properties
func (m *Manager) postTelemetryEvent(status map[string]interface{}) {
	// the old segment code was posting property source, so we add it, too
	// same for the other properties
	status["user_id"] = m.userIds.MetricsID()
	status["forgetful_metrics_id"] = m.userIds.ForgetfulMetricsID()
	status["sent_at"] = time.Now().Unix()
	status["source"] = "kited"
	status["os"] = runtime.GOOS
	status["client_version"] = m.clientVersion

	_ = m.telemetry.Track(context.Background(), m.userIds.MetricsID(), "kite_status", status)
}

// trackCodingSessionStart sends a tracking event when editing Python after a period of inactivity.
func (m *Manager) trackCodingSessionStart(evt *event.Event, editorEvt *component.EditorEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	lng := lang.FromFilename(evt.GetFilename())
	if now.Sub(m.lastEditTimeByLang[lng]) > inactivityInterval {
		expire, planEnd, plan, product := m.proxy.LicenseStatus()
		props := map[string]interface{}{
			"editor_used":          editorEvt.Source,
			"language":             lng.Name(),
			"install_id":           m.userIds.InstallID(),
			"license_expire":       expire.Unix(),
			"metrics_id":           m.userIds.MetricsID(),
			"plan_end":             planEnd.Unix(),
			"plan":                 plan,
			"product":              product,
			"trial_available":      m.proxy.TrialAvailable(),
			"time_since_last_edit": int64(now.Sub(m.lastEditTimeByLang[lng]).Truncate(time.Second).Seconds()),
		}
		kitectx.Go(func() error {
			clienttelemetry.Default.CIOOnly().Event("coding_session_started", props)
			return nil
		})
	}
	m.lastEditTimeByLang[lng] = now
}

// teamServerUserHash returns the hex-encoded hash of the configured enterprise server name.
// If no KTS server with a username@ part is configured, then an empty string is returned.
func (m *Manager) teamServerUserHash() string {
	if nameOrURL, _ := m.settings.Get(settings.KiteServer); nameOrURL != "" {
		parsedURL, err := kiteserver.ParseKiteServerURL(nameOrURL)
		if err == nil && parsedURL.User != nil {
			return fmt.Sprintf("%x", md5.Sum([]byte(parsedURL.User.Username())))
		}
	}
	return ""
}

func (m *Manager) detectUninstalledPlugins(status *plugins.PluginResponse) {
	if status == nil {
		return
	}

	for _, ed := range status.Plugins {
		if len(ed.Editors) == 0 || !ed.Running {
			continue
		}

		var pluginInstalled bool
		for _, e := range ed.Editors {
			if e.PluginInstalled {
				pluginInstalled = true
				break
			}
		}

		if !pluginInstalled {
			// Send an event to CIO
			tele.Event("uninstalled_plugin_detected", map[string]interface{}{
				"editor": ed.ID,
			})
			return
		}
	}
}
