package pythondocs

import (
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/text"
	"github.com/kiteco/kiteco/kite-golib/tfidf"
)

// TermIndex index methods using tokens
// in the documentation, docs, so posts and curation examples.
// During search time, it returns method names that contain
// any tokens in the search query.
// The field `DocNames` contains the doc id (names) in the corpus.
type TermIndex struct {
	Index     map[string][]string
	Counts    *tfidf.IDFCounter
	DocNames  []string
	MaxReturn int
}

// NewTermIndex returns a pointer to a new TermIndex
func NewTermIndex(corpus map[string][]string) *TermIndex {
	count := make(map[string]int)
	index := make(map[string][]string)
	seenNames := make(map[string]struct{})

	for name, docs := range corpus {
		seenNames[name] = struct{}{}
		seen := make(map[string]struct{})
		for _, doc := range docs {
			tokens := text.SearchTermProcessor.Apply(text.Tokenize(doc))
			for _, t := range tokens {
				if _, exists := seen[t]; exists {
					continue
				}
				seen[t] = struct{}{}
				// increase doc count for t
				count[t]++
				// index the method name
				index[t] = append(index[t], name)
			}
		}
		for _, part := range text.SearchTermProcessor.Apply(strings.Split(name, ".")) {
			if _, exists := seen[part]; exists {
				continue
			}
			seen[part] = struct{}{}
			count[part]++
			index[part] = append(index[part], name)
		}
	}

	var names []string
	for n := range seenNames {
		names = append(names, n)
	}

	return &TermIndex{
		Index:     index,
		Counts:    tfidf.TrainIDFCounter(len(corpus), count),
		MaxReturn: 100,
		DocNames:  names,
	}
}

type nameCount struct {
	name  string
	count float64
}

type byCount []*nameCount

func (bc byCount) Len() int           { return len(bc) }
func (bc byCount) Swap(i, j int)      { bc[j], bc[i] = bc[i], bc[j] }
func (bc byCount) Less(i, j int) bool { return bc[i].count < bc[j].count }

func (bc byCount) names() []string {
	var nameList []string
	for _, e := range bc {
		nameList = append(nameList, e.name)
	}
	return nameList
}

// Search returns methond names that are indexed for those
// given tokens
func (ti *TermIndex) Search(query string) []string {
	tokens := text.SearchTermProcessor.Apply(text.TokenizeNoCamel(query))

	var countList []*nameCount
	countsMap := make(map[string]*nameCount)

	for _, token := range tokens {
		for _, name := range ti.Index[token] {
			nc, exists := countsMap[name]
			if !exists {
				nc = &nameCount{name: name}
				countList = append(countList, nc)
				countsMap[name] = nc
			}
			nc.count += ti.Counts.Weight(token)
		}
	}
	sort.Sort(sort.Reverse(byCount(countList)))

	if len(countList) < ti.MaxReturn {
		return byCount(countList).names()
	}

	for i := ti.MaxReturn + 1; i < len(countList); i++ {
		if countList[i].count < countList[ti.MaxReturn].count {
			return byCount(countList).names()[:i]
		}
	}
	return byCount(countList).names()
}

// AllMethods returns the names of the docs that are served by this index.
// If the DocNames of the index is not set, it'll find all the doc names
// in the corpus and set the value of DocNames of the index.
func (ti *TermIndex) AllMethods() []string {
	if ti.DocNames == nil {
		seenNames := make(map[string]struct{})
		for _, docs := range ti.Index {
			for _, doc := range docs {
				seenNames[doc] = struct{}{}
			}
		}
		var names []string
		for n := range seenNames {
			names = append(names, n)
		}
		ti.DocNames = names
	}
	return ti.DocNames
}
