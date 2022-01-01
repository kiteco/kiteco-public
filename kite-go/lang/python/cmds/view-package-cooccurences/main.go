//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"time"

	arg "github.com/alexflint/go-arg"
	humanize "github.com/dustin/go-humanize"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

type cooccurences struct {
	Package    string
	Popularity int64
	Cooccuring map[string]int64
	InGraph    bool
	Node       *pythonimports.Node
}

type byPopularity []*cooccurences

func (bp byPopularity) Len() int      { return len(bp) }
func (bp byPopularity) Swap(i, j int) { bp[i], bp[j] = bp[j], bp[i] }
func (bp byPopularity) Less(i, j int) bool {
	if bp[i].Popularity == bp[j].Popularity {
		// sort in reverse by popularity so we want
		// names with the same popularity to be in alphabetical order
		return bp[i].Package > bp[j].Package
	}
	return bp[i].Popularity < bp[j].Popularity
}

type cooccurence struct {
	Package string
	Count   int64
	InGraph bool
	Node    *pythonimports.Node
}

type byCount []cooccurence

func (bc byCount) Len() int      { return len(bc) }
func (bc byCount) Swap(i, j int) { bc[i], bc[j] = bc[j], bc[i] }
func (bc byCount) Less(i, j int) bool {
	return bc[i].Count < bc[j].Count
}

func (a *app) sortCooccuring(c *cooccurences) []cooccurence {
	var cooccuring []cooccurence
	for pkg, count := range c.Cooccuring {
		node, inGraph := a.graph.Root.Members[pkg]
		cooccuring = append(cooccuring, cooccurence{
			Package: pkg,
			Count:   count,
			InGraph: inGraph,
			Node:    node,
		})
	}
	sort.Sort(sort.Reverse(byCount(cooccuring)))
	if len(cooccuring) > a.cutoff {
		cooccuring = cooccuring[:a.cutoff]
	}
	return cooccuring
}

func cooccurencesURL(pkg string) string {
	return fmt.Sprintf("/cooccurences/%s", pkg)
}

func nodeURL(node *pythonimports.Node) string {
	return fmt.Sprintf("http://graph.kite.com/node/%s", node.CanonicalName.String())
}

func comma(n int64) string {
	return humanize.Comma(n)
}

func color(node *pythonimports.Node) string {
	if node == nil {
		return "red"
	}
	return ""
}

type app struct {
	cooccurences map[string]*cooccurences
	templates    *templateset.Set
	graph        *pythonimports.Graph
	cutoff       int
}

func (a *app) handleTopLevel(w http.ResponseWriter, r *http.Request) {
	var cooccurs []*cooccurences
	for _, cooccur := range a.cooccurences {
		cooccurs = append(cooccurs, cooccur)
	}
	sort.Sort(sort.Reverse(byPopularity(cooccurs)))

	if len(cooccurs) > a.cutoff {
		cooccurs = cooccurs[:a.cutoff]
	}

	if err := a.templates.Render(w, "toplevel.html", map[string]interface{}{
		"Cooccurences": cooccurs,
	}); err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *app) handleCooccurences(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	cooccurences := a.cooccurences[slug]
	if cooccurences == nil {
		webutils.ReportNotFound(w, fmt.Sprintf("unknown ident `%s`", slug))
		return
	}

	sorted := a.sortCooccuring(cooccurences)

	if err := a.templates.Render(w, "cooccurences.html", map[string]interface{}{
		"Package":      slug,
		"Cooccurences": sorted,
		"InGraph":      cooccurences.InGraph,
		"Node":         cooccurences.Node,
	}); err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func main() {
	args := struct {
		Graph        string
		Cooccurences string
		Port         string
		InGraphOnly  bool `arg:"help:include only top level pacakges/modules that are in the import graph"`
		Cutoff       int  `arg:"help:include only the top n packages"`
	}{
		Graph:        pythonimports.DefaultImportGraph,
		Cooccurences: pythoncode.DefaultPackageCooccurences,
		Port:         ":3025",
		InGraphOnly:  true,
		Cutoff:       1000,
	}
	arg.MustParse(&args)

	start := time.Now()
	graph, err := pythonimports.NewGraph(args.Graph)
	if err != nil {
		log.Fatalf("error opening graph %s: %v\n", args.Graph, err)
	}

	pkgs, err := pythoncode.LoadGithubCooccurenceStats(args.Cooccurences)
	if err != nil {
		log.Fatalf("error loading signature patterns: %v", err)
	}

	stats := pythoncode.DefaultPackageStats
	if !args.InGraphOnly {
		stats = pythoncode.DefaultUnfilteredPackageStats
	}

	pkgStats, err := pythoncode.LoadGithubPackageStats(stats)
	if err != nil {
		log.Fatalf("error loading packag stats: %v \n", err)
	}

	c := make(map[string]*cooccurences)
	for pkg, stats := range pkgs {
		node, inGraph := graph.Root.Members[pkg]
		if args.InGraphOnly && !inGraph {
			continue
		}

		c[pkg] = &cooccurences{
			Package:    pkg,
			Cooccuring: stats,
			InGraph:    inGraph,
			Node:       node,
		}

		if ps, found := pkgStats[pkg]; found {
			c[pkg].Popularity = int64(ps.Count)
		}
	}

	app := &app{
		cooccurences: c,
		graph:        graph,
		cutoff:       args.Cutoff,
	}

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir}
	app.templates = templateset.NewSet(staticfs, "templates", template.FuncMap{
		"cooccurencesURL": cooccurencesURL,
		"comma":           comma,
		"sort":            app.sortCooccuring,
		"nodeURL":         nodeURL,
		"color":           color,
	})

	r := mux.NewRouter()
	r.HandleFunc("/cooccurences/{slug}", app.handleCooccurences).Methods("GET")
	r.PathPrefix("/static/").Handler(http.FileServer(staticfs))
	r.HandleFunc("/", app.handleTopLevel).Methods("GET")

	log.Println("Loading took", time.Since(start))
	log.Println("listening on " + args.Port)
	log.Fatal(http.ListenAndServe(args.Port, r))
}
