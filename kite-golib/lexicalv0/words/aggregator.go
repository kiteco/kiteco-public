package words

import (
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

const maxWordCount = 20e6

// Aggregator manages large counts by flushing to disk and managing
// merging once all counting has completed
type Aggregator struct {
	m            sync.Mutex
	counts       Counts
	maxWordCount int
	splits       int32
	splitsDir    string
}

// NewAggregator returns a new aggregator, using the provided
// directory for intermediate wordcount split files
func NewAggregator(tmpdir string) (*Aggregator, error) {
	err := os.MkdirAll(tmpdir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return &Aggregator{
		counts:       make(Counts),
		splits:       -1,
		splitsDir:    tmpdir,
		maxWordCount: maxWordCount,
	}, nil
}

// Add will increment count for word, with given extension, by n
func (a *Aggregator) Add(other Counts) {
	a.m.Lock()
	defer a.m.Unlock()
	a.counts.Add(other)
	if len(a.counts) > a.maxWordCount {
		err := a.flush(a.counts)
		if err != nil {
			log.Fatalln(err)
		}
		a.counts = make(Counts)
		debug.FreeOSMemory()
	}
}

// Merge will return a unified Counts object, applying the provided minCount filter
func (a *Aggregator) Merge(minCount int) (Counts, error) {
	return Merge(a.splitsDir, minCount)
}

// Flush ...
func (a *Aggregator) Flush() error {
	if len(a.counts) > 0 {
		return a.flush(a.counts)
	}
	return nil
}

func (a *Aggregator) flush(c Counts) error {
	v := atomic.AddInt32(&a.splits, 1)
	fn := filepath.Join(a.splitsDir, fmt.Sprintf("wordcounts-%05d.json.gz", v))

	f, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	fmt.Printf("flushing wordcounts with %d words to %s\n", len(c), fn)
	_, err = c.WriteTo(gz)
	if err != nil {
		return err
	}
	return nil
}

// --

// Merge ...
func Merge(dir string, minCount int) (Counts, error) {
	merger, err := newSplitMerger(dir)
	if err != nil {
		return nil, err
	}
	return merger.merge(minCount)
}
