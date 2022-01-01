package pythondocs

import (
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/diskmap"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

var (
	// DefaultSearchOptions contains default options for the Searcher.
	DefaultSearchOptions = SearchOptions{
		DocPath:               "s3://kite-emr/datasets/documentation/python/2016-08-02_15-42-00-PM/python.gob.gz",
		DocstringsPath:        "s3://kite-emr/datasets/documentation/python/2016-07-18_14-39-29-PM/pythondocstrings.gob.gz",
		DocstringsDiskmapPath: "s3://kite-emr/datasets/documentation/python/2017-01-19_15-03-23-PM/pythondocs.diskmap",
	}

	// Target cache size of ~1000 mb rounded to nearest power of 2. This will be placed in SearchOptions as a follow-up.
	cacheSize = 1 << 17
)

// SearchOptions are the options used to set up the searcher.
type SearchOptions struct {
	DocPath               string
	DocstringsPath        string
	DocstringsDiskmapPath string
}

// Identifier represents a single-identifier name together with the canonical name of its referrent
type Identifier struct {
	Name    string // Name is a single python identifier with no dot
	Ident   string // Ident is the canonical name of the node
	Rel     string // Rel is a fully qualified name for the node (not the canonical name)
	Kind    LangEntityKind
	HasDocs bool    // HasDocs is true if there is a LangEntity for this identifier
	Score   float64 // Score comes from the LangEntity, or is zero is there is no entity
}

type byIdentScore []Identifier

func (xs byIdentScore) Len() int           { return len(xs) }
func (xs byIdentScore) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }
func (xs byIdentScore) Less(i, j int) bool { return xs[i].Score < xs[j].Score }

// Result is the result of looking up a name in the docs corpus. It contains a
// LangEntity together with all ancestors and children
type Result struct {
	Ident      string      // Ident is the canonical name of the queried import graph node
	Entity     *LangEntity // Entity is nil if the node was found but it had no documentation
	Kind       LangEntityKind
	Ancestors  []Identifier
	Children   []Identifier
	References []Identifier
}

// FullIdent gets the canonical name of the node that the result contains documentation for
func (r Result) FullIdent() string {
	return r.Ident
}

// Corpus provides access to documentation
type Corpus struct {
	Entities map[*pythonimports.Node]*LangEntity
	dm       diskmap.Getter
	lru      *lru.Cache
	graph    *pythonimports.Graph
}

func loadDiskmap(opts SearchOptions) (diskmap.Getter, error) {
	localPath, downloadErr := fileutil.DownloadedFile(opts.DocstringsDiskmapPath)
	if downloadErr != nil {
		return nil, downloadErr
	}
	return diskmap.NewMap(localPath)
}

// LoadEntities returns a version of LoadCorpus without a diskmap initialized.
// Do not use. This will go away after the migration to the diskmap is complete.
func LoadEntities(graph *pythonimports.Graph, opts SearchOptions) (*Corpus, error) {
	entities := make(map[*pythonimports.Node]*LangEntity)
	var numNotFound, numEmpty, numTotal, numSkipped int
	for _, path := range []string{opts.DocstringsPath, opts.DocPath} {
		modules, err := NewModules(path)
		if err != nil {
			return nil, err
		}
		modules.Visit(func(entity *LangEntity) {
			numTotal++
			if entity == nil {
				numEmpty++
				return
			}

			ident := entity.FullIdent()
			if ident == "" {
				numEmpty++
				return
			}

			n, ok := graph.FindByID(entity.NodeID)
			if !ok || n == nil {
				// fall back to find by identifier / canonical name
				var err error
				n, err = graph.Find(ident)
				ok = err == nil
			}
			if !ok || n == nil {
				numNotFound++
				return
			}

			if prevEntity, ok := entities[n]; ok {
				prevIdent := prevEntity.FullIdent()
				// if prevEntity is locatable by identifier, then check that the CanonicalName matches the current ident
				// otherwise, the old logic would have thrown out prevEntity, so just merge it with the current entity
				if m, err := graph.Find(prevIdent); err == nil && m == n {
					if cn := n.CanonicalName; !cn.Empty() && ident != cn.String() {
						mod := strings.Split(ident, ".")[0]
						if cn.Head() != mod {
							log.Printf("docs name mismatch: %s != %s", cn.String(), ident)
							numSkipped++
							return
						}
					}
				}
				entity.merge(prevEntity)
			}

			entities[n] = entity
		})
	}

	log.Printf("loaded %d documentation entities, of which %d were empty, %d were missing from graph, %d were skipped due to conflicting names",
		numTotal, numEmpty, numNotFound, numSkipped)

	return &Corpus{
		Entities: entities,
		graph:    graph,
	}, nil
}

// LoadCorpus loads a corpus from the paths specified in opts
func LoadCorpus(graph *pythonimports.Graph, opts SearchOptions) (*Corpus, error) {
	corpus, err := LoadEntities(graph, opts)
	if err != nil {
		return nil, err
	}
	// We are temporarily going to disable diskmap due to excessive latencies with prefetching. We have to
	// re-asses whether this approach will work long term.
	/*
		dm, dmErr := loadDiskmap(opts)
		if dmErr != nil {
			return nil, dmErr
		}
		corpus.dm = dm
	*/
	cache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}
	corpus.lru = cache
	return corpus, nil
}

// Entity gets the entity for an import graph node, or nil if there is no documentation for that node
func (c *Corpus) Entity(n *pythonimports.Node) (*LangEntity, bool) {
	// TODO investigate who is passing nil to see if we can remove this check - @caleb
	if n == nil {
		return nil, false
	}
	defer docsEntityDuration.DeferRecord(time.Now())

	fromCache, foundCache := c.readCache(n)
	if foundCache {
		return fromCache, foundCache
	}

	// TODO(tarak): temporarily disabled due to completions prefetching access pattern
	// fromDM, foundDM := c.readDiskmap(n)

	fromIndex, foundIndex := c.readIndex(n)
	return fromIndex, foundIndex

	/*
		// Ensure the diskmap response equals the index response before we migrate entirely to the diskmap.
		if foundDM && foundIndex {
			if equalEntity(fromDM, fromIndex) {
				// only add a node to the cache if the diskmap entity was equal
				c.lru.Add(n.ID, fromDM)
				docsDiskmapMatchRate.Hit()
				return fromDM, foundDM
			}
			// log.Printf("expected %+v but got %+v", fromIndex, fromDM)
			docsDiskmapMatchRate.Miss()
			return fromIndex, foundIndex
		}

		return fromIndex, foundIndex
	*/
}

func equalEntity(diskEntity, memoryEntity *LangEntity) bool {
	// we mutate the memoryEntity because it's score is calculated later during boot. This is ugly, but
	// once we migrate to the diskmap it won't be needed anymore.
	memoryEntity.Score = diskEntity.Score
	return reflect.DeepEqual(diskEntity, memoryEntity)
}

func (c *Corpus) readIndex(n *pythonimports.Node) (*LangEntity, bool) {
	entity, found := c.Entities[n]
	if found {
		docsIndexRatio.Hit()
	} else {
		docsIndexRatio.Miss()
	}
	return entity, found
}

func (c *Corpus) readCache(node *pythonimports.Node) (*LangEntity, bool) {
	if obj, found := c.lru.Get(node.ID); found {
		docsCacheRatio.Hit()
		return obj.(*LangEntity), true
	}
	docsCacheRatio.Miss()
	return nil, false
}

func (c *Corpus) readDiskmap(n *pythonimports.Node) (*LangEntity, bool) {
	defer docsDiskmapDuration.DeferRecord(time.Now())
	key := strconv.FormatInt(n.ID, 10)
	var obj LangEntity
	err := diskmap.JSON.Get(c.dm, key, &obj)
	if err == nil {
		docsDiskmapRatio.Hit()
		return &obj, true
	}
	docsDiskmapRatio.Miss()
	return nil, false
}

// Ident checks for the existence of an ident and returns its canonical name if it does exist
func (c *Corpus) Ident(ident string) (string, bool) {
	node, err := c.graph.Find(ident)
	if err != nil || node == nil || node.CanonicalName.Empty() {
		return "", false
	}
	return node.CanonicalName.String(), true
}

// FindIdent walks to an identifier and returns its LangEntity together with its ancestors
// and children
func (c *Corpus) FindIdent(ident string) (*Result, bool) {
	// Find the node
	node, err := c.graph.Find(ident)
	if err != nil || node == nil || node.CanonicalName.Empty() {
		return nil, false
	}
	return c.Find(node)
}

// Find returns the LangEntity for a node together with its ancestors
// and children
func (c *Corpus) Find(node *pythonimports.Node) (*Result, bool) {
	// Look up the entity and use its name if present, otherwise use the node's
	// canonical name. Note that either of these can be different from the query ident.
	cn := node.CanonicalName

	// Only use the entity name if the canonical name is empty, because there are
	// doc entities attached to names like "IPython.compat.builtin_mod", which
	// resolves to "__builtin__", which means that if we start with the node for
	// __builtin__ then we may end up showing "IPython.compat.builtin_mod" at the
	// top of the docs flyout.
	//
	// Also, the canonical name is always used for response.FullName, which is then
	// used for the documentation title in the sidebar, so we should match that.
	entity, _ := c.Entity(node)
	if cn.Empty() && entity != nil {
		if ident := entity.FullIdent(); ident != "" {
			cn = pythonimports.NewDottedPath(ident)
		}
	}
	nodeName := cn.String()

	// Walk the path to get ancestors
	var ancestorParts []string
	if !cn.Empty() {
		ancestorParts = cn.Parts[:len(cn.Parts)-1]
	}

	var ancestors []Identifier
	cur := c.graph.Root
	var rel string
	for _, part := range ancestorParts {
		cur, _ = cur.Attr(part)
		if cur == nil {
			ancestors = nil
			break
		}

		if rel != "" {
			rel += "."
		}
		rel += part

		// get the kind
		var score float64
		kind := nodeToKind(cur.Classification)
		entity, hasdocs := c.Entity(cur)
		if hasdocs {
			kind = entity.Kind
			score = entity.Score
		}

		ancestors = append(ancestors, Identifier{
			Name:    part,
			Rel:     rel,
			Ident:   cur.CanonicalName.String(),
			Kind:    kind,
			HasDocs: hasdocs,
			Score:   score,
		})
	}

	// Get the children
	var children, references []Identifier
	for attr, child := range node.Members {
		entity, hasdocs := c.Entity(child)
		if !hasdocs {
			continue
		}

		childName := child.CanonicalName.String()
		f := Identifier{
			Name:    attr,
			Ident:   childName,
			Rel:     childName,
			Kind:    entity.Kind,
			HasDocs: true,
			Score:   entity.Score,
		}

		// If the descendant node was defined within this class or module then it
		// is a child, otherwise it is a reference.
		if strings.HasPrefix(childName, nodeName) {
			children = append(children, f)
		} else {
			references = append(references, f)
		}
	}

	// Sort
	sort.Sort(sort.Reverse(byIdentScore(children)))
	sort.Sort(sort.Reverse(byIdentScore(references)))

	// Construct final result
	return &Result{
		Ident:      nodeName,
		Kind:       nodeToKind(node.Classification),
		Entity:     entity,
		Ancestors:  ancestors,
		Children:   children,
		References: references,
	}, true
}
