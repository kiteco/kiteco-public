package data

import (
	"go/token"

	"github.com/kiteco/kiteco/kite-golib/stringindex"
)

// Selection encapsulates begin/end UTF-8 byte offsets
// End must be at least Begin
type Selection struct {
	Begin int `json:"begin"`
	End   int `json:"end"`
}

// NewSelection create a new Selection from 2 positions
func NewSelection(begin, end token.Pos) Selection {
	return Selection{
		Begin: int(begin),
		End:   int(end),
	}
}

// Cursor returns a position with equal Begin/End
func Cursor(n int) Selection {
	return Selection{n, n}
}

// Offset offsets position by a fixed integer value
func (s Selection) Offset(n int) Selection {
	return Selection{Begin: s.Begin + n, End: s.End + n}
}

// Len returns the number of bytes between Begin and End
func (s Selection) Len() int {
	return s.End - s.Begin
}

// Contains checks if s contains t
func (s Selection) Contains(t Selection) bool {
	return t.Begin >= s.Begin && t.End <= s.End
}

// Cursor returns s.End
func (s Selection) Cursor() int {
	return s.End
}

// EncodeOffsets encodes the Selection offsets according to the given text & encoding.
func (s *Selection) EncodeOffsets(text string, from, to stringindex.OffsetEncoding) error {
	conv := stringindex.NewConverter(text)
	begin, err := conv.EncodeOffset(s.Begin, from, to)
	if err != nil {
		return err
	}
	end, err := conv.EncodeOffset(s.End, from, to)
	if err != nil {
		return err
	}
	s.Begin = begin
	s.End = end
	return nil
}
