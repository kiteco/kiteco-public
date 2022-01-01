package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

const (
	selectedSiteKey       = "Kite-SelectedSite"
	targetRefreshInterval = 5 * time.Minute
)

type proxyHandlers struct {
	rw      sync.RWMutex
	targets []*proxyTarget
	ips     []string
	node    string
}

func (p *proxyHandlers) handleHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer handlerDuration.DeferRecord(start)

	target := p.selectTarget(w, r)
	target.httpproxy.ServeHTTP(w, r)

	// Record status code metrics
	// TODO(dane) - may be worthwhile to subdivide the metrics based on node delegated to
	if nw, ok := w.(negroni.ResponseWriter); ok {
		breakdown := responseCodes
		if r.URL.Path == "/http/events" {
			breakdown = eventResponseCodes
		}

		s := nw.Status()
		switch {
		case s >= 100 && s <= 199:
			breakdown.Hit("1xx")
		case s >= 200 && s <= 299:
			breakdown.Hit("2xx")
		case s >= 300 && s <= 399:
			breakdown.Hit("3xx")
		case s >= 400 && s <= 499:
			breakdown.Hit("4xx")
		case s >= 500 && s <= 599:
			if s == http.StatusBadGateway {
				breakdown.Hit("502")
				badGatewayCounter.Add(1)
				badGatewayDuration.RecordDuration(time.Since(start))

				// Update target-specific count
				atomic.AddInt64(&target.bgCount, int64(1))
				target.addBadGatewayPath(r.URL.Path)
			} else {
				breakdown.Hit("5xx")
			}
		default:
			breakdown.Hit("other")
		}
	}
}

func (p *proxyHandlers) handleReady(w http.ResponseWriter, r *http.Request) {
	p.rw.RLock()
	defer p.rw.RUnlock()

	var hasHealthy bool
	for _, target := range p.targets {
		if target.isHealthy() {
			hasHealthy = true
		}
	}
	if !hasHealthy {
		http.Error(w, fmt.Sprintf("%s is not ready", p.node), http.StatusServiceUnavailable)
		return
	}
}

func (p *proxyHandlers) selectTarget(w http.ResponseWriter, r *http.Request) *proxyTarget {
	p.rw.RLock()
	defer p.rw.RUnlock()

	var ipString string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		ipString = "error getting"
	} else {
		for _, ip := range addrs {
			ipString += ip.String() + "\n"
		}
	}

	var found bool
	var target *proxyTarget

	target, found = selectRandomTarget(p.targets)

	if !found {
		logger.Info("no target selected, using 503 handler")
		return serviceUnavailableTarget
	}

	w.Header().Set(selectedSiteKey, target.target.Host)

	return target
}

func (p *proxyHandlers) refreshLoop() {
	for range time.Tick(targetRefreshInterval) {
		err := p.refreshTargets()
		if err != nil {
			logger.Error("error refreshing targets:", err)
		}
	}
}

func (p *proxyHandlers) refreshTargets() error {
	logger.Infof("refreshing targets for %s", p.node)
	ips, err := net.LookupHost(p.node)
	if err != nil {
		return err
	}

	sort.Strings(p.ips)
	sort.Strings(ips)

	// If nothing has changed, don't change anything for this node..
	if reflect.DeepEqual(p.ips, ips) {
		return nil
	}

	logger.Infof("detected changes to %s targets...", p.node)
	logger.Info("previous targets:", p.ips)
	logger.Info("new targets:", ips)

	var newTargets []*proxyTarget
	for _, ip := range ips {
		t, err := newProxyTarget(ip)
		if err != nil {
			return err
		}
		newTargets = append(newTargets, t)
	}

	// Swap targets, carrying over health status of any overlapping nodes

	wasHealthy := make(map[string]bool)
	for _, t := range p.targets {
		t.stopHealthCheckLoop()
		if t.isHealthy() {
			wasHealthy[t.target.String()] = true
		}
	}

	for _, t := range newTargets {
		t.startHealthCheckLoop()
		if _, ok := wasHealthy[t.target.String()]; ok {
			t.health = true
		}
	}

	p.rw.Lock()
	defer p.rw.Unlock()

	p.targets = newTargets
	p.ips = ips

	return nil
}

var (
	badGatewayInterval  = time.Minute
	badGatewayThreshold = int64(5)
)

func (p *proxyHandlers) watchBadGateways() {
	t := time.NewTicker(badGatewayInterval)
	for range t.C {
		// Wrap in func for easier RLock/RUnlock semantics
		func() {
			p.rw.RLock()
			defer p.rw.RUnlock()

			// Iterate over *-node targets, determine whether they crossed the threshold
			for _, target := range p.targets {
				checkBadGateway(target)
			}
		}()
	}
}

func checkBadGateway(target *proxyTarget) {
	count := atomic.LoadInt64(&target.bgCount)
	if count > badGatewayThreshold {
		paths := target.getBadGatewayPaths()
		rollbar.Warning(fmt.Errorf("bad gateways exceeded threshold %d in 1 minute", badGatewayThreshold),
			count, target.ip, region, paths)
	}
	atomic.StoreInt64(&target.bgCount, int64(0))
	target.resetBadGatewayPaths()
}

func (p *proxyHandlers) handleGetBadGateway(w http.ResponseWriter, r *http.Request) {
	p.rw.RLock()
	defer p.rw.RUnlock()

	nodeToBG := make(map[string]bool)
	for _, target := range p.targets {
		nodeToBG[target.ip] = atomic.LoadInt64(&target.bgCount) > badGatewayThreshold
	}

	buf, err := json.Marshal(&nodeToBG)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (p *proxyHandlers) handleGetBadGatewayPaths(w http.ResponseWriter, r *http.Request) {
	p.rw.RLock()
	defer p.rw.RUnlock()

	nodeToBG := make(map[string]map[string]int)
	for _, target := range p.targets {
		nodeToBG[target.ip] = target.getBadGatewayPaths()
	}

	buf, err := json.Marshal(&nodeToBG)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

// --

type serviceUnavailableHandler struct{}

func (s serviceUnavailableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO(tarak): ServiceUnavailable or BadGateway?
	http.Error(w, "service unavailable", http.StatusServiceUnavailable)
}

var serviceUnavailableTarget = &proxyTarget{
	httpproxy: serviceUnavailableHandler{},
}
