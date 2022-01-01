//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/serialization"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

var showCmd = cmdline.Command{
	Name:     "show",
	Synopsis: "Display counts of missing modules and attributes in a web ui",
	Args: &showArgs{
		Port: ":3021",
	},
}

type attrsAndCount struct {
	Name  string
	Count int64
	Attrs map[string]int64
}

type byCountThenName []*attrsAndCount

func (b byCountThenName) Len() int      { return len(b) }
func (b byCountThenName) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byCountThenName) Less(i, j int) bool {
	if b[i].Count == b[j].Count {
		return b[i].Name > b[j].Name
	}
	return b[i].Count > b[j].Count
}

type attrCount struct {
	Attr  string
	Count int64
}

type byCountThenAttr []attrCount

func (b byCountThenAttr) Len() int { return len(b) }
func (b byCountThenAttr) Less(i, j int) bool {
	if b[i].Count == b[j].Count {
		return b[i].Attr > b[j].Attr
	}
	return b[i].Count > b[j].Count
}
func (b byCountThenAttr) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func sortedCounts(counts map[string]int64) []attrCount {
	var elems []attrCount
	for attr, count := range counts {
		elems = append(elems, attrCount{
			Attr:  attr,
			Count: count,
		})
	}
	sort.Sort(byCountThenAttr(elems))
	return elems
}

type app struct {
	MissingModules     map[string]int64
	MissingModuleAttrs []*attrsAndCount
	MissingClassAttrs  map[string]map[string]int64
	templates          *templateset.Set
}

func (a *app) handleTopLevel(w http.ResponseWriter, r *http.Request) {
	if err := a.templates.Render(w, "toplevel.html", map[string]interface{}{
		"MissingModules":     sortedCounts(a.MissingModules),
		"MissingModuleAttrs": a.MissingModuleAttrs,
		"MissingClassAttrs":  a.MissingClassAttrs,
	}); err != nil {
		webutils.ReportError(w, err.Error())
	}
}

type showArgs struct {
	Counts string `arg:"positional,required"`
	Port   string
}

func (args *showArgs) Handle() error {
	var counts counts
	if err := serialization.Decode(args.Counts, &counts); err != nil {
		return fmt.Errorf("error decoding counts `%s`: %v\n", args.Counts, err)
	}

	app := &app{
		MissingModules:    counts.MissingModules,
		MissingClassAttrs: counts.MissingClassAttrs,
	}

	// organize missing module attrs by module
	modAttrs := make(map[string]*attrsAndCount)
	for fqa, count := range counts.MissingModuleAttrs {
		var mod, attr string
		if pos := strings.LastIndex(fqa, "."); pos > 0 {
			mod = fqa[:pos]
			attr = fqa[pos+1:]
		}
		if mod == "" {
			log.Printf("bad module attr `%s`, skipping\n", fqa)
			continue
		}

		ac := modAttrs[mod]
		if ac == nil {
			ac = &attrsAndCount{
				Name:  mod,
				Attrs: make(map[string]int64),
			}
			modAttrs[mod] = ac
			app.MissingModuleAttrs = append(app.MissingModuleAttrs, ac)
		}
		ac.Count += count
		ac.Attrs[attr] += count
	}
	sort.Sort(byCountThenName(app.MissingModuleAttrs))

	// Construct static assets
	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir}
	app.templates = templateset.NewSet(staticfs, "templates", template.FuncMap{
		"prettyCount":  humanize.Comma,
		"sortedCounts": sortedCounts,
	})

	// Construct router
	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.FileServer(staticfs))
	r.HandleFunc("/", app.handleTopLevel).Methods("GET")

	// Listen
	log.Println("listening on " + args.Port)
	return http.ListenAndServe(args.Port, r)
}
