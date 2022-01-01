package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

const workerCount = 16

var provider string

func init() {
	// NOTE: temporary while we need to differentiate across aws and azure
	provider = envutil.MustGetenv("PROVIDER")
}

func poll() ([]*Metric, error) {
	var metrics []*Metric

	// get nodes as list of sourcers
	nodes, groupNodes, err := getNodes()
	if err != nil {
		return metrics, fmt.Errorf("error getting nodes: %v", err)
	}

	// fetch sources for all nodes
	pool := workerpool.New(workerCount)
	defer pool.Stop()
	for i := range nodes {
		node := nodes[i] // because the closure that we are adding to the pool does not get evaluated until after the loop
		pool.Add([]workerpool.Job{
			func() error { return node.fetchSources() },
		})
	}

	if err := pool.Wait(); err != nil {
		return metrics, fmt.Errorf("error in worker pool while getting sources: %v", err)
	}

	// fetch sources for group nodes
	for i := range groupNodes {
		groupNodes[i].fetchSources()
	}

	// convert node sources to metrics
	metrics = getMetrics(nodes)
	metrics = append(metrics, getMetrics(groupNodes)...)

	return metrics, nil
}

// loop contains all the logic for the ticker loop
func loop() {
	// poll for new metrics
	metrics, err := poll()
	if err != nil {
		log.Printf("\nerror polling: %v", err)
		return
	}
	// send all metrics
	//
	// NOTE: currently this is not parallelized because the senders we use are nonblocking - in the
	// future we may want to use the workerpool for this
	if err := sendAll(metrics); err != nil {
		log.Printf("\n%v", err)
		return
	}
	fmt.Print(".")
}

func main() {
	//test(testopts{include: []string{"mem"}})
	//return

	// start the loop immediately
	loop()
	t := time.NewTicker(time.Second)
	for range t.C {
		loop()
	}
}
