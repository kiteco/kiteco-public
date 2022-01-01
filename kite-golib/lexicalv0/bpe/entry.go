package bpe

// Entry represents an entry in the BPE vocab.
// During serialization/deserialization only one of BytePair/BytePairBytes will be
// set.
// - for entries that were constructed from a builder with `useBytes == true`
//   then it will be `BytePairBytes`, since we cannot directly serialize
//   these entries using strings because the JSON package does some weird
//   things when serializing strings that hold arbitrary bytes.
// - otherwise BytePair will be set.
type Entry struct {
	BytePair      string
	BytePairBytes []byte
	Count         int
}

// Pair returns a string representation of the byte pair, regardles of original format
func (e Entry) Pair() string {
	if e.BytePair == "" {
		return string(e.BytePairBytes)
	}
	return e.BytePair
}

// SortBytePair impelements sort.Interface to sort by byte-pair
type SortBytePair []Entry

// Len impelements sort.Interface
func (b SortBytePair) Len() int { return len(b) }

// Less impelements sort.Interface
func (b SortBytePair) Less(i, j int) bool {
	if len(b[i].Pair()) == len(b[j].Pair()) {
		return b[i].Pair() > b[j].Pair()
	}
	return len(b[i].Pair()) > len(b[j].Pair())
}

// Swap impelements sort.Interface
func (b SortBytePair) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortPopularity impelements sort.Interface to sort by byte-pair counts
type SortPopularity []Entry

// Len impelements sort.Interface
func (b SortPopularity) Len() int { return len(b) }

// Less impelements sort.Interface
func (b SortPopularity) Less(i, j int) bool {
	if b[i].Count == b[j].Count {
		// Fall back to lexicographic sort if counts are equal
		return SortBytePair(b).Less(i, j)
	}
	return b[i].Count > b[j].Count
}

// Swap impelements sort.Interface
func (b SortPopularity) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
