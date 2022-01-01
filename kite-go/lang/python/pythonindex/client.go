package pythonindex

import (
	"os"
	"sort"
	"strings"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/diskmap"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const defaultMaxResults = 6

var (
	// DefaultClientOptions contains default values for Client.
	DefaultClientOptions = ClientOptions{
		MinCoverage:         0.85,
		MinOccurrence:       15,
		IndexCuration:       true,
		IndexDocs:           false,
		UseStemmer:          true,
		GraphIdentifierPath: pythonimports.DefaultImportGraphIndex,
	}
)

type strFinder func(string) []string

// ClientOptions defines the parameters used by the client.
type ClientOptions struct {
	MinCoverage         float64
	MinOccurrence       int
	IndexCuration       bool
	IndexDocs           bool
	UseStemmer          bool
	GraphIdentifierPath string
}

// Client contains an inverted index that maps from parts of an identifier name to node (or nodeCount actually)
// in the import graph. This inverted index is used to support active search. It takes an active search query
// and returns node ids in the graph that the active search query refers to.
type Client struct {
	packageStats *index
	graph        *index
	curation     *index
	docs         *index
	suffixArray  *suffixArray

	opts *ClientOptions
}

// NewClient returns a pointer to a new client object.
func NewClient(manager pythonresource.Manager, packageStatsPath string, curated map[int64]*pythoncuration.Snippet, opts *ClientOptions) *Client {
	client := Client{
		opts: opts,
	}

	symbolToIdentCounts := make(map[string][]*IdentCount)
	client.packageStats = newPackageStatsIndex(manager, packageStatsPath, symbolToIdentCounts)

	client.graph = newGraphIndex(manager, symbolToIdentCounts, opts.GraphIdentifierPath)

	if opts.IndexCuration {
		client.curation = newCurationIndex(manager, curated, symbolToIdentCounts, opts.UseStemmer)
	}

	if opts.IndexDocs {
		client.docs = newDocsIndex(manager, symbolToIdentCounts, opts.UseStemmer)
	}

	// We build suffix array from packageStats and graph indices.
	var tokens []string
	for t := range client.packageStats.invertedIndex {
		tokens = append(tokens, t)
	}

	for t := range client.graph.invertedIndex {
		tokens = append(tokens, t)
	}

	client.suffixArray = newSuffixArray(tokens)

	return &client
}

// NewClientFromDiskmap returns a new client from a diskmap which contains the
// inverted index. An optional cache can also be provided.
// TODO(naman) unused: rm (and Cleanup) unless we decide to turn local code search back on
func NewClientFromDiskmap(dm *diskmap.Map, cache *lru.Cache, opts *ClientOptions) *Client {
	client := Client{
		opts:  opts,
		graph: newDiskmapIndex(dm, cache),
	}

	tokens, _ := dm.Keys()
	client.suffixArray = newSuffixArray(tokens)

	return &client
}

// Cleanup removes any temporary state associated with this Client
func (c *Client) Cleanup() error {
	if c.graph.diskIndex != nil {
		return os.Remove(c.graph.diskIndex.index.Path())
	}
	return nil
}

// QueryCompletionResult contains an identifier that completes an input query.
// It contains the identifier used to lookup the value as well as a version
// used for display.
type QueryCompletionResult struct {
	Display string
	Ident   string
}

// QueryCompletion returns identifiers that complete the input query.
func (c *Client) QueryCompletion(query string) []*QueryCompletionResult {
	return c.QueryCompletionLimit(query, defaultMaxResults)
}

// QueryCompletionLimit returns identifiers that complete the input query, up
// to the specified maximum number of results.
func (c *Client) QueryCompletionLimit(query string, limit int) []*QueryCompletionResult {
	tokens := text.Uniquify(strings.Split(text.Normalize(strings.ToLower(query)), " "))
	var identCounts []*IdentCount
	if c.packageStats != nil {
		identCounts = c.search(tokens, c.packageStats, c.suffixArray.prefixedBy)
	}
	if len(identCounts) == 0 {
		identCounts = c.search(tokens, c.graph, c.suffixArray.prefixedBy)
	}

	var completions []*QueryCompletionResult
	for _, ic := range c.prune(query, identCounts, limit) {
		if ic.Locator != "" {
			completions = append(completions, &QueryCompletionResult{
				Display: ic.Ident,
				Ident:   ic.Locator,
			})
		} else {
			completions = append(completions, &QueryCompletionResult{
				Display: ic.Ident,
				Ident:   ic.Ident,
			})
		}
	}

	return completions
}

// Search returns the identifier names that the query string refers to.
func (c *Client) Search(query string) []string {
	identCounts := c.SearchWithCount(query).IdentCounts

	var idents []string
	for _, nc := range identCounts {
		idents = append(idents, nc.Ident)
	}

	return idents
}

// SearchWithCount returns the node name (along with their counts on github) that the
// given query may refer to.
func (c *Client) SearchWithCount(query string) *CategorizedResults {
	// This can be replaced by text.Tokenize() if we also want to tokenize by "_".
	tokens := text.Uniquify(strings.Split(text.Normalize(strings.ToLower(query)), " "))

	var identCounts []*IdentCount

	// Try matching identifier names.
	if c.packageStats != nil {
		identCounts = c.search(tokens, c.packageStats, func(s string) []string {
			return []string{s}
		})
		if len(identCounts) > 0 {
			return &CategorizedResults{
				Source:      "package states",
				IdentCounts: c.prune(query, identCounts, defaultMaxResults),
			}
		}
	}

	// Try graph-based index.
	if c.graph != nil {
		identCounts = c.search(tokens, c.graph, func(s string) []string {
			return []string{s}
		})
		if len(identCounts) > 0 {
			return &CategorizedResults{
				Source:      "graph",
				IdentCounts: c.prune(query, identCounts, defaultMaxResults),
			}
		}
	}

	// Try curation-title based index
	if c.curation != nil {
		if c.curation.useStemmer {
			tokens = text.SearchTermProcessor.Apply(text.TokenizeWithoutCamelPhrases(query))
		}
		identCounts = c.search(tokens, c.curation, func(s string) []string {
			return []string{s}
		})
		if len(identCounts) > 0 {
			return &CategorizedResults{
				Source:      "curation",
				IdentCounts: c.prune(query, identCounts, defaultMaxResults),
			}
		}
	}

	// Try doc-based index
	if c.docs != nil {
		if c.docs.useStemmer {
			tokens = text.SearchTermProcessor.Apply(text.TokenizeWithoutCamelPhrases(query))
		}
		identCounts = c.search(tokens, c.docs, func(s string) []string {
			return []string{s}
		})
		if len(identCounts) > 0 {
			return &CategorizedResults{
				Source:      "doc",
				IdentCounts: c.prune(query, identCounts, defaultMaxResults),
			}
		}

	}

	return &CategorizedResults{
		Source: "none",
	}
}

// CategorizedResults wraps the search results with its source.
type CategorizedResults struct {
	Source      string
	IdentCounts []*IdentCount
}

// --

func (c *Client) search(tokens []string, idx *index, finder strFinder) []*IdentCount {
	intersect := make(map[*IdentCount]struct{})
	for i, t := range tokens {
		if i == 0 {
			for _, s := range finder(t) {
				cnts, _ := idx.find(s)
				for _, n := range cnts {
					intersect[n] = struct{}{}
				}
			}
			if len(intersect) == 0 {
				break
			}
			continue
		}
		subset := make(map[*IdentCount]struct{})
		for _, s := range finder(t) {
			cnts, _ := idx.find(s)
			for _, n := range cnts {
				if _, found := intersect[n]; found {
					subset[n] = struct{}{}
				}
			}
		}
		intersect = subset
		if len(intersect) == 0 {
			break
		}
	}
	var identCounts []*IdentCount
	for n := range intersect {
		identCounts = append(identCounts, n)
	}
	return identCounts
}

func (c *Client) prune(query string, identCounts []*IdentCount, limit int) []*IdentCount {
	seen := make(map[string]struct{})
	var uniqueIdents []*IdentCount

	for _, ic := range identCounts {
		if _, found := seen[ic.Ident]; !found {
			seen[ic.Ident] = struct{}{}
			uniqueIdents = append(uniqueIdents, ic)
		}
	}

	// Sort the candidates by their count.
	sort.Sort(sort.Reverse(byForcedCount(uniqueIdents)))

	var total int
	for _, nc := range uniqueIdents {
		total += nc.ForcedCount
	}
	threshold := float64(total) * c.opts.MinCoverage

	// Check whether query is the prefix of the identifier, if it is then
	// promote that identifier.
	query = strings.ToLower(query)
	var ptr int
	for i, nc := range uniqueIdents {
		if strings.HasPrefix(strings.ToLower(nc.Ident), query) {
			uniqueIdents[ptr], uniqueIdents[i] = uniqueIdents[i], uniqueIdents[ptr]
			ptr++
		}
	}

	var topCandidates []*IdentCount
	var acc int

	for _, nc := range uniqueIdents {
		if nc.ForcedCount >= c.opts.MinOccurrence || strings.HasPrefix(nc.Ident, "builtins") {
			topCandidates = append(topCandidates, nc)
		}
		acc += nc.ForcedCount
		if float64(acc) >= threshold {
			break
		}
	}

	if len(topCandidates) > limit {
		return topCandidates[:limit]
	}
	return topCandidates
}

// --

// IdentCount wraps a node name and its count on github.
// ForcedCount is the count that we force the identifier to have.
//
// For local code indexes, an IdentCount may contain a value locator too.
type IdentCount struct {
	Ident       string
	Count       int
	ForcedCount int
	Locator     string
}

type byForcedCount []*IdentCount

func (nc byForcedCount) Len() int           { return len(nc) }
func (nc byForcedCount) Swap(i, j int)      { nc[j], nc[i] = nc[i], nc[j] }
func (nc byForcedCount) Less(i, j int) bool { return nc[i].ForcedCount < nc[j].ForcedCount }
