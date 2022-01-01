package component

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

// AuthClient defines the functions to work with authentication data.
// Component interfaces must not depend on implementations
type AuthClient interface {
	Core

	// LicenseStatus returns the pro status and different information about the current active license of the user
	licensing.StatusGetter
	licensing.ProductGetter
	licensing.TrialAvailableGetter

	// LoggedInChan returns the channel were login events are posted
	LoggedInChan() chan *community.User

	// LoggedOutChan returns the channel were logout events are posted
	LoggedOutChan() chan struct{}

	// Parse returns an url which is "ref" resolved in relatively to the backend URL. See url.URL.Parse() for details.
	Parse(ref string) (*url.URL, error)

	// Get performs an authenticated GET request with the provided path relative to the target
	Getter

	// Post performs an authenticated POST request with the provided path relative to the target
	Poster

	// NewRequest parallels http.NewRequest, but takes in a path.
	// The proxy will fill in the full path relative to the target.
	NewRequest(method, path, contentType string, body io.Reader) (*http.Request, error)

	// Do performs the provided HTTP request
	Do(ctx context.Context, req *http.Request) (*http.Response, error)

	// ServeHTTP proxies requests to the backend server. It persists cookies between requests and between restarts.
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// Target gets the URL that this proxy talks to
	Target() *url.URL

	// SetTarget updates the URL of the backend server.
	// If the target should be removed then use UnsetTarget instead of this method.
	SetTarget(target *url.URL)

	// UnsetTarget will reset the target, preventing requests from being forwarded by the proxy
	UnsetTarget()

	// HasAuthCookie returns whether we are authenticated
	HasAuthCookie() bool

	// HasProductionTarget returns whether the target is a production backend
	HasProductionTarget() bool

	// FetchUser fetches the remote user
	FetchUser(ctx context.Context) (*community.User, error)

	// CachedUser returns the cached user object if it exists
	CachedUser() (*community.User, error)

	// GetUser returns the currently logged-in user. It returns an error if the user is not logged in.
	GetUser() (*community.User, error)

	// SetUser updates the current user
	SetUser(*community.User)

	// RefreshLicenses refreshes the cached licenses with the data of the remote backend
	RefreshLicenser
}

/* Smaller interfaces used for testing */

// Getter contains a method to make a GET request
type Getter interface {
	Get(ctx context.Context, path string) (*http.Response, error)
}

// Poster contains a method to make a POST request
type Poster interface {
	Post(ctx context.Context, path, contentType string, body io.Reader) (*http.Response, error)
}

// RefreshLicenser cotains a method to refresh licenses
type RefreshLicenser interface {
	RefreshLicenses(ctx context.Context) error
}
