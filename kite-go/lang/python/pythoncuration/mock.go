package pythoncuration

import (
	"log"

	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

func mockSnippet(method string) *Snippet {
	return &Snippet{
		Curated: &curation.Example{
			Snippet: &curation.CuratedSnippet{
				Title: method,
			},
			Result: &curation.ExecutionResult{},
		},
	}
}

// MockCuration creates a mocked-up searcher.
func MockCuration(graph *pythonimports.Graph, methods ...string) *Searcher {
	curatedMethodIndex := make(map[int64][]*Snippet)
	for _, method := range methods {
		node, err := graph.Find(method)
		if err != nil {
			log.Fatalf("error building curation from methods: %v\n", err)
		}
		curatedMethodIndex[node.ID] = []*Snippet{mockSnippet(method)}
	}

	return &Searcher{
		graph:              graph,
		curatedMap:         make(map[int64]*Snippet),
		relatedIndex:       make(map[int64][]int64),
		curatedMethodIndex: curatedMethodIndex,
		canonicalMap:       make(map[int64][]*Snippet),
		sampleFiles:        make(map[string][]byte),
		nodeMap:            make(map[int64]string),
	}
}

// MockCurationFromMap creates a mocked-up searcher from a map.
func MockCurationFromMap(graph *pythonimports.Graph, methods map[string]pythonimports.Kind) *Searcher {
	curatedMethodIndex := make(map[int64][]*Snippet)
	canonicalMap := make(map[int64][]*Snippet)
	for method, kind := range methods {
		node, err := graph.Find(method)
		if err != nil {
			log.Fatalf("error building curation from map: %v\n", err)
		}
		curatedMethodIndex[node.ID] = []*Snippet{mockSnippet(method)}
		if kind == pythonimports.Module {
			canonicalMap[node.ID] = append(canonicalMap[node.ID], mockSnippet(method))
		}
	}

	return &Searcher{
		graph:              graph,
		curatedMap:         make(map[int64]*Snippet),
		relatedIndex:       make(map[int64][]int64),
		curatedMethodIndex: curatedMethodIndex,
		canonicalMap:       canonicalMap,
		sampleFiles:        make(map[string][]byte),
		nodeMap:            make(map[int64]string),
	}
}
