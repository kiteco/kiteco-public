package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strings"

	humanize "github.com/dustin/go-humanize"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/serialization"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

type viewArgs struct {
	Results string `arg:"positional,help:missing or diff results to view"`
	Graph   string
	Port    string
}

var viewDiffCmd = cmdline.Command{
	Name:     "view-diff",
	Synopsis: "view diff results in web viewer",
	Args: &viewArgs{
		Graph: pythonimports.DefaultImportGraph,
		Port:  ":3029",
	},
}

type missing struct {
	Name       string
	AnyName    string
	Node       *pythonimports.Node
	PctMissing float64 // only valid for top level packages
	Missing    int64   // only valid for top level packages
	Total      int64   // only valid for top level packages
	Members    map[string]*missing
}

func newMissing(name, anyname string, node *pythonimports.Node) *missing {
	return &missing{
		Name:    name,
		AnyName: anyname,
		Node:    node,
		Members: make(map[string]*missing),
	}
}

type byPct []*missing

func (bp byPct) Len() int      { return len(bp) }
func (bp byPct) Swap(i, j int) { bp[i], bp[j] = bp[j], bp[i] }
func (bp byPct) Less(i, j int) bool {
	return bp[i].PctMissing < bp[j].PctMissing
}

func missURL(m *missing) string {
	return fmt.Sprintf("/missing/%s", m.AnyName)
}

func colorMissing(m *missing) string {
	if m.Node != nil && m.Node.Classification == pythonimports.Function {
		return "red"
	}
	return ""
}

type app struct {
	root          *missing
	templates     *templateset.Set
	missingByPath map[string]*missing
}

func (a *app) handleTopLevel(w http.ResponseWriter, r *http.Request) {
	var members []*missing
	for _, member := range a.root.Members {
		members = append(members, member)
	}
	sort.Sort(sort.Reverse(byPct(members)))

	if err := a.templates.Render(w, "diff/toplevel.html", map[string]interface{}{
		"Missing": members,
	}); err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *app) handleMissing(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	missing := a.missingByPath[slug]
	if missing == nil {
		webutils.ReportNotFound(w, fmt.Sprintf("unknown ident `%s`", slug))
		return
	}

	if err := a.templates.Render(w, "diff/missing.html", map[string]interface{}{
		"Missing": missing,
	}); err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (args *viewArgs) Handle() error {
	var pms []*pkgAndMissing
	err := serialization.Decode(args.Results, func(m *pkgAndMissing) {
		pms = append(pms, m)
	})
	if err != nil {
		return fmt.Errorf("error decoding results from %s: %v", args.Results, err)
	}

	graph, err := pythonimports.NewGraph(args.Graph)
	if err != nil {
		return fmt.Errorf("error loading graph %s: %v", args.Graph, err)
	}
	anynames := pythonimports.ComputeAnyPaths(graph)

	// populate root with names and import graph nodes
	missingByPath := make(map[string]*missing)
	root := newMissing("root", "root", nil)
	for _, pm := range pms {
		node := graph.Root.Members[pm.Pkg]
		if node == nil {
			log.Printf("no node for package %s, skipping\n", pm.Pkg)
			continue
		}
		m := newMissing(pm.Pkg, pm.Pkg, node)
		m.Missing = int64(len(pm.Missing))
		m.Total = pm.Total
		m.PctMissing = pm.Pct
		root.Members[pm.Pkg] = m
		missingByPath[pm.Pkg] = m

		for _, miss := range pm.Missing {
			parts := strings.Split(miss, ".")
			leafNode, leafMiss := node, root.Members[pm.Pkg]

			for i, part := range parts[1:] {
				if leafNode = leafNode.Members[part]; leafNode == nil {
					log.Printf("no node for %s, skipping", strings.Join(parts[:i+1], "."))
					break
				}

				anyname := anynames[leafNode].String()
				if leafMiss.Members[part] == nil {
					leafMiss.Members[part] = newMissing(part, anyname, leafNode)
				}
				leafMiss = leafMiss.Members[part]
				missingByPath[anyname] = leafMiss
			}
		}
	}

	app := app{
		root:          root,
		missingByPath: missingByPath,
	}

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir}
	app.templates = templateset.NewSet(staticfs, "templates", template.FuncMap{
		"len":     func(x interface{}) int { return reflect.ValueOf(x).Len() },
		"missURL": missURL,
		"nodeURL": nodeURL,
		"comma":   humanize.Comma,
		"ftoa":    ftoa,
		"color":   colorMissing,
	})

	r := mux.NewRouter()
	r.HandleFunc("/missing/{slug}", app.handleMissing).Methods("GET")
	r.PathPrefix("/static/").Handler(http.FileServer(staticfs))
	r.HandleFunc("/", app.handleTopLevel).Methods("GET")

	log.Println("listening on " + args.Port)
	return http.ListenAndServe(args.Port, r)
}
