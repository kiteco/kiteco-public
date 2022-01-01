package data

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
)

// Completion is a replace position (where to insert), and a snippet to insert.
// It is only meaningful when paired with an initial buffer state.
type Completion struct {
	Snippet Snippet   `json:"snippet"`
	Replace Selection `json:"replace"`
}

// Placeholder constants used to distinguish between concrete and abstract placeholders.
// We don't use this for now, but keep the logic in case we want to change it later.
const (
	PlaceholderBeginMark = ""
	PlaceholderEndMark   = ""
)

// Empty checks if c is an empty completion.
// Note non-empty completions may have no effect when applied to a specific Buffer.
func (c Completion) Empty() bool {
	return c.Snippet.Text == "" && c.Replace.Len() == 0
}

// Prepend is used to extend the snippet text for surfacing completions in editors where prefix matching determines completion ranking.
func (c Completion) Prepend(prefix string) Completion {
	plen := len(prefix)
	ptxt := prefix + c.Snippet.Text
	pbegin := c.Replace.Begin - plen

	// Shift placeholders to the right to accommodate new prefix
	ph := make([]Selection, 0)
	for _, p := range c.Snippet.placeholders {
		ph = append(ph, Selection{p.Begin + plen, p.End + plen})
	}

	return Completion{
		Snippet: Snippet{ptxt, ph},
		Replace: Selection{pbegin, c.Replace.End},
	}
}

// HasMultiIdents returns whether there are more than one {keyword | literal}
func (c Completion) HasMultiIdents() (bool, error) {
	words, err := pythonscanner.Lex([]byte(c.Snippet.Text), pythonscanner.DefaultOptions)
	if err != nil {
		return false, errors.Errorf("Could not lex completion.")
	}
	nIdents := 0
	for _, word := range words {
		if word.Token.IsKeyword() || word.Token.IsLiteral() {
			nIdents++
		}
		if nIdents > 1 {
			return true, nil
		}
	}
	return false, nil
}

// SingleToken is used to truncate an existing completion to one token.
func (c Completion) SingleToken() (Completion, error) {
	words, err := pythonscanner.Lex([]byte(c.Snippet.Text), pythonscanner.DefaultOptions)
	if err != nil {
		return c, errors.Errorf("Could not lex completion.")
	}
	we := int(words[0].End)
	ph := make([]Selection, 0)
	for _, p := range c.Snippet.placeholders {
		if p.End < we {
			ph = append(ph, p)
		}
	}
	return Completion{
		Snippet: Snippet{c.Snippet.Text[:we], ph},
		Replace: c.Replace,
	}, nil
}

// MustAfter panics if the completions cannot compose
func (c Completion) MustAfter(d Completion) Completion {
	e, err := c.After(d)
	if err != nil {
		panic(err)
	}
	return e
}

// After composes Completions: applying c.After(d) is equivalent to applying c after applying d
// TODO(naman) holy fuck, unit test this
func (c Completion) After(d Completion) (Completion, error) {
	first := d
	second := c

	// s0 is the initial buffer state
	// sD is the buffer state after insertion of first
	// sC is the buffer state after insertion of first & second

	// position in s1 of the text inserted by first
	s1First := Selection{
		Begin: first.Replace.Begin,
		End:   first.Replace.Begin + len(first.Snippet.Text),
	}

	repl := first.Replace
	var text string
	switch {
	case second.Replace.End < s1First.Begin, second.Replace.Begin > s1First.End:
		// no overlap: nothing to do
		return Completion{}, errors.Errorf("cannot compose disjoint completions")
	case s1First.Begin <= second.Replace.Begin && second.Replace.End <= s1First.End:
		// second is entirely within first
		overlapBegin := second.Replace.Begin - s1First.Begin
		overlapEnd := overlapBegin + second.Replace.Len()

		// replace is already correct
		text = first.Snippet.Text[:overlapBegin] + second.Snippet.Text + first.Snippet.Text[overlapEnd:]
	case second.Replace.Begin <= s1First.Begin && s1First.End <= second.Replace.End:
		// second entirely surrounds first
		repl.Begin = first.Replace.Begin - (s1First.Begin - second.Replace.Begin)
		repl.End = first.Replace.End + (second.Replace.End - s1First.End)
		text = second.Snippet.Text
	case second.Replace.Begin < s1First.Begin:
		// second overlaps first on the left
		lenBeforeOverlap := s1First.Begin - second.Replace.Begin
		overlapEnd := second.Replace.End - s1First.Begin

		repl.Begin = first.Replace.Begin - lenBeforeOverlap
		text = second.Snippet.Text + first.Snippet.Text[overlapEnd:]
	case second.Replace.End > s1First.End:
		// second overlaps first on the right
		lenAfterOverlap := second.Replace.End - s1First.End
		overlapBegin := second.Replace.Begin - s1First.Begin

		repl.End = first.Replace.End + lenAfterOverlap
		text = first.Snippet.Text[:overlapBegin] + second.Snippet.Text
	}

	// compute placeholder positions in s2
	var phs []Selection
	for _, ph := range first.Snippet.placeholders {
		ph1 := ph.Offset(first.Replace.Begin)
		if ph1.Begin >= second.Replace.Begin && ph1.End <= second.Replace.End {
			continue
		}
		if ph1.End <= second.Replace.Begin {
			ph2 := ph1
			phs = append(phs, ph2)
		}
		if ph1.Begin >= second.Replace.End {
			ph2 := ph1.Offset(len(second.Snippet.Text) - second.Replace.Len())
			phs = append(phs, ph2)
		}
		// otherwise the placeholder in s1 overlaps with the text replaced by second, so we drop it
	}

	for _, ph := range second.Snippet.placeholders {
		ph2 := ph.Offset(second.Replace.Begin)
		phs = append(phs, ph2)
	}

	// compute placeholder positions relative to start of final snippet text
	for i := range phs {
		phs[i] = phs[i].Offset(-repl.Begin)
	}

	sort.Slice(phs, func(i, j int) bool {
		// since all placeholders must be disjoint, just compare Begins
		return phs[i].Begin < phs[j].Begin
	})

	composed := Completion{
		Snippet: Snippet{Text: text, placeholders: phs},
		Replace: repl,
	}

	return composed, nil
}

func trailingIdent(s string) string {
	t := s
	for len(t) > 0 {
		r, sz := utf8.DecodeLastRuneInString(t)
		if r == utf8.RuneError || sz == 0 {
			// bad utf8
			break
		}
		if !pythonscanner.IsLetter(r) && !pythonscanner.IsDigit(r) {
			break
		}
		t = t[:len(t)-sz]
	}
	return s[len(t):]
}

// ExactCaseMatchPrecedingIdent checks if a completion matches the preceding identifier exactly (case sensitive).
// `c` is assumed to already have been validated.
func (c Completion) ExactCaseMatchPrecedingIdent(b SelectedBuffer) bool {
	typedPrefix := b.Buffer.TextAt(Selection{c.Replace.Begin, b.Selection.Begin})
	ident := trailingIdent(typedPrefix)
	if len(c.Snippet.Text) < len(typedPrefix) {
		return false
	}
	return c.Snippet.Text[len(typedPrefix)-len(ident):len(typedPrefix)] == ident
}

// ExactCaseMatchSuffix checks if a completion matches the suffix exactly (case sensitive).
// `c` is assumed to already have been validated.
func (c Completion) ExactCaseMatchSuffix(b SelectedBuffer) bool {
	typedSuffix := b.Buffer.TextAt(Selection{b.Selection.End, c.Replace.End})
	if typedSuffix == "" {
		return false
	}
	if !isAlphaNumeric(typedSuffix) {
		return false
	}
	return strings.HasSuffix(strings.TrimSpace(c.Snippet.Text), typedSuffix)
}

// Validate checks if the Completion "reasonably" completes the given Buffer/Selection.
// It does not check for "empty" completions.
// An extra filtering pass is necessary to remove completions with empty DisplayText().
func (c Completion) Validate(b SelectedBuffer) (Completion, bool) {
	if ((strings.HasPrefix(c.Snippet.Text, "[") && strings.HasSuffix(c.Snippet.Text, "]")) ||
		(strings.HasPrefix(c.Snippet.Text, "\"") && strings.HasSuffix(c.Snippet.Text, "\""))) &&
		len(c.Snippet.placeholders) == 0 {
		return c.validateDictCompletion(b)
	}

	if b.Selection.Len() > 0 {
		if c.Replace == b.Selection {
			return c, true
		}
		return Completion{}, false
	}

	if !c.Replace.Contains(b.Selection) {
		return Completion{}, false
	}

	// We don't trim the suffix, as that affects tabstop semantics.
	// Instead, we trim it for display purposes at the end.
	typedSuffix := b.Buffer.TextAt(Selection{b.Selection.End, c.Replace.End})
	// trim space from snippet text because keywords etc can insert spaces
	if !strings.HasSuffix(strings.TrimSpace(c.Snippet.Text), typedSuffix) {
		return Completion{}, false
	}

	typedPrefix := b.Buffer.TextAt(Selection{c.Replace.Begin, b.Selection.Begin})
	identBeforeCursor := strings.ToLower(trailingIdent(typedPrefix))
	nonIdentPrefix := typedPrefix[:len(typedPrefix)-len(identBeforeCursor)]

	// the non-identifier prefix has to be an exact match
	if !strings.HasPrefix(c.Snippet.Text, nonIdentPrefix) {
		return Completion{}, false
	}

	// TODO(naman) don't modify the prefix here, and instead do it in a single pass at render time
	var prefixAdded int
	if len(nonIdentPrefix) == 0 {
		// no non-ident prefix:
		// include the ident before the cursor in the display text
		extra := trailingIdent(b.Buffer.TextAt(Selection{0, c.Replace.Begin}))
		c.Snippet.Text = extra + c.Snippet.Text
		prefixAdded = len(extra)
	} else {
		// remove the non-ident prefix
		c.Snippet.Text = c.Snippet.Text[len(nonIdentPrefix):]
		prefixAdded = -len(nonIdentPrefix)
	}
	c.Replace.Begin -= prefixAdded

	// the identifier before the cursor has to be a case-insensitive match
	if len(c.Snippet.Text) < len(identBeforeCursor) {
		return Completion{}, false
	}
	if strings.ToLower(c.Snippet.Text[:len(identBeforeCursor)]) != identBeforeCursor {
		return Completion{}, false
	}

	// ensure that the non-identifier prefix didn't contain any placeholders
	if prefixAdded != 0 {
		// we must update the placeholders because we mutated the snippet text
		placeholders := make([]Selection, 0, len(c.Snippet.placeholders))
		bounds := Selection{0, len(c.Snippet.Text)}
		for _, ph := range c.Snippet.placeholders {
			ph = ph.Offset(prefixAdded)
			if !bounds.Contains(ph) {
				return Completion{}, false
			}
			placeholders = append(placeholders, ph)
		}
		c.Snippet.placeholders = placeholders
	}

	return c, true
}

func (c Completion) validateDictCompletion(buffer SelectedBuffer) (Completion, bool) {
	replacedText := buffer.Text()[c.Replace.Begin:c.Replace.End]
	snippetText := c.Snippet.Text

	// We don't want to trim right `.` to avoid strange replacement in the case myDict.get.
	replacedText = strings.TrimRight(replacedText, "[]\"'")
	replacedText = strings.TrimLeft(replacedText, ".[]\"'")
	snippetText = strings.Trim(snippetText, "[]\"'")

	return c, strings.HasPrefix(snippetText, replacedText)
}

// -

// DisplayOptions are the options that we can use to tweak our display text for different situations.
type DisplayOptions struct {
	TrimBeforeEmptyPH bool
	NoUnicode         bool
	// TODO get rid of with GGNN logic
	NoEmptyPH bool
}

// DisplayText computes the display text for the given completion.
// It assumes the completion has already been validated.
func (c Completion) DisplayText(b SelectedBuffer, opts DisplayOptions) string {
	c, ok := c.Validate(b)
	if !ok {
		return ""
	}

	snip := c.Snippet

	typedSuffix := b.Buffer.TextAt(Selection{b.Selection.End, c.Replace.End})
	if !isDictKey(c) && !isAlphaNumeric(typedSuffix) {
		snip.Text = snip.Text[:len(snip.Text)-len(typedSuffix)]
	}

	// check placeholder bounds and filter out invalid ones
	bounds := Selection{0, len(snip.Text)}
	var validPlaceholders []Selection
	for _, ph := range snip.placeholders {
		if !bounds.Contains(ph) {
			continue
		}

		// exclude empty placeholders at the very end
		if ph.Begin == len(snip.Text) {
			continue
		}

		if opts.TrimBeforeEmptyPH && ph.Len() == 0 {
			snip.Text = snip.Text[:ph.Begin]
			break
		}

		validPlaceholders = append(validPlaceholders, ph)
	}
	snip.placeholders = validPlaceholders

	ellipsis := "…"
	lineSep := " ⏎ "
	if opts.NoUnicode {
		lineSep = " \\n "
		ellipsis = "..."
	}
	display := snip.Text
	if !opts.NoEmptyPH {
		display = snip.Delimit("", "", ellipsis).Text
	}
	if !opts.NoUnicode {
		display = strings.Replace(display, "...", ellipsis, -1)
	}

	// trim all trailing white spaces on the right (including new lines)
	display = strings.TrimRightFunc(display, unicode.IsSpace)
	// trim the indentation/whitespace for each line of the completion
	parts := strings.Split(display, "\n")
	var trimmed []string
	// don't trim the leading (first line) whitespace
	trimmed = append(trimmed, strings.TrimRightFunc(parts[0], unicode.IsSpace))
	for i := 1; i < len(parts); i++ {
		trimmed = append(trimmed, strings.TrimSpace(parts[i]))
	}
	display = strings.Join(trimmed, lineSep)

	return display
}

// --

// EncodeOffsets encodes the Completion offsets according to the given text & encoding.
func (c *Completion) EncodeOffsets(text string, from, to stringindex.OffsetEncoding) error {
	err := c.Replace.EncodeOffsets(text, from, to)
	if err != nil {
		return err
	}
	return c.Snippet.EncodeOffsets(from, to)
}

// --

// python variable names can contain letters, digits, and underscore, but cannot begin with
// a digit
var pyVarPattern = regexp.MustCompile("^[a-zA-Z0-9_]*$")

func isAlphaNumeric(text string) bool {
	return pyVarPattern.MatchString(text)
}

func isDictKey(c Completion) bool {
	if ((strings.HasPrefix(c.Snippet.Text, "[") && strings.HasSuffix(c.Snippet.Text, "]")) ||
		(strings.HasPrefix(c.Snippet.Text, "\"") && strings.HasSuffix(c.Snippet.Text, "\""))) &&
		len(c.Snippet.placeholders) == 0 {
		return true
	}
	return false
}
