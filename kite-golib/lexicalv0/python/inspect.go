package python

import (
	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
)

// IndentInspect takes source code and the position of the start of current line
// and returns the indent string of current file and the depth of current line
func IndentInspect(buf []byte, pos int) (indent string, depth int, err error) {
	return render.IndentInspect(buf, pos, lang.Python, func(n *sitter.Node) bool {
		if render.SafeSymbol(n) == symBlock {
			parent := render.SafeParent(n)
			if render.SafeSymbol(render.SafeParent(parent)) == symModule {
				return true
			}
		}
		return false
	})
}
