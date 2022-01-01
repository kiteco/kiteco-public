package words

import (
	"encoding/json"
	"io"
	"math"
	"sort"
)

const (
	numExtsAlloc = 1
)

// CountByExt stores counts keyed by file extension
type CountByExt map[string]int

// Sum returns the total count across all extensions
func (c CountByExt) Sum() int {
	var total int
	for _, count := range c {
		total += count
	}
	return total
}

// Counts contains words mapped to counts keyed by file extension
type Counts map[string]CountByExt

// Hit increments counts for word w/ extension ext by count
func (cs Counts) Hit(word, ext string, count int) {
	if _, ok := cs[word]; !ok {
		cs[word] = make(CountByExt, numExtsAlloc)
	}
	cs[word][ext] += count
}

// Add merges counts with other
func (cs Counts) Add(other Counts) {
	for w, ce := range other {
		if _, ok := cs[w]; !ok {
			cs[w] = make(CountByExt, len(ce))
		}
		for ext, count := range ce {
			cs[w][ext] += count
		}
	}
}

// Clean removes entries with fewer than minCount entries
func (cs Counts) Clean(minCount int) Counts {
	for w, ce := range cs {
		if ce.Sum() < minCount {
			delete(cs, w)
		}
	}

	// Create a new map so we can release memory
	ncs := make(map[string]CountByExt, len(cs))
	for w, ce := range cs {
		ncs[w] = ce
	}

	return ncs
}

// Normalized will return word counts normalized by extension
func (cs Counts) Normalized(minCount int) map[string]int {
	// remove words with counts below threshold
	cs.Clean(minCount)

	// normalize counts such that the max value for all
	// extensions is maxAllExts, then take max across extensions
	// this has the effect of normalizing the counts and
	// not overly rewarding byte pairs that appear in many
	// extensions becuase the underlying languages are
	// correlated (e.g ts and js)
	maxByExt := make(map[string]int, numExtsAlloc)
	for _, ce := range cs {
		for ext, count := range ce {
			if count > maxByExt[ext] {
				maxByExt[ext] = count
			}
		}
	}

	var maxAllExts int
	for _, m := range maxByExt {
		if m > maxAllExts {
			maxAllExts = m
		}
	}

	finalCounts := make(map[string]int, len(cs))
	for w, ce := range cs {
		var max float64
		for ext, count := range ce {
			m := float64(maxAllExts) * float64(count) / float64(maxByExt[ext])
			if m > max {
				max = m
			}
		}
		finalCounts[w] = int(math.Round(max))
	}

	return finalCounts
}

type wordCountEntry struct {
	Word   string
	Counts map[string]int
}

// WriteTo ...
func (cs Counts) WriteTo(w io.Writer) (int64, error) {
	entries := make([]wordCountEntry, 0, len(cs))
	for w, ce := range cs {
		entries = append(entries, wordCountEntry{
			Word:   w,
			Counts: ce,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Word < entries[j].Word
	})

	js := json.NewEncoder(w)
	for _, entry := range entries {
		err := js.Encode(entry)
		if err != nil {
			return 0, err
		}
	}

	return 0, nil
}

// ReadFrom ...
func (cs Counts) ReadFrom(r io.Reader) (int64, error) {
	js := json.NewDecoder(r)
	for {
		var wce wordCountEntry
		err := js.Decode(&wce)
		if err == io.EOF {
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		cs[wce.Word] = wce.Counts
	}
}
