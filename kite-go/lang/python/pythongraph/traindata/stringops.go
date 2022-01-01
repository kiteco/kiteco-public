package traindata

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// SplitNameLiteral into subtokens
func SplitNameLiteral(lit string) []string {

	if IsSpecialToken(lit) {
		return []string{lit}
	}

	isUnderscore := func(c rune) bool {
		return c == '_'
	}

	runes := []rune(lit)

	var parts []string
	start, i := 0, 1
	for ; i < len(runes); i++ {
		prev := runes[i-1]
		cur := runes[i]

		var split bool
		switch {
		case isUnderscore(prev) && isUnderscore(cur):
			start = i + 1
			continue
		case isUnderscore(prev) && !isUnderscore(cur):
			start = i
			continue
		case !isUnderscore(prev) && isUnderscore(cur):
			split = true

		// neither cur, prev are '_'
		case unicode.IsDigit(prev) && unicode.IsDigit(cur):
			continue
		case unicode.IsDigit(prev) && !unicode.IsDigit(cur):
			split = true
		case !unicode.IsDigit(prev) && unicode.IsDigit(cur):
			split = true

		// neither cur, prev are digits
		case unicode.IsLower(prev) && unicode.IsLower(cur):
			continue
		case unicode.IsUpper(prev) && unicode.IsUpper(cur):
			continue
		case unicode.IsUpper(prev) && unicode.IsLower(cur):
			continue
		case unicode.IsLower(prev) && unicode.IsUpper(cur):
			split = true

		// at least one of cur, prev is non-latin
		case !unicode.Is(unicode.Latin, prev) && !unicode.Is(unicode.Latin, cur):
			continue
		case unicode.Is(unicode.Latin, prev) && !unicode.Is(unicode.Latin, cur):
			split = true
		case !unicode.Is(unicode.Latin, prev) && unicode.Is(unicode.Latin, cur):
			split = true

		default:
			rollbar.Error(fmt.Errorf("error splitting name literal"), map[string]interface{}{
				"i":     i,
				"start": start,
				"prev":  prev,
				"cur":   cur,
				"lit":   lit,
			})
			return []string{lit}
		}

		if split {
			if part := string(runes[start:i]); len(part) > 0 {
				parts = append(parts, strings.ToLower(part))
			}
			start = i

			if isUnderscore(cur) {
				start = i + 1
			}
		}
	}

	if start < len(runes) {
		if part := string(runes[start:i]); len(part) > 0 {
			parts = append(parts, strings.ToLower(part))
		}
	}

	if len(parts) == 0 {
		// only way this can happen is if name is all _
		return []string{strings.ToLower(lit)}
	}

	return parts
}
