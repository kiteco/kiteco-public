package pipeline

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type coordinator struct {
	clients []client
	pipe    Pipeline

	m          sync.Mutex
	state      RunState
	startedAt  time.Time
	finishedAt time.Time
	err        error
}

func (e *Engine) runAsCoordinator() (map[Aggregator]Sample, map[string]FeedStats, error) {
	endpoints := e.opts.ShardEndpoints

	if len(endpoints) == 0 {
		return nil, nil, fmt.Errorf("need to specify shard endpoints if running in coordinator mode")
	}

	log.Printf("running as coordinator, shard endpoints: %v", endpoints)

	clients := make([]client, 0, len(endpoints))
	for _, e := range endpoints {
		clients = append(clients, newClient(e))
	}

	coord := coordinator{
		clients: clients,
		pipe:    e.pipe,
		state:   StateWaiting,
	}
	if e.server != nil {
		e.server.setCoordinator(&coord)
	}

	log.Printf("waiting for shards to be ready")
	if err := coord.WaitForShardsReady(); err != nil {
		return nil, nil, err
	}

	log.Printf("starting shards")
	if err := coord.Start(); err != nil {
		return nil, nil, err
	}

	log.Printf("waiting for shards to be done")
	if err := coord.WaitForShardsDone(); err != nil {
		return nil, nil, err
	}

	log.Printf("aggregating shard results")
	results, err := coord.Results()
	if err != nil {
		return nil, nil, err
	}

	log.Printf("aggregating shard stats")
	stats, err := coord.FeedStats()
	coord.Done(err)
	if err != nil {
		return nil, nil, err
	}

	log.Printf("aggregated stats:")
	printStats(os.Stderr, stats)

	return results, stats, nil
}

// WaitForShardsReady waits for the shards to start their HTTP servers so that they can respond to requests
func (c *coordinator) WaitForShardsReady() error {
	// poll the shards until they're ready
	shardsReady := make(chan error, len(c.clients))
	for _, c := range c.clients {
		go func(c client) {
			for {
				status, err := c.Status()
				if err != nil {
					log.Printf("could not yet get status from shard %s: %v", c.endpoint, err)
					time.Sleep(5 * time.Second)
					continue
				}

				if status.Err != "" {
					shardsReady <- fmt.Errorf("shard %s has error: %s", c.endpoint, status.Err)
					return
				}

				if status.State != StateWaiting {
					shardsReady <- fmt.Errorf("shard %s in wrong state: %s", c.endpoint, status.State)
					return
				}

				log.Printf("shard %s is ready", c.endpoint)
				shardsReady <- nil
				return
			}
		}(c)
	}

	log.Printf("waiting for shards to be ready")
	var numReady int
	for err := range shardsReady {
		if err != nil {
			err = fmt.Errorf("error waiting for endpoint to be ready: %v", err)
			c.setError(err)
			return err
		}
		numReady++
		if numReady >= len(c.clients) {
			break
		}
	}

	return nil
}

// Start the pipeline on the shards
func (c *coordinator) Start() error {
	c.setStarted()

	// tell each shard to start running
	for i, client := range c.clients {
		req := StartRequest{
			Shard:       i,
			TotalShards: len(c.clients),
		}
		log.Printf("starting %s as shard %d/%d", client.endpoint, i, len(c.clients))
		if err := client.Start(req); err != nil {
			err = fmt.Errorf("error starting %s: %v", client.endpoint, err)
			c.setError(err)
			return err
		}
	}

	return nil
}

// WaitForShardsDone running the pipeline. If any shard failed to successfully finish the pipeline, returns an error
func (c *coordinator) WaitForShardsDone() error {
	shardsDone := make(chan error, len(c.clients))

	for _, c := range c.clients {
		go func(c client) {
			for {
				status, err := c.Status()
				if err != nil {
					shardsDone <- fmt.Errorf("error getting status for shard %s: %v", c.endpoint, err)
					return
				}

				if status.Err != "" {
					shardsDone <- fmt.Errorf("shard %s has error: %s", c.endpoint, status.Err)
					return
				}

				switch status.State {
				case StateRunning:
					time.Sleep(5 * time.Second)
				case StateFinished:
					shardsDone <- nil
					return
				default:
					shardsDone <- fmt.Errorf("shard %s in wrong state: %s", c.endpoint, status.State)
					return
				}
			}
		}(c)
	}

	var numDone int
	for err := range shardsDone {
		if err != nil {
			err = fmt.Errorf("error waiting for shards to be done: %v", err)
			c.setError(err)
			return err
		}
		numDone++
		if numDone >= len(c.clients) {
			break
		}
	}

	return nil
}

// Results aggregates the results for the aggregators across the shards
func (c *coordinator) Results() (map[Aggregator]Sample, error) {
	results := make(map[Aggregator]Sample)

	// aggregate the results from the shards
	// TODO: do we want to do a partial aggregation if some of the shards fail?
	for _, client := range c.clients {
		log.Printf("aggregating results from shard %s", client.endpoint)
		res, err := client.Results()

		if err != nil {
			err = fmt.Errorf("error getting results from shard %s: %v", client.endpoint, err)
			c.setError(err)
			return nil, err
		}

		for _, agg := range c.pipe.Aggregators() {
			log.Printf("=> %s", agg.Name())
			sr, found := res.SerializedResults[agg.Name()]
			if !found {
				err := fmt.Errorf("shard %s does not have results for aggregator %s",
					client.endpoint, agg.Name())
				c.setError(err)
				return nil, err
			}
			sample, err := agg.FromJSON(sr)
			if err != nil {
				err = fmt.Errorf("error deserializing results for aggregator %s from shard %s: %v",
					agg.Name(), client.endpoint, err)
				c.setError(err)
				return nil, err
			}
			out, err := agg.AggregateFromShard(results[agg], sample, client.endpoint)
			if err != nil {
				err = fmt.Errorf("error aggregating results of aggregator %s: %v", agg.Name(), err)
				c.setError(err)
				return nil, err
			}
			results[agg] = out
		}
	}

	if err := finalizeAggregators(c.pipe); err != nil {
		c.setError(err)
		return nil, err
	}

	return results, nil
}

// Done marks the coordinator as done, either with an error or not
func (c *coordinator) Done(err error) {
	if err != nil {
		c.setError(err)
		return
	}

	c.m.Lock()
	defer c.m.Unlock()
	c.setFinishedLocked()
}

// FeedStats gets the feed stats of each shard.
func (c *coordinator) FeedStats() (map[string]FeedStats, error) {
	type statsOrErr struct {
		Stats map[string]FeedStats
		Err   error
	}

	statsChan := make(chan statsOrErr, len(c.clients))
	for _, c := range c.clients {
		go func(c client) {
			for {
				fs, err := c.FeedStats()
				if err != nil {
					statsChan <- statsOrErr{
						Err: fmt.Errorf("error getting feed stats for shard %s: %v", c.endpoint, err),
					}
					return
				}
				statsChan <- statsOrErr{Stats: fs}
			}
		}(c)
	}

	var allStats []map[string]FeedStats
	for soe := range statsChan {
		if soe.Err != nil {
			// do not set the coordinator's error here because this method might be called
			err := fmt.Errorf("error getting stats for shard: %v", soe.Err)
			log.Printf("%v", err)
			return nil, err
		}

		allStats = append(allStats, soe.Stats)
		if len(allStats) >= len(c.clients) {
			break
		}
	}

	stats := AggregateStats(allStats)
	return stats, nil
}

// Status of the coordinator
func (c *coordinator) Status() runStatus {
	c.m.Lock()
	defer c.m.Unlock()

	return runStatus{
		State:      c.state,
		StartedAt:  c.startedAt,
		FinishedAt: c.finishedAt,
		Err:        c.err,
	}
}

func (c *coordinator) setError(err error) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.state == StateFinished {
		log.Printf("coordinator encountered subsequent error: %v", err)
		return
	}

	log.Printf("coordinator encountered first error: %v", err)
	c.setFinishedLocked()
	c.err = err
}

func (c *coordinator) setStarted() {
	c.m.Lock()
	defer c.m.Unlock()

	c.state = StateRunning
	c.startedAt = time.Now().UTC()
}

func (c *coordinator) setFinishedLocked() {
	c.state = StateFinished
	c.finishedAt = time.Now().UTC()
}
