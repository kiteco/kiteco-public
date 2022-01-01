package pythonindex

import (
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	maxLength = 1000
)

var (
	sentenceBoundary = regexp.MustCompile(`\.[\s$]`)
)

func newDocsIndex(manager pythonresource.Manager, symbolToIdentCounts map[string][]*IdentCount, useStemmer bool) *index {

	c := &index{
		useStemmer:    useStemmer,
		invertedIndex: make(map[string][]*IdentCount),
	}

	for pathStr, identCounts := range symbolToIdentCounts {
		for _, ic := range identCounts {
			sym, err := manager.PathSymbol(pythonimports.NewDottedPath(pathStr))
			if err != nil {
				continue
			}
			entity := manager.Documentation(sym)
			if entity == nil {
				continue
			}

			if typeSym, err := manager.Type(sym); err == nil {
				typePath := typeSym.Path()
				if len(typePath.Parts) == 2 {
					switch typePath.Head() {
					case "builtins":
						switch typePath.Parts[1] {
						case "int", "float", "log", "complex":
							continue
						}
					}
				}
			}
			// Index the incantations as entries for tokens in the titles.
			// We lower-case, stem, uniquify, and remove stop words from the title.

			var tokens []string
			if entity.HTML != "" {
				description := cleanHTML(entity.HTML)
				if len(description) > maxLength {
					description = description[:maxLength]
				}
				boundary := sentenceBoundary.FindStringIndex(description)
				if boundary == nil || len(description) == maxLength {
					continue
				}
				description = description[:boundary[0]]
				if c.useStemmer {
					tokens = text.SearchTermProcessor.Apply(text.Tokenize(description))
				} else {
					tokens = text.Uniquify(text.TokenizeWithoutCamelPhrases(strings.ToLower(description)))
				}
			}
			for _, t := range tokens {
				c.invertedIndex[t] = append(c.invertedIndex[t], ic)
			}
		}
	}

	return c
}
