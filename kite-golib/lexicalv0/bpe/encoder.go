package bpe

import (
	"encoding/json"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// Encoder ...
type Encoder struct {
	entries   []Entry
	vocab     []string
	vocabMap  map[string]int
	bpToEntry map[string]Entry

	// TODO: remove this once we move everything to the
	// byte based vocabs, for now we leave this here
	// so the localtraining pipeline works properly
	useBytes bool
}

// Entries ...
func (e *Encoder) Entries() []Entry {
	return copyVocab(e.entries)
}

// Entry ...
func (e *Encoder) Entry(bp string) (Entry, bool) {
	entry, ok := e.bpToEntry[bp]
	return entry, ok
}

// Vocab returns the list of vocabulary of the BPE encoder
func (e *Encoder) Vocab() []string {
	return e.vocab
}

// VocabMap ...
func (e *Encoder) VocabMap() map[string]int {
	return e.vocabMap
}

// NewEncoderFromVocab ...
func NewEncoderFromVocab(entries []Entry) *Encoder {
	sort.Stable(SortBytePair(entries))

	var useBytes bool
	if len(entries) > 0 {
		useBytes = len(entries[0].BytePairBytes) > 0
	}

	enc := &Encoder{
		entries:   entries,
		vocab:     vocabToList(entries),
		vocabMap:  vocabMap(entries),
		bpToEntry: bpToEntry(entries),
		useBytes:  useBytes,
	}
	return enc
}

// Size ...
func (e *Encoder) Size() int {
	return len(e.vocab)
}

// NewEncoder ...
func NewEncoder(mapping string) (*Encoder, error) {
	vocab, err := readVocab(mapping)
	if err != nil {
		return nil, err
	}
	return NewEncoderFromVocab(vocab), nil
}

// Encode ...
func (e *Encoder) Encode(words []string) []string {
	var tokens []string
	for _, w := range words {
		tokens = append(tokens, e.encodeWord(w)...)
	}
	return tokens
}

// EncodeIdx ...
func (e *Encoder) EncodeIdx(words []string) []int {
	var idx []int
	for _, enc := range e.Encode(words) {
		idx = append(idx, e.vocabMap[enc])
	}
	return idx
}

func (e *Encoder) encodeWord(word string) []string {
	return e.encodeWordImpl(word)
}

type subSolution struct {
	encodingLength int
	wordLength     int
}

func (e *Encoder) encodeWordImpl(word string) []string {
	quick := e.bruteForceEncode(word, 4)
	if quick != nil {
		return quick
	}

	subs := make([][]subSolution, len(word))
	for i := range subs {
		subs[i] = make([]subSolution, len(word)+1)
	}
	for k := 1; k <= len(word); k++ {
		for i := 0; i+k <= len(word); i++ {
			sub := word[i : i+k]
			if _, ok := e.vocabMap[sub]; ok {
				subs[i][i+k] = subSolution{
					encodingLength: 1,
					wordLength:     len(sub),
				}
				continue
			}
			if k == 1 {
				// no encoding is possible in this case
				return nil
			}
			sol := subSolution{
				// initialize with encodingLength larger than the minimal encoding length
				// minimal encoding length <= len(word)
				encodingLength: len(word) + 1,
			}
			for j := 1; j < k; j++ {
				left := subs[i][i+j].encodingLength
				right := subs[i+j][i+k].encodingLength
				cutLength := left + right
				if cutLength < sol.encodingLength {
					sol.wordLength = j
					sol.encodingLength = cutLength
				}
			}
			subs[i][i+k] = sol
		}
	}
	var sol []string
	var right int
	for right < len(word) {
		left := right
		right += subs[left][len(word)].wordLength
		sol = append(sol, word[left:right])
	}
	return sol
}

func (e *Encoder) bruteForceEncode(word string, maxLength int) []string {
	for i := 1; i <= maxLength; i++ {
		encoding := e.maybeEncode(word, i)
		if encoding != nil {
			return encoding
		}
	}
	return nil
}

func (e *Encoder) maybeEncode(word string, length int) []string {
	if length == 1 {
		if _, ok := e.vocabMap[word]; ok {
			return []string{word}
		}
		return nil
	}
	for i := 1; i < len(word); i++ {
		if _, ok := e.vocabMap[word[:i]]; !ok {
			continue
		}
		right := e.maybeEncode(word[i:], length-1)
		if right == nil {
			continue
		}
		var ans []string
		ans = append(ans, word[:i])
		ans = append(ans, right...)
		return ans
	}
	return nil
}

// --

func readVocab(mapping string) ([]Entry, error) {
	f, err := fileutil.NewCachedReader(mapping)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var vocab []Entry
	err = json.NewDecoder(f).Decode(&vocab)
	if err != nil {
		return nil, err
	}

	// set BytePair if needed
	for i, entry := range vocab {
		if entry.BytePair == "" {
			vocab[i].BytePair = string(entry.BytePairBytes)
		}
	}

	return vocab, nil
}

func vocabMap(vocab []Entry) map[string]int {
	m := make(map[string]int)
	for idx, v := range vocab {
		m[v.Pair()] = idx
	}
	return m
}

func bpToEntry(vocab []Entry) map[string]Entry {
	m := make(map[string]Entry)
	for _, v := range vocab {
		m[v.Pair()] = v
	}
	return m
}

func vocabToList(vocab []Entry) []string {
	var ret []string
	for _, v := range vocab {
		ret = append(ret, v.Pair())
	}
	return ret
}

func copyVocab(vocab []Entry) []Entry {
	dest := make([]Entry, len(vocab))
	copy(dest, vocab)
	return dest
}
