package pythonindex

import (
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

func newGraphIndex(graph pythonresource.Manager, symbolToIdentCounts map[string][]*IdentCount, path string) *index {
	invertedIndex := make(map[string][]*IdentCount)

	err := serialization.Decode(path, &invertedIndex)
	if err != nil {
		log.Printf("cannot load inverted index from %s:\n", path)
		return &index{
			invertedIndex: invertedIndex,
		}
	}

	seen := make(map[*IdentCount]struct{})
	for _, ics := range invertedIndex {
		for _, ic := range ics {
			if _, checked := seen[ic]; checked {
				continue
			}
			sym, err := graph.PathSymbol(pythonimports.NewDottedPath(ic.Ident))
			if err == nil {
				appendNode(sym.Canonical().PathString(), ic, symbolToIdentCounts)
			}
			seen[ic] = struct{}{}
		}
	}

	return &index{
		invertedIndex: invertedIndex,
	}
}
