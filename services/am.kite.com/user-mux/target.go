package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

var (
	targetPort = envutil.GetenvDefault("USER_NODE_SERVICE_PORT", "9090")
	debugPort  = envutil.GetenvDefault("USER_NODE_SERVICE_PORT_DEBUG", "9091")
)

const (
	healthCheckTolerance = 2
	healthCheckInterval  = 10 * time.Second
)

type proxyTarget struct {
	ip     string
	target *url.URL
	debug  *url.URL

	rw             sync.RWMutex
	health         bool
	healthyCount   int
	unhealthyCount int

	once              sync.Once
	healthContext     context.Context
	cancelHealthCheck context.CancelFunc

	httpproxy http.Handler

	client  *http.Client
	bgCount int64

	pathm sync.RWMutex
	paths map[string]int
}

func newProxyTarget(ip string) (*proxyTarget, error) {
	u := fmt.Sprintf("http://%s:%s/", ip, targetPort)
	target, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	u = fmt.Sprintf("http://%s:%s/", ip, debugPort)
	debug, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second, // using default via http.DefaultTransport
			}).Dial,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   30,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 5 * time.Minute,
		},
	}

	pt := &proxyTarget{
		ip:                ip,
		target:            target,
		debug:             debug,
		healthContext:     ctx,
		cancelHealthCheck: cancel,
		httpproxy:         httputil.NewSingleHostReverseProxy(target),
		client:            client,
	}

	if proxy, ok := pt.httpproxy.(*httputil.ReverseProxy); ok {
		transport := &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   5 * time.Second,  // 5 second connection timeout
				KeepAlive: 30 * time.Second, // using default via http.DefaultTransport
			}).Dial,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 5 * time.Minute,
		}
		proxy.Transport = newErrorHandlingRoundTripper(transport)
	}

	return pt, nil
}

func (p *proxyTarget) addBadGatewayPath(path string) {
	p.pathm.Lock()
	defer p.pathm.Unlock()
	p.paths[path]++
}

func (p *proxyTarget) resetBadGatewayPaths() {
	p.pathm.Lock()
	defer p.pathm.Unlock()
	p.paths = make(map[string]int)
}

func (p *proxyTarget) getBadGatewayPaths() map[string]int {
	p.pathm.RLock()
	defer p.pathm.RUnlock()

	ret := make(map[string]int)
	for k, v := range p.paths {
		ret[k] = v
	}
	return ret
}

func (p *proxyTarget) isHealthy() bool {
	p.rw.RLock()
	defer p.rw.RUnlock()
	return p.health
}

func (p *proxyTarget) setHealth(err error) {
	p.rw.Lock()
	defer p.rw.Unlock()

	if err == nil {
		p.healthyCount++
		p.unhealthyCount = 0
	} else {
		p.healthyCount = 0
		p.unhealthyCount++
	}

	if !p.health && p.healthyCount >= healthCheckTolerance {
		p.health = true
		p.healthyCount = 0
		logger.Info("target", p.target, "became healthy :)")
	}
	if p.health && p.unhealthyCount >= healthCheckTolerance {
		p.health = false
		p.unhealthyCount = 0
		logger.Info("target", p.target, "became unhealthy :(")
	}

	logger.Infof("set health called on %s: health: %t, healthyCount: %d, unhealthyCount: %d : %v",
		p.target.String(), p.health, p.healthyCount, p.unhealthyCount, err)
}

func (p *proxyTarget) stopHealthCheckLoop() {
	p.cancelHealthCheck()
}

func (p *proxyTarget) startHealthCheckLoop() {
	checkLoop := func() {
		logger.Info("starting health check for target", p.target.String())

		err := p.healthCheck()
		if err != nil {
			log.Printf("health check for %s failed: %s", p.target.String(), err)
		}

		ticker := time.NewTicker(healthCheckInterval)

		for {
			select {
			case <-ticker.C:
				err := p.healthCheck()
				if err != nil {
					log.Printf("health check for %s failed: %s", p.target.String(), err)
				}
			case <-p.healthContext.Done():
				logger.Info("stopping health check for target", p.target.String())
				return
			}
		}
	}

	// Make sure this only gets started once
	p.once.Do(func() {
		go checkLoop()
	})
}

func (p *proxyTarget) healthCheck() (err error) {
	defer func() {
		p.setHealth(err)
	}()

	var readyURL *url.URL
	readyURL, err = p.debug.Parse("/ready")
	if err != nil {
		return
	}

	var resp *http.Response
	resp, err = p.client.Get(readyURL.String())
	if err != nil {
		return
	}

	defer resp.Body.Close()

	_, err = io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("error consuming response from /ready: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("/ready returned response code: %d", resp.StatusCode)
	}

	return
}

// --

type errorHandlingRoundTripper struct {
	rt http.RoundTripper
}

func newErrorHandlingRoundTripper(rt http.RoundTripper) *errorHandlingRoundTripper {
	return &errorHandlingRoundTripper{rt: rt}
}

func (rt *errorHandlingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.rt == nil {
		panic("must set errorHandlingRoundTripper's tr")
	}

	res, err := rt.rt.RoundTrip(req)

	// Catch context canceled and unexpected EOF errors. These can be caused by the
	// client unexpectedly ending the connection. This should *NOT* result in a 502 Bad Gateway.
	// Instead, we respond with 408 Request Timeout.
	if err == context.Canceled || err == io.ErrUnexpectedEOF {
		return &http.Response{
			Status:     "408 Request Timeout",
			StatusCode: http.StatusRequestTimeout,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       ioutil.NopCloser(&bytes.Buffer{}),
			Request:    req,
			Header:     make(http.Header, 0),
		}, nil
	}

	// Catch connection reset by peer, return 500
	switch e := err.(type) {
	case *net.OpError:
		if strings.Contains(e.Err.Error(), "connection reset") {
			return &http.Response{
				Status:     "500 Internal Server Error",
				StatusCode: http.StatusInternalServerError,
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Body:       ioutil.NopCloser(&bytes.Buffer{}),
				Request:    req,
				Header:     make(http.Header, 0),
			}, nil
		}
	}

	if err != nil {
		rollbar.Warning(fmt.Errorf("uncaught roundtrip error in user-mux"), err)
	}

	return res, err
}
