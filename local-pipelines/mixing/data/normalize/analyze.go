package normalize

import (
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/python"
)

func process(code, cursor string) (data.SelectedBuffer, string, string) {
	parts := strings.Split(code, cursor)
	beforeCursor, afterCursor := parts[0], parts[1]
	beforeLines := strings.Split(beforeCursor, "\n")
	afterLines := strings.Split(afterCursor, "\n")
	before := beforeLines[len(beforeLines)-1]
	after := afterLines[0]
	buf := data.NewBuffer(beforeCursor)
	sel := data.Cursor(len(beforeCursor))
	return buf.Select(sel), before, after
}

func match(completion data.Snippet, before, after string) (matchMetrics, error) {
	line := before + after
	cursorPosition := len(before)

	// compute:
	// - totalSize: the length of the completion, excluding placeholders
	// - expression: a reg exp to find the completion
	var left, totalSize int
	var parts []string
	for _, ph := range completion.Placeholders() {
		right := ph.Begin
		parts = append(parts, regexp.QuoteMeta(completion.Text[left:right]))
		totalSize += right - left
		left = ph.End
	}
	parts = append(parts, regexp.QuoteMeta(completion.Text[left:]))
	totalSize += len(completion.Text) - left
	expression := strings.Join(parts, ".*")

	// find all matches for the completion in the line. there may be multiple matches.
	// for example if the line is "nums = sorted(set(n$ums))" and the completion is "nums"
	re, err := regexp.Compile(expression)
	if err != nil {
		return matchMetrics{}, err
	}
	locs := re.FindAllStringIndex(line, -1)

	// iterate through the matches found and check how many chars after
	// the cursor match, not counting characters in the prefix.
	var matchChars int
	for _, loc := range locs {
		if loc == nil {
			continue
		}
		prefixSize := cursorPosition - loc[0]
		if prefixSize < 0 {
			// the cursor is before the beginning of the completion,
			// so this is not a relevant match
			continue
		}
		diff := totalSize - prefixSize
		if diff <= 0 {
			// at least one of the following is the case:
			// - the cursor is after the end of the completion,
			//   so this match is not relevant.
			// - the completion provides no value
			continue
		}
		if diff > matchChars {
			matchChars = diff
		}
	}
	if matchChars == 0 {
		return matchMetrics{}, nil
	}
	var matchIdentifiers, matchKeywords int
	pyLexer := python.Lexer{}
	tokens, err := pyLexer.Lex([]byte(completion.Text))
	if err != nil {
		return matchMetrics{}, err
	}
	for _, token := range tokens {
		if pyLexer.IsType(lexer.IDENT, token) {
			matchIdentifiers++
			continue
		}
		if pyLexer.IsType(lexer.KEYWORD, token) {
			matchKeywords++
		}
	}
	return matchMetrics{
		characters:   matchChars,
		placeholders: len(completion.Placeholders()),
		identifiers:  matchIdentifiers,
		keywords:     matchKeywords,
	}, nil
}
