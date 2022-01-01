package clientapp

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/autostart"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/config"
	"github.com/kiteco/kiteco/kite-go/client/internal/activefile"
	"github.com/kiteco/kiteco/kite-go/client/internal/autosearch"
	"github.com/kiteco/kiteco/kite-go/client/internal/capture"
	"github.com/kiteco/kiteco/kite-go/client/internal/client"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/conversion/cohort"
	"github.com/kiteco/kiteco/kite-go/client/internal/cpuinfo"
	"github.com/kiteco/kiteco/kite-go/client/internal/debug"
	"github.com/kiteco/kiteco/kite-go/client/internal/desktoplogin"
	"github.com/kiteco/kiteco/kite-go/client/internal/health"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics"
	complmetrics "github.com/kiteco/kiteco/kite-go/client/internal/metrics/completions"
	"github.com/kiteco/kiteco/kite-go/client/internal/notifications"
	plugins "github.com/kiteco/kiteco/kite-go/client/internal/plugins_new"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-go/client/internal/proxy"
	"github.com/kiteco/kiteco/kite-go/client/internal/remotecontent"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	sidebarComp "github.com/kiteco/kiteco/kite-go/client/internal/sidebar"
	"github.com/kiteco/kiteco/kite-go/client/internal/status"
	"github.com/kiteco/kiteco/kite-go/client/internal/systeminfo"
	"github.com/kiteco/kiteco/kite-go/client/internal/updates"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/client/sidebar"
	"github.com/kiteco/kiteco/kite-go/client/startup"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/applesilicon"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/tfserving"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/kite-golib/telemetry"
	"github.com/shirou/gopsutil/host"
)

var (
	// ErrPortInUse is the error returned when Kite's HTTP port is in use.
	ErrPortInUse = errors.New("http port already in use")

	// ErrRunning is the error returned when kited was already running
	// but we simply want to launch the sidebar and exit quietly.
	ErrRunning = errors.New("already running")

	// ErrAutostartDisabled is the error returned when kited detected that it
	// was started at login/boot, but autostart was disabled
	ErrAutostartDisabled = errors.New("autostart is disabled")
)

const (
	// This is the port the kited HTTP server listens on. Only listen on the
	// local interface because exposing kited to the outside world would be a
	// security hazard, and also windows does not let you listen on external
	// interfaces without admin permissions. This is no longer configurable
	// via an environment var because the editors hardcode this value, so we
	// should too.
	httpPort = 46624
)

var (
	// this is overridden via the -X link flag in the build scripts
	gitCommit = "unknown commit"

	// flag to signal that we should use IDCC engine to power the
	// old completions endpoint
	useIDCCForOldCompletionsFlag = "USE_IDCC_FOR_OLD_COMPLETIONS"
)

// Start starts a client on 127.0.0.1:46624
func Start(opts *client.Options) (*client.Client, error) {
	c, _, err := StartPort(context.Background(), httpPort, false, opts)
	return c, err
}

// StartTestClient start a new test client on the given port, the host is 127.0.0.1. The returned HTTP server should be terminated in the test case.
func StartTestClient(ctx context.Context, port int, customOpts *client.Options, components ...component.Core) (*client.Client, *http.Server, *telemetry.MockClient, error) {
	if customOpts.Updater == nil {
		customOpts.Updater = updates.NewMockManager()
	}

	mockTrack := &telemetry.MockClient{}
	clienttelemetry.SetCustomTelemetryClient(mockTrack)

	c, server, err := StartPort(ctx, port, true, customOpts, components...)
	return c, server, mockTrack, err
}

// StartPort creates a kite application and returns the client.
func StartPort(ctx context.Context, port int, testCaseEnv bool, customOpts *client.Options, additionalComponents ...component.Core) (*client.Client, *http.Server, error) {
	if customOpts.Updater == nil {
		return nil, nil, errors.New("updater not defined in options")
	}

	mode := startup.GetMode(os.Args)

	var opts client.Options
	if customOpts != nil {
		opts = *customOpts
	}

	// initialize after we know its ok to startup...
	var err error
	var p *platform.Platform
	if testCaseEnv {
		p, err = platform.NewTestPlatformFeatures(opts.TestRootDir, opts.TestFeaturesOverride)
	} else {
		p, err = platform.NewPlatform()
	}
	if err != nil {
		return nil, nil, err
	}

	settingsFile := filepath.Join(p.KiteRoot, "settings.json")
	settingsMgr := settings.NewManager(settingsFile)
	settingsMgr.NotifyProxyValue(func(value string) {
		log.Println("proxy value changed, applying new settings")
		_ = proxy.Global.Configure(value)
	})

	// safe-guard on Linux: terminate early if autostart with the setting disabled is detected
	// on Linux, the kite-autostart service relies on kited to shutdown in this scenario
	if runtime.GOOS == "linux" && mode == startup.SystemBoot {
		autostartDisabled, _ := settingsMgr.GetBool(settings.AutostartDisabledKey)
		if autostartDisabled {
			return nil, nil, ErrAutostartDisabled
		}
	}

	if p.IsNewInstall {
		rand.Seed(time.Now().UnixNano())
		showChooseEngine := rand.Intn(100) < 0 // disabled
		settingsMgr.SetBool(settings.ChooseEngineKey, showChooseEngine)
	}

	// init the sidebar early with our settings, before we use the sidebar package
	sidebar.Init(settingsMgr)

	// init the autostart behavior based on the autostart setting value
	disabled, _ := settingsMgr.GetBool(settings.AutostartDisabledKey)
	if err := autostart.SetDisabled(disabled); err != nil {
		log.Println("error initializing autostart disabled:", err)
	}

	// start http server, listen explicitly so that we can return errors
	// do this before we initialize to avoid disrupting an existing kited instance
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		if (mode == startup.ManualLaunch || mode == startup.PluginLaunchWithSidebar) && !testCaseEnv {
			sidebar.Start()
			return nil, nil, ErrRunning
		}
		return nil, nil, ErrPortInUse
	}

	// Initialize the logger after we've determined that this is a unique instance of kited
	err = p.InitializeLogger()
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing logger: %v", err)
	}

	// Touch file indicating kited has run
	hasRunFile := filepath.Join(p.KiteRoot, "kited_has_run")
	f, err := os.Create(hasRunFile)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating hasRun file at %s: %s", hasRunFile, err.Error())
	}
	f.Close()

	cfg := config.GetConfiguration(p)
	hostPlatform, hostFamily, hostVersion, _ := host.PlatformInformation()

	log.Println("listening at port:", port)
	log.Println("machine ID:", p.MachineID)
	log.Println("install ID:", p.InstallID)
	log.Println("OS:", runtime.GOOS)
	log.Println("CPU:", cpuinfo.Get())
	log.Println("platform:", fmt.Sprintf("%s %s %s", hostPlatform, hostFamily, hostVersion))
	log.Println("root dir:", p.KiteRoot)
	log.Println("log file:", p.LogFile)
	log.Println("dev mode:", p.DevMode)
	log.Println("version:", p.ClientVersion)
	log.Println("configuration:", cfg.Name)

	// apply our client-wide proxy settings to the default HTTP client
	if err := proxy.Global.Configure(settingsMgr.GetProxyValue()); err != nil {
		log.Printf("error applying proxy configuration: %s", err.Error())
	}
	http.DefaultTransport = proxy.Global.DefaultTransport()
	log.Println("proxy:", proxy.Global.Value())

	// set up a sigpipe handler
	signalChan := make(chan os.Signal, 100)
	signal.Notify(signalChan, syscall.SIGPIPE)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case sig := <-signalChan:
				log.Printf("caught signal: %v. Ignoring.", sig)
			}
		}
	}()

	// get the version
	if p.IsDebugBuild && !testCaseEnv {
		log.Printf("debug build detected, setting ClientVersion to %s", gitCommit)
		p.ClientVersion = gitCommit
		clienttelemetry.EnableDev()
	}
	// setup user ids
	userIDs := userids.NewUserIDs(p.InstallID, p.MachineID)

	// setup rollbar
	rollbar.SetEnvironment(cfg.RollbarEnvironment)
	rollbar.SetToken(cfg.RollbarToken)
	rollbar.SetUserIDs(userIDs)

	// setup lifecycle segment token
	community.SetToken(cfg.MixpanelToken, cfg.CIOSiteID, cfg.CIOToken)

	sigMetrics := &metrics.SignaturesMetric{}
	completionMetrics := complmetrics.NewMetrics()
	watcherMetrics := &metrics.WatcherMetric{}

	// initialize the client
	clientURL := url.URL{Scheme: "http", Host: listener.Addr().String()}

	// reset the base options, leave custom options as passed
	opts.URL = clientURL
	opts.Platform = p
	opts.UserIDs = userIDs
	opts.Configuration = cfg
	opts.Languages = enabledLanguages(settingsMgr)
	opts.SigMetrics = sigMetrics
	opts.CompletionMetrics = completionMetrics
	opts.WatcherMetrics = watcherMetrics
	opts.TFServingMetrics = tfserving.GetMetrics()
	opts.Settings = settingsMgr

	if p.IsFeatureEnabled("REMOTE_MODELS") {
		opts.LocalOpts.RemoteModels = true
	}

	validator, err := client.NewLicenseValidator()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not create license validator")
	}
	if opts.LicenseStore == nil {
		opts.LicenseStore = licensing.NewStore(validator, userIDs.InstallID())
	}

	if !p.IsUnitTestMode {
		dismissed, ok := settingsMgr.GetBool(settings.ProLaunchNotificationDismissed)
		notifs := notifications.NewManager(ok && dismissed)
		additionalComponents = append(additionalComponents, notifs)
		opts.Notifs = notifs
	}

	statusMgr := status.NewManager()
	opts.Status = statusMgr
	additionalComponents = append(additionalComponents, statusMgr)

	cohortManager := cohort.NewManager()
	opts.Cohort = cohortManager
	additionalComponents = append(additionalComponents, cohortManager)

	remoteContentFile := filepath.Join(p.KiteRoot, "remotecontent.json")
	remoteContentManager := remotecontent.NewManager(remoteContentFile)
	opts.RemoteContent = remoteContentManager
	additionalComponents = append(additionalComponents, remoteContentManager)

	pluginsManager := plugins.NewManager(system.Options{
		DevMode:     p.DevMode,
		BetaChannel: p.IsFeatureEnabled("BETA_PLUGINS"),
	})
	opts.Plugins = pluginsManager
	additionalComponents = append(additionalComponents, pluginsManager)

	client, err := client.NewClient(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing client: %v", err)
	}

	additionalComponents = append(additionalComponents, settingsMgr)
	setupComponents(client, additionalComponents...)

	router := mux.NewRouter()
	client.Initialize(router)

	updatePlugins := !p.IsFeatureEnabled("PLUGIN_DEBUG") && !testCaseEnv
	go pluginsManager.BackgroundTask(ctx, 30*time.Minute, updatePlugins)

	segment := &segmentTracker{}

	handlers := []negroni.Handler{
		midware.NewRecovery(),
		midware.NewLogger(p.Logger),
		segment,
		negroni.Wrap(router),
	}

	if opts.TestHandlers != nil {
		handlers = opts.TestHandlers(handlers)
	}

	middleware := negroni.New(handlers...)

	hostPort := listener.Addr().String()
	server := &http.Server{Addr: hostPort, Handler: middleware}
	go server.Serve(listener)

	// Determine startup mode, and determine whether we want to start the sidebar
	// NOTE: This logic currently only applies to OS X
	if !testCaseEnv {
		log.Println("using startup mode:", mode.String())
		log.Println("using startup channel:", startup.GetChannel(os.Args))
		switch mode {
		case startup.ManualLaunch:
			_ = sidebar.Start()
		case startup.RelaunchAfterUpdate:
			_ = sidebar.StartIfPreviouslyVisible()
		case startup.SystemBoot:
			// Don't start the sidebar
		case startup.PluginLaunch:
			// Don't start the sidebar
		case startup.PluginLaunchWithSidebar:
			_ = sidebar.Start()
		case startup.SidebarRestart:
			// Don't start the sidebar
		default:
			// Don't start the sidebar
		}
	}

	return client, server, nil
}

func setupComponents(client *client.Client, additionalComponents ...component.Core) {
	desktop := desktoplogin.NewManager()
	client.AddComponent(desktop)

	debugMgr := debug.NewManager()
	client.AddComponent(debugMgr)

	systemInfo := systeminfo.NewManager()
	client.AddComponent(systemInfo)

	autosearch := autosearch.NewManager()
	client.AddComponent(autosearch)

	activefile := activefile.NewManager()
	client.AddComponent(activefile)

	health := health.NewManager()
	client.AddComponent(health)

	sb := sidebarComp.NewManager()
	client.AddComponent(sb)

	capt := capture.NewManager()
	client.AddComponent(capt)

	for _, c := range additionalComponents {
		client.AddComponent(c)
	}
}

func enabledLanguages(settingsMgr *settings.Manager) []lang.Language {
	// Only support Python when we're on Apple Silicon
	if applesilicon.Detected {
		return []lang.Language{
			lang.Python,
		}
	}
	langs := []lang.Language{
		lang.Bash,
		lang.C,
		lang.Cpp,
		lang.CSharp,
		lang.CSS,
		lang.Golang,
		lang.HTML,
		lang.Java,
		lang.JavaScript,
		lang.JSX,
		lang.Kotlin,
		lang.Less,
		lang.ObjectiveC,
		lang.PHP,
		lang.Python,
		lang.Ruby,
		lang.Scala,
		lang.TSX,
		lang.TypeScript,
		lang.Vue,
	}
	return langs
}

type segmentTracker struct {
	m     sync.RWMutex
	paths map[string]map[int]int
	count int
}

func (s *segmentTracker) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Autosearch needs to use a different codepath so that it can successfully
	// upgrade to a websocket connection
	socketpaths := map[string]struct{}{
		"/autosearch":   {},
		"/active-file":  {},
		"/codenav/push": {},
	}
	if _, issocket := socketpaths[r.URL.Path]; issocket {
		next.ServeHTTP(w, r)
		return
	}

	next.ServeHTTP(w, r)

	if r.URL.Path == "/clientapi/editor/completions" ||
		r.URL.Path == "/clientapi/editor/signatures" {
		switch nw := w.(type) {
		case negroni.ResponseWriter:
			s.incrPath(r.URL.Path, nw.Status())
		}
	}
}

func (s *segmentTracker) incrPath(path string, status int) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.paths == nil {
		s.paths = make(map[string]map[int]int)
	}
	if s.paths[path] == nil {
		s.paths[path] = make(map[int]int)
	}

	s.paths[path][status]++
	s.count++

	if s.count > flushThreshold {
		s.flushLocked()
	}
}

const flushThreshold = 50

func (s *segmentTracker) flushLocked() {
	paths := s.paths
	s.paths = nil
	s.count = 0

	clienttelemetry.KiteTelemetry("Client HTTP Batch", map[string]interface{}{
		"requests": paths,
	})
}
