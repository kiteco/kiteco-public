package data

import (
	"bytes"
	"encoding/json"

	"github.com/kiteco/kiteco/kite-golib/stringindex"
)

// Snippet pairs a string with a set of placeholder positions
type Snippet struct {
	Text string
	// placeholders is a sorted list of disjoint selections
	placeholders []Selection
}

// Placeholders returns the locations of all placeholders as Selections.
// It is sorted in increasing order of position. All placeholders are disjoint.
// It excludes the final "implicit" placeholder at the end of the text.
func (s Snippet) Placeholders() []Selection {
	return s.placeholders
}

// UnmarshalJSON implements json.Unmarshaler
func (s *Snippet) UnmarshalJSON(b []byte) error {
	var serdes struct {
		Text         string      `json:"text"`
		Placeholders []Selection `json:"placeholders"`
	}
	err := json.Unmarshal(b, &serdes)
	if err != nil {
		return err
	}
	s.Text = serdes.Text
	s.placeholders = serdes.Placeholders
	return nil
}

// MarshalJSON implements json.Marshaler
func (s Snippet) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"text":         s.Text,
		"placeholders": s.placeholders,
	})
}

// EncodeOffsets encodes the Snippet offsets according to the encoding.
func (s *Snippet) EncodeOffsets(from, to stringindex.OffsetEncoding) error {
	// don't mutate placeholders in-place, since it may be shared
	phs := make([]Selection, 0, len(s.placeholders))
	for _, ph := range s.placeholders {
		if err := ph.EncodeOffsets(s.Text, from, to); err != nil {
			return err
		}
		phs = append(phs, ph)
	}
	s.placeholders = phs
	return nil
}

// RemovePlaceholders removes the given placeholders from the Snippet.
// If a placeholder is not found, it is ignored.
func (s Snippet) RemovePlaceholders(toRemove ...Selection) Snippet {
	var idxs []int
	for i, ph := range s.placeholders {
		for _, sel := range toRemove {
			if ph == sel {
				idxs = append(idxs, i)
			}
		}
	}
	if idxs == nil {
		return s
	}

	var newPHs []Selection
	prevIdx := 0
	for _, idx := range idxs {
		newPHs = append(newPHs, s.placeholders[prevIdx:idx]...)
		// skip idx, start at idx+1
		prevIdx = idx + 1
	}
	newPHs = append(newPHs, s.placeholders[prevIdx:]...)

	s.placeholders = newPHs
	return s
}

// Delimit returns a new Snippet with the provided begin/end marks used to delimit placeholders in the text
func (s Snippet) Delimit(beginMark, endMark, emptyMark string) Snippet {
	res := Snippet{
		placeholders: make([]Selection, 0, len(s.placeholders)),
	}

	var b bytes.Buffer
	b.Grow(len(s.Text) + len(s.placeholders)*(len(beginMark)+len(endMark)))

	var cur int
	for _, ph := range s.placeholders {
		b.WriteString(s.Text[cur:ph.Begin])

		var newPH Selection
		newPH.Begin = b.Len()

		if ph.Len() == 0 {
			b.WriteString(emptyMark)
		} else {
			b.WriteString(beginMark)
			b.WriteString(s.Text[ph.Begin:ph.End])
			b.WriteString(endMark)
		}
		newPH.End = b.Len()

		cur = ph.End
		res.placeholders = append(res.placeholders, newPH)
	}
	b.WriteString(s.Text[cur:])
	res.Text = b.String()

	return res
}

// Iterate iterates over maximal substrings of the Snippet containing no internal placeholder boundaries.
// Empty placeholders are included. Empty non-placeholder substrings are excluded.
func (s Snippet) Iterate(f func(text string, ph bool) bool) bool {
	i := 0
	for _, ph := range s.placeholders {
		if i < ph.Begin && !f(s.Text[i:ph.Begin], false) {
			return false
		}
		if !f(s.Text[ph.Begin:ph.End], true) {
			return false
		}
		i = ph.End
	}
	if i < len(s.Text) && !f(s.Text[i:], false) {
		return false
	}
	return true
}
