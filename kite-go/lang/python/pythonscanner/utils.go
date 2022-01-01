package pythonscanner

import (
	"fmt"
	"strconv"
	"strings"
)

// IsValidIdent returns true in the case that the provided string
// is a valid python identifier (starts with a letter and contains only letters or digits), and false otherwise.
func IsValidIdent(ident string) bool {
	if len(ident) == 0 {
		return false
	}
	for i, r := range ident {
		if i == 0 {
			if !IsLetter(r) {
				return false
			}
		}
		if !IsLetter(r) && !IsDigit(r) {
			return false
		}
	}
	return true
}

// QuoteString quotes a string with the given quote (`'` or `"`).
// If a quote is not provided, a reasonable default is chosen.
// We may add support for raw, multiline, etc later.
func QuoteString(quote string, s string) string {
	if quote == "" {
		quote = `"`
		if strings.Contains(s, `"`) && !strings.Contains(s, `'`) {
			quote = `'`
		}
	}

	quoted := strconv.Quote(s)
	switch quote {
	case "\"":
		return quoted
	case "'":
		// Apart from the initial/final `"`, all other quotes are escaped (`\"`)
		// so we can blindly replace `\"` with `"` in the main string
		return "'" + strings.Replace(quoted[1:len(quoted)-1], `\"`, `"`, -1) + "'"
	default:
		panic(fmt.Sprintf("unhandled quote %v", quote))
	}
}
