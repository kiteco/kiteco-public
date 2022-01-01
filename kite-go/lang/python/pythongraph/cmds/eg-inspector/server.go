package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

type templateContent struct {
	Buffer             string
	Completions        *returnedComp
	Ordering           string
	EnableFiltering    bool
	SkippedCompletions int
	Provider           string
}

type app struct {
	templates     *templateset.Set
	rm            pythonresource.Manager
	defaultBuffer string
	models        *pythonmodels.Models
}

func toHTML(s interface{}) template.HTML {
	return template.HTML(fmt.Sprintf("%v", s))
}

func newApp(rm pythonresource.Manager, buffer string) (app, error) {
	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	models, err := pythonmodels.New(pythonmodels.DefaultOptions)
	fail(err)
	return app{
		templates:     templateset.NewSet(staticfs, "templates", template.FuncMap{"toHTML": toHTML}),
		rm:            rm,
		defaultBuffer: buffer,
		models:        models,
	}, nil
}

func parseHTMLBoolean(s string) bool {
	return s == "on"
}

func (a app) handleRequest(w http.ResponseWriter, r *http.Request) {

	var data templateContent
	data.Ordering = "alphabetical"
	data.Provider = "subtoken"
	switch r.Method {
	case "GET":
		data.Buffer = a.defaultBuffer
		// Nothing to do, we serve the template with it's 0 Completion
	case "POST":
		// Call ParseForm() to parse the raw query and update r.PostForm and r.Form.
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		buffer := r.FormValue("buffer")
		ordering := r.FormValue("ordering")
		filtering := parseHTMLBoolean(r.FormValue("enable-filtering"))
		provider := r.FormValue("provider")

		data.Buffer = buffer
		data.Provider = provider
		if provider == "callModel" {
			data.Completions, data.SkippedCompletions = a.getCallModelCompletions(buffer, ordering, filtering)
		} else {
			data.Completions, data.SkippedCompletions = a.getCompletions(buffer, ordering, filtering)
		}
		data.Ordering = ordering
		data.EnableFiltering = filtering
	}
	err := a.templates.Render(w, "root.html", data)
	if err != nil {
		log.Println(err)
	}
}
