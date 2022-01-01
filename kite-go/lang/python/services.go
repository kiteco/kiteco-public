package python

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/answers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonindex"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/lang/python/seo"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

var (
	// DefaultServiceOptions contains default values for Services.
	DefaultServiceOptions = ServiceOptions{
		Curation:                        pythoncuration.DefaultSearchOptions,
		Index:                           pythonindex.DefaultClientOptions,
		Batch:                           pythonbatch.DefaultOptions,
		ImportGraph:                     pythonimports.DefaultImportGraph,
		TypeInduction:                   typeinduction.DefaultClientOptions,
		NumDocumentationCuratedExamples: 5,
		KwargsData:                      pythoncode.DefaultKwargs,
		KwargsOptions:                   pythoncode.DefaultKwargsOptions,
		ResourceManager:                 pythonresource.DefaultOptions,
		ModelOptions:                    pythonmodels.DefaultOptions,
		SEOPath:                         seo.DefaultDataPath,
		AnswersPath:                     answers.DefaultPath,
	}
)

// --

// ServiceOptions holds the root directory and options for the Client
type ServiceOptions struct {
	// Curation contains initialization options for pythoncuration client.
	Curation pythoncuration.SearchOptions

	// Index contains options for pythonindex.Client.
	Index pythonindex.ClientOptions

	// Batch contains options for pythonbatch.BuilderLoader
	Batch pythonbatch.Options

	// ImportGraph is the location of the global python import graph data.
	ImportGraph string

	// TypeInduction contains Options for all datasets required for type induction
	TypeInduction typeinduction.Options

	// NumDocumentationCuratedExamples defines the maximum number of code examples
	// to return for a single documentation response
	NumDocumentationCuratedExamples int

	// KwargsData is the data for the possible **kwargs
	KwargsData string

	// KwargsOptions are the options for the possible **kwargs index
	KwargsOptions pythoncode.KwargsOptions

	// ResourceManager options
	ResourceManager pythonresource.Options

	// ModelOptions contains options for loading the models
	ModelOptions pythonmodels.Options

	// SEOPath points at data for SEO
	SEOPath string

	AnswersPath string
}

// --

// Services is a wrapper around all datasets that are available for Python.
type Services struct {
	Options         *ServiceOptions
	Curation        *pythoncuration.Searcher
	ImportGraph     *pythonimports.Graph
	Local           *localcode.Client
	InvertedIndex   *pythonindex.Client
	githubPrior     *pythoncode.GithubPrior
	BuilderLoader   *pythonbatch.BuilderLoader
	ResourceManager pythonresource.Manager
	Models          *pythonmodels.Models
	SEOData         seo.Data
	Answers         *answers.Index
}

// NewServices builds a new client with the provided options.
func NewServices(opts *ServiceOptions, local *localcode.Client) (*Services, error) {
	if opts == nil {
		opts = &ServiceOptions{}
		*opts = DefaultServiceOptions
	}

	s := &Services{
		Options: opts,
		Local:   local,
	}
	defer s.timer(time.Now(), "python.NewServices")

	pool := workerpool.New(4)
	defer pool.Stop()
	start := time.Now()

	// Everything depends on the import graph, except for import graph sources
	// so load those two first.
	pool.Add([]workerpool.Job{
		func() error { return s.loadImportGraph(*opts) },
		func() error { return s.initResourceManager(*opts) }, // TODO separate symbol graph loading from the rest of the resource manager
		func() error { return s.loadSEO(*opts) },
		func() error { return s.loadAnswers(*opts) },
	})

	if err := pool.Wait(); err != nil {
		return nil, err
	}

	s.timer(start, "phase one ====")
	start = time.Now()

	// These datasets load fast, and cover the dependncies for loadInvertedIndex. This lets us
	// load the inverted index in parallel with other expensive datasets e.g signature patterns
	// and arg specs.
	pool.Add([]workerpool.Job{
		func() error { return s.loadCuration(s.ImportGraph, *opts) },
	})

	if err := pool.Wait(); err != nil {
		return nil, err
	}

	s.timer(start, "phase two ====")
	start = time.Now()

	pool.Add([]workerpool.Job{
		func() error {
			return s.loadInvertedIndex(s.ResourceManager, s.Curation, *opts)
		},
		func() error {
			return s.loadModels(*opts)
		},
	})

	if err := pool.Wait(); err != nil {
		return nil, err
	}

	s.timer(start, "phase three ====")

	err := s.setupLocalcode(*opts)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Close releases all resources. Services can't be used again after calling Close.
func (s *Services) Close() {
	s.ResourceManager.Close()
}

// Reset releases resources
func (s *Services) Reset() {
	s.ResourceManager.Reset()
	s.Models.Reset()
}

// FilterCuratedExampleReferences filters out fully-qualified name references
// that have no documentation and also removes overlaps from for-loops
func (s *Services) FilterCuratedExampleReferences(resp *response.CuratedExample) {
	for _, part := range [][]*response.CuratedExampleSegment{resp.Prelude, resp.Main, resp.Postlude} {
		for _, seg := range part {
			if seg.Type != "code" {
				continue
			}
			annotation, ok := seg.Annotation.(*response.CodeAnnotation)
			if !ok {
				continue
			}
			var refs []interface{}
			var prevEnd int
			for _, iref := range annotation.References {
				ref, ok := iref.(response.PythonReference)
				if !ok {
					continue
				}
				if ref.Begin >= ref.End || ref.Begin < prevEnd {
					continue
				}
				if ref.NodeType != "import" && ref.NodeType != "attribute" && (ref.NodeType != "name" || ref.Instance) {
					continue
				}
				if n, _ := s.ImportGraph.Find(ref.FullyQualifiedName); n == nil {
					continue
				}
				refs = append(refs, ref)
				prevEnd = ref.End
			}
			annotation.References = refs
		}
	}
}

// findDocumentation constructs a PythonDocumentation response from an import
// graph node. If the node is from a local graph, then this function also
// returns the associated local Documentation object.
func (s *Services) findDocumentation(ctx kitectx.Context, vb valueBundle, buf *bufferIndex) (*pythonlocal.Documentation, *response.PythonDocumentation) {
	ctx.CheckAbort()

	val, idx := vb.val, vb.idx
	if val == nil {
		return nil, nil
	}
	if buf != nil {
		if doc, err := buf.Documentation(val); err == nil {
			return doc, pythonlocal.DocumentationResponse(doc)
		}
	}
	if idx != nil {
		if doc, err := idx.Documentation(ctx, val); err == nil {
			return doc, pythonlocal.DocumentationResponse(doc)
		}
	}

	// Check the resource manager
	for _, val := range pythontype.Disjuncts(kitectx.TODO(), val) {
		var sym pythonresource.Symbol
		switch val := val.(type) {
		case pythontype.External:
			sym = val.Symbol()
		case pythontype.ExternalInstance:
			sym = val.TypeExternal.Symbol()
		}
		if sym.Nil() {
			continue
		}
		docs := s.ResourceManager.Documentation(sym)
		if docs != nil {
			return nil, &response.PythonDocumentation{
				Description: docs.Text, // TODO(tarak): Is this supposed to be HTML or text? Undocumented
				StructuredDoc: &response.PythonStructuredDoc{
					Description: docs.HTML, // This is expected to be HTML if available
				},
			}
		}
	}

	return nil, nil
}

// CuratedExamplesResponseFromIDs gets an array of curated code examples by an array of IDs
// It ignores ids that can't be found, instead choosing to return a partial set of that which
// was requested
func (s *Services) CuratedExamplesResponseFromIDs(ids []int64) []*response.CuratedExample {
	if s.Curation == nil {
		return nil
	}
	var resps []*response.CuratedExample

	for _, id := range ids {
		snippet, found := s.Curation.FindByID(id)
		if found {
			resp := pythoncuration.SnippetToResponse(snippet, s.Curation)
			s.FilterCuratedExampleReferences(resp)
			resps = append(resps, resp)
		}
	}
	return resps
}

// CuratedExampleResponseFromID gets a curated code example by ID
func (s *Services) CuratedExampleResponseFromID(id int64) (*response.CuratedExample, bool) {
	if s.Curation == nil {
		return nil, false
	}

	snippet, found := s.Curation.FindByID(id)
	if !found {
		return nil, false
	}
	resp := pythoncuration.SnippetToResponse(snippet, s.Curation)
	s.FilterCuratedExampleReferences(resp)
	return resp, true
}

// --

// ServicesHandler wraps a Services object and provides HTTP access to the
// underlying functionality
type ServicesHandler struct {
	services *Services
}

// NewServicesHandler builds ServicesHandlers from Services objects
func NewServicesHandler(services *Services) (*ServicesHandler, error) {
	if services == nil {
		return nil, fmt.Errorf("error building python handler with null services")
	}

	return &ServicesHandler{
		services: services,
	}, nil
}

// HandleCuratedExamples gets an array of curated code examples ids set in
// the query
// it expects a querystring ala: "id=1&id=2&id=3&..."
func (h *ServicesHandler) HandleCuratedExamples(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	strIds := strings.Split(values.Get("id"), ",")
	var ids []int64
	for _, strID := range strIds {
		id, err := strconv.ParseInt(strID, 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("bad id: %s", strID), http.StatusBadRequest)
			return
		}
		ids = append(ids, id)
	}

	resp := h.services.CuratedExamplesResponseFromIDs(ids)
	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// HandleCuratedExample gets a curated code example by ID
func (h *ServicesHandler) HandleCuratedExample(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("bad id: %s", vars["id"]), http.StatusBadRequest)
		return
	}

	resp, exists := h.services.CuratedExampleResponseFromID(id)
	if !exists {
		http.Error(w, fmt.Sprintf("no curated example with id: %d", id), http.StatusNotFound)
		return
	}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}
