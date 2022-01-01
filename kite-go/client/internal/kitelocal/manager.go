package kitelocal

import (
	"context"
	"log"
	"net/http"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/driver"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/completions"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/editorapi"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/filesystem"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/indexing"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/ksgexperiment"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/navigation"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/internal/signatures"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics"
	complmetrics "github.com/kiteco/kiteco/kite-go/client/internal/metrics/completions"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/event"
	lexicalapi "github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	pythonapi "github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

const (
	eventPoolSize      = 2
	responseBufferSize = 50
	idleTimeoutMinutes = 15
)

// compile-time check that we implement the intended components
var (
	_ = component.PluginEventer((*Manager)(nil))
)

// Options defines the settings of the kitelocal component
type Options struct {
	// IndexedDir is the directory to index, default to $HOME
	IndexedDir string

	// Dists is used by tests to customize the loaded data sets, pass nil to load the default sets, i.e. all set
	Dists []keytypes.Distribution

	// Disable dynamic loading of distributions other than those provided via dists
	DisableDynamicLoading bool

	// SignatureMetrics are used to track popular patterns performance
	SignatureMetrics *metrics.SignaturesMetric

	// CompletionsMetrics are used to track popular patterns performance
	CompletionsMetrics *complmetrics.MetricsByLang

	// WatcherMetrics are used to track the number of active watches
	WatcherMetrics *metrics.WatcherMetric

	// ProSelectedMetrics are used to track the number of pro completions selected
	SmartSelectedMetrics *metrics.SmartSelectedMetrics

	// RemoteModels enables TFServing-based remote models
	RemoteModels bool

	// RemoteResourceManager defines the optional endpoint, where the remote resource manager lives
	RemoteResourceManager string
}

// Manager currently controls or houses all logic associated with kitelocal
type Manager struct {
	opts      Options
	ctx       context.Context
	ctxCancel func()

	pythonServices *python.Services

	Responses chan *response.Root

	userIDs     userids.IDs
	components  *component.Manager
	permissions component.PermissionsManager
	authClient  component.AuthClient
	platform    *platform.Platform

	pool           *workerpool.Pool
	eventProcessor *eventProcessor

	fileProcessor *fileProcessor

	fs          *filesystem.Manager
	indexer     *indexing.Context
	editorAPI   *editorapi.Manager
	signatures  *signatures.Manager
	completions *completions.Manager
	ksg         *ksgexperiment.Manager
	codenav     *navigation.Manager

	// indexedDir is the directory which is walked and indexed by kite local, defaults to $HOME
	indexedDir            string
	cacheRoot             string
	dists                 []keytypes.Distribution
	disableDynamicLoading bool
	debug                 bool

	idleTimerLock sync.Mutex
	idleTimer     *time.Timer
}

// NewManager creates a new Manager
func NewManager(components *component.Manager, opts Options) (*Manager, error) {
	indexedDir := opts.IndexedDir
	// default to $HOME on Windows and macOS, don't watch a root dir on Linux
	if indexedDir == "" {
		if runtime.GOOS == "linux" {
			log.Printf("watching files on-demand on Linux, ignoring indexedDir")
		} else {
			usr, err := user.Current()
			if err != nil {
				log.Println("unable to get current user:", err)
				return nil, err
			}
			indexedDir = usr.HomeDir
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		ctx:                   ctx,
		ctxCancel:             cancel,
		opts:                  opts,
		Responses:             make(chan *response.Root, responseBufferSize),
		components:            components,
		indexedDir:            indexedDir,
		dists:                 opts.Dists,
		disableDynamicLoading: opts.DisableDynamicLoading,
	}

	return m, nil
}

// Name implements component.Core
func (m *Manager) Name() string {
	return "kitelocal"
}

// Initialize implements component.Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
	err := datadeps.Enable()
	if err != nil {
		panic(err)
	}
	datadeps.SetLocalOnly()

	tfThreads, err := opts.Settings.GetInt(settings.TFThreadsKey)
	if err != nil {
		tfThreads = 1
	}
	log.Printf("using tf_threads value of %d", tfThreads)
	tensorflow.SetTensorflowThreadpoolSize(tfThreads)

	// Don't initialize pool in NewManager because it will start unecessary goroutines
	m.pool = workerpool.NewWithCtx(m.ctx, eventPoolSize)
	m.platform = opts.Platform
	loadOpts := LoadOptions{
		Blocking:               opts.Platform.IsUnitTestMode, // blocking init in unit test environment
		Dists:                  m.dists,
		DisableDynamicLoading:  m.disableDynamicLoading,
		RemoteResourcesManager: m.opts.RemoteResourceManager,
	}
	pythonServices, err := LoadPythonServices(m.ctx, loadOpts)
	if err != nil {
		panic(err)
	}
	m.pythonServices = pythonServices

	debug.SetGCPercent(15)

	m.userIDs = opts.UserIDs
	m.permissions = opts.Permissions
	m.authClient = opts.AuthClient
	m.eventProcessor = newEventProcessor(opts.Permissions, opts.Settings.GetMaxFileSizeBytes)

	m.fs = filesystem.NewManager(filesystem.Options{
		RootDir:        m.indexedDir,
		KiteDir:        opts.Platform.KiteRoot,
		DutyCycle:      0.15,
		WatcherMetrics: m.opts.WatcherMetrics,
	})

	m.indexer = indexing.NewContext(m.ctx, pythonServices, m.fs, m.userIDs)

	metricsDisabledSetting, _ := opts.Settings.GetBool(settings.MetricsDisabledKey)
	metricsDisabled := metricsDisabledSetting || opts.Platform.IsUnitTestMode

	fileDriverDebug := opts.Platform.IsFeatureEnabled("DEBUG_FILEDRIVER")
	if fileDriverDebug {
		log.Println("Running File Driver in Debug mode")
	}
	m.fileProcessor = newFileProcessor(pythonServices, m.indexer, m.userIDs, metricsDisabled, fileDriverDebug)

	m.indexer.DriverProvider = m.fileProcessor

	m.editorAPI = editorapi.NewManager(m.fileProcessor, pythonServices, m.indexer)
	m.signatures = signatures.NewManager(m.fileProcessor, signatures.Options{
		Metric: m.opts.SignatureMetrics,
	})

	modelOpts := lexicalmodels.DefaultModelOptions
	kiteServerHost, _ := opts.Settings.Get(settings.KiteServer)
	switch {
	case kiteServerHost != "":
		log.Println("Kite Server found:", kiteServerHost)
		modelOpts = modelOpts.WithRemoteModels(kiteServerHost)

	case m.opts.RemoteModels:
		log.Println("Enabling Remote Models")
		modelOpts = modelOpts.WithRemoteModels(lexicalmodels.DefaultRemoteHost)
	}

	lexicalModels, err := lexicalmodels.NewModels(modelOpts)
	if err != nil {
		panic(err)
	}
	opts.Status.SetModels(lexicalModels)

	m.codenav, err = navigation.NewManager(filepath.Join(opts.Platform.KiteRoot, "git-cache.json"))
	if err != nil {
		panic(err)
	}
	opts.Status.SetNav(m.codenav)

	completionsOpts := completions.Options{
		Metrics:              m.opts.CompletionsMetrics,
		SmartSelectedMetrics: m.opts.SmartSelectedMetrics,
		PythonOptions: pythonapi.Options{
			ResourceManager:           pythonServices.ResourceManager,
			Models:                    pythonServices.Models,
			LexicalModels:             lexicalModels,
			LocalContext:              m.indexer,
			GGNNSubtokenEnabled:       m.platform.GGNNSubtokenEnabled,
			GGNNSubtokenEnabledByFlag: m.platform.GGNNSubtokenEnabledByFlag,
		},
		LexicalOptions: lexicalapi.Options{
			Models: lexicalModels,
		},
		ModelOptions: modelOpts,
	}

	m.completions = completions.NewManager(m.fileProcessor, completionsOpts)

	m.ksg = ksgexperiment.NewManager()

	m.editorAPI.Initialize(opts)
	m.signatures.Initialize(opts)
	m.completions.Initialize(opts)
	m.fs.Initialize(opts)
	m.ksg.Initialize(opts)
	m.codenav.Initialize(opts)

	m.idleTimerLock.Lock()
	defer m.idleTimerLock.Unlock()
	m.idleTimer = time.AfterFunc(idleTimeoutMinutes*time.Minute, func() {
		log.Printf("releasing resources due to user idle after %d minutes", idleTimeoutMinutes)
		m.reset()
	})

	debug.FreeOSMemory()
}

// reset releases resources of this manager
func (m *Manager) reset() {
	m.completions.Reset()
	m.fileProcessor.reset()
	m.indexer.Reset()
	pythonparser.PurgeParseCache()
	m.pythonServices.Reset()
	m.eventProcessor.reset()

	debug.FreeOSMemory()
}

// Provider returns the driver.Provider
func (m *Manager) Provider() driver.Provider {
	return m.fileProcessor
}

// Terminate implements component.Terminater
func (m *Manager) Terminate() {
	m.idleTimer.Stop()

	m.reset()

	m.pool.Stop()
	m.fs.Terminate()
	m.indexer.Terminate()
	m.codenav.Terminate()

	m.pythonServices.Close()

	m.ctxCancel()

	close(m.Responses)
}

// PluginEvent implements component.PluginEventer
func (m *Manager) PluginEvent(*component.EditorEvent) {
	m.idleTimerLock.Lock()
	defer m.idleTimerLock.Unlock()
	m.idleTimer.Stop()
	m.idleTimer.Reset(idleTimeoutMinutes * time.Minute)
	m.fs.StartWalk()
}

// ProcessedEvent implements component.ProcessedEventer
func (m *Manager) ProcessedEvent(evt *event.Event, editorEvt *component.EditorEvent) {
	m.completions.ProcessedEvent(evt, editorEvt)
	m.codenav.ProcessedEvent(evt, editorEvt)
}

// SettingUpdated implements component.Settings
func (m *Manager) SettingUpdated(key, value string) {
	if key == "setup_completed" && value == "true" {
		m.fs.StartWalk()
	}
	if key == settings.KiteServer {
		m.completions.ToggleRemote(value)
	}
}

// SettingDeleted implements component.Settings
func (m *Manager) SettingDeleted(key string) {
	if key == settings.KiteServer {
		m.completions.ToggleRemote("")
	}
}

// TestFlush waits until all the jobs in the worker pool are finished. It implements component TestFlusher
func (m *Manager) TestFlush(ctx context.Context) {
	m.indexer.TestFlush(ctx)
	_ = m.pool.Wait()
}

// RegisterHandlers implements component.Handlers
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/editor/event", m.handleEditorEvent).Methods("POST")
	mux.HandleFunc("/clientapi/iskitelocal", m.handleIsKiteLocal).Methods("GET")

	m.codenav.RegisterHandlers(mux)
	m.editorAPI.RegisterHandlers(mux)
	m.signatures.RegisterHandlers(mux)
	m.completions.RegisterHandlers(mux)
	m.ksg.RegisterHandlers(mux)
}

// handleIsKiteLocal is the handler for /clientapi/iskitelocal
func (m *Manager) handleIsKiteLocal(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// --

func (m *Manager) logf(msg string, objs ...interface{}) {
	if m.debug {
		log.Printf("!! "+msg, objs...)
	}
}
