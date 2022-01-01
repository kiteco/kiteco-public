package localcode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/gorilla/mux"
)

type requestClient struct {
	workers *workerGroup
	client  *http.Client
}

func newRequestClient(workers *workerGroup) *requestClient {
	return &requestClient{
		workers: workers,
		client: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second, // using default via http.DefaultTransport
				}).Dial,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				ResponseHeaderTimeout: 5 * time.Minute,
			},
		},
	}
}

func (a *requestClient) requestArtifact(input userMachineFile) error {
	var req artifactRequests
	req.Requests = append(req.Requests, &artifactRequest{
		UserID:   input.UserID,
		Machine:  input.Machine,
		Filename: input.Filename,
	})

	i := a.workers.shard(input.UserID)

	buf, err := json.Marshal(req)
	if err != nil {
		log.Println("localcode.requestClient: error marshalling artifact requests:", err)
		return err
	}

	updateURL, err := a.workers.url(i, "/artifacts/submit-requests")
	if err != nil {
		log.Printf("localcode.requestClient: error creating submit requests url for shard %d: %s", i, err)
		return err
	}

	resp, err := a.client.Post(updateURL.String(), "application/json", bytes.NewReader(buf))
	if err != nil {
		log.Println("localcode.requestClient: error sending artifact request POST:", err)
		return err
	}
	defer resp.Body.Close()

	return nil
}

// --

type requestServer struct {
	m        sync.Mutex
	requests map[userMachine]*userRequest
}

func newRequestServer() *requestServer {
	return &requestServer{
		requests: make(map[userMachine]*userRequest),
	}
}

func (a *requestServer) setupRoutes(mux *mux.Router) {
	mux.HandleFunc("/artifacts/submit-requests", a.handleRequests).Methods("POST")
	mux.HandleFunc("/queue", a.handleQueue).Methods("GET")
}

func (a *requestServer) handleQueue(w http.ResponseWriter, r *http.Request) {
	a.m.Lock()
	defer a.m.Unlock()

	var requests []*userRequest
	for _, request := range a.requests {
		requests = append(requests, request)
	}

	sort.Sort(requestQueue(requests))

	tabw := tabwriter.NewWriter(w, 10, 10, 10, ' ', 0)
	defer tabw.Flush()

	tabw.Write([]byte("#\tselected\tin queue\tuid\tmachine\t# files\n"))
	for idx, req := range requests {
		var selected string
		if !req.selectedAt.IsZero() {
			selected = time.Since(req.selectedAt).String()
		}
		tabw.Write([]byte(fmt.Sprintf("%d\t%s\t%s\t%d\t%s\t%d\n",
			idx+1, selected, time.Since(req.firstRequested), req.UserID, req.Machine, len(req.Files))))
	}
}

type artifactRequests struct {
	Requests []*artifactRequest
}

func (a *requestServer) handleRequests(w http.ResponseWriter, r *http.Request) {
	defer handleRequestsDuration.DeferRecord(time.Now())

	var requests artifactRequests
	err := json.NewDecoder(r.Body).Decode(&requests)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	a.m.Lock()
	defer a.m.Unlock()
	for _, request := range requests.Requests {
		// Note this is INDEPENDENT of client-side tracking of firstRequsted
		if ur, ok := a.requests[request.userMachine()]; !ok {
			log.Printf("localcode.Worker (%d, %s): received request for %s", request.UserID, request.Machine, request.Filename)
			a.requests[request.userMachine()] = &userRequest{
				UserID:  request.UserID,
				Machine: request.Machine,
				Files: map[string]time.Time{
					request.Filename: time.Now(),
				},
				firstRequested: time.Now(),
			}
		} else {
			ur.Files[request.Filename] = time.Now()
		}
	}
}

type requestQueue []*userRequest

func (a requestQueue) Len() int      { return len(a) }
func (a requestQueue) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a requestQueue) Less(i, j int) bool {
	return a[i].firstRequested.Before(a[j].firstRequested)
}

func (a *requestServer) next() (*userRequest, bool) {
	a.m.Lock()
	defer a.m.Unlock()

	queueSizeCount.Record(int64(len(a.requests)))

	var requests []*userRequest
	for _, request := range a.requests {
		if request.selectedAt.IsZero() {
			requests = append(requests, request)
		}
	}

	if len(requests) == 0 {
		return nil, false
	}

	sort.Sort(requestQueue(requests))

	selected := requests[0]
	selected.selectedAt = time.Now()

	requestToSelectedDuration.RecordDuration(time.Since(selected.firstRequested))

	return selected, true
}

func (a *requestServer) completed(req *userRequest) {
	a.m.Lock()
	defer a.m.Unlock()
	delete(a.requests, req.userMachine())
	requestToCompletedDuration.RecordDuration(time.Since(req.firstRequested))
}

// --

type artifactRequest struct {
	UserID   int64
	Machine  string
	Filename string

	// server-side tracking
	selectedAt time.Time

	// client & server, independently
	firstRequested time.Time
}

func (u *artifactRequest) userMachine() userMachine {
	return userMachine{UserID: u.UserID, Machine: u.Machine}
}

func (u *artifactRequest) userMachineFile() userMachineFile {
	return userMachineFile{UserID: u.UserID, Machine: u.Machine, Filename: u.Filename}
}

type userRequest struct {
	UserID  int64
	Machine string
	Files   map[string]time.Time

	// server-side tracking
	selectedAt     time.Time
	firstRequested time.Time
}

func (u *userRequest) userMachine() userMachine {
	return userMachine{UserID: u.UserID, Machine: u.Machine}
}

func (u *userRequest) latestFile() string {
	var latest string
	var latestTs time.Time

	for fn, ts := range u.Files {
		if ts.After(latestTs) {
			latestTs = ts
			latest = fn
		}
	}

	return latest
}
