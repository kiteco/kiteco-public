package driver

import (
	"container/heap"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const floatDelta = 0.001

// newScheduler ...
func newScheduler(newWorkCB func()) scheduler {
	return scheduler{
		newWorkCB: newWorkCB,
		specs:     make(map[data.SelectedBufferHash]*speculationState),
	}
}

// - Work Lifecycle API

type provisionStatus int

const (
	statusPending provisionStatus = iota
	statusInProgress
	statusComplete
	statusNotApplicable
)

// workItem encapsulates a unit of work as a buffer, selection, and provider
type workItem struct {
	data.SelectedBuffer `json:"selected_buffer"`
	Provider            pythonproviders.Provider  `json:"provider"`
	GGNNPredictor       *pythongraph.PredictorNew `json:"-"`
}

type workItemHash struct {
	data.SelectedBufferHash
	provider pythonproviders.Provider
}

func (w workItem) Hash() workItemHash {
	return workItemHash{w.SelectedBuffer.Hash(), w.Provider}
}

// HasWork checks if a workItem is available
func (s *scheduler) HasWork() bool {
	return len(s.queue) > 0
}

// GetWork gets the next scheduled workItem from the queue.
// Before the work starts, WorkStarting should be called.
func (s *scheduler) GetWork() workItem {
	work := s.queue[0]
	if work.status != statusPending {
		panic("getting work that is not pending ")
	}
	return work.workItem
}

// WorkStatus returns the provisionStatus for a specific workItem
func (s *scheduler) WorkStatus(work workItem) provisionStatus {
	return s.get(work.SelectedBuffer).get(work.Provider).status
}

// WorkStarting informs the scheduler that progress is about to start for a specific workItem.
// If the work is already started or complete, we panic.
func (s *scheduler) WorkStarting(work workItem, cancel kitectx.CancelFunc) error {
	st := s.get(work.SelectedBuffer)

	ps := st.get(work.Provider)
	if ps.status != statusPending {
		//return //TODO remove and switch back to panic once we'll understand why that happen
		//panic("work already started")
		err := errors.New("error starting work %v", ps.status)
		return err
	}

	ps.setPriority(schedulerHeap{s}, -1)
	ps.status = statusInProgress
	ps.cancel = cancel

	// increase priority of already-queued "similar" work to a large priority,
	// so that the driver can effectively cache expensive computations
	for _, ps := range st.provisions {
		if ps.isQueued() {
			priority := float64(100)
			if s.opts.GGNNSubtokenEnabled {
				priority = float64(10000)
			}
			ps.increasePriority(schedulerHeap{s}, priority)
		}
	}
	return nil
}

// GotCompletion should be called when a completion is generated; it may internally schedule more work.
func (s *scheduler) GotCompletion(work workItem, source data.SelectedBuffer, compl pythonproviders.MetaCompletion) {
	compl.MixingMeta.Provider = pythonproviders.ProviderJSON{Provider: work.Provider}
	c := Completion{
		Meta:   compl,
		Target: source.Buffer.Replace(compl.Replace, compl.Snippet.Text),
	}

	st := s.get(source)
	st.get(work.Provider).addCompletion(c)

	if st.depth < 0 {
		return
	}

	s.rescheduleCompletion(work.Provider, c, st.depth, st.score)
}

// WorkIncomplete should be called when a workItem is no longer in progress, but it is also not complete.
// This is typically due to a context expiry.
func (s *scheduler) WorkIncomplete(work workItem) {
	s.get(work.SelectedBuffer).get(work.Provider).status = statusPending
}

// WorkComplete should be called when progress on a workItem is complete.
func (s *scheduler) WorkComplete(work workItem) {
	s.get(work.SelectedBuffer).get(work.Provider).status = statusComplete
}

// WorkNotApplicable should be called when a workItem returns with a ProviderNotApplicableError
func (s *scheduler) WorkNotApplicable(work workItem) {
	s.get(work.SelectedBuffer).get(work.Provider).status = statusNotApplicable
}

// - Control API

// getInvCompletion checks that new is the result of replacing old.Selection with some text
// and returns an "inverse" completion that transforms new back to old.
func getInvCompletion(old, new data.SelectedBuffer) (data.Completion, error) {
	prefix := old.Buffer.TextAt(data.Selection{End: old.Selection.Begin})
	selected := old.Buffer.TextAt(old.Selection)
	suffix := old.Buffer.TextAt(data.Selection{Begin: old.Selection.Begin, End: old.Buffer.Len()})

	newText := new.Buffer.Text()
	if len(prefix)+len(suffix) > len(newText) || !strings.HasPrefix(newText, prefix) || !strings.HasSuffix(newText, suffix) {
		return data.Completion{}, errors.Errorf("buffer mismatch")
	}

	replace := data.Selection{Begin: len(prefix), End: len(newText) - len(suffix)}
	if !replace.Contains(new.Selection) {
		return data.Completion{}, errors.Errorf("selection out of bounds")
	}

	return data.Completion{Replace: replace, Snippet: data.Snippet{Text: selected}}, nil
}

func (s *scheduler) copyCompletions(old, new data.SelectedBuffer) {
	// This is not quite complete, but it is subjectively good enough for now:
	//
	// If old is `foo‸` with existing "dots" completion `foo._`, and new is `foo.‸`,
	// then we'll copy the dots completion over to new, and then regenerate all attributes,
	// even though attributes may already live on the nested `foo._` state.
	//
	// We may be able to fix this by doing something recursive here, but consideration is needed.
	inv, err := getInvCompletion(old, new)
	if err != nil {
		return
	}
	oldState := s.specs[old.Hash()]
	if oldState == nil {
		return
	}

	newState := s.get(new)
	for p, ps := range oldState.provisions {
		for _, cc := range ps.completions {
			for _, c := range cc {
				composed, err := c.Meta.Completion.After(inv)
				if err != nil {
					continue
				}
				composed, ok := composed.Validate(new)
				if !ok {
					continue
				}
				c.Meta.Completion = composed
				newState.get(p).addCompletion(c)
			}
		}
	}
}

func (s *scheduler) Update(opts ScheduleOptions, old, new data.SelectedBuffer) {
	s.opts = opts
	if s.opts.GGNNSubtokenEnabled {
		s.UpdateWithGGNN(opts, old, new)
	} else {
		s.UpdateWithoutGGNN(opts, old, new)
	}
}
func (s *scheduler) UpdateWithGGNN(opts ScheduleOptions, old, new data.SelectedBuffer) {
	if old.Hash() == new.Hash() {
		return // nothing to do
	}

	for len(s.queue) > 0 {
		// go in reverse for efficiency, since removing from the end it constant time
		s.queue[len(s.queue)-1].setPriority(schedulerHeap{s}, -1)
	}

	for _, st := range s.specs {
		if st.depth >= 0 {
			st.lastOrphaned = time.Now()
			st.depth = -1
		}
	}

	s.copyCompletions(old, new)

	s.reschedule(new, 0, 1, true)

	// cancel orphaned work
	for _, st := range s.specs {
		if st.depth < 0 {
			for _, ps := range st.provisions {
				if ps.cancel != nil {
					ps.cancel()
				}
			}
		}
	}

	return
}

func (s *scheduler) UpdateWithoutGGNN(opts ScheduleOptions, old, new data.SelectedBuffer) {
	st := s.get(new)
	if st.depth == 0 {
		return // nothing to do
	}

	for len(s.queue) > 0 {
		// go in reverse for efficiency, since removing from the end it constant time
		s.queue[len(s.queue)-1].setPriority(schedulerHeap{s}, -1)
	}

	for _, st := range s.specs {
		if st.depth >= 0 {
			st.lastOrphaned = time.Now()
			st.depth = -1
		}
	}

	s.copyCompletions(old, new)

	s.reschedule(new, 0, 1, true)

	// cancel orphaned work
	for _, st := range s.specs {
		if st.depth < 0 {
			for _, ps := range st.provisions {
				if ps.cancel != nil {
					ps.cancel()
				}
			}
		}
	}

	return
}

// Prune prunes inaccessible & sufficiently old speculationState
func (s *scheduler) Prune(d time.Duration) {
	now := time.Now()
	for k, st := range s.specs {
		if st.depth < 0 && now.Sub(st.lastOrphaned) > d {
			delete(s.specs, k)
			for _, ps := range st.provisions {
				if ps.cancel != nil {
					ps.cancel()
				}
			}
		}
	}
}

// - Test fixture generation

// Fixture contains the info necessary to recreate the scheduler cache state in tests.
type Fixture struct {
	Cache []CacheEntry `json:"cache"`
}

// CacheEntry encapsulates the info necessary to add completions from a fixture to the cache.
type CacheEntry struct {
	Source      data.SelectedBuffer          `json:"source"`
	Provider    pythonproviders.ProviderJSON `json:"provider"`
	Completions []Completion                 `json:"completions"`
}

// ToFixture returns a test fixture of the scheduler cache.
func (s *scheduler) ToFixture() Fixture {
	f := Fixture{}
	for _, st := range s.specs {
		for p := range st.provisions {
			ps := st.get(p)
			var completions []Completion
			for _, comps := range ps.completions {
				completions = append(completions, comps...)
			}
			if completions != nil {
				f.Cache = append(f.Cache, CacheEntry{
					Source:      ps.SelectedBuffer,
					Provider:    pythonproviders.ProviderJSON{Provider: p},
					Completions: completions,
				})
			}
		}
	}
	return f
}

// FromFixture populates the scheduler cache from a test fixture.
func (s *scheduler) FromFixture(f Fixture) {
	for _, entry := range f.Cache {
		st := s.get(entry.Source)
		p := st.get(entry.Provider.Provider)
		for _, c := range entry.Completions {
			p.addCompletion(c)
		}
	}
}

// -

type scheduler struct {
	opts ScheduleOptions

	global    pythonproviders.Global
	newWorkCB func()

	specs map[data.SelectedBufferHash]*speculationState
	queue []*provisionState
}

func (s *scheduler) get(sb data.SelectedBuffer) *speculationState {
	hash := sb.Hash()
	st := s.specs[hash]
	if st == nil {
		st = &speculationState{
			SelectedBuffer: sb,
			depth:          -1,
		}
		s.specs[hash] = st
	}
	return st
}

type speculationState struct {
	data.SelectedBuffer
	provisions map[pythonproviders.Provider]*provisionState

	depth        int
	score        float64
	lastOrphaned time.Time // time when the depth was last set to -1
}

func (st *speculationState) get(p pythonproviders.Provider) *provisionState {
	ps := st.provisions[p]
	if ps == nil {
		ps = &provisionState{
			workItem: workItem{
				SelectedBuffer: st.SelectedBuffer,
				Provider:       p,
			},
			priority: -1,
			hIndex:   -1,
		}

		if st.provisions == nil {
			st.provisions = make(map[pythonproviders.Provider]*provisionState)
		}
		st.provisions[p] = ps
	}
	return ps
}

// Completion encapsulates the meta completion and target buffer
type Completion struct {
	Meta   pythonproviders.MetaCompletion `json:"meta"`
	Target data.Buffer                    `json:"target"`
}

// speculate takes a completion and generates new buffer states to speculate on
func (c Completion) speculate() []data.SelectedBuffer {
	var specs []data.SelectedBuffer
	for _, sel := range c.Meta.Snippet.Placeholders() {
		sel = sel.Offset(c.Meta.Replace.Begin)
		specs = append(specs, c.Target.Select(sel))
	}
	endCursor := data.Cursor(c.Meta.Replace.Begin + len(c.Meta.Snippet.Text))
	specs = append(specs, c.Target.Select(endCursor))
	return specs
}

type provisionState struct {
	workItem
	priority float64
	// hIndex is -1 if the provisionState is not in the heap
	// if hIndex >= 0, then status == statusPending
	hIndex int

	completions map[data.BufferHash][]Completion
	cancel      kitectx.CancelFunc
	status      provisionStatus
}

func (ps *provisionState) isQueued() bool {
	return ps.hIndex >= 0
}

func (ps *provisionState) addCompletion(c Completion) {
	if ps.completions == nil {
		ps.completions = make(map[data.BufferHash][]Completion)
	}
	ps.completions[c.Target.Hash()] = append(ps.completions[c.Target.Hash()], c)
}

func (ps *provisionState) setPriority(h heap.Interface, priority float64) {
	enqueue := priority >= -floatDelta && ps.status == statusPending
	ps.priority = priority
	if ps.hIndex < 0 {
		if enqueue {
			heap.Push(h, ps)
		}
	} else if enqueue {
		heap.Fix(h, ps.hIndex)
	} else {
		heap.Remove(h, ps.hIndex)
	}
}
func (ps *provisionState) increasePriority(h heap.Interface, priority float64) {
	if priority <= ps.priority {
		return
	}
	ps.setPriority(h, priority)
}

// - heap implementation

type schedulerHeap struct {
	*scheduler
}

func (s schedulerHeap) Len() int {
	return len(s.queue)
}
func (s schedulerHeap) Less(i, j int) bool {
	// only non-negative values are valid
	return s.queue[i].priority > s.queue[j].priority
}
func (s schedulerHeap) Swap(i, j int) {
	s.queue[i], s.queue[j] = s.queue[j], s.queue[i]
	s.queue[i].hIndex = i
	s.queue[j].hIndex = j
}
func (s schedulerHeap) Push(pwI interface{}) {
	ps := pwI.(*provisionState)
	if ps.priority < -floatDelta {
		panic("adding negative priority work to queue")
	}
	if ps.status != statusPending {
		panic("trying to push not pending work to the queue")
	}
	ps.hIndex = len(s.queue)
	s.queue = append(s.queue, ps)
	s.newWorkCB()
}
func (s schedulerHeap) Pop() interface{} {
	ret := s.queue[len(s.queue)-1]
	s.queue = s.queue[:len(s.queue)-1]
	ret.hIndex = -1
	return ret
}
