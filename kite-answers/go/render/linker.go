package render

import (
	"fmt"
	"regexp"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

/*
 Syntax for links to Kite docs. We capture the link text in [] as well as the symbol path.
 [`baz`](kite-sym:foo.bar.baz)
*/
var linkRE = regexp.MustCompile("(?i)\\[([^\\[\\]]+?)\\]\\(kite-sym:([\\w\\d\\.]+?)\\)")

func link(links []Link, rm pythonresource.Manager, input []byte) ([]byte, []Link) {
	linkIndices := linkRE.FindAllSubmatchIndex(input, -1)
	offset := 0
	for _, indices := range linkIndices {
		if indices == nil {
			continue
		}
		// Update valid indices w.r.t transformed links
		for i := range indices {
			indices[i] += offset
		}
		// Grab link text and path
		text := string(input[indices[2]:indices[3]])
		path := string(input[indices[4]:indices[5]])

		// Validate path and construct link
		sym := validatePath(rm, path)
		links = append(links, Link{path, sym})

		var link []byte
		if !sym.Nil() {
			link = linkify(path, text)
		} else {
			// If invalid, transform into <a> without a href, so preview CSS can highlight decorate.
			link = []byte(fmt.Sprintf("<a>%s</a>", text))
		}
		// Replace Markdown link syntax with transformed link.
		var prefix = make([]byte, indices[0])
		copy(prefix, input[0:indices[0]])
		suffix := input[indices[1]:len(input)]
		input = append(prefix, link...)
		input = append(input, suffix...)
		offset += len(link) - (indices[1] - indices[0])
	}
	return input, links
}

// Validates candidate string and other combination's in Kite's Python resource manager.
func validatePath(rm pythonresource.Manager, candidate string) pythonresource.Symbol {
	// Check path without mods
	sym, err := rm.PathSymbol(pythonimports.NewDottedPath(candidate))
	if err == nil {
		return sym
	}
	return pythonresource.Symbol{}
}

// Constructs Kite Doc link given path and text
func linkify(path string, text string) []byte {
	url := fmt.Sprintf("https://kite.com/python/docs/%s", path)
	link := []byte(fmt.Sprintf("[%s](%s)", text, url))
	return link
}
