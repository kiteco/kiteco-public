package client

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof" // importing this so debug handlers are registered
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/autostart"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/config"
	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal"
	local_permissions "github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/permissions"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics/completions"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics/livemetrics"
	"github.com/kiteco/kiteco/kite-go/client/internal/network"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/client/internal/statusicon"
	"github.com/kiteco/kiteco/kite-go/client/internal/updates"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/tfserving"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/kite-golib/telemetry"
)

// PreviousLogsSuffix is the suffix for logs that are from previous runs of kited
const PreviousLogsSuffix = "bak"

// ErrNotAuthenticated is the error returned if no user is authenticated.
var ErrNotAuthenticated = errors.New("not authenticated")

var docsTarget = "https://" + domains.DocsAPI

// Options contains options for the client
type Options struct {
	URL               url.URL
	Platform          *platform.Platform
	Configuration     config.Configuration
	Updater           updates.Manager
	Network           component.NetworkManager
	Notifs            component.NotificationsManager
	Languages         []lang.Language
	SigMetrics        *metrics.SignaturesMetric
	CompletionMetrics *completions.MetricsByLang
	WatcherMetrics    *metrics.WatcherMetric
	TFServingMetrics  *tfserving.Metrics
	UserIDs           *userids.UserIDs
	Plugins           component.PluginsManager
	Settings          component.SettingsManager
	Cohort            component.CohortManager
	Status            component.StatusManager
	LicenseStore      *licensing.Store
	RemoteContent     component.RemoteContentManager

	// init options to pass to the kite local component
	LocalOpts kitelocal.Options

	// allows to override the root directory for test cases. This is not used in production.
	TestRootDir string
	// allows to override the features status of the platform features
	TestFeaturesOverride map[string]bool

	// optional function to wrap or replace the global HTTP handler with a custom handler
	TestHandlers func([]negroni.Handler) []negroni.Handler
}

// Client represents the core of the client logic that brokers connections
// between the backend, client daemon, and UI interfaces.
type Client struct {
	URL           *url.URL
	Configuration *config.Configuration
	Platform      *platform.Platform
	components    *component.Manager
	AuthClient    component.AuthClient
	DocsClient    http.Handler
	Plugins       component.PluginsManager
	Settings      component.SettingsManager
	Cohort        component.CohortManager
	Permissions   component.PermissionsManager
	Network       component.NetworkManager
	Notifs        component.NotificationsManager
	Status        component.StatusManager
	RemoteContent component.RemoteContentManager

	kitelocal *kitelocal.Manager
	Metrics   component.MetricsManager
	Updater   updates.Manager
	UserIDs   *userids.UserIDs

	mu         sync.Mutex
	wg         sync.WaitGroup
	cancelFunc context.CancelFunc

	logsUploaded bool
	testReady    int32
}

// NewClient returns a new client object
func NewClient(opts Options) (*Client, error) {
	clienttelemetry.SetClientVersion(opts.Platform.ClientVersion)
	rollbar.SetClientVersion(opts.Platform.ClientVersion)
	telemetry.SetClientVersion(opts.Platform.ClientVersion)

	var componentMgr *component.Manager
	if opts.Platform.IsUnitTestMode {
		componentMgr = component.NewTestManager()
	} else {
		componentMgr = component.NewManager()
	}

	// currently env KITED_PYTHON_REMOTE="host:port" controls this setting
	if pythonRemoveEnv, ok := os.LookupEnv("KITED_PYTHON_REMOTE"); ok {
		opts.LocalOpts.RemoteResourceManager = pythonRemoveEnv
	}

	opts.UserIDs.SetLocal(true)
	smartSelectedMetrics := metrics.NewSmartSelectedMetrics()
	opts.LocalOpts.SmartSelectedMetrics = smartSelectedMetrics
	opts.LocalOpts.SignatureMetrics = opts.SigMetrics
	opts.LocalOpts.CompletionsMetrics = opts.CompletionMetrics
	opts.LocalOpts.WatcherMetrics = opts.WatcherMetrics
	klocal, err := kitelocal.NewManager(componentMgr, opts.LocalOpts)
	if err != nil {
		return nil, errors.Errorf("error building kitelocal manager: %s", err)
	}
	componentMgr.Add(klocal)
	componentMgr.Add(opts.Settings)

	// set up network connectivity manager, use mgr from opts first and fallback to live network manager
	networkMgr := opts.Network
	if networkMgr == nil {
		networkMgr = network.NewManager(componentMgr)
	}
	componentMgr.Add(networkMgr)

	// set up metrics
	metrics := livemetrics.NewManager(opts.SigMetrics, opts.CompletionMetrics, opts.WatcherMetrics, smartSelectedMetrics, opts.TFServingMetrics)
	componentMgr.Add(metrics)

	permMgr := local_permissions.NewManager(opts.Languages, metrics.PermissionsRequest)
	componentMgr.Add(permMgr)

	// set up the proxy for backend requests
	p := auth.NewClient(opts.LicenseStore)
	componentMgr.Add(p)

	// set up the reverse proxy for docs requests
	docsURL, err := url.Parse(docsTarget)
	if err != nil {
		return nil, fmt.Errorf("error parsing docs target url %s: %s", docsTarget, err)
	}
	d := httputil.NewSingleHostReverseProxy(docsURL)

	// register our updater, it provides HTTP routes
	componentMgr.Add(opts.Updater)

	c := &Client{
		URL:           &opts.URL,
		UserIDs:       opts.UserIDs,
		Configuration: &opts.Configuration,
		Platform:      opts.Platform,
		components:    componentMgr,
		kitelocal:     klocal,
		Metrics:       metrics,
		AuthClient:    p,
		DocsClient:    d,
		Plugins:       opts.Plugins,
		Network:       networkMgr,
		Settings:      opts.Settings,
		Cohort:        opts.Cohort,
		Permissions:   permMgr,
		Updater:       opts.Updater,
		Notifs:        opts.Notifs,
		Status:        opts.Status,
		RemoteContent: opts.RemoteContent,
	}
	return c, nil
}

// TestReady indicates whether the client has completed initialization (used in tests)
func (c *Client) TestReady() bool {
	return atomic.LoadInt32(&c.testReady) == 1
}

// AddComponent adds a new component to the Client
func (c *Client) AddComponent(comp component.Core) error {
	return c.components.Add(comp)
}

// Components returns all components which were added to the component manager
func (c *Client) Components() []component.Core {
	return c.components.Components()
}

// TestComponentManager returns the component manager, for tests only
func (c *Client) TestComponentManager() *component.Manager {
	return c.components
}

// Initialize initializes the components registered with Client
func (c *Client) Initialize(router *mux.Router) {
	// systray can't be shutdown, i.e. the UI loop can't be stopped
	// running more than one test method will lead to unpredictable segmentation faults
	// therefore we have to disable the systray component in unit tests
	if _, headless := os.LookupEnv("KITE_HEADLESS"); !headless && !c.Platform.IsUnitTestMode {
		ui := statusicon.NewManager(c.Updater)
		c.components.Add(ui)
	}

	c.Settings.AddNotificationTarget(c.components)
	c.Settings.AddNotificationTargetKey(settings.ServerKey, c.observeServer)
	c.Settings.AddNotificationTargetKey(settings.AutostartDisabledKey, c.observeAutostart)

	// init our components
	c.components.Initialize(component.InitializerOptions{
		KitedURL:      c.URL,
		Configuration: c.Configuration,
		AuthClient:    c.AuthClient,
		DocsClient:    c.DocsClient,
		License:       c.AuthClient,
		Permissions:   c.Permissions,
		Plugins:       c.Plugins,
		Settings:      c.Settings,
		Cohort:        c.Cohort,
		Metrics:       c.Metrics,
		Platform:      c.Platform,
		Network:       c.Network,
		UserIDs:       c.UserIDs,
		Notifs:        c.Notifs,
		Status:        c.Status,
		RemoteContent: c.RemoteContent,
	})

	// setup http routes
	c.components.RegisterHandlers(router)
}

// observeServer watches for server settings changes
func (c *Client) observeServer(server string) {
	if c.Connected() {
		c.Disconnect()
	}

	if server != "" {
		go c.Connect(server)
	}
}

// observeAutostart watches for autostart settings changes
func (c *Client) observeAutostart(disabled string) {
	if err := autostart.SetDisabled(disabled == "true"); err != nil {
		log.Printf("error setting autostart disabled to %s: %v", disabled, err)
	}
}

// Connected returns whether the client is Connected
func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cancelFunc != nil
}

// Disconnect disconnects the client
func (c *Client) Disconnect() {
	c.mu.Lock()
	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}
	c.mu.Unlock()

	// NOTE: Be sure to unlock BEFORE waiting, since defer statements in Connect
	// require the lock to update the state of c.connected.
	c.wg.Wait()
}

// Shutdown terminates the client. It can not be reused after this method was called.
func (c *Client) Shutdown() {
	if c.Connected() {
		c.Disconnect()
	}

	// shutdown tracking
	clienttelemetry.Close()

	c.components.Terminate()
}

// Connect will connect the client to the given host port
func (c *Client) Connect(url string) error {
	defer func() {
		// On panic, notify rollbar and restart the top-level Connect method.
		if err := recover(); err != nil {
			// TODO(juan): is it safe to use
			// GetUser here? are we confidant all
			// locks have been released?
			uid := c.UserIDs.UserID()
			email := c.UserIDs.Email()

			rollbar.PanicRecovery(err, url, uid, email)
			// avoid consuming 100% of cpu if there is a panic every time
			time.Sleep(100 * time.Millisecond)
			log.Println("restarting client...")
			go c.Connect(url)
		}
	}()

	c.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel
	c.mu.Unlock()

	c.wg.Add(1)
	defer c.wg.Done()
	return c.processHTTP(ctx, url)
}
