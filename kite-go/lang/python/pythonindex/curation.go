package pythonindex

import (
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	minWordCount = 5
)

func newCurationIndex(rm pythonresource.Manager, curated map[int64]*pythoncuration.Snippet,
	symbolToIdentCounts map[string][]*IdentCount, useStemmer bool) *index {

	c := &index{
		useStemmer: useStemmer,
	}

	c.invertedIndex = make(map[string][]*IdentCount)

	for _, cs := range curated {
		nameToIdentCount := make(map[string]*IdentCount)

		// Get the incantations used in this example that belong to the example's indended package.
		for _, inc := range cs.Snippet.Incantations {
			// Create only one IdentCount object for an incantation name.
			if _, found := nameToIdentCount[inc.ExampleOf]; found {
				continue
			}
			if strings.Split(inc.ExampleOf, ".")[0] == cs.Curated.Snippet.Package {
				if ic, found := findIdentCount(rm, inc.ExampleOf, symbolToIdentCounts); found {
					nameToIdentCount[inc.ExampleOf] = &IdentCount{
						Ident:       ic.Ident,
						Count:       ic.Count,
						ForcedCount: ic.ForcedCount,
					}
				}
			}
		}
		// Index the incantations as entries for tokens in the titles.
		// We lower-case, stem, uniquify, and remove stop words from the title.
		var tokens []string
		if c.useStemmer {
			tokens = text.SearchTermProcessor.Apply(text.Tokenize(cs.Curated.Snippet.Title))
		} else {
			tokens = text.Uniquify(text.TokenizeWithoutCamelPhrases(strings.ToLower(cs.Curated.Snippet.Title)))
		}
		for _, t := range tokens {
			for _, inc := range cs.Snippet.Incantations {
				if strings.Split(inc.ExampleOf, ".")[0] != cs.Curated.Snippet.Package {
					continue
				}
				ic, found := nameToIdentCount[inc.ExampleOf]
				if !found {
					continue
				}
				c.invertedIndex[t] = append(c.invertedIndex[t], ic)
			}
		}
	}

	for token, ics := range c.invertedIndex {
		if len(ics) < minWordCount {
			delete(c.invertedIndex, token)
		}
	}

	return c
}

// we will not need this function once static analysis is used in the curation pipeline.
func findIdentCount(rm pythonresource.Manager, inc string, symbolToIdentCounts map[string][]*IdentCount) (*IdentCount, bool) {

	sym, err := resolve(rm, inc)
	if err != nil {
		return nil, false
	}

	for _, ic := range symbolToIdentCounts[sym.Canonical().PathString()] {
		if ic.Ident == inc {
			return ic, true
		}
	}

	return nil, false
}
