//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	arg "github.com/alexflint/go-arg"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/highlight"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

type app struct {
	templates *templateset.Set
	records   recordIndex
	sources   *pythoncode.HashToSourceIndex
}

func parseLocator(params url.Values) (locator, error) {
	hash := params.Get("hash")
	cursorParam := params.Get("cursor")

	if len(hash) == 0 && len(cursorParam) == 0 {
		return locator{}, nil
	} else if len(hash) == 0 {
		return locator{}, fmt.Errorf("need hash if cursor specified")
	} else if len(cursorParam) == 0 {
		return locator{}, fmt.Errorf("need hash if cursor specified")
	}

	var err error
	cursor, err := strconv.ParseInt(cursorParam, 10, 64)
	if err != nil {
		return locator{}, fmt.Errorf("could not parse cursor param (%s): %v", cursorParam, err)
	}

	return locator{Hash: hash, Cursor: cursor}, nil
}

func (a app) locIndex(loc locator) (int, error) {
	// If no locator specified, return the first record
	var idx int
	if loc == (locator{}) {
		return 0, nil
	}
	idx, err := a.records.Index(loc)
	if err != nil {
		return 0, err
	}
	return idx, nil
}

func (a app) handleRoot(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	queryLoc, err := parseLocator(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	idx, err := a.locIndex(queryLoc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	rec := a.records.Get(idx)
	loc := rec.Locator()

	next := idx + 1
	if next >= a.records.Count() {
		next = 0
	}
	prev := idx - 1
	if prev < 0 {
		prev = a.records.Count() - 1
	}

	src, err := a.sources.SourceFor(loc.Hash)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting source for hash %s: %v", loc.Hash, err), http.StatusNotFound)
		return
	}

	code, err := highlight.Highlight(string(src), loc.Cursor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type renderComp struct {
		Name string
		Prob float64
	}

	comps := make([]renderComp, 0, len(rec.Probs))
	for i, name := range rec.Sample.Meta.CompIdentifiers {
		comps = append(comps, renderComp{
			Name: name,
			Prob: rec.Probs[i],
		})
	}

	sort.Slice(comps, func(i, j int) bool {
		return comps[i].Prob > comps[j].Prob
	})

	err = a.templates.Render(w, "root.html", map[string]interface{}{
		"Locator":     loc,
		"Source":      template.HTML(code),
		"Completions": comps,
		"PrevURL":     locURL(a.records.Get(prev).Locator()),
		"NextURL":     locURL(a.records.Get(next).Locator()),
		"CurIdx":      idx,
		"Count":       a.records.Count(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func locURL(loc locator) string {
	return fmt.Sprintf("/?hash=%s&cursor=%d#cursor", url.QueryEscape(loc.Hash), loc.Cursor)
}

func main() {
	args := struct {
		VisData     string `arg:"required"`
		Port        int
		SourceCache string
	}{
		Port:        4321,
		SourceCache: "/data",
	}
	arg.MustParse(&args)

	records, err := newRecordIndex(args.VisData)
	if err != nil {
		log.Fatalln(err)
	}

	sources, err := pythoncode.NewHashToSourceIndex(pythoncode.HashToSourceIndexPath, args.SourceCache)
	if err != nil {
		log.Fatalln(err)
	}

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates := templateset.NewSet(staticfs, "templates", nil)

	a := app{
		templates: templates,
		records:   records,
		sources:   sources,
	}

	r := mux.NewRouter()
	r.HandleFunc("/", a.handleRoot)

	log.Printf("listening on port %d", args.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", args.Port), r))
}
