package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/client/token"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/kiteserver"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/remotectrl"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

const (
	defaultHTTPTimeout = 10 * time.Second
)

var (
	// ProDefaultPlan instantiates a "typical" pro PlanResponse

	connectionTimeout       = 5 * time.Second
	connectionCheckInterval = 10 * time.Minute
	licenseCheckInterval    = 12 * time.Hour
	sessionKey              = "kite-session"

	errUserNotAvailable = errors.New("user is not available")
)

// NewClient creates a new SessionedProxy object, storing
// session information at the location provided.
func NewClient(store *licensing.Store) *Client {
	c := &Client{
		loggedInChan:  make(chan *community.User, 100),
		loggedOutChan: make(chan struct{}, 100),
		token:         token.NewToken(),
		httpTimeout:   defaultHTTPTimeout,
	}

	c.licenseStore = store

	return c
}

// NewTestClient returns an AuthClient configured for unit tests.
// It uses a shorter HTTP timeout and a temporary authority, for example.
func NewTestClient(httpTimeout time.Duration) *Client {
	testAuthority, err := licensing.NewTestAuthority()
	if err != nil {
		panic(err)
	}
	store := licensing.NewStore(testAuthority.CreateValidator(), "")

	c := NewClient(store)
	// TODO(naman) I think this is unnecessary, since it's reset during initialization.
	c.userCacheFile = filepath.Join(os.TempDir(), "user")
	c.httpTimeout = httpTimeout
	return c
}

// Client represents a connection to the backend. It will reflect the current
// authenticated state of the connection.
type Client struct {
	httpTimeout time.Duration

	loggedInChan  chan *community.User
	loggedOutChan chan struct{}

	machineID string
	token     *token.Token

	filepath        string
	licenseFilepath string
	debug           bool

	permissions component.PermissionsManager
	platform    *platform.Platform
	metrics     component.MetricsManager
	network     component.NetworkManager
	settings    component.SettingsManager
	cohort      component.CohortManager

	licenseStore        *licensing.Store
	licenseExistingUser bool

	// the time when we did an auth check based on Tick
	lastTickCheck       time.Time
	lastConnectionCheck time.Time
	lastLicenseCheck    time.Time

	mu sync.RWMutex
	// guarded by mu
	client *http.Client
	// guarded by mu
	target *url.URL
	// guarded by mu
	proxy *httputil.ReverseProxy
	// guarded by mu
	user *community.User
	// not guarded
	userIDs userids.IDs

	// guarded by mu
	remoteChannel  string
	remoteListener *remotectrl.Listener
	// not guarded
	remoteMsgLog   *log.Logger
	remoteMsgSet   sync.Map
	remoteHandlers []remotectrl.Handler

	// connection monitoring
	openedConnections int64
	closedConnections int64

	userCacheFile  string
	userCacheMutex sync.RWMutex
}

// Name implements component.Name
func (c *Client) Name() string {
	return "authclient"
}

// Initialize implements Initializer. It's called to setup this component
func (c *Client) Initialize(opts component.InitializerOptions) {
	c.permissions = opts.Permissions
	c.platform = opts.Platform
	c.metrics = opts.Metrics
	c.network = opts.Network
	c.settings = opts.Settings
	c.cohort = opts.Cohort

	c.filepath = filepath.Join(opts.Platform.KiteRoot, "session.json")
	c.licenseFilepath = filepath.Join(opts.Platform.KiteRoot, "license.json")
	c.machineID = opts.Platform.MachineID
	c.debug = opts.Platform.DevMode

	c.userCacheFile = filepath.Join(opts.Platform.KiteRoot, "user")
	c.userIDs = opts.UserIDs

	c.initializeRemote(opts)

	// restore auth from last session
	err := c.restoreAuth()
	if err != nil && !opts.Platform.IsUnitTestMode {
		log.Println(err)
	}

	if err := c.licenseStore.LoadFile(c.licenseFilepath); err != nil && !os.IsNotExist(err) {
		log.Printf("Unable to load license data from file %s: %v", c.licenseFilepath, err)
		rollbar.Error(errors.Errorf("unable to load license data: %s", err))
	} else {
		log.Printf("Successfully loaded %d licenses", c.licenseStore.Size())
	}

	kiteServerHost, _ := opts.Settings.Get(settings.KiteServer)

	// This can block initialization if the server is invalid, so do in a goroutine
	kitectx.Go(func() error {
		c.SettingUpdated(settings.KiteServer, kiteServerHost)
		return nil
	})
}

// SettingUpdated implements component.Settings
func (c *Client) SettingUpdated(key, value string) {
	if key == settings.KiteServer {
		_, _, err := kiteserver.GetHealth(value)
		c.mu.Lock()
		defer c.mu.Unlock()
		c.licenseStore.KiteServer = err == nil
	}
}

// SettingDeleted implements component.Settings
func (c *Client) SettingDeleted(key string) {
	if key == settings.KiteServer {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.licenseStore.KiteServer = false
	}
}

// GetProduct implements licensing.ProductGetter
func (c *Client) GetProduct() licensing.Product {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// return c.licenseStore.Product()
	return licensing.Pro
}

// TrialAvailable implements AuthClient
func (c *Client) TrialAvailable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// return c.licenseStore.TrialAvailable()
	return false
}

// LoggedInChan returns the channel where login events are posted. It implements AuthClient.
func (c *Client) LoggedInChan() chan *community.User {
	return c.loggedInChan
}

// LoggedOutChan returns the channel where logout events are posted. It implements AuthClient.
func (c *Client) LoggedOutChan() chan struct{} {
	return c.loggedOutChan
}

// RegisterHandlers implements component.Handlers
func (c *Client) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/user", c.handleUser)
	mux.HandleFunc("/clientapi/license-info", c.handleLicenseInfo).Methods("GET")
	mux.HandleFunc("/clientapi/default-email", c.handleDefaultEmail)
	mux.HandleFunc("/clientapi/login", c.handleLogin)
	mux.HandleFunc("/clientapi/logout", c.handleLogout)
	mux.HandleFunc("/clientapi/create-account", c.handleCreateAccount)
	mux.HandleFunc("/clientapi/create-passwordless", c.handleCreatePasswordlessAccount)
	mux.HandleFunc("/clientapi/authenticate", c.handleAuthenticate)

	mux.PathPrefix("/api/account/").Handler(c)

	// proxy this route to user-node for the client to query user-node availability
	mux.Handle("/ping", c)

	// Deprecated
	// TODO remove when handlePlan query will be removed from plugins
	mux.HandleFunc("/clientapi/plan", c.handlePlan).Methods("GET")
}

// Parse returns an url which is "ref" resolved in relatively to the backend URL. See url.URL.Parse() for details.
func (c *Client) Parse(ref string) (*url.URL, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.target == nil {
		return nil, fmt.Errorf("target url not available")
	}

	return c.target.Parse(ref)
}

// GoTick implements component Ticker.
func (c *Client) GoTick(ctx context.Context) {
	if time.Since(c.lastConnectionCheck) >= connectionCheckInterval {
		defer func() {
			c.lastConnectionCheck = time.Now()
		}()

		unclosed := c.getOpenConnections()
		if unclosed > 5 {
			rollbar.Warning(fmt.Errorf("auth.Client detected unclosed HTTP responses"), unclosed)
		}
	}

	if time.Since(c.lastLicenseCheck) >= licenseCheckInterval {
		defer func() {
			c.lastLicenseCheck = time.Now()
		}()

		err := c.RefreshLicenses(ctx)
		if err != nil && err != errUserNotAvailable {
			log.Printf("scheduled license refresh error: %v", err)
		}
	}
}

// SetUser sets the user, nil indicates that the user has logged out
func (c *Client) SetUser(user *community.User) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.user = user
	c.resetRemoteListenerLocked()

	// cache user if user is not nil. This means we don't invalidate the user
	// cache unless a valid login occurs.
	if user != nil {
		err := c.cacheUser(user)
		if err != nil {
			log.Printf("user caching error: %v", err)
		}

		c.licenseStore.SetUserID(c.user.IDString())
		_ = c.refreshLicensesLocked(context.Background())
		return
	}
	c.licenseStore.ClearUser()
}

// GetUser returns the currently logged in user or nil if there's no user
// read-locks mutex "mu"
func (c *Client) GetUser() (*community.User, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.user == nil {
		return nil, errUserNotAvailable
	}
	return c.user, nil
}

// LicenseStatus computes the current ProStatus (active, grace_period, inactive) based on the licenses available in licensestore
// It also return expirationDate, subscriptionEnd, current plan and product
func (c *Client) LicenseStatus() (time.Time, time.Time, licensing.Plan, licensing.Product) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// return c.licenseStore.LicenseStatus()
	return time.Time{}, time.Time{}, licensing.ProIndefinite, licensing.Pro
}

// RefreshLicenses loads the current licenses from the remote server and updates the license store
// Use refreshLicensesLocked when the mutex was already locked
func (c *Client) RefreshLicenses(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.refreshLicensesLocked(ctx)
}

func (c *Client) refreshLicensesLocked(ctx context.Context) (err error) {
	tokens, err := c.getRemoteTokensLocked(ctx)
	if err != nil {
		return err
	}

	// user lifecycle event
	defer func() {
		exp, planEnd, plan, product := c.licenseStore.LicenseStatus()
		go clienttelemetry.Update(map[string]interface{}{
			"license_expire": exp.Unix(),
			"plan_end":       planEnd.Unix(),
			"plan":           plan,
			"product":        product,
		})
	}()

	defer c.licenseStore.SaveFile(c.licenseFilepath)

	// reset to defaults before applying new values
	c.licenseStore.ClearAll()
	if c.user != nil {
		c.licenseStore.SetUserID(c.user.IDString())
	}

	for _, token := range tokens {
		err := c.licenseStore.Add(token)
		if err != nil {
			log.Printf("error adding license token: %v", err)
		}
	}

	return nil
}

// --

// client returns the HTTP client used by this auth client. The returned value can be nil. Used for testing.
func (c *Client) httpClient() *http.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.client
}

// getProxy returns the currently used proxy. It read-locks mutex "mu" to retrieve it.
func (c *Client) getProxy() *httputil.ReverseProxy {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.proxy
}

// saveAuth saves the cookie information to a json file on disk
func (c *Client) saveAuth() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var kiteCookies []*http.Cookie
	if c.client != nil && c.client.Jar != nil {
		cookies := c.client.Jar.Cookies(c.target)
		for _, cookie := range cookies {
			if cookie.Name == sessionKey {
				kiteCookies = append(kiteCookies, cookie)
			}
		}
	}

	buf, err := json.Marshal(kiteCookies)
	if err != nil {
		return errors.Errorf("error marshaling auth: %v", err)
	}

	err = ioutil.WriteFile(c.filepath, buf, os.ModePerm)
	if err != nil {
		return errors.Errorf("error writing auth: %v", err)
	}

	return nil
}

// restores the cookies by reading the json file on disk
// It assumes that all required locks were acquired before it's called
func (c *Client) restoreAuth() error {
	buf, err := ioutil.ReadFile(c.filepath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error reading auth: %v", err)
	}

	var cookies []*http.Cookie
	err = json.Unmarshal(buf, &cookies)
	if err != nil {
		return fmt.Errorf("error unmarshalling auth: %v", err)
	}

	if c.client != nil && c.client.Jar != nil {
		c.client.Jar.SetCookies(c.target, cookies)
	}

	return nil
}

// getOpenConnections returns the number of currently open connections
func (c *Client) getOpenConnections() int64 {
	return atomic.LoadInt64(&c.openedConnections) - atomic.LoadInt64(&c.closedConnections)
}
