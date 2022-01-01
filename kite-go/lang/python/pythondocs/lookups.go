package pythondocs

import (
	"math"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	minLogProb = -30
)

// IdentifierLookupTable builds an inverted index to allow us to look up identifier names that contain all query tokens.
type IdentifierLookupTable struct {
	index    map[string][]string
	prior    *pythoncode.GithubPrior
	nameToID map[string]int64
}

// NewIdentifierLookupTable returns a pointer to a new IdentifierLookupTable object.
func NewIdentifierLookupTable(graph *pythonimports.Graph, packageStats map[string]pythoncode.PackageStats) *IdentifierLookupTable {
	index := make(map[string][]string)
	nameToID := make(map[string]int64)

	// We only look at identifiers that exist in the github corpus
	for _, stats := range packageStats {
		for _, m := range stats.Methods {
			node, _ := graph.Find(m.Ident)
			// We ignore any identifier names that can't be found in the graph.
			if node != nil {
				tokens := text.TokenizeWithoutCamelPhrases(m.Ident)
				for _, t := range tokens {
					index[t] = append(index[t], m.Ident)
				}
				nameToID[m.Ident] = node.ID
			}
		}
	}

	for t, names := range index {
		index[t] = text.Uniquify(names)
	}

	return &IdentifierLookupTable{
		index:    index,
		nameToID: nameToID,
		prior:    pythoncode.NewGithubPrior(graph, packageStats),
	}
}

type identProb struct {
	ident               string
	prob                float64
	optimizedMatchRatio float64
	matchRatio          float64
}

func (i *identProb) score() float64 {
	if i.optimizedMatchRatio > 0 {
		return math.Log(i.optimizedMatchRatio) + i.prob
	}
	return minLogProb + i.prob
}

// lookup finds identifier names that contain all the query tokens.
func (l *IdentifierLookupTable) lookup(query string) []*identProb {
	tokens := text.TokenizeWithoutCamelPhrases(query)
	intersect := make(map[string]struct{})

	for i, t := range tokens {
		if i == 0 {
			for _, name := range l.index[t] {
				intersect[name] = struct{}{}
			}
		} else {
			newSet := make(map[string]struct{})
			for _, name := range l.index[t] {
				if _, exists := intersect[name]; !exists {
					continue
				}
				newSet[name] = struct{}{}
			}
			intersect = newSet
		}
		if len(intersect) == 0 {
			break
		}
	}

	var candidates []*identProb
	for n := range intersect {
		var prob float64
		if l.prior != nil {
			prob = l.prior.Find(n)
		}
		candidates = append(candidates, &identProb{
			ident:               n,
			prob:                prob,
			matchRatio:          computeMatchRatio(text.TokenizeWithoutCamelPhrases(n), tokens),
			optimizedMatchRatio: computeMatchRatio(tokenize(n), tokens),
		})
	}
	sort.Sort(byScore(candidates))

	candidates = l.checkExactMatch(candidates)
	candidates = l.uniquify(candidates)
	return candidates
}

// uniquify removes candidates that correspond to the same node in the graph.
func (l *IdentifierLookupTable) uniquify(candidates []*identProb) []*identProb {
	var uniqueCandidates []*identProb
	seenIDs := make(map[int64]struct{})
	for _, c := range candidates {
		if _, seen := seenIDs[l.nameToID[c.ident]]; !seen {
			seenIDs[l.nameToID[c.ident]] = struct{}{}
			uniqueCandidates = append(uniqueCandidates, c)
		}
	}
	return uniqueCandidates
}

// FindWithScore returns identifier names that contain all the query tokens.
func (l *IdentifierLookupTable) FindWithScore(query string) []*ranking.DataPoint {
	candidates := l.lookup(query)
	var data []*ranking.DataPoint
	for _, c := range candidates {
		data = append(data, &ranking.DataPoint{
			Name:  c.ident,
			Score: c.score(),
		})
	}
	return data
}

// Find returns identifier names that contain all the query tokens.
func (l *IdentifierLookupTable) Find(query string) []string {
	candidates := l.lookup(query)
	var names []string
	for _, c := range candidates {
		names = append(names, c.ident)
	}
	return names
}

// checkExactMatch checks whether there are identifiers that exactly match
// the query tokens. If there are, it'll truncate the list of candidates
// to just those.
func (l *IdentifierLookupTable) checkExactMatch(candidates []*identProb) []*identProb {
	var exactMatches []*identProb
	for _, c := range candidates {
		if c.matchRatio == 1.0 {
			exactMatches = append(exactMatches, c)
		}
	}
	if len(exactMatches) > 0 {
		return exactMatches
	}
	return candidates
}

type byScore []*identProb

func (candidates byScore) Len() int      { return len(candidates) }
func (candidates byScore) Swap(i, j int) { candidates[j], candidates[i] = candidates[i], candidates[j] }

// Less implements how the order of two identProb objects should be determined. We implement it in a way
// that a < b if a should be ranked higher (more relevant) than b.
// There are two steps in the sorting algorithm:
// 1) sort by match ratio.
// 2) sort by probability.
func (candidates byScore) Less(i, j int) bool {
	if candidates[i].optimizedMatchRatio == candidates[j].optimizedMatchRatio {
		return candidates[i].prob > candidates[j].prob
	}
	return candidates[i].optimizedMatchRatio > candidates[j].optimizedMatchRatio
}

// --

// tokenize checks whether any camel phrase is seen if the
// characters in the phrases are lower cased. We don't
// tokenize this type of camel phrases.
// For example, zipfile.ZipFile --> we won't split ZipFile into zip and file.
func tokenize(s string) []string {
	seen := make(map[string]struct{})
	nonCamelPhrases := make(map[string]struct{})

	for _, t := range strings.Split(s, ".") {
		if len(text.TokenizeCamel(t)) == 1 {
			nonCamelPhrases[t] = struct{}{}
			for _, n := range text.TokenizeWithoutCamelPhrases(t) {
				seen[n] = struct{}{}
			}
		}
	}

	// Tokenize camel phrases if needed.
	for _, t := range strings.Split(s, ".") {
		if len(text.TokenizeCamel(t)) != 1 {
			if _, found := nonCamelPhrases[strings.ToLower(t)]; !found {
				for _, n := range text.TokenizeWithoutCamelPhrases(t) {
					seen[n] = struct{}{}
				}
			}
		}
	}

	var tokens []string
	for t := range seen {
		tokens = append(tokens, t)
	}
	return tokens
}

func computeMatchRatio(nameTokens, queryTokens []string) float64 {
	seen := make(map[string]struct{})
	for _, n := range nameTokens {
		seen[n] = struct{}{}
	}

	var count int
	for _, t := range queryTokens {
		if _, found := seen[t]; found {
			count++
		}
	}

	return float64(count) / float64(len(seen))
}
