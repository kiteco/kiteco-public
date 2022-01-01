package bpe

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

// Builder ...
type Builder struct {
	m        sync.Mutex
	words    map[string]*tokenizedWord
	vocab    map[string]struct{}
	enc      *Encoder
	debug    bool
	mergeLog []MergedPair
	useBytes bool
}

// NewBuilder ...
func NewBuilder(useBytes bool) *Builder {
	return &Builder{
		words:    make(map[string]*tokenizedWord),
		vocab:    make(map[string]struct{}),
		useBytes: useBytes,
	}
}

// NewBuilderWithVocab ...
func NewBuilderWithVocab(vocab string) (*Builder, error) {
	enc, err := NewEncoder(vocab)
	if err != nil {
		return nil, err
	}
	return NewBuilderFromEncoder(enc), nil
}

// NewBuilderFromEncoder ...
func NewBuilderFromEncoder(enc *Encoder) *Builder {
	b := NewBuilder(enc.useBytes)
	b.enc = enc

	for _, v := range b.enc.vocab {
		b.vocab[v] = struct{}{}
	}
	return b
}

// LoadOptions specifies options wrt loading words
type LoadOptions struct {
	TopP      float64
	WordCount int
}

// LoadWords loads words saved at filename
func (b *Builder) LoadWords(filename string, opts LoadOptions) error {
	b.debug = true

	b.printf("loading words from %s", filename)

	var words []BuilderWordCount
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, &words)
	if err != nil {
		return err
	}

	sort.Slice(words, func(i, j int) bool {
		return words[i].Count > words[j].Count
	})

	b.printf("found %d words, loading...", len(words))

	var total int
	for _, word := range words {
		total += word.Count
	}

	var loaded int
	for idx, word := range words {
		tw := b.getTokenizedWord(string(word.Word))
		tw.incr(word.Count)
		loaded += word.Count
		top := float64(loaded) / float64(total)
		if idx%100000 == 0 {
			b.printf("loaded %d (%.04f, top %.04f, wordcount %d)...", idx, float64(idx)/float64(len(words)), top, word.Count)
		}
		if opts.WordCount > 0 && word.Count <= opts.WordCount {
			b.printf("stopped at %d out of %d, reached wordcount %d (%d out of %d, %.04f)", idx, len(words), opts.WordCount, loaded, total, top)
			break
		}
		if opts.TopP > 0 && top >= opts.TopP {
			b.printf("stopped at %d out of %d, reached top %.04f (%d out of %d, %d)", idx, len(words), opts.TopP, loaded, total, word.Count)
			break
		}
	}

	b.printf("finished!")

	return nil
}

// Add ...
func (b *Builder) Add(words []string) {
	for _, w := range words {
		tw := b.getTokenizedWord(w)
		tw.incr(1)
	}
}

func (b *Builder) getTokenizedWord(w string) *tokenizedWord {
	b.m.Lock()
	defer b.m.Unlock()
	t, ok := b.words[w]
	if !ok {
		var toks []string
		if b.enc == nil {
			toks = splitWord(w, b.useBytes)
		} else {
			toks = b.enc.Encode([]string{w})
		}
		t = newTokenizedWord(toks)
		b.words[w] = t
		for _, tok := range toks {
			b.vocab[tok] = struct{}{}
		}
	}
	return t
}

// Words ...
func (b *Builder) Words() int {
	return len(b.words)
}

// Vocab ...
func (b *Builder) Vocab() []Entry {
	b.m.Lock()
	defer b.m.Unlock()

	// Grab counts of all current tokens
	counts := b.CurrentTokens()

	// Make sure we include tokens that may not be "in use" anymore
	// that were added during vocab generation
	for tok := range b.vocab {
		if _, ok := counts[tok]; !ok {
			counts[tok] = 0
		}
	}

	var vocab []Entry
	for bp, count := range counts {
		vocab = append(vocab, Entry{BytePair: bp, Count: count})
	}

	sort.Stable(SortBytePair(vocab))

	if b.useBytes {
		// if we used bytes then we cannot directly serialize
		// these entries using strings because the JSON package does some weird
		// things when serializing strings that hold arbitrary bytes.
		for i, e := range vocab {
			vocab[i].BytePairBytes = []byte(e.BytePair)
			vocab[i].BytePair = ""
		}
	}

	return vocab
}

// WriteTo ...
func (b *Builder) WriteTo(w io.Writer) (int64, error) {
	vocab := b.Vocab()

	// MarshalIndent to make it slightly easier to read
	buf, err := json.MarshalIndent(vocab, "", "  ")
	if err != nil {
		return 0, err
	}

	n, err := w.Write(buf)
	return int64(n), err
}

// MergeOptions ...
type MergeOptions struct {
	Iterations       int
	MinPairFrequency int
	MaxVocabSize     int
	Logging          bool
	Concurrency      int
	CheckpointDir    string
}

// --

// Merge ...
func (b *Builder) Merge(opts MergeOptions) error {
	b.debug = opts.Logging
	if opts.Concurrency == 0 {
		opts.Concurrency = 1
	}

	b.printf("constructing pairs...")
	pairs := make(map[MergedPair]int)
	for _, tw := range b.words {
		for p, locs := range tw.tokenPairs {
			pairs[p] += tw.wordCount * len(locs)
		}
	}

	pac := &pairCountHeap{}
	pairToPac := make(map[MergedPair]*pairAndCount)
	for p, c := range pairs {
		c := &pairAndCount{pair: p, count: c}
		pairToPac[p] = c
		heap.Push(pac, c)
	}

	heap.Init(pac)

	if opts.CheckpointDir != "" {
		b.saveWords(opts.CheckpointDir)
	}

	b.printf("starting merge...")

	var start time.Time
	pool := workerpool.New(opts.Concurrency)
	for i := 0; opts.Iterations == 0 || i < opts.Iterations; i++ {
		topPair := heap.Pop(pac).(*pairAndCount)
		pairCount := topPair.count
		pairToMerge := topPair.pair

		b.printf("[iter: %d] vocab: %d, words: %d, most frequent pair count: %d, (%s % x,%s % x) took %s",
			i, len(b.vocab), len(b.words), pairCount, pairToMerge.Parent1, pairToMerge.Parent1, pairToMerge.Parent2, pairToMerge.Parent2, time.Since(start))
		b.mergeLog = append(b.mergeLog, pairToMerge)

		start = time.Now()

		// Do this here so we have some stats to log within checkpoint
		if opts.CheckpointDir != "" && len(b.vocab)%500 == 0 {
			b.checkpoint(opts.CheckpointDir)
		}

		if opts.MaxVocabSize > 0 && len(b.vocab) >= opts.MaxVocabSize {
			return nil
		}

		if opts.MinPairFrequency > 0 && pairCount <= opts.MinPairFrequency {
			return nil
		}

		// Terminate if we don't have any pairs that occur more than once
		if pairCount == 1 {
			return nil
		}

		// Merge that pair

		// Merge it wherever it occurs, constructing new pairs/counts as we go
		var wg sync.WaitGroup
		wg.Add(1)
		deltasChan := make(chan map[MergedPair]int, 10*opts.Concurrency)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			for deltas := range deltasChan {
				for p, delta := range deltas {
					if newPac, ok := pairToPac[p]; !ok {
						newPac = &pairAndCount{pair: p, count: delta}
						pairToPac[p] = newPac
						heap.Push(pac, newPac)
					} else {
						pairToPac[p].count += delta
						heap.Fix(pac, pairToPac[p].index)
					}
				}
			}
		}(&wg)

		var jobs []workerpool.Job
		for _, tw := range b.words {
			if !tw.contains(pairToMerge) {
				continue
			}
			localTW := tw
			jobs = append(jobs, func() error {
				localTW.mergePair(pairToMerge)
				deltas := localTW.findPairDeltas()
				deltasChan <- deltas
				return nil
			})
		}

		pool.AddBlocking(jobs)
		err := pool.Wait()
		if err != nil {
			return err
		}

		close(deltasChan)
		wg.Wait()

		// Do a quick sanity check: the pair we just merged should have a count of zero. If not,
		// something bad happened and we should panic.
		newCount := pairToPac[pairToMerge].count
		if newCount != 0 {
			b.printf("constructing pairs...")
			newPairs := make(map[MergedPair]int)
			for _, tw := range b.words {
				for p, locs := range tw.tokenPairs {
					newPairs[p] += tw.wordCount * len(locs)
				}
			}
			panic(errors.Errorf("pair (%s,%s) still has count of %d, originally had %d, recomputed: %d",
				pairToMerge.Parent1, pairToMerge.Parent2, newCount, pairCount, newPairs[pairToMerge]))
		}

		// Add pair to vocab
		b.vocab[pairToMerge.Joined] = struct{}{}
	}

	return nil
}

// MergeLog saves which words get merged into which one
func (b *Builder) MergeLog() []MergedPair {
	return b.mergeLog
}

// CurrentVocab ...
func (b *Builder) CurrentVocab() map[string]struct{} {
	return b.vocab
}

// CurrentTokens returns a map from the current token to the count
func (b *Builder) CurrentTokens() map[string]int {
	tokens := make(map[string]int)
	for _, word := range b.words {
		for _, token := range word.tokens() {
			tokens[token] += word.wordCount
		}
	}
	return tokens
}

func (b *Builder) checkpoint(dir string) {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}

	fn := fmt.Sprintf("ident-vocab-%d-entries.bpe", len(b.vocab))
	b.printf("[checkpoint] writing vocab w/ size %d to %s", len(b.vocab), fn)

	f, err := os.Create(filepath.Join(dir, fn))
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	_, err = b.WriteTo(f)
	if err != nil {
		log.Fatalln(err)
	}
}

// BuilderWordCount ...
type BuilderWordCount struct {
	// we cannot directly serialize
	// entry because the JSON package does some weird things when
	// serializing strings that hold arbitrary bytes...
	Word  []byte
	Count int
}

func (b *Builder) saveWords(dir string) {
	var words []BuilderWordCount
	for word, tw := range b.words {
		words = append(words, BuilderWordCount{[]byte(word), tw.wordCount})
	}

	sort.Slice(words, func(i, j int) bool {
		return words[i].Count > words[j].Count
	})

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}

	fn := "wordcounts.json"
	b.printf("[checkpoint] writing words and counts to %s", fn)

	buf, err := json.MarshalIndent(words, "", " ")
	if err != nil {
		log.Fatalln(err)
	}

	err = ioutil.WriteFile(filepath.Join(dir, fn), buf, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
}

// --

type tokenizedWord struct {
	tokenized  []string
	tokenPairs map[MergedPair][][2]int
	wordCount  int

	origPairs     map[MergedPair][][2]int
	origTokenized []string
}

// newTokenizedWord constructs a tokenized word with the provided tokenization
func newTokenizedWord(word []string) *tokenizedWord {
	tw := &tokenizedWord{
		tokenized: append([]string{}, word...),
	}
	tw.computePairs()
	return tw
}

func (t *tokenizedWord) tokens() []string {
	return t.tokenized
}

func (t *tokenizedWord) contains(p MergedPair) bool {
	_, ok := t.tokenPairs[p]
	return ok
}

func (t *tokenizedWord) incr(val int) {
	t.wordCount += val
}

// TODO(tarak): This is where all the heavy lifting happens. Can probably make this more
// efficient; can certainly parallelize where its called in Merge. But works for now.
func (t *tokenizedWord) mergePair(p MergedPair) {
	// Copy prev state
	copy(t.origTokenized, t.tokenized)
	t.origPairs = make(map[MergedPair][][2]int)
	for k, v := range t.tokenPairs {
		t.origPairs[k] = v
	}

	for {
		pairs := t.tokenPairs[p]
		if len(pairs) == 0 {
			return
		}

		pairIdx := pairs[0]
		orig := t.tokenized
		t.tokenized = t.tokenized[:pairIdx[0]]
		t.tokenized = append(t.tokenized, orig[pairIdx[0]]+orig[pairIdx[1]])
		t.tokenized = append(t.tokenized, orig[pairIdx[1]+1:]...)
		t.computePairs()
	}
}

func (t *tokenizedWord) findPairDeltas() map[MergedPair]int {
	deltas := make(map[MergedPair]int)
	for p, origLocs := range t.origPairs {
		newLocs := t.tokenPairs[p]
		deltas[p] = (len(newLocs) - len(origLocs)) * t.wordCount
	}
	for p, newLocs := range t.tokenPairs {
		if _, ok := deltas[p]; !ok {
			origLocs := t.origPairs[p]
			deltas[p] = (len(newLocs) - len(origLocs)) * t.wordCount
		}
	}
	return deltas
}

func (t *tokenizedWord) computePairs() {
	t.tokenPairs = make(map[MergedPair][][2]int, len(t.tokenized)-1)
	var lastChunk string
	for idx, r := range t.tokenized {
		if idx == 0 {
			lastChunk = r
			continue
		}

		p := MergedPair{lastChunk, r, lastChunk + r}
		t.tokenPairs[p] = append(t.tokenPairs[p], [2]int{idx - 1, idx})
		lastChunk = r
	}
}

// MergedPair has the two parents and the joined entry
type MergedPair struct {
	Parent1 string
	Parent2 string
	Joined  string
}

type pairAndCount struct {
	pair  MergedPair
	count int
	index int
}

func (b *Builder) printf(msg string, objs ...interface{}) {
	if b.debug {
		log.Printf(msg, objs...)
	}
}

type pairCountHeap []*pairAndCount

func (h pairCountHeap) Len() int { return len(h) }

func (h pairCountHeap) Less(i, j int) bool {
	if h[i].count == h[j].count {
		return h[i].pair.Joined > h[j].pair.Joined
	}
	return h[i].count > h[j].count
}

func (h pairCountHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *pairCountHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	n := len(*h)
	pac := x.(*pairAndCount)
	pac.index = n
	*h = append(*h, pac)
}

func (h *pairCountHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	return x
}

func splitWord(w string, useBytes bool) []string {
	if useBytes {
		// raw bytes
		toks := make([]string, 0, len(w))
		for i := 0; i < len(w); i++ {
			// NOTE: do not do `string(w[i])`, this does some weird things that can potentially add bytes ...
			// E.g: calling `fmt.Printf("%x -> %x", byte('\xe2'), string(byte('\xe2')))` yields `e2 -> c3a2`
			conv := string([]byte{w[i]})
			toks = append(toks, conv)
		}
		return toks
	}

	// utf8 code points
	return strings.Split(w, "")
}
