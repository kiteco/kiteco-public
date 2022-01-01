package pythondocs

import (
	"strconv"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/diskmap"
	"github.com/kiteco/kiteco/kite-golib/diskmap/mock"
)

// MockDocumentation constructs an empty canonical docs structure
func MockDocumentation(graph *pythonimports.Graph, methods ...string) *Corpus {
	corpus := &Corpus{
		Entities: make(map[*pythonimports.Node]*LangEntity),
		graph:    graph,
	}
	for _, method := range methods {
		n, _ := graph.Find(method)
		if n == nil {
			continue
		}
		corpus.Entities[n] = &LangEntity{
			Ident: method,
			Kind:  UnknownKind,
			Doc:   "docs for " + method,
			StructuredDoc: &StructuredDoc{
				Ident:           method,
				DescriptionHTML: "<body><p>" + method + "</p></body>",
			},
		}
	}
	corpus.dm = createDiskmap(corpus.Entities)
	cache, err := lru.New(100)
	if err != nil {
		panic(err)
	}
	corpus.lru = cache
	return corpus
}

func createDiskmap(entities map[*pythonimports.Node]*LangEntity) diskmap.Getter {
	m := make(map[string][]byte)
	for node, entity := range entities {
		key := strconv.FormatInt(node.ID, 10)
		bytes, err := diskmap.JSON.Marshal(entity)
		if err != nil {
			// nothing sensible to do for this test function besides immediately alerting the tester
			panic(err)
		}
		m[key] = bytes
	}
	return mock.New(m)
}
