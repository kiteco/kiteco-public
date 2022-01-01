package driver

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const (
	speculationTimeLimit = 4 * time.Second
	// TODO(naman) increase this once the scheduler becomes smart enough to not starve high priority work with low priority work
	numAsyncWorkers = 1
	gcInterval      = 4 * time.Second
)

// Driver manages completions state for a given buffer
type Driver struct {
	cancel kitectx.CancelFunc

	inputsCache *lru.Cache

	cond *sync.Cond
	*driver
}
type driver struct {
	// the zero value indicates a "full stop" (the Driver is stopped, and no further work should be triggered)
	// a non-zero value indicates when the Driver will pause next, or when it started its ongoing pause.
	pauseAt time.Time

	global    pythonproviders.Global
	root      data.SelectedBuffer
	rootPyCtx *python.Context
	sched     scheduler
	doneChans map[workItemHash]chan struct{}
}
type lockedInputs struct {
	inputs   pythonproviders.Inputs
	err      error
	computed bool
	lock     sync.RWMutex
}

// New initializes a new Driver; no Completions will be generated until Update is called for the first time.
func New() Driver {
	d := Driver{
		cond:   sync.NewCond(&sync.Mutex{}),
		driver: &driver{},
	}
	d.inputsCache, _ = lru.New(2 * numAsyncWorkers)
	d.pauseAt = time.Now()
	d.sched = newScheduler(d.cond.Signal)
	d.doneChans = make(map[workItemHash]chan struct{})

	f, cancel := kitectx.Background().ClosureWithCancel(func(ctx kitectx.Context) error {
		var wg sync.WaitGroup
		wg.Add(numAsyncWorkers)
		for i := 0; i < numAsyncWorkers; i++ {
			kitectx.Go(func() error {
				defer wg.Done()
				d.workLoop(ctx)
				return nil
			})
		}

		wg.Add(1)
		kitectx.Go(func() error {
			defer wg.Done()
			d.garbageCollector(ctx)
			return nil
		})

		wg.Wait()
		return nil
	})

	d.cancel = cancel
	go f()

	return d
}

func (d Driver) getInputs(ctx kitectx.Context, global pythonproviders.Global, b, root data.SelectedBuffer, rootPyCtx *python.Context) (pythonproviders.Inputs, error) {
	ctx.CheckAbort()

	hash := b.Hash()

	var locked *lockedInputs
	var added bool
	for !added {
		if inpsIF, ok := d.inputsCache.Get(hash); ok {
			locked = inpsIF.(*lockedInputs)
			break
		}

		if locked == nil {
			locked = &lockedInputs{}
		}
		added, _ = d.inputsCache.ContainsOrAdd(hash, locked)
	}

	locked.lock.Lock()
	defer locked.lock.Unlock()
	if !locked.computed {
		if rootPyCtx != nil && b.Hash() == root.Hash() {
			locked.inputs, locked.err = pythonproviders.NewInputsFromPyCtx(ctx, global, b, false, rootPyCtx, d.sched.opts.GGNNSubtokenEnabled)
		} else {
			locked.inputs, locked.err = pythonproviders.NewInputs(ctx, global, b, false, d.sched.opts.GGNNSubtokenEnabled)
		}
		locked.computed = true
	}
	return locked.inputs, locked.err
}

func (d Driver) garbageCollector(ctx kitectx.Context) {
	tick := time.NewTicker(gcInterval)

	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			func() {
				d.cond.L.Lock()
				defer d.cond.L.Unlock()
				d.sched.Prune(gcInterval)
			}()
		case <-ctx.AbortChan():
			ctx.Abort()
		}
	}
}

// prepareWorkLocked marks the work as started with the scheduler,
// creates a channel for listeners to listen for completions,
// and returns a function that will execute the work.
// prepareWorkLocked assumes the global lock is held, but the returned function should be called without holding the lock.
func (d Driver) prepareWorkLocked(ctx kitectx.Context, work workItem) func() error {
	global := d.global
	root := d.root
	rootPyCtx := d.rootPyCtx

	workFn, cancel := ctx.ClosureWithCancel(func(ctx kitectx.Context) error {
		inps, err := d.getInputs(ctx, global, work.SelectedBuffer, root, rootPyCtx)
		if err != nil {
			return err
		}
		if work.GGNNPredictor != nil {
			inps.GGNNPredictor = work.GGNNPredictor
		}
		return work.Provider.Provide(ctx, global, inps, func(ctx kitectx.Context, source data.SelectedBuffer, compl pythonproviders.MetaCompletion) {
			d.cond.L.Lock()
			defer d.cond.L.Unlock()
			ctx.CheckAbort()
			d.sched.GotCompletion(work, source, compl)
		})
	})

	err := d.sched.WorkStarting(work, cancel)
	if err != nil {
		return func() error { return err }
	}

	hash := work.Hash()
	doneChan := make(chan struct{})
	d.doneChans[hash] = doneChan

	return func() (err error) {
		defer func() {
			d.cond.L.Lock()
			defer d.cond.L.Unlock()

			switch err.(type) {
			case kitectx.ContextExpiredError:
				d.sched.WorkIncomplete(work)
			case data.ProviderNotApplicableError:
				d.sched.WorkNotApplicable(work)
			default:
				d.sched.WorkComplete(work)
			}

			// notify listeners of work completion
			close(doneChan)
			delete(d.doneChans, hash)
		}()

		return workFn()
	}
}

func (d Driver) unpauseForLocked(limit time.Duration) error {
	if d.pauseAt == (time.Time{}) { // full stop
		return errors.Errorf("Driver already stopped")
	}

	d.pauseAt = time.Now().Add(limit)
	// signal to async workers that they may resume speculation, if stopped due to a time limit
	d.cond.Broadcast()
	return nil
}

// Options for Driver.Update
type Options struct {
	MixOptions
	ScheduleOptions

	// AsyncTimeout indicates how long to run asynchronous speculation before pausing the Driver.
	// If a zero value if provided, we choose a reasonable default.
	AsyncTimeout time.Duration

	// BlockDebug should be only used for debug & testing.
	BlockDebug bool

	// BlockTimeoutLexical indicates how long to block update for the lexical provider before returning completions.
	BlockTimeoutLexical time.Duration

	// UnitTestMode is used for unit tests
	UnitTestMode bool
}

// Update updates and returns completions for the given SelectedBuffer
func (d Driver) Update(ctx kitectx.Context, opts Options, global pythonproviders.Global, buf data.SelectedBuffer, pyctx *python.Context, requestCompletions bool, metricFn data.EngineMetricsCallback) ([]data.NRCompletion, error) {
	if requestCompletions || opts.UnitTestMode { // wait for completions in unit tests
		return d.updateBlocking(ctx, opts, global, buf, pyctx, metricFn)
	}
	return d.updateNonBlocking(ctx, opts, global, buf, pyctx)
}

func (d Driver) updateBlocking(ctx kitectx.Context, opts Options, global pythonproviders.Global, buf data.SelectedBuffer, pyctx *python.Context, metricFn data.EngineMetricsCallback) ([]data.NRCompletion, error) {
	// indicates if we should block on lexical completions until the first of either
	// the timeout or the lexical provider completes.
	blockLexical := opts.BlockTimeoutLexical != 0 && !opts.BlockDebug

	// this set tracks which providers self-report as not applicable for the given buffer state
	notApplicableSet := sync.Map{}

	var doneChans []chan struct{}
	var lexicalDone chan struct{}
	err := func() error {
		d.cond.L.Lock()
		defer d.cond.L.Unlock()

		if err := d.updateHelper(opts, global, buf, pyctx); err != nil {
			return err
		}

		// kick off any unstarted work for the chosen providers and compute the list of channels to block on
		for p := range blockingProviders {
			work := workItem{buf, p, nil}

			switch d.sched.WorkStatus(work) {
			case statusComplete:
				continue // no need to block on complete work
			case statusNotApplicable:
				notApplicableSet.Store(p.Name(), struct{}{})
				continue
			case statusPending:
				// shadow the provider to prevent race issues
				p := p
				// work not started; prepare it (mark it as started)
				workFn := d.prepareWorkLocked(ctx, work)
				// and kick off a new goroutine to execute it
				kitectx.Go(func() error {
					err := workFn()
					switch err.(type) {
					case data.ProviderNotApplicableError:
						notApplicableSet.Store(p.Name(), struct{}{})
					}
					return err
				})
			case statusInProgress:
			default:
				panic("unhandled provisionStatus")
			}

			// at this point work must already be started, but not complete, so we can get a done channel
			c := d.doneChans[work.Hash()]
			if c == nil {
				panic("no done channel for in-progress work")
			}

			if _, ok := p.(pythonproviders.Lexical); ok && blockLexical {
				lexicalDone = c
			} else {
				doneChans = append(doneChans, c)
			}
		}
		return nil
	}()

	if err != nil {
		return nil, err
	}

	// Wait for all the blocking providers
	for _, c := range doneChans {
		select {
		case <-ctx.AbortChan():
			ctx.Abort()
		case <-c:
		}
	}

	// Wait for lexical or timeout, if applicable
	if blockLexical && lexicalDone != nil {
		timeout := time.NewTimer(opts.BlockTimeoutLexical)
		defer timeout.Stop()

		select {
		case <-ctx.AbortChan():
			ctx.Abort()
		case <-lexicalDone:
		case <-timeout.C:
		}
	}

	if opts.BlockDebug {
		d.waitUntilPaused(ctx)
	}

	compls := func() []data.NRCompletion {
		d.cond.L.Lock()
		defer d.cond.L.Unlock()
		return d.sched.Mix(ctx, opts.MixOptions, pyctx, global, buf)
	}()

	// All blocking is done, invert the not-applicable set to determine which providers accepted the current buffer.
	// We assume that if a provider didn't return a NotApplicable error, that qualifies acceptance. This includes
	// situations such as timeouts, panics, and context expiry.
	// We assume that Lexical has returned quickly if it is NotApplicable, in the case where we don't block for it.
	acceptedProviders := make(map[data.ProviderName]struct{})
	for p := range blockingProviders {
		if _, present := notApplicableSet.Load(p.Name()); !present {
			// this blocking provider was not marked as not-applicable, so because all blocking providers are run
			// we know that it accepted this buffer state.
			acceptedProviders[p.Name()] = struct{}{}
		}
	}

	fulfillingProviders := make(map[data.ProviderName]struct{})
	for _, compl := range compls {
		fulfillingProviders[compl.Provider] = struct{}{}
	}

	if metricFn != nil {
		kitectx.Go(func() error {
			metricFn(acceptedProviders, fulfillingProviders)
			return nil
		})
	}

	return compls, nil
}

func (d Driver) updateNonBlocking(ctx kitectx.Context, opts Options, global pythonproviders.Global, buf data.SelectedBuffer, pyctx *python.Context) ([]data.NRCompletion, error) {
	err := func() error {
		d.cond.L.Lock()
		defer d.cond.L.Unlock()

		if err := d.updateHelper(opts, global, buf, pyctx); err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		return nil, err
	}

	if opts.BlockDebug {
		d.waitUntilPaused(ctx)
	}

	return nil, nil
}

func speculationLimit(opts Options) time.Duration {
	if opts.AsyncTimeout == 0 {
		if opts.BlockDebug {
			return 24 * time.Hour
		}
		return speculationTimeLimit
	}
	return opts.AsyncTimeout
}

func (d Driver) updateHelper(opts Options, global pythonproviders.Global, buf data.SelectedBuffer, pyctx *python.Context) error {
	if err := d.unpauseForLocked(speculationLimit(opts)); err != nil {
		return err
	}

	// update the schedule
	d.sched.Update(opts.ScheduleOptions, d.root, buf)

	// update the stored root
	d.root = buf
	d.rootPyCtx = pyctx
	d.global = global
	return nil
}

// waitUntilPaused waits for all async workers to pause if
// 1. no scheduled work remains, or if
// 2. the workers have been paused, or if
// 3. the workers have been stopped by Cleanup
func (d Driver) waitUntilPaused(ctx kitectx.Context) {
	var doneChans []chan struct{}
	var hasWork, paused bool

	for {
		func() {
			d.cond.L.Lock()
			defer d.cond.L.Unlock()
			for _, c := range d.doneChans {
				doneChans = append(doneChans, c)
			}
			hasWork = d.sched.HasWork()
			paused = time.Since(d.pauseAt) > 0
		}()

		if (paused || !hasWork) && len(doneChans) == 0 {
			return
		}

		for _, c := range doneChans {
			select {
			case <-ctx.AbortChan():
				ctx.Abort()
			case <-c:
			}
		}
		doneChans = doneChans[:0]
	}
}

func (d Driver) workLoop(ctx kitectx.Context) {
	for {
		var workFn func() error
		func() {
			d.cond.L.Lock()
			defer d.cond.L.Unlock()
			ctx.CheckAbort()

			for time.Since(d.pauseAt) > 0 || !d.sched.HasWork() {
				d.cond.Wait()
				ctx.CheckAbort()
			}

			workFn = d.prepareWorkLocked(ctx, d.sched.GetWork())
		}()
		// don't CheckAbort between prepareWorkLocked() and workFn(),
		// since we *must* call workFn in order for state to be consistent.
		workFn()

		ctx.CheckAbort()
	}
}

// Cleanup stops all work and cleans up resources associated with the Driver
// After Cleanup is called, the Driver should be thrown away. If
func (d Driver) Cleanup() {
	d.cond.L.Lock()
	defer d.cond.L.Unlock()

	d.pauseAt = time.Time{} // full stop

	d.cancel()
	d.cond.Broadcast()
}

// Reset stops all work and cleans up resources associated with the Driver.
// It blocks until all ongoing background work has been finished.
// After Reset is called, the Driver must not be used or reused again.
func (d Driver) Reset() {
	d.Cleanup()

	d.waitUntilPaused(kitectx.Background())
}

// CreateSchedulerFixture returns a test fixture from the scheduler cache.
func (d Driver) CreateSchedulerFixture() Fixture {
	return d.sched.ToFixture()
}
