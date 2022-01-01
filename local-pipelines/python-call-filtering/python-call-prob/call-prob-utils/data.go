package callprobutils

import (
	"math/rand"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

// SampleInputs contains all required information to generate samples to train the callprob filtering model
type SampleInputs struct {
	Hash   string
	Cursor int64
	RAST   *pythonanalyzer.ResolvedAST
	Sym    pythonresource.Symbol
	// UserTyped contains the buffer after the attribute dot and is used to determine which (if any) of the
	// call completions were actually entered
	UserTyped []byte
	UserCall  *pythonast.CallExpr
	CallComps []pythongraph.PredictedCall
	ScopeSize int
}

// SampleTag ...
func (SampleInputs) SampleTag() {}

// RNG provide a random number generator generator
type RNG struct {
	R *rand.Rand
	m sync.Mutex
}

// Random provides a new random number generator with a seed generated from the master generator
func (r *RNG) Random() *rand.Rand {
	r.m.Lock()
	defer r.m.Unlock()

	seed := r.R.Int63()
	return rand.New(rand.NewSource(seed))
}

// NewRNG create a new random number generator. The seed is used to initialize a master random number generator
// that will then be used to generate the seeds of the generate created at each call of Random()
func NewRNG(seed int64) *RNG {
	return &RNG{
		R: rand.New(rand.NewSource(seed)),
	}
}
