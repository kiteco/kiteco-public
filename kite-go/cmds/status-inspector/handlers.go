//go:generate go-bindata -debug -o bindata.go templates

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/deployments"
	"github.com/kiteco/kiteco/kite-golib/status"
)

type handlers struct {
	dm         sync.Mutex
	production []deployments.Region
	staging    []deployments.Region

	m              sync.Mutex
	prodstatus     *status.Status
	stagingstatus  *status.Status
	clientstatus   *status.Status
	stagingclients *status.Status
}

func newHandlers() *handlers {
	production, err := deployments.Production()
	if err != nil {
		log.Println("error finding production deployments:", err)
	}

	staging, err := deployments.Staging()
	if err != nil {
		log.Println("error finding staging deployments:", err)
	}

	h := &handlers{
		production: production,
		staging:    staging,
	}

	go h.poll()
	go h.trackDeployments()

	return h
}

func (h *handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	var release string
	var s *status.Status

	h.m.Lock()
	s = h.prodstatus
	h.m.Unlock()

	h.dm.Lock()
	if len(h.production) > 0 {
		release = h.production[0].Release
	}
	h.dm.Unlock()

	status.Render(release, s, w, r)
}

func (h *handlers) handleStaging(w http.ResponseWriter, r *http.Request) {
	var release string
	var s *status.Status

	h.m.Lock()
	s = h.stagingstatus
	h.m.Unlock()

	h.dm.Lock()
	if len(h.staging) > 0 {
		release = h.staging[0].Release
	}
	h.dm.Unlock()

	status.Render(release+" (staging)", s, w, r)
}

func (h *handlers) handleClients(w http.ResponseWriter, r *http.Request) {
	var s *status.Status

	h.m.Lock()
	s = h.clientstatus
	h.m.Unlock()

	// HACK: filter out any non-client sections
	if s != nil {
		for name := range s.Sections {
			if !strings.HasPrefix(name, "client/") {
				delete(s.Sections, name)
			}
		}
	}

	status.Render("clients", s, w, r)
}

func (h *handlers) handleStagingClients(w http.ResponseWriter, r *http.Request) {
	var s *status.Status

	h.m.Lock()
	s = h.stagingclients
	h.m.Unlock()

	// HACK: filter out any non-client sections
	if s != nil {
		for name := range s.Sections {
			if !strings.HasPrefix(name, "client/") {
				delete(s.Sections, name)
			}
		}
	}

	status.Render("clients (staging)", s, w, r)
}

func (h *handlers) trackDeployments() {
	for range time.NewTicker(5 * time.Minute).C {
		production, err := deployments.Production()
		if err != nil {
			log.Println("error finding production deployments:", err)
		}
		staging, err := deployments.Staging()
		if err != nil {
			log.Println("error finding staging deployments:", err)
		}

		log.Println("updating deployments")
		h.dm.Lock()
		h.production = production
		h.staging = staging
		h.dm.Unlock()
	}
}

func (h *handlers) poll() {
	update := func() {
		fmt.Print(".")
		h.dm.Lock()
		p, s := h.production, h.staging
		h.dm.Unlock()

		var stagingstatus *status.Status

		prodstatus := h.update(p)

		// If prod and staging are pointing to the same release, don't re-poll
		if len(p) > 0 && len(s) > 0 && p[0].Release == s[0].Release {
			stagingstatus = prodstatus
		} else {
			stagingstatus = h.update(s)
		}

		clientstatus := h.aggregateClients(p)
		stagingclients := h.aggregateClients(s)

		h.m.Lock()
		defer h.m.Unlock()
		h.prodstatus = prodstatus
		h.stagingstatus = stagingstatus
		h.clientstatus = clientstatus
		h.stagingclients = stagingclients
	}

	update()
	for range time.NewTicker(time.Minute).C {
		update()
	}
}

func (h *handlers) update(deployments []deployments.Region) *status.Status {
	var m sync.Mutex
	var statuses []*status.Status

	for _, deployment := range deployments {
		var ips []string
		ips = append(ips, deployment.UsernodeIPs...)
		ips = append(ips, deployment.LocalCodeWorkerIPs...)
		ips = append(ips, deployment.UserMuxIPs...)

		var wg sync.WaitGroup
		wg.Add(len(ips))

		var missing int
		for _, serverIP := range ips {
			go func(serverIP string) {
				defer wg.Done()
				log.Println("fetching", deployment.Region, serverIP)
				server, err := url.Parse(fmt.Sprintf("http://%s:9091/", serverIP))
				if err != nil {
					log.Println("error parsing url for", serverIP, "err:", err)
					missing++
					return
				}
				s, err := status.Poll(server)
				if err != nil {
					log.Println(err)
					missing++
					return
				}

				m.Lock()
				defer m.Unlock()
				statuses = append(statuses, s)
			}(serverIP)
		}

		wg.Wait()

		if missing > 0 {
			section := status.NewSection("!! missing statuses")
			counter := section.Counter(deployment.Region)
			counter.Set(int64(missing))
		}

		log.Println("finished with", deployment.Region)
	}

	log.Println("aggregating", len(statuses), "statuses")
	statuses = append(statuses, status.Get())
	return status.Aggregate(statuses)
}

func (h *handlers) aggregateClients(deployments []deployments.Region) *status.Status {
	type tmp struct {
		UID    int64
		Email  string
		Status *status.Status
	}

	var statuses []*status.Status
	for _, deployment := range deployments {
		var ips []string
		ips = append(ips, deployment.UsernodeIPs...)

		for _, serverIP := range ips {
			server, err := url.Parse(fmt.Sprintf("http://%s:9091/metrics/statuses", serverIP))
			if err != nil {
				log.Println("error parsing url for", serverIP, "err:", err)
				return nil
			}
			resp, err := http.Get(server.String())
			if err != nil {
				log.Println(err)
				return nil
			}

			dec := json.NewDecoder(resp.Body)
			for {
				var t tmp
				err := dec.Decode(&t)
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					break
				}
				if err != nil {
					log.Println(err)
					break
				}
				statuses = append(statuses, t.Status)
			}

			err = resp.Body.Close()
			if err != nil {
				log.Println(err)
			}
		}
	}

	return status.Aggregate(statuses)
}
