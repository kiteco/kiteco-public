package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type httpRequestResponse struct {
	Timestamp   string            `json:"timestamp"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	QueryParams map[string]string `json:"query"`
	Body        string            `json:"body"`

	ResponseStatus int    `json:"response_status"`
	ResponseBody   string `json:"response_body"`
}

type callHistory struct {
	mu       sync.Mutex
	requests []httpRequestResponse

	// contains the number of currently processed requests
	wg sync.WaitGroup
}

func newCallHistory() *callHistory {
	return &callHistory{
		requests: []httpRequestResponse{},
	}
}

// Name implements component Core
func (h *callHistory) Name() string {
	return "call-history"
}

// RegisterHandlers implements component RequestHandlers
func (h *callHistory) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/testapi/request-history", h.handleRequestHistory).Methods("GET")
	mux.HandleFunc("/testapi/request-history/reset", h.handleRequestHistoryReset).Methods("POST", "GET")
}

func (h *callHistory) recordCall(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if strings.HasPrefix(r.URL.Path, "/testapi/") {
		next.ServeHTTP(rw, r)
		return
	}

	h.wg.Add(1)
	defer h.wg.Done()

	start := time.Now().Format(time.RFC3339)

	var body []byte
	var err error
	if r.GetBody != nil {
		reader, err := r.GetBody()
		if err != nil {
			log.Printf("error while retrieving request body: %v\n", err)
		}
		if body, err = ioutil.ReadAll(reader); err != nil {
			log.Printf("error while reading request body: %v\n", err)
		}
	} else if r.Body != nil {
		if body, err = ioutil.ReadAll(r.Body); err != nil {
			log.Printf("error while reading request body: %v\n", err)
		}

		// replace request with a new request which contains a body which is not yet read
		if r, err = http.NewRequest(r.Method, r.URL.String(), bytes.NewReader(body)); err != nil {
			log.Printf("error while wrapping request: %v\n", err)
			return
		}
	}

	resp := newResponseRecorder(rw)
	next.ServeHTTP(resp, r)

	if resp.status == 0 {
		log.Printf("no response was written for %s\n", r.URL.Path)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// kited is never using multiple values for a query param
	singleQuery := make(map[string]string)
	for k, v := range r.URL.Query() {
		singleQuery[k] = v[0]
	}

	h.requests = append(h.requests, httpRequestResponse{
		Timestamp:      start,
		Method:         r.Method,
		Path:           r.URL.Path,
		QueryParams:    singleQuery,
		Body:           string(body),
		ResponseStatus: resp.status,
		ResponseBody:   string(resp.body),
	})
}

// handleRequestHistory response with the current HTTP request/response history as JSON. It waits for active requests to finish before the history is returned.
func (h *callHistory) handleRequestHistory(w http.ResponseWriter, request *http.Request) {
	// wait for active requests to finish
	h.wg.Wait()

	h.mu.Lock()
	defer h.mu.Unlock()

	w.Header().Add("Content-Type", "application/json")

	r, err := json.Marshal(h.requests)
	if err != nil {
		http.Error(w, "error marshalling request history", http.StatusInternalServerError)
		return
	}

	w.Write(r)
}

func (h *callHistory) handleRequestHistoryReset(writer http.ResponseWriter, request *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.requests = []httpRequestResponse{}
	writer.Write([]byte("ok"))
}
