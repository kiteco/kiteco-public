package proxy

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
)

var debug bool

// Proxy encapsulates a proxy configuration
type Proxy struct {
	lock     sync.RWMutex
	provider provider
}

// Global is a Proxy, that defaults to detecting the proxy from the environment.
// This provides a safe fallback for settings of older clients.
var Global = NewProxy()

// NewProxy creates a Proxy that defaults to detecting the proxy from the environment.
func NewProxy() *Proxy {
	p := new(Proxy)
	p.Configure("")
	return p
}

// ForTransport is a function which can be used as a proxy provider for http.Transport
func (p *Proxy) ForTransport(r *http.Request) (*url.URL, error) {
	return p.proxyForURL(r.URL)
}

// IsProxied returns whether a proxy would be used for the given URL
func (p *Proxy) IsProxied(u string) (bool, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return false, err
	}
	proxyURL, err := p.proxyForURL(parsedURL)
	return proxyURL != nil, err
}

// Value returns the currently set Proxy value
// It returns either direct, environment, or a manually set proxy URL
func (p *Proxy) Value() string {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.provider.String()
}

// Configure configures the Proxy.
func (p *Proxy) Configure(value string) error {
	switch value {
	case "", settings.EnvironmentProxySentinel:
		p.lock.Lock()
		defer p.lock.Unlock()

		p.provider = newEnvProvider()

	case settings.NoProxySentinel:
		p.lock.Lock()
		defer p.lock.Unlock()

		p.provider = directProvider{}

	default:
		proxyURL, err := url.Parse(value)
		if err != nil {
			return err
		}

		p.lock.Lock()
		defer p.lock.Unlock()

		p.provider = manualProvider{proxyURL}
	}

	return nil
}

// DefaultTransport returns a new instance of http.Transport.
// It's' equivalent to http.DefaultTransport, but uses the Proxy configuration.
func (p *Proxy) DefaultTransport() *http.Transport {
	return &http.Transport{
		// modified proeprties
		Proxy: Global.ForTransport,
		// properties copied from http.DefaultTransport
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func (p *Proxy) proxyForURL(url *url.URL) (*url.URL, error) {
	if host := url.Hostname(); host == "127.0.0.1" || host == "::1" || host == "localhost" {
		return nil, nil
	}

	p.lock.RLock()
	defer p.lock.RUnlock()

	if debug {
		start := time.Now()
		defer func() {
			log.Printf("proxyForURL() duration for %s: %s", url.String(), time.Now().Sub(start).String())
		}()
	}

	return p.provider.ProxyForURL(url)
}
