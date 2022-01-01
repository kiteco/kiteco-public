package words

import (
	"compress/gzip"
	"container/heap"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type splitMerger struct {
	splits []*splitReader
	heap   wordCountHeap
}

func newSplitMerger(dir string) (*splitMerger, error) {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var splits []*splitReader
	for _, fi := range fis {
		r, err := newSplitReader(filepath.Join(dir, fi.Name()))
		if err != nil {
			return nil, err
		}
		splits = append(splits, r)
	}

	merger := &splitMerger{
		splits: splits,
	}

	err = merger.init()
	if err != nil {
		return nil, err
	}

	return merger, nil
}

func (s *splitMerger) init() error {
	heap.Init(&s.heap)
	for idx, split := range s.splits {
		wce, err := split.next()
		if err != nil {
			return err
		}
		heap.Push(&s.heap, &wordCountHeapEntry{
			index: idx,
			wce:   wce,
		})
	}
	return nil
}

func (s *splitMerger) merge(minCount int) (Counts, error) {
	wordcount := make(Counts)

	// Set lastWord so we know when we've finished consuming a word
	var lastWord string
	var counts CountByExt

	first := true

	for !s.done() {
		// Pop entry, replenish from same split
		entry := heap.Pop(&s.heap).(*wordCountHeapEntry)
		err := s.pushIndex(entry.index)
		if err != nil {
			return nil, err
		}

		// If the word has changed, and satisfies minCount, add to wordcount,
		// and reset lastWord and counts.
		if first {
			counts = make(CountByExt)
			first = false
		} else if lastWord != entry.wce.Word {
			if counts.Sum() > minCount {
				wordcount[lastWord] = counts
			}
			counts = make(CountByExt)
		}

		lastWord = entry.wce.Word

		// Accumulate counts
		for ext, c := range entry.wce.Counts {
			counts[ext] += c
		}
	}

	// Make sure to flush last word if it meets minCount
	if counts.Sum() > minCount {
		wordcount[lastWord] = counts
	}

	return wordcount, nil
}

func (s *splitMerger) pushIndex(i int) error {
	r := s.splits[i]
	if r == nil {
		return nil
	}

	wce, err := r.next()
	if err == io.EOF {
		// If we've reached the end of the file, close it
		// and set it to nil so we can ignore in the future
		s.splits[i].close()
		s.splits[i] = nil
		return nil
	}
	if err != nil {
		return err
	}

	heap.Push(&s.heap, &wordCountHeapEntry{
		index: i,
		wce:   wce,
	})

	return nil
}

func (s *splitMerger) done() bool {
	for _, r := range s.splits {
		if r != nil {
			return false
		}
	}
	return len(s.heap) == 0
}

// --

type wordCountHeapEntry struct {
	index int
	wce   wordCountEntry
}

type wordCountHeap []*wordCountHeapEntry

func (h wordCountHeap) Len() int           { return len(h) }
func (h wordCountHeap) Less(i, j int) bool { return h[i].wce.Word < h[j].wce.Word }
func (h wordCountHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *wordCountHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(*wordCountHeapEntry))
}

func (h *wordCountHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (h *wordCountHeap) Peek() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	return x
}

// --

type splitReader struct {
	f  *os.File
	gz *gzip.Reader
	js *json.Decoder
}

func newSplitReader(fn string) (*splitReader, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	js := json.NewDecoder(gz)

	return &splitReader{
		f:  f,
		gz: gz,
		js: js,
	}, nil
}

func (s *splitReader) next() (wordCountEntry, error) {
	var wce wordCountEntry
	err := s.js.Decode(&wce)
	return wce, err
}

func (s *splitReader) close() error {
	err := s.gz.Close()
	if err != nil {
		return err
	}
	err = s.f.Close()
	if err != nil {
		return err
	}
	return nil
}
