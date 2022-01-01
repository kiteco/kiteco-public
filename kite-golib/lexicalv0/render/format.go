package render

import (
	"bytes"
	"io"
	"strings"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// OffsetMapping maps a byte offset position before and after a call to Prettify.
// For each piece of content (usually a token), we keep both the start and end positions
type OffsetMapping struct {
	StartBefore, StartAfter, EndBefore, EndAfter int
}

// MatchOption is an option for matching start or end of a token when using position mapping
type MatchOption bool

const (
	// MatchStart ...
	MatchStart = MatchOption(true)
	// MatchEnd ...
	MatchEnd = MatchOption(false)
)

type prettifyFn func(io.Writer, []byte, *sitter.Node) ([]OffsetMapping, error)

// We we transform positions mappings, startOfToken controls
// if we match the start or the end of token.
// For example, if we are at `x :=$`, and the completion is `make(map[int]bool)`
// The prettifier will render the snippet as `x := make(map[int]bool)`
// In this case, we want to match the end of the `:=` token, so that we include the
// leading space in the returned completion like ` make(map[int]bool)`
// However, if we are at `x := $`, then we want to match the start of
// `make` token because we the user has already typed the space.
func transformPosition(pos int, startOfToken MatchOption, mappings []OffsetMapping) int {
	newPos := -1
	for _, mapping := range mappings {
		var before int
		var after int
		if startOfToken {
			before = mapping.StartBefore
			after = mapping.StartAfter
		} else {
			before = mapping.EndBefore
			after = mapping.EndAfter
		}

		if before == pos {
			newPos = after
			break
		}
	}
	return newPos
}

// placeholder selections are relative to the snippet, but the mappings of
// the prettified source is relative to the full parsed buffer. This placeholder-
// specific transform function adjusts the positions accordingly. The phOffset
// value should be the start of the replacement selection in the buffer.
//
// E.g. given the following buffer and replacement selection ($):
//   if (x) { $ }
//            ^ replacement start = 9
// And the following snippet before prettify with placeholder ($):
//   return yz($)
//             ^ placeholder start = 10
// The actual placeholder position before prettify is:
//   if (x) { return yz($) }
//                      ^ ph start + repl start = 19
// And let's say the prettifier adds a space before the call and inside
// the parens, the post-prettifying looks like this:
//   if (x) { return yz ( $) }
//                        ^ post-prettify start = 21
// The offset is then subtracted to get the final, post-prettify, relative
// to the snippet, placeholder position:
//   return yz ( $)
//               ^ post-prettify start - repl start = 12
func transformPlaceholderWithPrettifiedMappings(ph data.Selection,
	originalBegin, newBegin int, match MatchOption, mappings []OffsetMapping) data.Selection {
	// note that the placeholder offset may be different after the prettifying,
	// e.g. if there is extraneous whitespace in the original source, it will
	// get removed after prettifying and the snippet's start position will be
	// smaller than before prettifying - for that reason we get the placeholder's
	// offset post-prettify by transforming its offset itself.
	ph.Begin += originalBegin
	ph.End += originalBegin
	newPHBegin := transformPosition(ph.Begin, match, mappings)
	newPHEnd := transformPosition(ph.End, MatchEnd, mappings)
	if newPHBegin == -1 || newPHEnd == -1 {
		return data.Selection{Begin: -1, End: -1}
	}
	newPHBegin -= newBegin
	newPHEnd -= newBegin
	return data.Selection{Begin: newPHBegin, End: newPHEnd}
}

// FormatCompletion pretty-prints the snippet's text and returns the
// transformed data.Completion with placeholders' position adjusted.
func FormatCompletion(input string, c data.Completion, lang *sitter.Language, match MatchOption, prettify prettifyFn) data.Snippet {
	if c.Replace.Begin < 0 || c.Replace.End > len(input) || c.Replace.End < c.Replace.Begin {
		rollbar.Error(errors.New("invalid completion replace bounds"), c)
		return c.Snippet
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)
	src := []byte(input[:c.Replace.Begin] + c.Snippet.Text + input[c.Replace.End:])
	tree := parser.Parse(src)
	defer tree.Close()

	var buf bytes.Buffer
	// There might be situations that confuse the pretty printer due to invalid AST tree
	// If that happens, just return what the traditional rendering gives us
	mappings, err := prettify(&buf, src, tree.RootNode())
	if err != nil {
		return c.Snippet
	}
	// adjust the end, because the replace selection is based on what's in
	// the user's buffer, but what was prettified is the user's buffer
	// concatenated with the snippet.
	repl := c.Replace
	repl.End += len(c.Snippet.Text) - (repl.End - repl.Begin)

	repl.Begin = transformPosition(repl.Begin, match, mappings)
	repl.End = transformPosition(repl.End, MatchEnd, mappings)
	if repl.Begin == -1 || repl.End == -1 {
		return c.Snippet
	}

	newSnippet := buf.String()[repl.Begin:repl.End]
	if strings.TrimSpace(newSnippet) == "" {
		// handle cases where the treesitter could not properly parse the resulting code,
		// in which case we return the snippet as it was - better than returning an
		// empty string.
		newSnippet = c.Snippet.Text
	}

	// adjust the placeholders' positions - the placeholders are relative to the
	// inserted snippet, but the mappings we have are related to the whole input,
	// so in order to translate the placeholders, we have to first adjust them
	// to the snippet's position in the whole input, and then translate them to
	// the prettified mappings, and adjust them back to be relative to the new
	// snippet.
	phs := make([]data.Selection, 0, len(c.Snippet.Placeholders()))
	var last int
	for _, ph := range c.Snippet.Placeholders() {
		var match MatchOption
		// For nonempty tokens, we look for start and end
		if ph.Begin != ph.End {
			match = true
		}
		newph := transformPlaceholderWithPrettifiedMappings(ph, c.Replace.Begin, repl.Begin, match, mappings)
		if newph.Begin == -1 || newph.End == -1 {
			return c.Snippet
		}
		if newph.Begin < last || newph.End > len(newSnippet) || newph.End < newph.Begin {
			// ignore invalid placeholders that overlap, are out of bounds or
			// go backwards - something must've gone wrong either with the
			// original placeholders, or in the mappings during prettifying
			rollbar.Error(errors.New("FormatCompletion: invalid placeholder"), ph, newph, last, newSnippet)
			continue
		}
		last = newph.End
		phs = append(phs, newph)
	}

	// punch holes in the snippet based on the adjusted placeholder positions
	parts := make([]string, 0, 2*len(phs)+1)
	var lastIx int
	for _, ph := range phs {
		parts = append(parts, newSnippet[lastIx:ph.Begin])
		parts = append(parts, data.Hole(newSnippet[ph.Begin:ph.End]))
		lastIx = ph.End
	}
	parts = append(parts, newSnippet[lastIx:])

	return data.BuildSnippet(strings.Join(parts, ""))
}
