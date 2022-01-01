package render

import (
	"bytes"
	"regexp"
)

// See https://spec.commonmark.org/0.29/#fenced-code-blocks
// We don't remove indentation of the code block contents if the opening fence is indented,
// but otherwise we should be compliant with the spec.

var codeFenceRE = regexp.MustCompile("(?m)^ {0,3}([`~]{3,})")
var headlineRE = regexp.MustCompile(`(?m)^(?:[\t\f\r ]*\r?\n)* {0,3}# (.+)(?: #+)?\r?$`)

// splitOnCodeFence searches for the next code fence in the given buffer slice. It returns before, fence, info, after where
// - before is the buffer slice before the next code fence
// - fence is the buffer slice containing the ```, ~~~, etc
// - info is the buffer slice containing the text immediately following the fence, trimmed of spaces
// - after is the buffer slice starting at the line following the fence
// return md, nil, nil, nil if no code fence is found
// if openingFence is passed in, search for a matching closing fence
func splitOnCodeFence(md []byte, openingFence []byte) (before []byte, fence []byte, info []byte, after []byte) {
	if len(bytes.TrimSpace(md)) == 0 {
		return nil, nil, nil, nil
	}

	var start int
	for start < len(md) {
		idxs := codeFenceRE.FindSubmatchIndex(md[start:])
		if idxs == nil {
			return md, nil, nil, nil
		}
		start = start + idxs[3] // so that `continue` restarts the search after this candidate

		before := md[:idxs[0]]
		if len(bytes.TrimSpace(before)) == 0 {
			before = nil // don't return extraneous whitespace
		}

		fence := md[idxs[2]:idxs[3]]
		if len(openingFence) > 0 && (len(fence) < len(openingFence) || openingFence[0] != fence[0]) {
			continue // closing fence should use the same character as the opening, and have at least as many characters
		}

		eol := idxs[3]
		for eol < len(md) && md[eol] != '\r' && md[eol] != '\n' {
			eol++
		}
		info := bytes.TrimSpace(md[idxs[3]:eol])
		if fence[0] == '`' && bytes.Contains(info, []byte{'`'}) {
			continue // info strings can't contain ` if the fence character is `
		}
		if len(openingFence) > 0 && len(info) > 0 {
			continue // closing fences shouldn't have info strings
		}

		after := md[eol:]
		if bytes.HasPrefix(after, []byte("\r\n")) {
			after = after[2:]
		} else if len(after) > 0 { // starts with one of "\r" or "\n" by construction
			after = after[1:]
		}
		if len(bytes.TrimSpace(after)) == 0 {
			after = nil // don't return extraneous whitespace
		}

		return before, fence, info, after
	}

	return md, nil, nil, nil
}

func splitOnCodeBlock(md []byte) (before []byte, info []byte, code []byte, after []byte) {
	before, openingFence, info, afterOpening := splitOnCodeFence(md, nil)
	code, _, _, after = splitOnCodeFence(afterOpening, openingFence)
	return before, info, code, after
}

func splitOnHeadline(md []byte) (headline []byte, after []byte) {
	// only look for a heading before a code fence
	before, _, _, _ := splitOnCodeFence(md, nil)

	index := headlineRE.FindIndex(before)
	if index == nil {
		return []byte{}, md
	}
	headline = md[0:index[1]]
	after = md[index[1]:len(md)]
	return headline, after
}
