package mockserver

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/config"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/conversion/cohort"
	"github.com/kiteco/kiteco/kite-go/client/internal/conversion/monetizable"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/permissions"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/community/account"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/userids"
)

// NewTestClientServer returns a new TestClientServer which accepts the given validUsers.
// Returns an error if the setup failed
func NewTestClientServer(validUsers map[string]string) (*TestClientServer, error) {
	return NewTestClientServerFeatures(validUsers, nil)
}

// NewTestClientServerFeatures returns a new TestClientServer which accepts the given validUsers.
// Returns an error if the setup failed
func NewTestClientServerFeatures(validUsers map[string]string, featureOverride map[string]bool) (*TestClientServer, error) {
	return NewTestClientServerRootFeatures("", validUsers, featureOverride)
}

// NewTestClientServerRootFeatures returns a new TestClientServer which accepts the given Kite root & validUsers.
// Returns an error if the setup failed
func NewTestClientServerRootFeatures(root string, validUsers map[string]string, featureOverride map[string]bool) (*TestClientServer, error) {
	kited, err := NewKitedTestServer()
	if err != nil {
		return nil, err
	}

	backend, err := NewBackend(validUsers)
	if err != nil {
		return nil, err
	}

	kitedClient := NewKitedClient(kited.URL)

	p, err := platform.NewTestPlatformFeatures(root, featureOverride)
	if err != nil {
		return nil, err
	}

	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	basePath, err := ioutil.TempDir(usr.HomeDir, "kite-temp-files")
	if err != nil {
		return nil, err
	}

	t := &TestClientServer{
		Kited:                 kited,
		KitedClient:           kitedClient,
		Backend:               backend,
		Components:            component.NewTestManager(),
		Platform:              p,
		quitChan:              make(chan int, 1),
		ReadLoginLogoutEvents: true,
		Languages:             []lang.Language{lang.Python},
		BasePath:              basePath,
	}
	if root == "" {
		t.AddCleanupAction(func() {
			os.RemoveAll(t.Platform.KiteRoot)
		})
	}
	return t, nil
}

// TestClientServer bundles a mock backend server, a mock kited server and a client to talk to the kited server
type TestClientServer struct {
	Kited       *KitedTestServer
	Backend     *MockBackendServer
	Components  *component.Manager
	AuthClient  component.AuthClient
	Permissions component.PermissionsManager
	Settings    component.SettingsManager
	Metrics     component.MetricsManager
	Platform    *platform.Platform
	Network     component.NetworkManager

	// languages supported by our kited mock, defaults to Python
	Languages []lang.Language

	KitedClient *KitedClient

	quitChan chan int

	ReadLoginLogoutEvents bool

	cleanupActions []func()

	// guards kiteUser
	mu       sync.Mutex
	kiteUser *community.User

	BasePath string
}

// AddCleanupAction registers adds a new cleanup action which will be called by the Close() method
func (t *TestClientServer) AddCleanupAction(a func()) {
	t.cleanupActions = append(t.cleanupActions, a)
}

// Close releases resources used by the TestClientServer
func (t *TestClientServer) Close() {
	t.quitChan <- 0

	t.Components.Terminate()
	t.Kited.Close()
	t.Backend.Close()

	for _, a := range t.cleanupActions {
		a()
	}

	os.RemoveAll(t.BasePath)
}

// SendAccountCreationRequest posts the email and password to the server. It optionally waits until the account creation has been processed by the mockserver
func (t *TestClientServer) SendAccountCreationRequest(email, password string, waitForCreation bool) (*http.Response, error) {
	return t.KitedClient.SendAccountCreationRequest(email, password, waitForCreation)
}

// SendLoginRequest posts the email and password to the server. It optionally waits until the login has been processed by the mockserver
func (t *TestClientServer) SendLoginRequest(email, password string, waitForLogin bool) (*http.Response, error) {
	return t.KitedClient.SendLoginRequest(email, password, waitForLogin)
}

// SendLogoutRequest is a helper method to logout
func (t *TestClientServer) SendLogoutRequest(waitForLogout bool) (*http.Response, error) {
	return t.KitedClient.SendLogoutRequest(waitForLogout)
}

// DoKitedGet is a helper method to send a HTTP GET request to kited
func (t *TestClientServer) DoKitedGet(path string) (*http.Response, error) {
	return t.KitedClient.Get(path)
}

// DoKitedPost is a helper method to send a HTTP POST request to kited
func (t *TestClientServer) DoKitedPost(path string, body io.Reader) (*http.Response, error) {
	return t.KitedClient.Post(path, body)
}

// DoKitedPut is a helper method to send a HTTP PUT request to kited
func (t *TestClientServer) DoKitedPut(path string, body io.Reader) (*http.Response, error) {
	return t.KitedClient.Put(path, body)
}

// MockNetworkManager is a mock NetworkManager, who's needed to avoid a
// network->auth->mockserver->network import cycle
type MockNetworkManager struct {
	online      bool
	kitedOnline bool
}

// Name implements interface Core
func (m *MockNetworkManager) Name() string {
	return "network"
}

// SetOnline sets the network to online or offline based on the value of the bool
func (m *MockNetworkManager) SetOnline(val bool) {
	m.online = val
}

// SetOffline sets the network to online or offline based on the value of the bool
func (m *MockNetworkManager) SetOffline(val bool) {
	m.online = !val
}

// Online implements interface NetworkManager
func (m *MockNetworkManager) Online() bool {
	return m.online
}

// CheckOnline implements interface component.NetworkManager
func (m *MockNetworkManager) CheckOnline(ctx context.Context) bool {
	return m.online
}

// KitedOnline implements interface component.NetworkManager
func (m *MockNetworkManager) KitedOnline() bool {
	return m.kitedOnline
}

// KitedInitialized implements interface KitedEventer
func (m *MockNetworkManager) KitedInitialized() {
	m.kitedOnline = true
}

// KitedUninitialized implements interface KitedEventer
func (m *MockNetworkManager) KitedUninitialized() {
	m.kitedOnline = false
}

// NewMockNetworkManager returns a new MockNetworkManager
func NewMockNetworkManager() component.NetworkManager {
	return &MockNetworkManager{
		online: true,
	}
}

// SetupWithCustomAuthClient configures performs a default setup, but uses a custom auth client
// This is useful for tests which would otherwise create an import cycle, e.g. to test metrics which uses the TestClientServer
// which used metrics if there wasn't a way to pass a custom auth client
func (t *TestClientServer) SetupWithCustomAuthClient(authClient component.AuthClient, components ...component.Core) error {
	s := settings.NewTestManager()

	metrics := metrics.NewMockManager()
	permMgr := permissions.NewManager(t.Languages, nil)

	return t.SetupComponents(authClient, s, permMgr, metrics, components...)
}

// SetupComponents configures the components registered in the mocked Kited http server
func (t *TestClientServer) SetupComponents(auth component.AuthClient, settings component.SettingsManager, permissions component.PermissionsManager, metrics component.MetricsManager, components ...component.Core) error {
	t.AuthClient = auth
	t.Permissions = permissions
	t.Settings = settings
	t.Metrics = metrics

	userIds := userids.NewUserIDs(t.Platform.InstallID, t.Platform.MachineID)

	// setup components
	if auth != nil {
		t.Components.Add(auth)
	}

	if permissions != nil {
		t.Components.Add(permissions)
	}

	if settings != nil {
		t.Components.Add(settings)
		settings.AddNotificationTarget(t.Components)
	}

	if metrics != nil {
		t.Components.Add(metrics)
	}

	cohort := cohort.NewTestManager(
		&monetizable.SegmenterMock{
			IsMonetizableReturns: true,
		},
	)
	t.Components.Add(cohort)

	var network component.NetworkManager
	if t.Network != nil {
		network = t.Network
	} else {
		network = NewMockNetworkManager()
	}

	t.Components.Add(network)

	for _, c := range components {
		t.Components.Add(c)
	}

	configuration := config.GetConfiguration(t.Platform)

	t.Components.Initialize(component.InitializerOptions{
		KitedURL:      t.Kited.URL,
		Configuration: &configuration,
		AuthClient:    auth,
		License:       auth,
		Cohort:        cohort,
		Permissions:   permissions,
		Settings:      settings,
		Metrics:       metrics,
		Platform:      t.Platform,
		Network:       network,
		UserIDs:       userIds,
	})

	// register HTTP handlers
	t.Components.RegisterHandlers(t.Kited.Router)

	// make sure to empty the (blocking) login / logout channels, if available
	if t.ReadLoginLogoutEvents && auth != nil {
		go t.handleAuthLoop()
	}

	if auth != nil {
		auth.SetTarget(t.Backend.URL)
	}

	if auth != nil {
		// emulates logic in client/http.go
		hasCookie := auth.HasAuthCookie()
		remoteUser, remoteErr := auth.FetchUser(context.Background())
		localUser, localErr := auth.CachedUser()
		switch {
		// If we were able to authenticate remotely, log in. This means the user has a valid
		// session and we were able to fetch the user object remotely
		case remoteErr == nil:
			auth.LoggedInChan() <- remoteUser
			// User will identify in the select loop below via normal login flow

		// If we have an auth cookie and a cached user, treat the user as logged in. This means
		// the user had a valid session before, and a user object was cached during that session.
		// But we currently cannot fetch the user remotely (i.e user is offline)
		case hasCookie && localErr == nil:
			auth.LoggedInChan() <- localUser
			// User will identify in the select loop below via normal login flow
		}
	}

	clienttelemetry.SetCustomTelemetryClient(nil)
	clienttelemetry.SetClientVersion("1.0.0-unit-test")

	return nil
}

// CurrentUser returns the user which is currently logged into kited
func (t *TestClientServer) CurrentUser() (*community.User, error) {
	return t.KitedClient.CurrentUser()
}

// CurrentPlan returns the plan for the user which is currently logged into kited
func (t *TestClientServer) CurrentPlan() (*account.PlanResponse, error) {
	return t.KitedClient.CurrentPlan()
}

func (t *TestClientServer) handleAuthLoop() {
	for {
		t.HandleAuthEvent()
	}
}

// HandleAuthEvent handles a single pending login or logout events
func (t *TestClientServer) HandleAuthEvent() {
	select {
	case <-t.quitChan:
		return

	case user := <-t.AuthClient.LoggedInChan():
		log.Printf("Login of user %s", user.Name)
		t.AuthClient.SetUser(user)
		t.Components.LoggedIn()
		t.mu.Lock()
		t.kiteUser = user
		t.mu.Unlock()

		// make sure that we do not set 0 as userID
		uids := userids.NewUserIDs("", "test-case-machine")
		uids.SetUser(user.ID+1, "", true)
		clienttelemetry.SetUserIDs(uids)

	case <-t.AuthClient.LoggedOutChan():
		log.Printf("User logged out")
		t.Components.LoggedOut()
		t.mu.Lock()
		t.kiteUser = nil
		t.mu.Unlock()
	}
}

// GetFilePath returns a sub-path of the Whitelisted base path which is suitable for the current platform
func (t *TestClientServer) GetFilePath(path ...string) string {
	all := append([]string{t.BasePath}, path...)
	return filepath.Join(all...)
}
