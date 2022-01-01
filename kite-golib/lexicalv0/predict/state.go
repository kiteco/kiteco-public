package predict

import (
	"log"
	"math/rand"
	"time"
)

// State ... rename to Prediction
type State struct {
	EmbeddedContext   []int64
	UnembeddedContext []int64
	PredictedContext  [][]int64
	Prefix            string
	Rand              *rand.Rand

	Search SearchConfig

	prm *PartialRunModel

	queryCalled int
	incremental chan Predicted

	// TODO: remove
	originalContext []int64

	// used for metrics
	// TODO: find a better place for this.
	curatedTokens map[int]bool
}

// newPredictState creates a new predict state.
// If randomSeed < 0, it is based off the current time.
func newPredictState(context []int64, prefix string, randomSeed int64, search SearchConfig) *State {
	if randomSeed < 0 {
		randomSeed = time.Now().UnixNano()
	}
	return &State{
		EmbeddedContext: context,
		Prefix:          prefix,
		Rand:            rand.New(rand.NewSource(randomSeed)),
		Search:          search,
		originalContext: context,
	}
}

// close will close any partial runs associated with this state
func (s *State) close() error {
	if s.prm != nil {
		err := s.prm.Close()
		if err != nil {
			log.Printf("error closing partial run model: %v", err)
		}
		return err
	}
	return nil
}
