package auth

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

// Get performs an authenticated GET request with the provided path relative to the target
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	req, err := c.makeRequestLocked("GET", path, "", nil, true)
	if err != nil {
		return nil, err
	}

	return c.doHTTPLocked(ctx, req)
}

// getNoHMAC performs an session-only GET request with the provided path relative to the target
func (c *Client) getNoHMAC(ctx context.Context, path string) (*http.Response, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.getNoHMACLocked(ctx, path)
}

// getNoHMACLocked performs an session-only GET request with the provided path relative to the target
func (c *Client) getNoHMACLocked(ctx context.Context, path string) (*http.Response, error) {
	req, err := c.makeRequestLocked("GET", path, "", nil, false)
	if err != nil {
		return nil, err
	}

	return c.doHTTPLocked(ctx, req)
}

// Post performs an authenticated POST request with the provided path relative to the target
func (c *Client) Post(ctx context.Context, path, contentType string, body io.Reader) (*http.Response, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	req, err := c.makeRequestLocked("POST", path, contentType, body, true)
	if err != nil {
		return nil, err
	}

	return c.doHTTPLocked(ctx, req)
}

// postForm performs an authenticated POST request with the provided path relative to the target
func (c *Client) postForm(ctx context.Context, url string, data url.Values) (*http.Response, error) {
	return c.Post(ctx, url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// NewRequest parallels http.NewRequest, but takes in a path. The proxy will fill in the full path relative to the target.
func (c *Client) NewRequest(method, path, contentType string, body io.Reader) (*http.Request, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.makeRequestLocked(method, path, contentType, body, true)
}

// Do performs the provided HTTP request, it adds the HMAC headers
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.doHTTPLocked(ctx, c.addAuth(req, true))
}

// doHTTPLocked performs the HTTP request, it does not change the request's properties.
func (c *Client) doHTTPLocked(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c.client == nil {
		return nil, errors.New("http client not set")
	}

	resp, err := c.client.Do(req.WithContext(ctx))
	if err == nil {
		c.token.UpdateFromHeader(resp.Header)
	}
	return c.wrap(resp), err
}

// --

// makeRequestLocked returns a pointer to a new http request object
func (c *Client) makeRequestLocked(method, path string, contentType string, body io.Reader, hmac bool) (*http.Request, error) {
	if c.target == nil {
		return nil, errors.Errorf("target not defined")
	}

	endpoint, err := c.target.Parse(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, endpoint.String(), body)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return c.addAuth(req, hmac), nil
}

// addAuth should be idempotent
func (c *Client) addAuth(req *http.Request, hmac bool) *http.Request {
	req.Header.Set(community.MachineHeader, c.machineID)

	if hmac {
		c.token.AddToHeader(req.Header)
	}

	return req
}

// wrap wraps a response to update the number of open connections
func (c *Client) wrap(resp *http.Response) *http.Response {
	if resp == nil {
		return nil
	}

	atomic.AddInt64(&c.openedConnections, 1)
	resp.Body = &bodyWrapper{body: resp.Body, client: c}
	return resp
}

// bodyWrapper is a simple wrapper to decrement to number of open connections on Close()
type bodyWrapper struct {
	body   io.ReadCloser
	client *Client
}

func (b *bodyWrapper) Read(buf []byte) (int, error) {
	return b.body.Read(buf)
}

func (b *bodyWrapper) Close() error {
	atomic.AddInt64(&b.client.closedConnections, 1)
	return b.body.Close()
}
