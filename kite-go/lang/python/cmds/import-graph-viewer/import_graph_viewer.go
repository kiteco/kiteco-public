//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	arg "github.com/alexflint/go-arg"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/templateset"
	"github.com/kr/pretty"
)

func toJSON(x interface{}) string {
	buf, _ := json.MarshalIndent(x, "", "  ")
	return string(buf)
}

// searchResult represents a result rendered by the /search endpoint
type searchResult struct {
	Label template.HTML
	Node  *pythonimports.Node
}

type member struct {
	Parent *pythonimports.Node
	Attr   string
	Child  *pythonimports.Node
}

type byAttr []member

func (xs byAttr) Len() int           { return len(xs) }
func (xs byAttr) Less(i, j int) bool { return attrLess(xs[i].Attr, xs[j].Attr) }
func (xs byAttr) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }

type byName []pythonimports.FlatMember

func (xs byName) Len() int           { return len(xs) }
func (xs byName) Less(i, j int) bool { return attrLess(xs[i].Attr, xs[j].Attr) }
func (xs byName) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }

func countPrefix(s string, ch rune) int {
	for i, c := range s {
		if c != ch {
			return i
		}
	}
	return len(s)
}

func attrLess(a, b string) bool {
	ap := countPrefix(a, '_')
	bp := countPrefix(b, '_')
	if ap == bp {
		return strings.ToLower(a) < strings.ToLower(b)
	}
	return ap < bp
}

// Sort the the members for a node, with underscores and double-underscores at the end
func sortMembers(members map[string]*pythonimports.Node) []member {
	var xs []member
	for k, v := range members {
		xs = append(xs, member{
			Attr:  k,
			Child: v,
		})
	}
	sort.Sort(byAttr(xs))
	return xs
}

// App encapsulates the live application state
type App struct {
	graph           *pythonimports.Graph
	flat            map[int64]*pythonimports.FlatNode
	incoming        map[int64][]member
	templates       *templateset.Set
	slugByNode      map[*pythonimports.Node]pythonimports.DottedPath
	nodeByCanonical map[string]*pythonimports.Node
}

func (a *App) handleLookup(w http.ResponseWriter, r *http.Request) {
	ident := mux.Vars(r)["ident"]
	node, err := a.graph.Find(ident)
	if err != nil {
		http.Error(w, fmt.Sprintf("unknown node %s: %v", ident, err), http.StatusNotFound)
		return
	}
	http.Redirect(w, r, a.nodeURL(node), http.StatusTemporaryRedirect)
}

func (a *App) handleLookupID(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["nodeid"]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("unparseable id %s: %v", idStr, err), http.StatusBadRequest)
		return
	}
	node, ok := a.graph.FindByID(id)
	if !ok {
		http.Error(w, fmt.Sprintf("unknown node %d", id), http.StatusNotFound)
		return
	}
	http.Redirect(w, r, a.nodeURL(node), http.StatusTemporaryRedirect)
}

func (a *App) handleTopLevel(w http.ResponseWriter, r *http.Request) {
	err := a.templates.Render(w, "toplevel.html", map[string]interface{}{
		"Packages": a.graph.PkgToNode,
	})
	if err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *App) handleNode(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	node, found := a.nodeByCanonical[slug]
	if !found {
		var err error
		node, err = a.graph.Find(slug)
		if err != nil {
			http.Error(w, fmt.Sprintf("unknown node %s: %v", slug, err), http.StatusNotFound)
			return
		}
	}

	// Print the raw flat node
	var raw string
	if flat, found := a.flat[node.ID]; found {
		sort.Sort(byName(flat.Members))
		raw = pretty.Sprintf("%# v", *flat)
	}

	// get bases from actual node so that we can
	// also get bases for skeleton nodes.
	var bases []int64
	for _, base := range node.Bases {
		if base != nil {
			bases = append(bases, base.ID)
		}
	}

	// Render the template
	err := a.templates.Render(w, "node.html", map[string]interface{}{
		"Node":     node,
		"Raw":      raw,
		"Incoming": a.incoming[node.ID],
		"Bases":    bases,
	})
	if err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *App) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")
	if query == "" {
		http.Error(w, "missing query parameter", http.StatusBadRequest)
		return
	}

	re, err := regexp.Compile(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("search query was not a regular expression: %v", err), http.StatusBadRequest)
		return
	}

	var results []searchResult
	for i := range a.graph.Nodes {
		node := &a.graph.Nodes[i]
		s := node.CanonicalName.String()
		if m := re.FindStringSubmatchIndex(s); len(m) > 0 {
			begin, end := m[0], m[1]
			label := template.HTML(fmt.Sprintf("%s<b>%s</b>%s", s[:begin], s[begin:end], s[end:]))
			results = append(results, searchResult{
				Label: label,
				Node:  node,
			})
		}
	}

	// Render the template
	err = a.templates.Render(w, "search.html", map[string]interface{}{
		"Results": results,
	})
	if err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *App) nodeName(node *pythonimports.Node) string {
	if !node.CanonicalName.Empty() {
		return node.CanonicalName.String()
	}
	return fmt.Sprintf("{Node %d}", node.ID)
}

func (a *App) nodeURL(node *pythonimports.Node) string {
	if slug, ok := a.slugByNode[node]; ok {
		return fmt.Sprintf("/node/%s", slug.String())
	}
	return ""
}

func main() {
	args := struct {
		Port          string
		Path          string
		SkipSkeletons bool
	}{
		Port: ":3021",
		Path: pythonimports.DefaultImportGraph,
	}
	arg.MustParse(&args)

	// Load flat nodes
	flatNodes, err := pythonimports.LoadFlatGraph(args.Path)
	if err != nil {
		log.Fatal(err)
	}

	// Index flat nodes by ID
	flatByID := make(map[int64]*pythonimports.FlatNode)
	for _, flat := range flatNodes {
		flatByID[flat.ID] = flat
	}

	// Build full graph
	graph := pythonimports.NewGraphFromNodes(flatNodes)
	if !args.SkipSkeletons {
		if err := pythonskeletons.UpdateGraph(graph); err != nil {
			log.Fatal(err)
		}
	}

	// Compute incoming edges
	incoming := make(map[int64][]member)
	for i := range graph.Nodes {
		node := &graph.Nodes[i]
		for attr, child := range node.Members {
			if child == nil {
				continue
			}
			incoming[child.ID] = append(incoming[child.ID], member{
				Parent: node,
				Attr:   attr,
				Child:  child,
			})
		}
	}

	// Compute slugs
	log.Println("computing anypaths...")
	slugByNode := pythonimports.ComputeAnyPaths(graph)

	// Compute map from canonical names to nodes because some canonical names do not resolve to that node
	nodeByCanonical := make(map[string]*pythonimports.Node)
	for i := range graph.Nodes {
		nodeByCanonical[graph.Nodes[i].CanonicalName.String()] = &graph.Nodes[i]
	}

	// Construct app
	app := App{
		graph:           graph,
		flat:            flatByID,
		incoming:        incoming,
		slugByNode:      slugByNode,
		nodeByCanonical: nodeByCanonical,
	}

	// Construct static assets
	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	app.templates = templateset.NewSet(staticfs, "templates", template.FuncMap{
		"len":  func(x interface{}) int { return reflect.ValueOf(x).Len() },
		"lit":  func(x string) string { return fmt.Sprint(x) },
		"json": toJSON,
		"sort": sortMembers,
		"name": app.nodeName,
		"url":  app.nodeURL,
	})

	// Construct router
	r := mux.NewRouter()
	r.HandleFunc("/node/{slug}", app.handleNode).Methods("GET")
	r.PathPrefix("/static/").Handler(http.FileServer(staticfs))
	r.HandleFunc("/", app.handleTopLevel).Methods("GET")
	r.HandleFunc("/search", app.handleSearch).Methods("GET")
	r.HandleFunc("/lookup/{ident:.+}", app.handleLookup).Methods("GET")
	r.HandleFunc("/id/{nodeid}", app.handleLookupID).Methods("GET")

	// Listen
	log.Println("listening on " + args.Port)
	log.Fatal(http.ListenAndServe(args.Port, r))
}
