package pythoncuration

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/serialization"
	yaml "gopkg.in/yaml.v1"
)

var (
	// DefaultSearchOptions contains default options for the Searcher
	DefaultSearchOptions = SearchOptions{
		CurationRoot:             "s3://kite-emr/datasets/curated-snippets/2016-08-22_13-36-00-PM/",
		PassiveRankerRoot:        "s3://kite-emr/rankers/passive-code-examples/2016-01-25_05-23-48-PM",
		CollectionToPackagesPath: "s3://kite-emr/datasets/collection-to-package/2016-03-01_12-52-48-PM/collections.yaml",
		LRUCacheSize:             100,
		BenchmarkMode:            false,
	}
)

// Searcher handles all curated example retrival requests
type Searcher struct {
	cache              *lru.Cache
	curatedMap         map[int64]*Snippet
	relatedIndex       map[int64][]int64
	curatedMethodIndex map[int64][]*Snippet
	tracedReferences   map[int64]*dynamicanalysis.ResolvedSnippet
	canonicalMap       map[int64][]*Snippet
	sampleFiles        map[string][]byte

	graph   *pythonimports.Graph
	nodeMap map[int64]string

	passiveRanker *Ranker

	benchmarkMode bool
}

// SearchOptions contain the configuration of the searcher
type SearchOptions struct {
	CurationRoot             string
	PassiveRankerRoot        string
	CollectionToPackagesPath string

	LRUCacheSize  int
	BenchmarkMode bool
}

// NewSearcher returns a pointer to a Searcher object
func NewSearcher(graph *pythonimports.Graph, opts *SearchOptions) (*Searcher, error) {
	if opts == nil {
		opts = &SearchOptions{}
		*opts = DefaultSearchOptions
	}
	cache, err := lru.New(opts.LRUCacheSize)
	if err != nil {
		return nil, fmt.Errorf("error creating lru cache: %v", err)
	}

	searcher := &Searcher{
		cache:              cache,
		curatedMap:         make(map[int64]*Snippet),
		relatedIndex:       make(map[int64][]int64),
		tracedReferences:   make(map[int64]*dynamicanalysis.ResolvedSnippet),
		curatedMethodIndex: make(map[int64][]*Snippet),
		canonicalMap:       make(map[int64][]*Snippet),
		sampleFiles:        make(map[string][]byte),
		nodeMap:            make(map[int64]string),
	}

	curatedSnippets := fileutil.Join(opts.CurationRoot, "curated-snippets.emr")
	tracedReferences := fileutil.Join(opts.CurationRoot, "traced-references.json.gz")
	relatedExamples := fileutil.Join(opts.CurationRoot, "related-examples.json.gz")
	sampleFiles := fileutil.Join(opts.CurationRoot, "sample-files.json.gz")

	if err := searcher.loadSampleFiles(sampleFiles); err != nil {
		return nil, fmt.Errorf("error loading sample files: %v", err)
	}
	if err := searcher.loadCurated(curatedSnippets, opts.CollectionToPackagesPath, graph); err != nil {
		return nil, err
	}
	if err := searcher.loadTracedReferences(tracedReferences); err != nil {
		return nil, fmt.Errorf("error loading traced-references file from %s: %v", tracedReferences, err)
	}
	searcher.indexCurated(graph)
	if err := searcher.loadRelatedExamples(relatedExamples); err != nil {
		return nil, err
	}

	log.Println("loaded", len(searcher.curatedMap), "curated snippets")
	log.Println(len(searcher.relatedIndex), "curated snippets have related examples")

	var passiveRanker *Ranker
	if opts.PassiveRankerRoot != "" {
		rankerPath := fileutil.Join(opts.PassiveRankerRoot, "model.json")
		featurerPath := fileutil.Join(opts.PassiveRankerRoot, "featurer.gob")

		var err error
		passiveRanker, err = NewRankerFromFile(rankerPath, featurerPath)

		if err != nil {
			return nil, fmt.Errorf("cannot load passive ranking model: %v", err)
		}
	}
	searcher.passiveRanker = passiveRanker
	searcher.benchmarkMode = opts.BenchmarkMode
	searcher.UseGraph(graph)

	return searcher, nil
}

func (s *Searcher) loadSampleFiles(path string) error {
	file, err := fileutil.NewCachedReader(path)
	if err != nil {
		return fmt.Errorf("error loading sample files from %s: %v", path, err)
	}
	defer file.Close()
	gzip, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("error reading as gzip: %v", err)
	}
	defer gzip.Close()
	r := json.NewDecoder(gzip)
	err = r.Decode(&s.sampleFiles)
	if err != nil {
		return fmt.Errorf("error decoding map of sample file data: %v", err)
	}
	return nil
}

// Find returns code examples that contain the method as one of its incantations.
func (s *Searcher) Find(method string) ([]*Snippet, bool) {
	node, err := s.graph.Find(method)
	if err != nil || node == nil {
		return nil, false
	}
	return s.Examples(node)
}

// Examples returns code examples that contain references to the given node
// path is used to rank the examples by relevance and can be empty
func (s *Searcher) Examples(node *pythonimports.Node) ([]*Snippet, bool) {
	cacheKey := fmt.Sprintf("find-%d", node.ID)
	if snippets, cached := s.checkCache(cacheKey); cached {
		return snippets, true
	}

	method := node.CanonicalName.String()
	snippets, found := s.curatedMethodIndex[node.ID]
	if s.passiveRanker != nil {
		var references []*dynamicanalysis.ResolvedSnippet
		for _, snip := range snippets {
			references = append(references, s.tracedReferences[snip.Curated.Snippet.SnippetID])
		}
		snippets = s.passiveRanker.Rank(method, snippets, references)
		// Promote canonical examples.
		if index := strings.Index(method, "."); index != -1 {
			pkg := method[:index]
			if pkgNode, err := s.graph.Find(pkg); err == nil {
				if canonicals, found := s.canonicalMap[pkgNode.ID]; found {
					s.promoteCanonical(canonicals, snippets)
				}
			}
		}
	}

	// Try canonical examples.
	if !found {
		snippets, found = s.Canonical(method)
		sort.Sort(rankCanonicals(snippets))
	}
	s.putCache(cacheKey, snippets)
	return snippets, found
}

// promoteCanonical checks whether any of the snippets are canonical examples.
// If they're, move them to the top.
func (s *Searcher) promoteCanonical(canonicals, snippets []*Snippet) []*Snippet {
	var promoted []*Snippet
	var regular []*Snippet

	for _, snip := range snippets {
		var isCanonical bool
		for _, c := range canonicals {
			if c == snip {
				isCanonical = true
				break
			}
		}
		if isCanonical {
			promoted = append(promoted, snip)
		} else {
			regular = append(regular, snip)
		}
	}
	sort.Sort(rankCanonicals(promoted))

	// Copy snippets in the new order.
	snippets = snippets[:0]
	for _, s := range promoted {
		snippets = append(snippets, s)
	}
	for _, s := range regular {
		snippets = append(snippets, s)
	}
	return snippets
}

type rankCanonicals []*Snippet

// We rank canonical examples by
// 1) Code width
// 2) Code Width
// 3) Title Lenght
// 4) Title alphabetically
func (b rankCanonicals) Len() int      { return len(b) }
func (b rankCanonicals) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b rankCanonicals) Less(i, j int) bool {
	if b[i].Snippet.Width == b[j].Snippet.Width {
		if b[i].Snippet.NumLines == b[j].Snippet.NumLines {
			if len(b[i].Curated.Snippet.Title) == len(b[j].Curated.Snippet.Title) {
				return b[i].Curated.Snippet.Title < b[j].Curated.Snippet.Title
			}
			return len(b[i].Curated.Snippet.Title) < len(b[j].Curated.Snippet.Title)
		}
		return b[i].Snippet.NumLines < b[j].Snippet.NumLines
	}
	return b[i].Snippet.Width < b[j].Snippet.Width
}

// FindByID returns a single curated snippet by its database ID or nil if the
// snippet does not exist
func (s *Searcher) FindByID(id int64) (*Snippet, bool) {
	snippet, found := s.curatedMap[id]
	return snippet, found
}

// Related returns related code examples for a curated snippet
func (s *Searcher) Related(snippet *Snippet) []*Snippet {
	var relatedSnippets []*Snippet
	for _, id := range s.relatedIndex[snippet.Curated.Snippet.SnippetID] {
		if snip, found := s.curatedMap[id]; found {
			relatedSnippets = append(relatedSnippets, snip)
		}
	}
	return relatedSnippets
}

// Canonical returns code examples that are considered canonical for the provided package.
func (s *Searcher) Canonical(pkg string) ([]*Snippet, bool) {
	node, err := s.graph.Find(pkg)
	if err != nil {
		return nil, false
	}
	snippets, exists := s.canonicalMap[node.ID]
	return snippets, exists
}

// AllCurated returns a map containing all curated snippets in a map indexed by ID
func (s *Searcher) AllCurated() map[int64]*Snippet {
	return s.curatedMap
}

// UseGraph builds an internal canonicalization map using the provided import graph.
// This will be used for all subsequent `Find` operations.
func (s *Searcher) UseGraph(graph *pythonimports.Graph) {
	s.graph = graph
	for _, curated := range s.curatedMap {
		for _, inc := range curated.Snippet.Incantations {
			node, err := s.graph.Find(inc.ExampleOf)
			if err != nil {
				continue
			}
			s.nodeMap[node.ID] = inc.ExampleOf
		}
	}
}

// --

func (s *Searcher) canonicalize(method string) string {
	if s.graph != nil {
		if node, err := s.graph.Find(method); err == nil {
			if name, exists := s.nodeMap[node.ID]; exists {
				method = name
			}
		}
	}
	return method
}

func (s *Searcher) checkCache(query string) ([]*Snippet, bool) {
	if s.benchmarkMode {
		return nil, false
	}
	if s.cache == nil {
		return nil, false
	}
	if val, exists := s.cache.Get(query); exists {
		return val.([]*Snippet), true
	}
	return nil, false
}

func (s *Searcher) putCache(query string, snippets []*Snippet) {
	if !s.benchmarkMode && s.cache != nil && len(snippets) > 0 {
		s.cache.Add(query, snippets)
	}
}

type curatedByScore []*Snippet

func (p curatedByScore) Len() int      { return len(p) }
func (p curatedByScore) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p curatedByScore) Less(i, j int) bool {
	return len(p[i].Snippet.Incantations) < len(p[j].Snippet.Incantations)
}

func (s *Searcher) indexCurated(graph *pythonimports.Graph) {
	type apparatusSpec struct {
		RelevantTo []string `yaml:"relevant_to"`
	}

	for id, traced := range s.tracedReferences {
		cs, found := s.curatedMap[id]
		// Can't find the curated example by its id.
		if !found {
			continue
		}

		var snippetPackageNodes []*pythonimports.Node
		for _, pkg := range strings.Split(cs.Curated.Snippet.RelevantPackages, ",") {
			node, err := graph.Find(pkg)
			if err != nil {
				continue
			}
			snippetPackageNodes = append(snippetPackageNodes, node)
		}

		// Index this example for the relevant identifers
		var spec apparatusSpec
		if err := yaml.Unmarshal([]byte(cs.Curated.Snippet.ApparatusSpec), &spec); err == nil {
			for _, ref := range spec.RelevantTo {
				refNode, err := graph.Find(ref)
				if err != nil {
					continue
				}
				s.curatedMethodIndex[refNode.ID] = append(s.curatedMethodIndex[refNode.ID], cs)
			}
			// The relevant_to list is a white list that shows what identifiers an example
			// is relevant to. If such a list is provided, then we do not index this example
			// for other identifiers.
			if len(spec.RelevantTo) > 0 {
				continue
			}
		}

		seen := make(map[int64]struct{})
		for _, ref := range traced.References {
			refNode, err := graph.Find(ref.FullyQualifiedName)
			if err != nil {
				continue
			}
			if _, exists := seen[refNode.ID]; exists {
				continue
			}
			index := strings.Index(ref.FullyQualifiedName, ".")
			if index == -1 {
				continue
			}
			refPackageNode, err := graph.Find(ref.FullyQualifiedName[:index])
			if err != nil {
				continue
			}
			for _, snippetPackageNode := range snippetPackageNodes {
				if refPackageNode == snippetPackageNode {
					s.curatedMethodIndex[refNode.ID] = append(s.curatedMethodIndex[refNode.ID], cs)
					break
				}
			}
			seen[refNode.ID] = struct{}{}
		}
	}
	for _, c := range s.curatedMethodIndex {
		sort.Sort(sort.Reverse(curatedByScore(c)))
	}
}

func (s *Searcher) loadTracedReferences(path string) error {
	err := serialization.Decode(path, func(traced *dynamicanalysis.ResolvedSnippet) {
		s.tracedReferences[traced.SnippetID] = traced
	})
	return err
}

func (s *Searcher) loadCurated(curated, mappingPath string, graph *pythonimports.Graph) error {
	relevantPackages := loadCollectionToPackages(mappingPath)
	type apparatusSpec struct {
		Canonical bool `yaml:"canonical"`
	}

	file, err := fileutil.NewCachedReader(curated)
	if err != nil {
		return fmt.Errorf("error loading curated snippets from %s: %v", curated, err)
	}
	defer file.Close()
	r := awsutil.NewEMRIterator(file)

	for r.Next() {
		var cs Snippet
		if err := json.Unmarshal(r.Value(), &cs); err != nil {
			return err
		}
		if cs.Curated == nil {
			log.Printf("[warning] loaded a Snippet with Curated=nil:\n%+v\n", cs)
			continue
		}
		if cs.Curated.Snippet == nil {
			log.Printf("[warning] loaded a Snippet with Curated.Snippet=nil:\n%+v\n", cs)
			continue
		}
		if _, ok := s.curatedMap[cs.Curated.Snippet.SnippetID]; ok {
			log.Printf("[warning] loading code snippets with the same id: %d\n", cs.Curated.Snippet.SnippetID)
			continue
		}

		// Find relevant packages for cs.Curated.Snippet.Package
		packages := []string{cs.Curated.Snippet.Package}
		if cp, found := relevantPackages[cs.Curated.Snippet.Package]; found {
			packages = cp.RelevantPackages
		}

		var spec apparatusSpec
		if err := yaml.Unmarshal([]byte(cs.Curated.Snippet.ApparatusSpec), &spec); err == nil {
			if spec.Canonical {
				for _, pkg := range packages {
					if pkgNode, err := graph.Find(pkg); err == nil {
						s.canonicalMap[pkgNode.ID] = append(s.canonicalMap[pkgNode.ID], &cs)
					}
				}
			}
		}

		cs.Curated.Snippet.RelevantPackages = strings.Join(packages, ",")
		s.curatedMap[cs.Curated.Snippet.SnippetID] = &cs
	}
	if err := r.Err(); err != nil {
		return fmt.Errorf("error reading emr: %v", err)
	}
	return nil
}

func (s *Searcher) loadRelatedExamples(path string) error {
	file, err := fileutil.NewCachedReader(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decomp, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer decomp.Close()

	decoder := json.NewDecoder(decomp)

	for {
		var re curation.RelatedExamples
		err := decoder.Decode(&re)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		s.relatedIndex[re.SnippetID] = re.Examples
	}
	return nil
}
