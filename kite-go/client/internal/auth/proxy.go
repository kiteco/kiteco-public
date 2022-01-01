package auth

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/proxy"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

var isTravis = os.Getenv("CI") == "true"

var prodTargets = []string{
	fmt.Sprintf("https://%s/", domains.Alpha),
}

type responseWriterRecorder struct {
	http.ResponseWriter
	bytes  int
	status int
}

func (r *responseWriterRecorder) Write(buf []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(buf)
	r.bytes += n
	return n, err
}

func (r *responseWriterRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// ServeHTTP proxies requests to the backend server.
// It persists cookies between requests and between restarts.
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := c.Target()
	client := c.httpClient()
	if target == nil || client == nil {
		log.Println("proxy target not set, ignoring request for", r.URL.Path)
		http.Error(w, "proxy target not set", http.StatusInternalServerError)
		return
	}

	// Set the machine header
	r.Header.Set(community.MachineHeader, c.machineID)

	// Remove any cookies.
	stripCookies(r.Header)

	// Set our own cookies
	prevCookies := client.Jar.Cookies(target)
	for _, c := range prevCookies {
		r.AddCookie(c)
	}

	// Add any hmac info we might have
	c.token.AddToHeader(r.Header)

	// handle proxy request
	r.Host = target.Host
	rw := &responseWriterRecorder{ResponseWriter: w}

	proxy := c.getProxy() //retrieve proxy with lock to be safe against races by SetTarget()/UnsetTarget()
	proxy.ServeHTTP(rw, r)

	if c.metrics != nil {
		if region := w.Header().Get("Kite-Region"); region != "" {
			c.metrics.SetRegion(region)
		}
	}

	// Update hmac tokens from the response
	c.token.UpdateFromHeader(w.Header())

	// Update cookies from the response
	newCookies := extractCookies(w.Header())
	if len(newCookies) > 0 {
		client.Jar.SetCookies(target, newCookies)
		err := c.saveAuth()
		if err != nil {
			log.Println(err)
		}
	}
}

//Target returns the currently used URL. It's returning nil if there's no active backend
func (c *Client) Target() *url.URL {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.target
}

// UnsetTarget will reset the target, preventing requests from being forwarded
// by the proxy
func (c *Client) UnsetTarget() {
	log.Printf("Unsetting target URL")

	c.mu.Lock()
	defer c.mu.Unlock()

	c.unsetTargetInternal()
}

// SetTarget updates the URL of the backend server.
// If the target should be removed then use UnsetTarget instead of this method.
func (c *Client) SetTarget(target *url.URL) {
	c.mu.Lock()
	defer c.mu.Unlock()

	//release resources
	c.unsetTargetInternal()

	//setup new target, proxy and http client
	transport := &http.Transport{
		// patched properties
		Proxy: proxy.Global.ForTransport,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.debug,
		},
		DialContext: (&net.Dialer{
			// this is our own value
			Timeout:   connectionTimeout,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		// copy of Go's default settings (see http.DefaultTransport)
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	c.target = target
	c.proxy = httputil.NewSingleHostReverseProxy(c.target)
	c.proxy.Transport = transport

	// if cookiejar.New failed then we will just pass a nil jar to http.Client
	jar := newCookieJar()
	if jar == nil {
		rollbar.Error(errors.New("error creating cookiejar"))
	}

	c.client = &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	if err := c.restoreAuth(); err != nil {
		log.Printf("error in restoreAuth() while updating new target URL: %s, %s", err.Error(), c.filepath)
	}

	// This is a terrible spot to put this,
	// but necessary to handle unauthenticated users.
	// Once we refactor auth, we should try to move this.
	c.resetRemoteListenerLocked()
}

// unsetTargetInternal releases resources used by proxy and http
// it expects that any necessary mutex has been acquired before it's called
func (c *Client) unsetTargetInternal() {
	//release resources
	if c.proxy != nil {
		c.proxy.Transport.(*http.Transport).CloseIdleConnections()
	}
	c.proxy = nil

	if c.client != nil {
		c.client.Transport.(*http.Transport).CloseIdleConnections()
	}
	c.client = nil

	c.target = nil
}

// HasAuthCookie returns whether we are authenticated
func (c *Client) HasAuthCookie() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil && c.client.Jar != nil {
		for _, cookie := range c.client.Jar.Cookies(c.target) {
			if cookie.Name == sessionKey {
				return true
			}
		}
	}
	return false
}

// HasProductionTarget returns whether the target is a production backend
func (c *Client) HasProductionTarget() bool {
	target := c.Target()
	if target == nil {
		return false
	}
	for _, t := range prodTargets {
		if target.String() == t {
			return true
		}
	}
	return false
}

// --
func stripCookies(header http.Header) {
	key := http.CanonicalHeaderKey("cookie")
	if header.Get(key) != "" {
		header.Del(key)
	}
}

func extractCookies(header http.Header) []*http.Cookie {
	response := http.Response{Header: header}
	return response.Cookies()
}
