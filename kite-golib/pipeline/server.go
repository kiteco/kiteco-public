//go:generate bash -c "go-bindata $BINDATAFLAGS -pkg pipeline -o bindata.go templates/..."

package pipeline

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/codegangsta/negroni"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

type server struct {
	runner      *runner
	coordinator *coordinator

	pipe Pipeline
	opts EngineOptions

	templates *templateset.Set

	m sync.Mutex
}

func newServer(pipe Pipeline, opts EngineOptions) *server {
	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}

	renderError := func(err error) string {
		if err == nil {
			return ""
		}
		return err.Error()
	}

	templates := templateset.NewSet(staticfs, "templates", template.FuncMap{
		"renderError": renderError,
	})

	return &server{
		pipe:      pipe,
		opts:      opts,
		templates: templates,
	}
}

func (s *server) HandleFeedStats(w http.ResponseWriter, r *http.Request) {
	fs, err := s.feedStats()
	if err != nil {
		s.internalError(w, fmt.Errorf("could not get feed stats: %v", err))
		return
	}

	buf, err := json.Marshal(fs)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshalling stats: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (s *server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	status := s.runStatus()
	var errStr string
	if status.Err != nil {
		errStr = status.Err.Error()
	}

	resp := StatusResponse{
		State: status.State,
		Err:   errStr,
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshaling JSON: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (s *server) HandleResults(w http.ResponseWriter, r *http.Request) {
	runner := s.getRunner()
	if runner == nil {
		http.Error(w, "no runner present", http.StatusBadRequest)
		return
	}

	results, err := runner.GetResults()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	serializedResults := make(map[string][]byte, len(results))
	for agg, res := range results {
		buf, err := json.Marshal(res)
		if err != nil {
			http.Error(w, fmt.Sprintf("error serializing results of %s", agg.Name()),
				http.StatusInternalServerError)
			return
		}
		serializedResults[agg.Name()] = buf
	}

	resp := ResultsResponse{
		SerializedResults: serializedResults,
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshaling JSON: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (s *server) HandleStart(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := decode(r.Body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.TotalShards <= 0 || req.Shard < 0 || req.Shard >= req.TotalShards {
		http.Error(w, fmt.Sprintf("bad Shard/TotalShards params: %d, %d",
			req.Shard, req.TotalShards), http.StatusBadRequest)
		return
	}

	if err := s.startShard(req.Shard, req.TotalShards); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) SetRunner(runner *runner) {
	s.m.Lock()
	defer s.m.Unlock()
	s.runner = runner
}

func (s *server) getRunner() *runner {
	s.m.Lock()
	defer s.m.Unlock()
	return s.runner
}

func (s *server) setCoordinator(coordinator *coordinator) {
	s.m.Lock()
	defer s.m.Unlock()
	s.coordinator = coordinator
}

func (s *server) getCoordinator() *coordinator {
	s.m.Lock()
	defer s.m.Unlock()
	return s.coordinator
}

func (s *server) startShard(shard int, totalShards int) error {
	s.m.Lock()
	defer s.m.Unlock()

	if s.runner != nil {
		return fmt.Errorf("server already has a runner")
	}

	runner, err := newRunner(s.pipe, shard, totalShards, s.opts)
	if err != nil {
		return err
	}
	s.runner = runner

	err = s.runner.Start()
	if err != nil {
		return err
	}

	return nil
}

func (s *server) feedStats() (map[string]FeedStats, error) {
	if s.opts.Role != Coordinator {
		// If we're operating in standalone mode or as a shard, we can just get the stats directly
		s.m.Lock()
		defer s.m.Unlock()
		if s.runner == nil {
			return nil, fmt.Errorf("pipeline not running")
		}
		return s.runner.stats.Stats(), nil
	}

	// otherwise we need to aggregate the stats from the shards
	coord := s.getCoordinator()
	if coord == nil {
		return nil, fmt.Errorf("coordinator not set")
	}
	return coord.FeedStats()
}

func (s *server) runStatus() runStatus {
	if s.opts.Role != Coordinator {
		s.m.Lock()
		defer s.m.Unlock()
		if s.runner == nil {
			return runStatus{State: StateWaiting}
		}
		return s.runner.Status()
	}

	coord := s.getCoordinator()
	if coord == nil {
		return runStatus{State: StateWaiting}
	}
	return coord.Status()
}

func decode(r io.Reader, v interface{}) error {
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return fmt.Errorf("json decode error: %v", err)
	}
	return nil
}

// Listen starts the server in a goroutine. If the server fails and quitOnFail is true, the entire process will quit.
func (s *server) Listen(port int, quitOnFail bool) {
	r := mux.NewRouter()
	// UI methods
	r.HandleFunc("/", s.HandleRoot).Methods("GET")
	r.HandleFunc("/feed-errors", s.HandleFeedErrors).Methods("GET")

	// API methods
	r.HandleFunc("/api/start", s.HandleStart).Methods("POST")
	r.HandleFunc("/api/status", s.HandleStatus).Methods("GET")
	r.HandleFunc("/api/results", s.HandleResults).Methods("GET")
	r.HandleFunc("/api/feed-stats", s.HandleFeedStats).Methods("GET")

	r.PathPrefix("/debug/").Handler(http.DefaultServeMux)

	neg := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(contextutil.BasicLogger()),
		negroni.Wrap(r),
	)

	log.Printf("attempting to start server on port http://localhost:%d", port)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), neg)
		log.Printf("server could not bind to port :%d: %v", port, err)
		if quitOnFail {
			log.Fatal(err)
		}
	}()
}
