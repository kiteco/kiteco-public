package javascript

import (
	"strings"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
)

// FindIndentation returns 0 if no indent could be found, > 0 = number of spaces, < 0 = number of tabs.
func FindIndentation(buf string) int {
	indentStr := render.FindIndentationFromSource(buf)
	if c := strings.Count(indentStr, "\t"); c > 0 {
		return -c
	}

	return strings.Count(indentStr, " ")
}
