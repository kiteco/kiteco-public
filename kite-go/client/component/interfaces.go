package component

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/config"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

// Core is the base interface used for components, it provides a name
type Core interface {
	Name() string
}

// InitializerOptions provides the values passed to Initialize()
type InitializerOptions struct {
	KitedURL      *url.URL
	Configuration *config.Configuration
	AuthClient    AuthClient
	DocsClient    http.Handler
	License       interface {
		licensing.ProductGetter
		licensing.StatusGetter
		licensing.TrialAvailableGetter
	}
	Settings      SettingsManager
	Cohort        CohortManager
	Permissions   PermissionsManager
	Plugins       PluginsManager
	Metrics       MetricsManager
	Platform      *platform.Platform
	Network       NetworkManager
	UserIDs       userids.IDs
	Notifs        NotificationsManager
	Status        StatusManager
	RemoteContent RemoteContentManager
}

// Initializer is called during a client's setup to initialize a comopnent with the base URL, the auth client and the
// managers of settings and permisions
type Initializer interface {
	Initialize(opts InitializerOptions)
}

// Terminater is called on shutdown.
type Terminater interface {
	Terminate() // ... I'll be back ...
}

// UserAuth provides methods which are called after the user logged in or logged out.
type UserAuth interface {
	LoggedIn()
	LoggedOut()
}

// NetworkEventer provides methods which are called after an according change in network connectivity
type NetworkEventer interface {
	NetworkOnline()
	NetworkOffline()
}

// KitedEventer provides methods which are called after an according change in kited application status
type KitedEventer interface {
	KitedInitialized()
	KitedUninitialized()
}

// PluginEventer provides a method which is called after an editor send an event.
type PluginEventer interface {
	PluginEvent(*EditorEvent)
}

// ProcessedEventer provides a method which is called after an event has been sent to the backend
type ProcessedEventer interface {
	ProcessedEvent(*event.Event, *EditorEvent)
}

// EventResponser provides methods which are called after the backend processed an event
type EventResponser interface {
	EventResponse(*response.Root)
}

// Settings provides method which are called after a value was modified or removed
type Settings interface {
	SettingUpdated(string, string)
	SettingDeleted(string)
}

// Handlers is implemented by a component which provides http routes
type Handlers interface {
	RegisterHandlers(mux *mux.Router)
}

// Ticker can be used to repeatedly update a component
type Ticker interface {
	// GoTick is called at a regular interval so that components can perform regularly repeated operations.
	// The component should exit when the context expires. GoTick is called in a goroutine and may run
	// concurrently with other component methods.
	GoTick(ctx context.Context)
}

// TestFlusher is implemented by a component which supports flush for tests
type TestFlusher interface {
	TestFlush(ctx context.Context)
}
