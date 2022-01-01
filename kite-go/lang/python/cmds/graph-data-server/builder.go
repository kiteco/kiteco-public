package main

import (
	"fmt"
	"log"
	"sync/atomic"

	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

const (
	// number of concurrent tasks that can be produced at a time
	maxConcurrentTasks  = 8
	maxCompletedBatches = 10
)

type builder struct {
	batchSize int

	samples chan *sample

	pool *workerpool.Pool

	quit chan struct{}

	buildCount    int64
	buildErrCount int64
	BuildOKCount  int64
}

type buildFunc func() (*sample, error)

func newBuilder(batchSize int, build buildFunc) *builder {
	b := &builder{
		batchSize: batchSize,
		pool:      workerpool.New(maxConcurrentTasks),
		samples:   make(chan *sample, maxCompletedBatches*batchSize),
		quit:      make(chan struct{}),
	}

	go func() {
		for {
			select {
			case <-b.quit:
				return
			default:
				// make sure to use add blocking here so that
				// we do not keep spawing go routines until we can
				// start actuall doing the jobs
				b.pool.AddBlocking([]workerpool.Job{func() error {
					atomic.AddInt64(&b.buildCount, 1)

					sample, err := build()
					if err != nil {
						atomic.AddInt64(&b.buildErrCount, 1)
						log.Println(err)
						return nil
					}
					atomic.AddInt64(&b.BuildOKCount, 1)

					b.samples <- sample
					return nil
				}})
			}
		}
	}()

	return b
}

func (b *builder) Cleanup() {
	// shut down the feeder go routine
	close(b.quit)

	// shut downt he worker pool
	b.pool.Stop()

	// drain sample channel so the worker go routines finish
	go func() {
		for range b.samples {

		}
	}()
}

func (b *builder) GetBatch() ([]*sample, error) {
	batch := make([]*sample, 0, b.batchSize)
	for i := 0; i < b.batchSize; i++ {
		select {
		case <-b.quit:
			return nil, fmt.Errorf("session was shut down")
		case s := <-b.samples:
			batch = append(batch, s)
		}
	}
	return batch, nil
}
