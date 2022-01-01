package proxy

import (
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMode(t *testing.T) {
	p := NewProxy()
	assert.Equal(t, settings.EnvironmentProxySentinel, p.Value())
	err := p.Configure("")
	require.NoError(t, err)
	assert.Equal(t, settings.EnvironmentProxySentinel, p.Value())
}

func TestManualMode(t *testing.T) {
	// setup
	p := NewProxy()
	p.Configure("http://localhost:8080")

	expected, err := url.Parse("http://localhost:8080")
	require.NoError(t, err)

	// http request
	r, err := http.NewRequest("GET", "http://alpha.kite.com/api/ping", nil)
	require.NoError(t, err)
	actual, _ := p.ForTransport(r)
	require.Equal(t, expected, actual)

	// https request
	r, err = http.NewRequest("GET", "https://alpha.kite.com/api/ping", nil)
	require.NoError(t, err)
	actual, _ = p.ForTransport(r)
	require.Equal(t, expected, actual)
}

func TestDirectMode(t *testing.T) {
	// setup
	p := NewProxy()
	p.Configure(settings.NoProxySentinel)

	// http request
	r, err := http.NewRequest("GET", "http://alpha.kite.com/api/ping", nil)
	require.NoError(t, err)
	proxyURL, _ := p.ForTransport(r)
	require.Nil(t, proxyURL)

	// https request
	r, err = http.NewRequest("GET", "https://alpha.kite.com/api/ping", nil)
	require.NoError(t, err)
	proxyURL, _ = p.ForTransport(r)
	require.Nil(t, proxyURL)
}

func TestEnvironmentMode(t *testing.T) {
	// setup
	p := NewProxy()
	p.Configure(settings.EnvironmentProxySentinel)
	defer localEnvOverride("http_proxy", "http://localhost:9090")()
	defer localEnvOverride("https_proxy", "http://localhost:9090")()

	expected, err := url.Parse("http://localhost:9090")
	require.NoError(t, err)

	// http request
	r, err := http.NewRequest("GET", "http://alpha.kite.com/api/ping", nil)
	require.NoError(t, err)
	actual, _ := p.ForTransport(r)
	require.Equal(t, expected, actual)

	// https request
	r, err = http.NewRequest("GET", "https://alpha.kite.com/api/ping", nil)
	require.NoError(t, err)
	actual, _ = p.ForTransport(r)
	require.Equal(t, expected, actual)
}

// sets a new value for an environment variable
// returns a function which sets the old value
func localEnvOverride(key, value string) func() {
	old := os.Getenv(key)
	_ = os.Setenv(key, value)

	return func() {
		_ = os.Setenv(key, old)
	}
}
