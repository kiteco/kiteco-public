package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
)

const (
	eps = 1e-9
)

type symbolProb struct {
	// symbol that the user requested
	symbol string
	// canonicalize the symbol when requesting hashes
	canonicalize bool
	// cumulativeProb is the end of the symbol's cumulative probability range.
	// i.e. the first symbol's cumulativeProb is (probability of that symbol), and the last symbol's
	// cumulativeProb is 1.
	cumulativeProb float64
}

func (sp symbolProb) String() string {
	return fmt.Sprintf("<%s canonicalize?%v>", sp.symbol, sp.canonicalize)
}

// distribution represents a discrete distribution of symbols.
type distribution struct {
	// cumulative distribution of the symbols - a list of symbols and their associated cumulative probabilities
	cumulative []symbolProb
	rand       *rand.Rand
}

// newDistribution creates a distribution; weights should a map of each symbol to its relative probability
// (i.e. if the weight of one symbol is twice that of another, it is twice as likely to be drawn)
func newDistribution(weights traindata.SymbolDist, seed int64) (distribution, error) {
	// Ensure that the samples returned for each session are deterministic
	r := newRandom(seed)

	var cumulativeProb float64

	var cumulative []symbolProb

	if len(weights) == 0 {
		return distribution{}, fmt.Errorf("need to have at least one symbol in distribution")
	}

	// make sure the structure of the distribution is deterministic
	var symbols []string
	for sym := range weights {
		symbols = append(symbols, sym)
	}
	sort.Strings(symbols)

	var totalWeight float64
	for sym, se := range weights {
		if se.Weight < 0 {
			return distribution{}, fmt.Errorf("symbol %s has negative weight %f", sym, se.Weight)
		}
		totalWeight += se.Weight
	}

	if math.Abs(totalWeight-eps) == 0 {
		return distribution{}, fmt.Errorf("at least one weight needs to be non-zero")
	}

	for _, sym := range symbols {
		se := weights[sym]
		prob := se.Weight / totalWeight
		cumulativeProb += prob
		cumulative = append(cumulative, symbolProb{
			symbol:         sym,
			canonicalize:   se.Canonicalize,
			cumulativeProb: cumulativeProb,
		})
	}

	if math.Abs(cumulativeProb-1.0) > eps {
		return distribution{}, fmt.Errorf("distribution sums to %f", cumulativeProb)
	}

	return distribution{
		rand:       r,
		cumulative: cumulative,
	}, nil
}

func (d *distribution) Draw() string {
	cumulativeProb := d.rand.Float64()
	for _, symProb := range d.cumulative {
		if cumulativeProb > symProb.cumulativeProb {
			continue
		}
		return symProb.symbol
	}
	return d.cumulative[len(d.cumulative)-1].symbol
}

func newRandom(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}
