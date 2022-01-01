package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/health"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/lytics/slackhook"
)

var slack *slackhook.Client

func init() {
	slackURL := os.Getenv("SLACK_WEBHOOK_URL")
	if slackURL != "" {
		slack = slackhook.New(slackURL)
	}
}

// pollStatus groups together polling data about
// one endpoint.
type pollStatus struct {
	name     string
	endpoint string
	status   health.Status

	lastPolled time.Time
	lastError  error
	alerted    bool

	// timeErrored is the time at which the most recent error was first noticed
	timeErrored time.Time
	gracePeriod time.Duration
}

// pollTracker is a map of all tracked endpoints, and has
// a poll method that can be run to check all endpoints.
type pollTracker struct {
	mutex   sync.Mutex
	pollMap map[string]*pollStatus
	client  *http.Client
}

func newPollTracker(endpoints []endpoint) *pollTracker {
	pollMap := make(map[string]*pollStatus)
	for _, ep := range endpoints {
		log.Println("registering", ep.Name, "at", ep.Endpoint)
		pollMap[ep.Name] = &pollStatus{
			name:        ep.Name,
			endpoint:    ep.Endpoint,
			gracePeriod: time.Duration(ep.GracePeriod) * time.Second,
		}
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	return &pollTracker{
		pollMap: pollMap,
		client:  httpClient,
	}
}

// poll will poll all registered endpoints
func (p *pollTracker) poll() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for name, status := range p.pollMap {
		status.lastPolled = time.Now()
		endpoint := fmt.Sprintf("%s%s", strings.TrimRight(status.endpoint, "/"), health.Endpoint)
		resp, err := p.client.Get(endpoint)
		if err != nil {
			status.status = health.StatusUnreachable
			if status.lastError == nil {
				// If this is the first interval at which the error is noticed,
				// update the errored time
				status.timeErrored = time.Now()
			}
			status.lastError = err
		} else {
			var r health.Response
			err := json.NewDecoder(resp.Body).Decode(&r)
			if err != nil {
				status.status = health.StatusNone
			} else {
				status.status = r.StatusCode
			}
			status.lastError = nil
		}

		if status.status == health.StatusUnreachable {
			// Only alert if the grace period is over.
			// Once alerted, we won't alert again unless the endpoint has become
			// reachable again (see else clause below where alerted is set to false again).
			if !status.alerted && time.Now().Sub(status.timeErrored) > status.gracePeriod {
				rollbar.Critical(fmt.Errorf("%s unreachable: %s", status.name, status.lastError.Error()))
				if slack != nil {
					slack.Simple(fmt.Sprintf("%s unreachable: %s", status.name, status.lastError.Error()))
				}
				status.alerted = true
			}
		} else {
			// Since the endpoint became reachable again, we should alert if it goes down again
			status.alerted = false
		}
		log.Println("checked", name, "at", endpoint,
			"status:", status.status.String(), "lastErr:", status.lastError)
	}
}

// htmlStatus is a convenience struct to pass along template data
type htmlStatus struct {
	StatusCode health.Status
	Status     string
	Name       string
	Endpoint   string
	LastPolled string
	LastError  error
	Alerted    bool
}

// Sort by name

type byName []htmlStatus

func (b byName) Len() int           { return len(b) }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byName) Less(i, j int) bool { return b[i].Name < b[j].Name }

// ServeHTTP is a simple web handler to report the status of registered endpoints
func (p *pollTracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	timeFmt := "Jan 2, 2006 at 3:04pm"
	var tmplData []htmlStatus
	for _, status := range p.pollMap {
		tmplData = append(tmplData, htmlStatus{
			StatusCode: status.status,
			Status:     status.status.String(),
			Name:       status.name,
			Endpoint:   status.endpoint,
			LastPolled: status.lastPolled.Format(timeFmt),
			LastError:  status.lastError,
			Alerted:    status.alerted,
		})
	}

	data, err := Asset("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("status").Parse(string(data))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sort.Sort(byName(tmplData))
	err = tmpl.Execute(w, tmplData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
