package text

import (
	"unicode/utf8"
)

// SplitWithOpts ...
func SplitWithOpts(text string, mergeKeywords bool) []string {
	var words []string
	var leftCC, rightCC cc
	var leftRune, rightRune rune
	var consumed int

	var width int
	for i := 0; i < len(text); i += width {
		rightRune, width = utf8.DecodeRuneInString(text[i:])
		rightCC = charCategory(rightRune)

		canMerge := merge(leftCC, leftRune, rightCC, rightRune)
		if i+width >= len(text) {
			if i == 0 {
				// single rune -> single word
				words = append(words, text)
			} else if canMerge {
				// final rune is same category as previous rune -> include as single word
				words = append(words, text[consumed:])
			} else {
				// final rune is different category than previous rune -> 2 separate words
				words = append(words, text[consumed:i])
				words = append(words, text[i:])
			}
			consumed = len(text)
		} else if i != 0 && !canMerge {
			// need i != 0 since leftCC, leftRune not initialized when i == 0
			words = append(words, text[consumed:i])
			consumed = i
		}

		leftCC = rightCC
		leftRune = rightRune
	}

	if !mergeKeywords {
		return words
	}

	// heuristics to ensure that keywords usually end up as a single word,
	// NOTE: this is an approximation, doing this exactly
	// would require actually lexing the text according to the appropriate grammar
	words = mergeWords(words,
		func(curr, next string) bool { return keywords[curr+next] },
		func(curr, next, nextNext string) bool { return keywords[curr+next+nextNext] },
	)

	// heuristics to allow known keywords to merge with horizontal spaces that follow
	// the keyword
	words = mergeWords(words, merge2Spaces, merge3Spaces)

	return words
}

// Split the provided string into words based on a predefined set of rules
func Split(text string) []string {
	return SplitWithOpts(text, false)
}

func merge2Spaces(curr, next string) bool {
	if keywords[curr] && isHSpaceWord(next) {
		return true
	}
	return false
}

func merge3Spaces(curr, next, nextNext string) bool {
	nextSpace := isHSpaceWord(next)
	nextNextSpace := isHSpaceWord(nextNext)
	switch {
	case keywords[curr] && nextSpace && nextNextSpace:
		return true
	default:
		return false
	}
}

type merge2Fn func(curr, next string) bool
type merge3Fn func(curr, next, nextNext string) bool

func mergeWords(words []string, merge2 merge2Fn, merge3 merge3Fn) []string {
	if len(words) == 0 {
		return words
	}

	if merge2 == nil {
		merge2 = func(string, string) bool {
			return false
		}
	}
	if merge3 == nil {
		merge3 = func(string, string, string) bool {
			return false
		}
	}

	merged := make([]string, 0, len(words))
	for i := 0; i < len(words); {
		curr := words[i]
		if i+1 < len(words) {
			next := words[i+1]
			if i+2 < len(words) {
				nextNext := words[i+2]
				if merge3(curr, next, nextNext) {
					merged = append(merged, curr+next+nextNext)
					i += 3
					continue
				}
			}
			if merge2(curr, next) {
				merged = append(merged, curr+next)
				i += 2
				continue
			}
		}

		merged = append(merged, curr)
		i++
	}

	return merged
}

// merge must be transitive, e.g if merge(a,b) && merge(b,c) then must have merge(a,c),
// this is pretty permissive. The main cases we care about are ensuring that:
//  - we always split on spaces
//  - identifiers and punctuation are split
// since these are the most common cases that lead to suboptimal allocation of vocab slots.
func merge(leftCC cc, leftRune rune, rightCC cc, rightRune rune) bool {
	switch {
	case isLetterOrNumber(leftCC) && isPunct(rightCC):
		if rightRune == '_' || rightRune == '-' || rightRune == '#' {
			return true
		}
		return false
	case isLetterOrNumber(rightCC) && isPunct(leftCC):
		if leftRune == '_' || leftRune == '-' || leftRune == '#' {
			return true
		}
		return false
	case isSpace(leftCC) && isSpace(rightCC):
		// split horizontal and vertical spaces
		if leftCC != rightCC {
			return false
		}
		return true
	case isSpace(leftCC) || isSpace(rightCC):
		// always split on spaces, if both are not a space
		return false
	case leftCC == symb && rightCC != symb && !isPunct(rightCC):
		return false
	case !isPunct(leftCC) && leftCC != symb && rightCC == symb:
		return false
	default:
		return true
	}
}
