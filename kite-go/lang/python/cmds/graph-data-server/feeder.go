package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

const hashLimit = 5000

// symInfo contains information related to each input symbol
type symInfo struct {
	// Symbol the user requested
	Symbol pythonresource.Symbol
	// hashes available for this symbol
	hashes []string
	// idx to track which hash to pull to produce the next sample
	idx int
	m   sync.Mutex
}

func (s *symInfo) Invalidate(hash string) {
	s.m.Lock()
	defer s.m.Unlock()

	idx := -1
	for i, h := range s.hashes {
		if h == hash {
			idx = i
			break
		}
	}

	if idx == -1 {
		// this can happen if we have two concurrent requets
		// that invalidate the same hash
		return
	}

	before := s.hashes[:idx]
	after := s.hashes[idx+1:]

	s.hashes = append(before, after...)

	switch {
	case s.idx == idx && s.idx == len(s.hashes):
		// the can happen if s.idx was pointing to
		// the last hash and we removed the last hash,
		// so we cycle back to the first hash
		s.idx = 0
	case s.idx > idx:
		s.idx--
	}
}

func (s *symInfo) Next() string {
	s.m.Lock()
	defer s.m.Unlock()

	if len(s.hashes) == 0 {
		return ""
	}

	h := s.hashes[s.idx]
	s.idx = (s.idx + 1) % len(s.hashes)

	return h
}

// feeder is used to iterate over a distribution of symbols and return a sampleSeed, from which the sample
// can be unambiguously calculated.
type feeder struct {
	dist distribution
	// partition limits the samples returned such that their hash falls into a specific interval
	partition interval

	// symbol string -> info for that symbol
	symInfo map[string]*symInfo

	// canonical symbol string => all input symbols that map to this canonical symbol
	canonicalToSymbols map[string][]string

	rand       *rand.Rand
	randomSeed int64
}

func newSymInfo(sp symbolProb, seed int64, sc pythoncode.SymbolContext, partition interval, res *resources) (*symInfo, error) {
	// get the symbol the user requested
	sym, err := getSymbol(sp.symbol, res.rm)
	if err != nil {
		return nil, fmt.Errorf("could not find symbol %v in resource manager: %v", sp, err)
	}

	hashCounts, err := res.store.HashesFor(sym, sp.canonicalize, true)
	if err != nil {
		return nil, fmt.Errorf("could not find hashes for symbol %v", sp)
	}
	if len(hashCounts) == 0 {
		return nil, fmt.Errorf("no hashes available for symbol %v", sp)
	}

	// Get a pseudo-random permutation of the stored hashes
	var hashes []string
	r := newRandom(seed)
	for _, idx := range r.Perm(len(hashCounts)) {
		hc := hashCounts[idx]
		if ShouldUseHash(hc, sc) {
			hashes = append(hashes, hc.Hash)
		}

		// TODO: hacky, we artificially limit the number of hashes we consider
		// for anyseed to avoid running out of memory when we build the entire
		// distribution.
		if len(hashes) == hashLimit {
			break
		}
	}

	if len(hashes) == 0 {
		return nil, fmt.Errorf("no relevant hashes available for symbol %v in context %v", sp, sc)
	}

	// find the subset of the hashes for the given partition
	low := int(float64(len(hashes)) * partition.Low)
	high := int(float64(len(hashes)) * partition.High)
	partitionHashes := hashes[low:high]

	return &symInfo{
		Symbol: sym,
		hashes: partitionHashes,
	}, nil
}

func newFeeder(dist distribution, seed int64, sc pythoncode.SymbolContext, partition interval, res *resources) (*feeder, error) {
	info := make(map[string]*symInfo)
	canonicalToSymbols := make(map[string][]string)

	// collect all hashes for each symbol along with any other needed meta data
	log.Printf("Loading %d symbols", len(dist.cumulative))
	for _, sp := range dist.cumulative {
		si, err := newSymInfo(sp, seed, sc, partition, res)
		if err != nil {
			return nil, err
		}

		// we always need to canonicalize this because the
		// pythongraph operates on canonical symbols
		pathStr := si.Symbol.Canonical().PathString()

		canonicalToSymbols[pathStr] = append(canonicalToSymbols[pathStr], sp.symbol)

		info[sp.symbol] = si
	}

	return &feeder{
		dist:               dist,
		partition:          partition,
		symInfo:            info,
		canonicalToSymbols: canonicalToSymbols,
		rand:               newRandom(seed),
		randomSeed:         seed,
	}, nil
}

func (f *feeder) next() (sampleSeed, error) {
	symbol := f.dist.Draw()

	info := f.symInfo[symbol]

	hash := info.Next()
	if hash == "" {
		return sampleSeed{}, fmt.Errorf("no partitioned hashes available for symbol %s", symbol)
	}

	return sampleSeed{
		Symbol: info.Symbol,
		Hash:   hash,
		Random: f.rand.Int(),
	}, nil
}

func (f *feeder) Next() (sampleSeed, error) {
	return f.next()
}

func (f *feeder) Invalidate(symbol pythonresource.Symbol, hash string) {
	f.symInfo[symbol.PathString()].Invalidate(hash)
}

func getSymbol(symbol string, rm pythonresource.Manager) (pythonresource.Symbol, error) {
	sym, err := rm.PathSymbol(pythonimports.NewDottedPath(symbol))
	if err != nil {
		return pythonresource.Symbol{}, fmt.Errorf("error finding symbol %s: %v", symbol, err)
	}
	if sym.PathString() != symbol {
		return pythonresource.Symbol{}, fmt.Errorf("resource manger path symbol %s != input symbol path %s", sym.PathString(), symbol)
	}
	return sym, nil
}
