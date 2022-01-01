package data

import (
	"bytes"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// HolePH is an alias for HoleWithPlaceholderMarks
func HolePH(text string) string {
	return HoleWithPlaceholderMarks(text)
}

// HoleWithPlaceholderMarks returns a Snippet string (see ForFormat) for a single placeholder containing the given text with brackets
// Use the HolePH alias instead. This name is annoyingly long.
func HoleWithPlaceholderMarks(text string) string {
	if len(text) == 0 {
		return Hole(text)
	}
	return Hole(PlaceholderBeginMark + text + PlaceholderEndMark)
}

// Hole returns a Snippet string (see ForFormat) for a single placeholder containing the given text without brackets
func Hole(text string) string {
	return Snippet{
		Text: text,
		placeholders: []Selection{{
			Begin: 0,
			End:   len(text),
		}},
	}.ForFormat()
}

// NewSnippet build a plain snippet without any placeholder
func NewSnippet(str string) Snippet {
	return Snippet{Text: str}
}

// BuildSnippet builds a new snippet from str produced with Snippet.ForFormat & fmt
func BuildSnippet(str string) Snippet {
	var s Snippet
	err := s.parseDelimited(str, internalBeginMark, internalEndMark)
	if err != nil {
		panic(err)
	}
	return s
}

// ForFormat renders a Snippet for formatting with fmt
func (s Snippet) ForFormat() string {
	return s.Delimit(internalBeginMark, internalEndMark, internalBeginMark+internalEndMark).Text
}

var (
	internalBeginMark = "\002"
	internalEndMark   = "\003"
)

func (s *Snippet) parseDelimited(str, beginMark, endMark string) error {
	var b bytes.Buffer
	var phs []Selection

	findUnmatched := func() int {
		for i := len(phs) - 1; i >= 0; i-- {
			if phs[i].End < 0 {
				return i
			}
		}
		return -1
	}

	for {
		begin := strings.Index(str, beginMark)
		end := strings.Index(str, endMark)
		if begin < 0 && end < 0 {
			b.WriteString(str)
			break
		}
		if end < 0 {
			return errors.Errorf("unmatched begin mark")
		}

		if begin >= 0 && begin < end {
			b.WriteString(str[:begin])
			phs = append(phs, Selection{Begin: b.Len(), End: -1})
			str = str[begin+len(beginMark):]
		} else {
			b.WriteString(str[:end])
			i := findUnmatched()
			if i < 0 {
				return errors.Errorf("unmatched end mark")
			}
			phs[i].End = b.Len()
			str = str[end+len(endMark):]
		}
	}
	if findUnmatched() >= 0 {
		return errors.Errorf("unmatched begin mark")
	}

	*s = Snippet{
		Text:         b.String(),
		placeholders: phs,
	}
	return nil
}
