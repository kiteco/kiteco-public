package workerpool

import (
	"context"
	"fmt"
	"sync"
)

// Job is a simple function wrapper for representing units of work
type Job func() error

// Pool managages a collection of goroutines to run Jobs
type Pool struct {
	ctx    context.Context
	wg     sync.WaitGroup
	cancel context.CancelFunc
	queue  chan Job

	m           sync.Mutex
	errsGuarded []error
}

// New returns a new Pool with n workers
func New(n int) *Pool {
	return NewWithCtx(context.Background(), n)
}

// NewWithCtx returns a new Pool with n workers
func NewWithCtx(ctx context.Context, n int) *Pool {
	ctx, cancel := context.WithCancel(ctx)

	p := Pool{
		ctx:    ctx,
		cancel: cancel,
		queue:  make(chan Job),
	}
	for i := 0; i < n; i++ {
		go p.worker(i)
	}

	return &p
}

// Add adds the provided jobs
func (l *Pool) Add(jobs []Job) {
	l.wg.Add(len(jobs))
	go func() {
		l.addBlocking(jobs)
	}()
}

// AddBlocking adds the provided jobs to the queue
// and blocks until all of the jobs have been added.
func (l *Pool) AddBlocking(jobs []Job) {
	l.wg.Add(len(jobs))
	l.addBlocking(jobs)
}

// the jobs should already be added to the wait group before this is called
func (l *Pool) addBlocking(jobs []Job) {
	var processed int
	for _, j := range jobs {
		select {
		case l.queue <- j:
			processed++
		case <-l.ctx.Done():
			break
		}
	}

	// mark the rest of the jobs as done with the wait group
	l.wg.Add(processed - len(jobs))
}

// Stop stops the workers, discarding any unfinished work. Worker goroutines may
// continue to process the current job they were working on, and then exit.
func (l *Pool) Stop() {
	l.cancel() // cancel workers
}

// Wait will wait for all pending Jobs to complete
func (l *Pool) Wait() error {
	l.wg.Wait()
	return l.Err()
}

// Errors contains all the errors returned by workers, if any
type Errors struct {
	Errors []error
}

// Error implements error
func (e Errors) Error() string {
	return fmt.Sprintf("workerpool: pool errors: %+v", e.Errors)
}

// Err returns any errors returned by Jobs.
func (l *Pool) Err() error {
	l.m.Lock()
	defer l.m.Unlock()

	if len(l.errsGuarded) == 0 {
		return nil
	}
	return Errors{Errors: l.errsGuarded}
}

// --

func (l *Pool) worker(i int) {
	for {
		select {
		case j := <-l.queue:
			if err := j(); err != nil {
				l.m.Lock()
				l.errsGuarded = append(l.errsGuarded, err)
				l.m.Unlock()
			}
			l.wg.Done()
		case <-l.ctx.Done():
			return
		}
	}
}
