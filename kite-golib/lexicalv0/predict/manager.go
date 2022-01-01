package predict

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

const (
	contextPlacholderOpName      = "placeholders/context"
	contextMaskPlaceholderOpName = "placeholders/context_mask"
	langsPlaceholderOpName       = "placeholders/langs"
)

var (
	errNoPredictionSlotsRemaining = errors.New("no prediction slots remaining")
)

var gid = int32(0)

type partialRunManager struct {
	pr     *tensorflow.PartialRun
	hp     HParams
	search SearchConfig

	debug bool
	id    int32

	m                     sync.Mutex
	wg                    sync.WaitGroup
	numPredictsDone       int
	numPredictsReserved   int
	closed                bool
	initialEmbedCompleted bool
}

func newPartialRunManager(pr *tensorflow.PartialRun, hp HParams) *partialRunManager {
	manager := &partialRunManager{
		pr: pr,
		hp: hp,
		id: atomic.AddInt32(&gid, 1),
	}

	manager.logf("new partial run")
	return manager
}

func (p *partialRunManager) EmbedInitialContext(feedFunc feederFunc) (interface{}, error) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		return nil, errors.New("partial run is closed")
	}

	if p.initialEmbedCompleted {
		return nil, errors.New("initial embed already completed")
	}

	feed, fetchOp := feedFunc(-1, false)
	p.logf("embedding initial context with op: %s", fetchOp)

	res, err := p.pr.Run(feed, []string{fetchOp}, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "error embedding initial context")
	}

	p.initialEmbedCompleted = true

	return res[fetchOp], nil
}

func (p *partialRunManager) slotsDoneLocked() int {
	return p.numPredictsDone
}

// Close blocks until all outstanding jobs are completed, then it calls
// feedFunc(numSlotsDone), while guarded by the lock.
// TODO:
//   - this is pretty nasty, currently the only way to force a cleanup is
//     to run all of the fetches associated with the partial run
func (p *partialRunManager) Close(feedFunc feederFunc) error {
	p.logf("closing when done")
	p.wg.Wait()

	p.m.Lock()
	defer p.m.Unlock()

	p.logf("closing")

	if p.closed {
		return nil
	}

	if p.hp.NumPredictionSlots-p.numPredictsDone < 1 {
		// we have already used all the slots so tensorflow
		// will clean up this partial run so it is not safe
		// to call run anymore
		return nil
	}
	p.closed = true

	feeds := make(map[string]interface{})
	var fetches []string
	for slot := p.slotsDoneLocked(); slot < p.hp.NumPredictionSlots; slot++ {
		feed, fetch := feedFunc(slot, true)
		for k, v := range feed {
			feeds[k] = v
		}

		fetches = append(fetches, fetch)
	}

	if !p.initialEmbedCompleted {
		feed, fetch := feedFunc(-1, true)
		for k, v := range feed {
			feeds[k] = v
		}
		fetches = append(fetches, fetch)
	}

	_, err := p.pr.Run(feeds, fetches, nil)
	if err != nil {
		if strings.Contains(err.Error(), "session is closed") {
			// TODO: this case can come up in unit tests when the underlying session
			// is closed before the partial run is closed.
			// One possible fix for this would be to add a close method to the tensorflow.PartialRun
			// type (SEE https://github.com/kiteco/tensorflow/pull/3) but the current tensorflow c api
			// does not actually do anything
			// when the partial run is closed (https://github.com/tensorflow/tensorflow/blob/master/tensorflow/c/c_api.cc#L2347)
			// so this will give client's the false impression that they are not responsible for resource cleanup.
			// So for now we just check the error message manually
			return nil
		}
		return errors.Wrapf(err, "unable to do final run")
	}
	return nil

}

// slot == -1 for initial run, isForClose == true is a special flag indicating
// that we are requesting the ops in order to close the partial run
type feederFunc func(slot int, isForClose bool) (map[string]interface{}, string)

func (p *partialRunManager) RunNextSlot(feedFunc feederFunc) (interface{}, error) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		p.logf("runNextSlot called, but partial run is closed")
		return nil, errors.New("partial run is closed, no more calls to run allowed")
	}

	// Check if we've used up all the slots. NOTE: this is different from slotsRemaining,
	// which tracks *reserved* slots
	if p.hp.NumPredictionSlots-p.numPredictsDone < 1 {
		p.logf("runNextSlotCalled, but no slots remaining")
		return nil, errNoPredictionSlotsRemaining
	}

	if !p.initialEmbedCompleted {
		p.logf("runNextSlot called, but initial embed has not been completed")
		return nil, errors.New("must perform initial embed before using run")
	}

	slot := p.numPredictsDone
	p.numPredictsDone++
	defer p.wg.Done()

	feeds, fetch := feedFunc(slot, false)

	p.logf("runNextSlot successfully run")
	res, err := p.pr.Run(feeds, []string{fetch}, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "error running op %s", fetch)
	}
	return res[fetch], nil
}

// Reserve partial run slots
func (p *partialRunManager) Reserve(slots int) bool {
	p.m.Lock()
	defer p.m.Unlock()
	return p.reserveLocked(slots)
}

func (p *partialRunManager) reserveLocked(slots int) bool {
	// Check if we have enough slots
	if p.slotsRemainingLocked() < slots {
		p.logf("failed to reserve %d slots, more than available (%d)", slots, p.slotsRemainingLocked())
		return false
	}

	// reserve slots
	p.numPredictsReserved += slots
	p.wg.Add(slots)
	p.logf("reserved %d slots, %d remaining", slots, p.slotsRemainingLocked())
	return true
}

// Release any slots that were not used
func (p *partialRunManager) Release(unused int) bool {
	p.m.Lock()
	defer p.m.Unlock()
	if unused > p.numPredictsReserved {
		p.logf("failed to release %d slots, more than has been reserved (%d)", unused, p.numPredictsReserved)
		return false
	}

	// release slots
	p.numPredictsReserved -= unused
	p.wg.Add(-unused)
	p.logf("released %d unused slots, now %d remaining", unused, p.slotsRemainingLocked())
	return true
}

// SlotsRemaining in the PRM
func (p *partialRunManager) SlotsRemaining() int {
	p.m.Lock()
	defer p.m.Unlock()
	return p.slotsRemainingLocked()
}

func (p *partialRunManager) slotsRemainingLocked() int {
	return p.hp.NumPredictionSlots - p.numPredictsReserved
}

func (p *partialRunManager) logf(msg string, args ...interface{}) {
	if p.debug {
		prefix := fmt.Sprintf("[partial_run id:%d embedded:%t closed:%t %d/%d/%d] ",
			p.id, p.initialEmbedCompleted, p.closed,
			p.numPredictsDone, p.numPredictsReserved, p.hp.NumPredictionSlots)
		log.Printf(prefix+msg, args...)
	}
}
