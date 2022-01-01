package answers

import (
	"compress/gzip"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-answers/go/render"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// DefaultPath for KA index
const DefaultPath = "s3://kite-data/kite-answers/2020-03-27T18:44:45.json.gz"

// Content bundles rendered data with a canonical slug
type Content struct {
	render.Rendered
	Canonical string `json:"canonical"`
}

// Index contains a list of Content, and a map where multiple slugs can refer
// to the corresponding index in the list of contents.
type Index struct {
	Content []Content      `json:"content,omitempty"`
	Slugs   map[string]int `json:"slugs,omitempty"`

	// Links index KA slugs to display by documentation page.
	// The keys are SEO-normalized paths (as per the kite-go/lang/python/seo package).
	Links map[pythonimports.Hash][]editorapi.AnswersLink `json:"links"`
}

// Load loads an index
func Load(path string) (*Index, error) {
	jsongzR, err := fileutil.NewCachedReader(path)
	if err != nil {
		return nil, err
	}
	defer jsongzR.Close()

	jsonR, err := gzip.NewReader(jsongzR)
	if err != nil {
		return nil, err
	}

	var i Index
	if err := json.NewDecoder(jsonR).Decode(&i); err != nil {
		return nil, err
	}
	return &i, nil
}

// HandleHTTP handles HTTP requests with a mux variable `{slug}`
func (idx *Index) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]
	i, ok := idx.Slugs[slug]
	if !ok {
		http.Error(w, "Invalid slug: "+slug, http.StatusNotFound)
		return
	}

	out := idx.Content[i]
	buf, err := json.Marshal(out)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}
