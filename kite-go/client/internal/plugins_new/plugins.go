//go:generate go-bindata -o bindata.go -pkg plugins onboarding/...

package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/atom"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/jetbrains"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/jupyterlab"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/neovim"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/spyder"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/sublime"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/vim"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/vscode"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/kitestatus"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
)

const settingsCheckInterval = 60 * time.Minute

var spyderSuboptimalSettingsStatus = kitestatus.GetBooleanDefault("spyder_suboptimal_settings", false)

const jupyterExt = "ipynb"

// isVscodeEnabled returns false on WSL, and true in all other cases.
// Related: https://github.com/Microsoft/WSL/issues/423
func isVscodeEnabled() bool {
	name, err := ioutil.ReadFile("/proc/sys/kernel/osrelease")
	if err == nil && strings.Contains(strings.ToLower(string(name)), "microsoft") {
		return false
	}

	name, err = ioutil.ReadFile("/proc/version")
	if err == nil && strings.Contains(strings.ToLower(string(name)), "microsoft") {
		return false
	}

	return true
}

// Manager manages enabled plugins.
type Manager struct {
	options        system.Options
	pluginManagers map[string]editor.Plugin
	// Defines the sort order of the enabled plugins
	keys []string
	// detected editors
	editors *editorsList
	// Encountered editors
	settings component.SettingsManager

	m sync.Mutex

	encountered map[string]bool
	metrics     component.MetricsManager
	platform    *platform.Platform

	onboardingDir       string
	onboardingEventSent map[string]struct{}

	lastSettingsCheck time.Time

	mruEditor string
}

// NewManager returns the plugin manager which handles HTTP requests for the enabled plugins.
// enableGoLand is temporary, the plugin will be enabled for all users in the future
func NewManager(options system.Options) *Manager {
	processMgr := process.NewManager()

	// initialize editor.Plugins
	plugins := append(
		[]editor.Plugin{
			getPlugin(atom.NewManager(processMgr)),
			getPlugin(sublime.NewManager(processMgr)),
			getPlugin(neovim.NewManager(processMgr)),
			getPlugin(vim.NewManager(processMgr)),
			getPlugin(spyder.NewManager(processMgr)),
			getPlugin(jupyterlab.NewManager()),
		},
		getPlugins(jetbrains.NewJetBrainsManagers(processMgr, options.BetaChannel))...,
	)

	if isVscodeEnabled() {
		plugins = append(plugins, getPlugin(vscode.NewManager(processMgr)))
	}

	var manager = &Manager{
		options:        options,
		pluginManagers: buildPluginMap(plugins),
		// the order of editors in the response
		keys:        buildPluginKeys(plugins),
		encountered: make(map[string]bool),

		onboardingDir:       os.TempDir(),
		onboardingEventSent: make(map[string]struct{}),
	}
	// make sure that all keys in the sort order spec exist
	for _, key := range manager.keys {
		_, found := manager.pluginManagers[key]
		if !found {
			panic(fmt.Sprintf("could not find %s in the managers map", key))
		}
	}

	return manager
}

// NewTestManager returns the plugin manager which handles HTTP requests for the enabled plugins.
func NewTestManager(options system.Options, plugins ...editor.Plugin) *Manager {
	return &Manager{
		options:        options,
		pluginManagers: buildPluginMap(plugins),
		// the order of editors in the response
		keys:          buildPluginKeys(plugins),
		encountered:   make(map[string]bool),
		onboardingDir: os.TempDir(),
	}
}

// Name implements component Core
func (m *Manager) Name() string {
	return "plugins"
}

// Initialize implements component Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.settings = opts.Settings
	m.metrics = opts.Metrics
	// init must not fail if there's no value yet for 'encountered_editors'
	if err := m.settings.GetObj("encountered_editors", &m.encountered); err != nil {
		log.Printf("error loading 'encountered_editors' value: %s", err.Error())
		m.encountered = make(map[string]bool)
	}
	m.platform = opts.Platform

	m.editors = newEditorsList(filepath.Join(m.platform.KiteRoot, "editors.json"))
	_ = m.editors.load()
}

// GoTick implements component GoTicker
// every 60 minutes, it updates the kitestatus property "spyder_suboptimal_settings"
func (m *Manager) GoTick(ctx context.Context) {
	if time.Since(m.lastSettingsCheck) > settingsCheckInterval {
		log.Printf("activated GoTick in plugin manager")
		defer func() {
			m.lastSettingsCheck = time.Now()
		}()

		if optimalSettings, _, err := spyder.SettingsStatus(ctx, m.pluginManagers[spyder.ID]); err == nil {
			log.Printf("updating Kite_status flag for spyder settings: suboptimal = %v", !optimalSettings)
			spyderSuboptimalSettingsStatus.SetBool(!optimalSettings)
		}
	}
}

// InstalledEditors implements component.PluginsManager
func (m *Manager) InstalledEditors() map[string]struct{} {
	if purged := m.editors.purgeDetected(); purged > 0 {
		_ = m.editors.save()
	}

	installed := make(map[string]struct{})
	for _, manager := range m.pluginManagers {
		editors, errObj := m.detectEditors(context.Background(), manager)
		if errObj != nil {
			continue
		}
		if len(editors) > 0 {
			installed[manager.ID()] = struct{}{}
		}
	}
	return installed
}

// JetbrainsInstalledProductIDs returns form ["GO", "IC", "PY"]
func (m *Manager) JetbrainsInstalledProductIDs() []string {
	return m.installedProductIDs()
}

// RegisterHandlers implements component Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/plugins", m.handleStatusAll).Methods("GET")
	mux.HandleFunc("/clientapi/plugins", m.handleStatusAllPost).Methods("POST")
	mux.HandleFunc("/clientapi/plugins/installed", m.handleUninstallAll).Methods("DELETE")
	// Interactive onboarding
	mux.HandleFunc("/clientapi/plugins/onboarding_file", m.handleOnboardingFile).Methods("GET")
	// Editor encounters
	mux.HandleFunc("/clientapi/plugins/encountered", m.handleGetEncountered).Methods("GET")
	mux.HandleFunc("/clientapi/plugins/encountered", m.handleSaveEncountered).Methods("POST")

	// Exposes last used editor
	mux.HandleFunc("/clientapi/plugins/most_recent", m.handleMRU).Methods("GET")

	// automatically installed plugins
	mux.HandleFunc("/clientapi/plugins/auto_installed", m.handleGetAutoInstalled).Methods("GET")
	mux.HandleFunc("/clientapi/plugins/auto_installed", m.handleResetAutoInstalled).Methods("DELETE")

	mux.HandleFunc("/clientapi/plugins/{id}", m.handleStatus).Methods("GET")
	mux.HandleFunc("/clientapi/plugins/{id}", m.handleInstall).Methods("POST")
	mux.HandleFunc("/clientapi/plugins/{id}", m.handleUninstall).Methods("DELETE")
	mux.HandleFunc("/clientapi/plugins/{id}/update", m.handleUpdate).Methods("POST")
	mux.HandleFunc("/clientapi/plugins/{id}/open", m.handleOpenFile).Methods("POST")

	// request to help Copilot with Kite's optimized settings for Spyder
	mux.HandleFunc("/clientapi/plugins/spyder/optimalSettings", m.handleSpyderSettingsStatus).Methods("GET")
	mux.HandleFunc("/clientapi/plugins/spyder/optimalSettings", m.handleApplySpyderSettings).Methods("POST")
}

// ProcessedEvent implements ProcessedEventer
func (m *Manager) ProcessedEvent(event *event.Event, editorEvent *component.EditorEvent) {
	m.recordMRUEditor(editorEvent)

	if !m.isOnboardingEvent(event) {
		return
	}

	if action := event.GetAction(); action != "edit" && action != "selection" {
		return
	}

	selections := event.GetSelections()
	if len(selections) == 0 {
		return
	}

	var offset int64 = -1
	for _, s := range selections {
		end := s.GetEnd()
		if end > offset {
			offset = end
		}
	}

	text := event.GetText()
	conv := stringindex.NewConverter(text)
	maxLength := conv.BytesFromRunes(len(text) - 1)
	// onboarding is completed when offset >= 80% of content length
	maxOffset := int64(float64(maxLength) * 0.8)
	if offset >= maxOffset {
		props := make(map[string]interface{}, 1)
		props[fmt.Sprintf("%s_onboarding_completed", editorEvent.Source)] = true

		m.metrics.UpdateUser(props)
	}
}

// HandleStatusAll retrieves a list of plugin installations on this machine.
func (m *Manager) handleStatusAll(w http.ResponseWriter, r *http.Request) {
	if m.platform.IsUnitTestMode {
		// return an empty plugin response because we don't want to block the unit tests
		writeJSON(w, http.StatusOK, PluginResponse{})
		return
	}

	// keep and return only those paths which still exist on disk
	if purged := m.editors.purgeDetected(); purged > 0 {
		_ = m.editors.save()
	}

	var statuses []*PluginStatus
	for _, manager := range m.pluginManagers {
		installs, errObj := m.detectEditors(r.Context(), manager)
		if errObj != nil {
			log.Println(errObj)
			continue
		}
		statuses = append(statuses, m.markEncountered(status(r.Context(), manager, installs)))
	}
	payload := PluginResponse{
		Plugins: statuses,
	}
	writeJSON(w, http.StatusOK, payload)
}

// HandleStatusAllPost retrieves a list of plugin installations on this machine.
// It also install editors when the corresponding setting is enabled
func (m *Manager) handleStatusAllPost(w http.ResponseWriter, r *http.Request) {
	if m.platform.IsUnitTestMode {
		// return an empty plugin response because we don't want to block the unit tests
		writeJSON(w, http.StatusOK, PluginResponse{})
		return
	}

	// update data on running editors
	m.detectRunning(r.Context())

	// keep only those paths which still exist on disk
	if purged := m.editors.purgeDetected(); purged > 0 {
		_ = m.editors.save()
	}

	// detect (common locations and running)
	paths := make(map[string][]string)
	for id, manager := range m.pluginManagers {
		running, err := manager.DetectRunningEditors(r.Context())
		if err != nil {
			log.Printf("error detecting running editors: %v", err)
		}

		detected, _ := manager.DetectEditors(r.Context())
		if err != nil {
			log.Printf("error detecting editors: %v", err)
		}

		paths[id] = shared.DedupePaths(append(running, detected...))
	}

	// auto-install for all, this takes care of the setting and encountered editors
	m.autoInstallPlugins(r.Context(), paths)

	// update status, this has to happen after the automatic installation
	var statuses []*PluginStatus
	for id, manager := range m.pluginManagers {
		editors := shared.MapEditors(r.Context(), paths[id], manager)
		statuses = append(statuses, m.markEncountered(status(r.Context(), manager, editors)))
	}

	// return new status
	payload := PluginResponse{
		Plugins: statuses,
	}
	writeJSON(w, http.StatusOK, payload)
}

// HandleUninstallAll deletes every installed plugin on the system.
func (m *Manager) handleUninstallAll(w http.ResponseWriter, r *http.Request) {
	var errors []*errorResponse
	var statuses []*PluginStatus
	for _, manager := range m.pluginManagers {
		installs, errObj := m.detectEditors(r.Context(), manager)
		if errObj != nil {
			errors = append(errors, errObj)
			continue
		}
		for _, install := range installs {
			err := manager.Uninstall(r.Context(), install.Path)
			if err != nil {
				errorObj := newErrorResponse(fmt.Sprintf("Failed to uninstall %s at path %s", manager.Name(), install.Path), err)
				errors = append(errors, errorObj)
			}
		}
		statuses = append(statuses, m.markEncountered(status(r.Context(), manager, installs)))
	}
	payload := uninstallAllResponse{
		PluginResponse: PluginResponse{
			Plugins: statuses,
		},
		Errors: errors,
	}
	writeJSON(w, http.StatusOK, &payload)
}

// HandleStatus returns the status of an editor family.
func (m *Manager) handleStatus(w http.ResponseWriter, r *http.Request) {
	manager, found := m.getPluginManager(r)
	if !found {
		http.NotFound(w, r)
		return
	}
	installs, errObj := m.detectEditors(r.Context(), manager)
	if errObj != nil {
		writeJSON(w, http.StatusConflict, &errObj)
		return
	}
	writeJSON(w, http.StatusOK, m.markEncountered(status(r.Context(), manager, installs)))
}

// HandleInstall executes the plugin install command for a given editor.
func (m *Manager) handleInstall(w http.ResponseWriter, r *http.Request) {
	// This is a workaround for the Windows setup flow when the user has a
	// running editor that has the plugin already installed. In this case we
	// use the install_only flag to indicate that we can ignore errors
	// caused by running editors.
	ignoreProcessRunning := r.FormValue("install_only") == "true"
	// Check manager and path
	manager, found := m.getPluginManager(r)
	if !found {
		http.Error(w, fmt.Sprintf("no manager found for %s", mux.Vars(r)["id"]), http.StatusNotFound)
		return
	}
	path := editorPath(r)
	if path == "" {
		http.Error(w, "Editor path not specified.", http.StatusBadRequest)
		return
	}

	// don't install if the editor is running and installation while running is not allowed
	// only return if this error is not to be ignored
	if cfg := manager.InstallConfig(r.Context()); cfg.Running && !cfg.InstallWhileRunning && !ignoreProcessRunning {
		errorObj := errorResponse{
			Title:  fmt.Sprintf("Failed to install %s at path %s", manager.Name(), path),
			Detail: "a process is running",
		}
		writeJSON(w, http.StatusConflict, &errorObj)
		return
	}

	err := manager.Install(r.Context(), path)
	if err != nil {
		clienttelemetry.KiteTelemetry("Editor Plugin Install Failed", map[string]interface{}{
			"editor": manager.ID(),
			"path":   path,
			"error":  err,
		})
		m.reportInstallError("error installing plugin", path, manager, err)
		errorObj := newErrorResponse(fmt.Sprintf("Failed to install %s at path %s", manager.Name(), path), err)
		writeJSON(w, http.StatusConflict, errorObj)
		return
	}

	clienttelemetry.KiteTelemetry("Editor Plugin Installed", map[string]interface{}{
		"editor": manager.ID,
		"path":   path,
	})
	installs, errObj := m.detectEditors(r.Context(), manager)
	if errObj != nil {
		writeJSON(w, http.StatusConflict, &errObj)
		return
	}
	writeJSON(w, http.StatusOK, m.markEncountered(status(r.Context(), manager, installs)))
}

// HandleUninstall executes the plugin uninstall command for a given editor.
func (m *Manager) handleUninstall(w http.ResponseWriter, r *http.Request) {
	// Check manager and path
	manager, found := m.getPluginManager(r)
	if !found {
		http.Error(w, fmt.Sprintf("no manager found for %s", mux.Vars(r)["id"]), http.StatusNotFound)
		return
	}
	path := editorPath(r)
	if path == "" {
		http.Error(w, "Editor path not specified.", http.StatusBadRequest)
		return
	}

	// return early if the editor is running and uninstall while running is not allowed
	if cfg := manager.InstallConfig(r.Context()); cfg.Running && !cfg.UninstallWhileRunning {
		writeJSON(w, http.StatusConflict, &errorResponse{
			Title:  fmt.Sprintf("Failed to uninstall %s at path %s", manager.Name(), path),
			Detail: "a process is running",
		})
		return
	}

	// Uninstall
	err := manager.Uninstall(r.Context(), path)
	if err != nil {
		clienttelemetry.KiteTelemetry("Editor Plugin Uninstall Failed", map[string]interface{}{
			"editor": manager.ID(),
			"path":   path,
			"error":  err,
		})
		errorObj := errorResponse{
			Title:  fmt.Sprintf("Failed to uninstall %s at path %s", manager.Name(), path),
			Detail: err.Error(),
		}
		writeJSON(w, http.StatusConflict, &errorObj)
		return
	}
	// Setup response
	clienttelemetry.KiteTelemetry("Editor Plugin Uninstalled", map[string]interface{}{
		"editor": manager.ID(),
		"path":   path,
	})
	installs, errObj := m.detectEditors(r.Context(), manager)
	if errObj != nil {
		writeJSON(w, http.StatusConflict, &errObj)
		return
	}
	writeJSON(w, http.StatusOK, m.markEncountered(status(r.Context(), manager, installs)))
}

// HandleUpdate executes the plugin update command for a given editor.
func (m *Manager) handleUpdate(w http.ResponseWriter, r *http.Request) {
	manager, found := m.getPluginManager(r)
	if !found {
		http.Error(w, fmt.Sprintf("no manager found for %s", mux.Vars(r)["id"]), http.StatusNotFound)
		return
	}

	// return early if update while running isn't allowed
	if cfg := manager.InstallConfig(r.Context()); cfg.Running && !cfg.UpdateWhileRunning {
		writeJSON(w, http.StatusConflict, &errorResponse{
			Title:  fmt.Sprintf("Failed to update %s", manager.Name()),
			Detail: "a process is running",
		})
		return
	}

	log.Printf("updating %s...", manager.Name())
	installs, errObj := m.detectEditors(r.Context(), manager)
	if errObj != nil {
		writeJSON(w, http.StatusConflict, &errObj)
		return
	}
	for _, install := range installs {
		if !manager.IsInstalled(r.Context(), install.Path) {
			continue
		}
		err := manager.Update(r.Context(), install.Path)
		if err != nil {
			p := newErrorResponse(fmt.Sprintf("Failed to update %s", manager.Name()), err)
			writeJSON(w, http.StatusConflict, &p)
			return
		}
	}
}

// handleGetEncountered returns the list of encountered editors
func (m *Manager) handleGetEncountered(w http.ResponseWriter, r *http.Request) {
	m.m.Lock()
	defer m.m.Unlock()
	writeJSON(w, http.StatusOK, m.encountered)
}

// handleSaveEncountered adds new editors to the list of encountered editors
// and returns the new list
func (m *Manager) handleSaveEncountered(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not read body: %s", err.Error()), http.StatusBadRequest)
		return
	}
	var encountered []string
	if err = json.Unmarshal(body, &encountered); err != nil {
		http.Error(w, fmt.Sprintf("could not decode body: %s", err.Error()), http.StatusBadRequest)
		return
	}

	m.m.Lock()
	defer m.m.Unlock()
	for _, new := range encountered {
		m.encountered[new] = true
	}
	if err = m.settings.SetObj("encountered_editors", m.encountered); err != nil {
		http.Error(w, fmt.Sprintf("error saving encountered editors: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, m.encountered)
}

// handleGetAutoInstalled returns the list of ids of automatically installed plugins
func (m *Manager) handleGetAutoInstalled(w http.ResponseWriter, r *http.Request) {
	// access setting key under lock
	m.m.Lock()
	defer m.m.Unlock()

	var ids []string
	_ = m.settings.GetObj(settings.AutoInstalledPluginIDsKey, &ids)

	if len(ids) == 0 {
		writeJSON(w, http.StatusNotFound, nil)
	} else {
		sort.Strings(ids)
		writeJSON(w, http.StatusOK, ids)
	}
}

// handleResetAutoInstalled clears the list of automatically installed plugins
func (m *Manager) handleResetAutoInstalled(w http.ResponseWriter, r *http.Request) {
	// access setting key under lock
	m.m.Lock()
	defer m.m.Unlock()

	var ids []string
	_ = m.settings.GetObj(settings.AutoInstalledPluginIDsKey, &ids)

	// now delete it
	_ = m.settings.Delete(settings.AutoInstalledPluginIDsKey)

	// return the value retrieved before we deleted it
	if len(ids) == 0 {
		writeJSON(w, http.StatusNotFound, nil)
	} else {
		sort.Strings(ids)
		writeJSON(w, http.StatusOK, ids)
	}
}

// handleOnboardingFile checks to see if the interactive onboarding file is in
// its specified location. If it is not, then it adds it
// it then returns the path to that file
func (m *Manager) handleOnboardingFile(w http.ResponseWriter, r *http.Request) {
	langParam := r.URL.Query().Get("language")
	// default to python
	if langParam == "" {
		langParam = "python"
	}

	language := lang.FromName(langParam)
	supported := map[lang.Language]bool{
		lang.Python:     true,
		lang.Golang:     true,
		lang.JavaScript: true,
	}
	if !supported[language] {
		http.Error(w, fmt.Sprintf("input language is not valid: %s", langParam), http.StatusNotFound)
		return
	}

	editorID := r.URL.Query().Get("editor")

	fileBytes, err := m.getOnboardingData(editorID, runtime.GOOS, language)
	if err != nil {
		clienttelemetry.Event("live_onboarding_creation_failed", map[string]interface{}{
			"editor": editorID,
		})
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// WriteFile truncates existing files
	fp := m.getOnboardingFilepath(language, editorID)
	if err = ioutil.WriteFile(fp, fileBytes, os.ModePerm); err != nil {
		clienttelemetry.Event("live_onboarding_creation_failed", map[string]interface{}{
			"path":   fp,
			"editor": editorID,
		})
		http.Error(w, fmt.Sprintf("error creating and writing onboarding file: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	successSent := func() bool {
		m.m.Lock()
		defer m.m.Unlock()
		if _, successSent := m.onboardingEventSent[editorID]; successSent {
			return true
		}
		m.onboardingEventSent[editorID] = struct{}{}
		return false
	}()
	if !successSent {
		clienttelemetry.Event("live_onboarding_creation_succeeded", map[string]interface{}{
			"path":   fp,
			"editor": editorID,
		})
	}

	// do we want to track invocations of this endpoint, or just file creations?
	writeJSON(w, http.StatusOK, fp)
}

func (m *Manager) getOnboardingData(editor, os string, language lang.Language) ([]byte, error) {
	ext := language.Extension()
	if ext == "go" {
		// this is a workaround, .go files would be picked up by the compiler
		ext = "golang"
	}

	if editor == "jupyter" {
		ext = jupyterExt
	}

	filenames := []string{
		fmt.Sprintf("onboarding/%s/kite_tutorial_%s_%s.%s", editor, editor, os, ext),
		fmt.Sprintf("onboarding/%s/kite_tutorial_%s.%s", editor, editor, ext),
	}

	// pick the first filename which exists in our onboarding assets
	for _, filename := range filenames {
		data, err := Asset(filename)
		if err == nil {
			return data, nil
		}
	}

	return nil, errors.Errorf("no onboarding file exists for editor / language / OS", filenames)
}

func (m *Manager) getOnboardingFilepath(language lang.Language, editor string) string {
	ext := language.Extension()
	if editor == "jupyter" {
		ext = jupyterExt
	}
	return filepath.Join(m.onboardingDir, fmt.Sprintf("kite_tutorial.%s", ext))
}

func (m *Manager) isOnboardingEvent(ev *event.Event) bool {
	language := lang.FromFilename(ev.GetFilename())
	if language != lang.Python && language != lang.Golang {
		return false
	}

	switch runtime.GOOS {
	case "windows":
		unixPath, err := localpath.ToUnix(m.getOnboardingFilepath(language, ""))
		return err == nil && strings.ToLower(ev.GetFilename()) == strings.ToLower(unixPath)
	default:
		return ev.GetFilename() == m.getOnboardingFilepath(language, "")
	}
}

type openFileRequest struct {
	EditorPath string `json:"path"`
	Filename   string `json:"filename"`
	Line       int    `json:"line,omitempty"`

	// for telemetry
	BlockRank int `json:"block_rank,omitempty"`
	FileRank  int `json:"file_rank,omitempty"`
}

func (m *Manager) handleOpenFile(w http.ResponseWriter, r *http.Request) {
	manager, found := m.getPluginManager(r)
	if !found {
		http.NotFound(w, r)
		return
	}

	defer r.Body.Close()

	var req openFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("could not decode body: %s", err.Error()), http.StatusBadRequest)
		return
	}

	_, err := manager.OpenFile(r.Context(), mux.Vars(r)["id"], req.EditorPath, req.Filename, req.Line)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, struct{}{})

	if req.BlockRank != 0 {
		clienttelemetry.EventWithKiteTelemetry("code_finder_code_block_opened", map[string]interface{}{
			"editor":     mux.Vars(r)["id"],
			"block_rank": req.BlockRank,
			"file_rank":  req.FileRank,
		})
	} else {
		clienttelemetry.EventWithKiteTelemetry("code_finder_file_opened", map[string]interface{}{
			"editor":    mux.Vars(r)["id"],
			"file_rank": req.FileRank,
		})
	}
}

// handleSpyderSettingsStatus returns true if the optimized settings are applied to all Spyder editors
// it returns false if non-optimal settings are configured for at least one Spyder editor
func (m *Manager) handleSpyderSettingsStatus(w http.ResponseWriter, r *http.Request) {
	optimalSettings, runningEditor, err := spyder.SettingsStatus(r.Context(), m.pluginManagers[spyder.ID])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var response struct {
		OptimalSettings bool `json:"optimalSettings"`
		RunningEditor   bool `json:"runningEditor"`
	}
	response.OptimalSettings = optimalSettings
	response.RunningEditor = runningEditor

	writeJSON(w, http.StatusOK, response)
}

// handleApplySpyderSettings applies the optimized settings for all detected installations of Spyder where
// the settings may be applied
func (m *Manager) handleApplySpyderSettings(w http.ResponseWriter, r *http.Request) {
	mgr := m.pluginManagers["spyder"]
	if mgr == nil {
		http.Error(w, "unable to find spyder plugin manager", http.StatusInternalServerError)
		return
	}

	if err := spyder.ApplyOptimalSettings(r.Context(), mgr); err != nil {
		http.Error(w, fmt.Sprintf("error applying spyder settings: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, "applied optimized to spyder configurations")
}

func (m *Manager) recordMRUEditor(editorEvent *component.EditorEvent) {
	m.m.Lock()
	defer m.m.Unlock()

	m.mruEditor = editorEvent.Source
}

func (m *Manager) handleMRU(w http.ResponseWriter, r *http.Request) {
	m.m.Lock()
	defer m.m.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{"editor": m.mruEditor})
}
