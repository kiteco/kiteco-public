package pipeline

import (
	"fmt"
	"io"
	"log"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
)

const (
	defaultPort = 3111
)

// Role defines how the engine behaves
type Role string

const (
	// Standalone is used when the pipeline is run on a single instance; this is the default role
	Standalone Role = "standalone"
	// Coordinator connects to one or more shard instances, starts the pipeline on the instances and aggregates the
	// results once the instances finish
	Coordinator Role = "coordinator"
	// Shard runs the pipeline on a partition of the source data; an Coordinator instance is responsible for starting
	// the execution and retrieving the results
	Shard Role = "shard"
)

// EngineOptions for configuring the pipeline
type EngineOptions struct {
	Role Role

	// NumWorkers defines the number of worker goroutines that are launched to process the records.
	NumWorkers int
	// RunDBPath may contain the S3 directory to which the results of the pipeline will be written.
	RunDBPath string
	// RunName is a label that is applied to the run when saving to the RunDB.
	RunName string

	Logger io.Writer

	// NoServer, if true, keeps the engine's HTTP server from starting.
	NoServer bool

	// Port defines the port for the engine's HTTP server to bind to. If absent, 3111 is used as the default.
	// The engine will attempt to spin up an HTTP server on the port, which can be used for monitoring the status
	// of the pipeline.
	// If the engine is run in a Shard role, the pipeline will fail if the server cannot bind to the port, because
	// the server is necessary for communication between the coordinator and shards.
	Port int

	// ShardEndpoints defines the host:port pairs of the shard servers with which the Coordinator communicates. This
	// needs to be non-empty if the engine is running as the Coordinator role.
	ShardEndpoints []string

	// OnlyKeys is a map of source name to a set of allowed keys. If the entry for a given source is present, only
	// keys matching the provided ones will be passed through the pipeline. This is useful for debugging - if a
	// panic or error is seen for a particular key, this provides a means to isolate the failure.
	OnlyKeys map[string][]string
}

var (
	// DefaultEngineOptions for configuring the pipeline
	DefaultEngineOptions = EngineOptions{
		NumWorkers: 16,
	}
)

// Engine executes a pipeline.
type Engine struct {
	pipe   Pipeline
	opts   EngineOptions
	server *server
	rdb    rundb.RunDB
	ri     rundb.RunInfo
}

// NewEngine from the specified pipeline
func NewEngine(pipe Pipeline, opts EngineOptions) (*Engine, error) {
	var rdb rundb.RunDB
	if opts.RunDBPath != "" {
		var err error
		rdb, err = rundb.NewRunDB(opts.RunDBPath)
		if err != nil {
			return nil, err
		}
	}

	if opts.Role == "" {
		opts.Role = Standalone
	}

	if err := pipe.Validate(); err != nil {
		return nil, fmt.Errorf("error validating pipeline: %v", err)
	}

	if opts.Port == 0 {
		opts.Port = defaultPort
	}

	if opts.Role == Shard && opts.NoServer {
		return nil, fmt.Errorf("NoServer cannot be set for Shard role")
	}
	var server *server
	if !opts.NoServer {
		server = newServer(pipe, opts)
		quitOnFail := opts.Role == Shard // the pipeline should fail if a shard can't start its server
		server.Listen(opts.Port, quitOnFail)
	}

	return &Engine{
		pipe:   pipe,
		opts:   opts,
		server: server,
		rdb:    rdb,
	}, nil
}

// Run the pipeline, returning a map of each Aggregator to its aggregated output.
func (e *Engine) Run() (map[Aggregator]Sample, error) {
	info := rundb.NewRunInfo(e.rdb, e.pipe.Name, e.opts.RunName)

	info.SetStatus(rundb.StatusStarted)
	info.Params = e.pipe.Params

	printParams(e.pipe.Params)

	if err := e.saveRunInfo(info); err != nil {
		return nil, err
	}

	results, stats, err := e.execute()
	if err != nil {
		info.SetStatus(rundb.StatusError)
		info.Error = err.Error()
		if err := e.saveRunInfo(info); err != nil {
			return nil, err
		}
		return nil, err
	}
	info.FeedStats = stats

	if e.pipe.ResultsFn != nil {
		info.Results = e.pipe.ResultsFn(results)
	}

	info.SetStatus(rundb.StatusFinished)
	if err := e.saveRunInfo(info); err != nil {
		return nil, err
	}

	return results, nil
}

func (e *Engine) execute() (map[Aggregator]Sample, map[string]FeedStats, error) {
	switch e.opts.Role {
	case Standalone:
		return e.runAsStandalone()
	case Coordinator:
		return e.runAsCoordinator()
	case Shard:
		return e.runAsShard() // this will block forever
	default:
		return nil, nil, fmt.Errorf("unrecognized role: %s", e.opts.Role)
	}
}

func (e *Engine) runAsStandalone() (map[Aggregator]Sample, map[string]FeedStats, error) {
	runner, err := newRunner(e.pipe, 0, 1, e.opts)
	if err != nil {
		return nil, nil, err
	}

	if e.server != nil {
		e.server.SetRunner(runner)
	}

	if err := runner.Start(); err != nil {
		return nil, nil, err
	}

	runner.Wait()
	status := runner.Status()

	log.Printf("runner finished, state=%s, err=%v", status.State, status.Err)
	if status.Err != nil {
		return nil, nil, err
	}

	res, err := runner.GetResults()
	stats := runner.stats.Stats()

	if err := finalizeAggregators(e.pipe); err != nil {
		return nil, nil, err
	}

	return res, stats, err
}

func (e *Engine) runAsShard() (map[Aggregator]Sample, map[string]FeedStats, error) {
	if e.opts.Port == 0 {
		return nil, nil, fmt.Errorf("need to specify port if running in shard role")
	}

	log.Printf("running as shard on port %d", e.opts.Port)

	// If the pipeline is running as a shard, all we do here is block. The server will be responsible for handling
	// requests coming in from the coordinator and starting the pipeline
	log.Printf("blocking forever")
	for {
		time.Sleep(20 * time.Second)
	}
}

func (e *Engine) saveRunInfo(info rundb.RunInfo) error {
	if e.opts.Role == Shard {
		log.Printf("Role is Shard, not updating info")
		return nil
	}

	if e.rdb == (rundb.RunDB{}) {
		log.Printf("RunDB not configured, not updating info")
		return nil
	}

	return e.rdb.SaveRun(info)
}

func printParams(params map[string]interface{}) {
	if len(params) == 0 {
		return
	}

	type param struct {
		Name  string
		Value interface{}
	}
	var sorted []param
	for k, v := range params {
		sorted = append(sorted, param{Name: k, Value: v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	log.Printf("Params:")
	for _, p := range sorted {
		log.Printf("%s = %v", p.Name, p.Value)
	}
}

func finalizeAggregators(p Pipeline) error {
	// finalize the aggregators
	for _, agg := range p.Aggregators() {
		log.Printf("finalizing aggregator %s", agg.Name())
		if err := agg.Finalize(); err != nil {
			return fmt.Errorf("error finalizing aggregator %s: %v", agg.Name(), err)
		}
	}
	return nil
}
