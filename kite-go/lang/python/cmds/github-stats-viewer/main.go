//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"

	arg "github.com/alexflint/go-arg"
	humanize "github.com/dustin/go-humanize"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

type stats struct {
	Name    string
	Path    string
	Count   int
	Members map[string]*stats
	InGraph bool
	AnyName string
	Node    *pythonimports.Node
}

func newStats(graph *pythonimports.Graph, anynames map[*pythonimports.Node]pythonimports.DottedPath, path string, count int) *stats {
	name := path
	if pos := strings.LastIndex(name, "."); pos > -1 {
		name = path[pos+1:]
	}
	s := &stats{
		Name:    name,
		Path:    path,
		Count:   count,
		Members: make(map[string]*stats),
	}

	if node, err := graph.Find(path); err == nil && node != nil {
		s.Node = node
		anyname := anynames[node]
		s.AnyName = anyname.String()
		s.InGraph = true
	}
	return s
}

type byCountByName []*stats

func (b byCountByName) Len() int      { return len(b) }
func (b byCountByName) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byCountByName) Less(i, j int) bool {
	if b[i].Count == b[j].Count {
		return b[i].Name < b[j].Name
	}
	return b[i].Count < b[j].Count
}

type searchResult struct {
	Label template.HTML
	Stats *stats
}

type srByCountByName []searchResult

func (s srByCountByName) Len() int      { return len(s) }
func (s srByCountByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s srByCountByName) Less(i, j int) bool {
	if s[i].Stats.Count == s[j].Stats.Count {
		return s[i].Stats.Name < s[j].Stats.Name
	}
	return s[i].Stats.Count < s[j].Stats.Count
}

func statsURL(stats *stats) string {
	return fmt.Sprintf("/stats/%s", stats.Path)
}

func comma(n int) string {
	return humanize.Comma(int64(n))
}

func color(stats *stats) string {
	if stats.InGraph {
		return ""
	}
	return "red"
}

type app struct {
	// all stats
	root        *stats
	statsByPath map[string]*stats
	templates   *templateset.Set
	anynames    map[*pythonimports.Node]pythonimports.DottedPath
	cutoff      int
}

func (a *app) sortMembers(members map[string]*stats) []*stats {
	var ss []*stats
	for _, s := range members {
		ss = append(ss, s)
	}
	sort.Sort(sort.Reverse(byCountByName(ss)))

	if len(ss) > a.cutoff {
		ss = ss[:a.cutoff]
	}

	return ss
}

func (a *app) walkStats(path string) *stats {
	parts := strings.Split(path, ".")
	leaf := a.root
	for _, part := range parts {
		leaf = leaf.Members[part]
		if leaf == nil {
			break
		}
	}
	return leaf
}

func (a *app) handleLookup(w http.ResponseWriter, r *http.Request) {
	ident := mux.Vars(r)["ident"]
	stats := a.walkStats(ident)
	if stats == nil {
		webutils.ReportNotFound(w, fmt.Sprintf("unknown ident %s", ident))
		return
	}
	http.Redirect(w, r, statsURL(stats), http.StatusTemporaryRedirect)
}

func (a *app) handleTopLevel(w http.ResponseWriter, r *http.Request) {
	pkgs := a.sortMembers(a.root.Members)

	err := a.templates.Render(w, "toplevel.html", map[string]interface{}{
		"Stats": pkgs,
	})
	if err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *app) handleStats(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	stats := a.statsByPath[slug]
	if stats == nil {
		webutils.ReportNotFound(w, fmt.Sprintf("unknown ident `%s`", slug))
		return
	}

	// Render the template
	err := a.templates.Render(w, "stats.html", map[string]interface{}{
		"Stats": stats,
	})
	if err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *app) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")
	if query == "" {
		webutils.ReportBadRequest(w, "missing query parameter")
		return
	}

	re, err := regexp.Compile(query)
	if err != nil {
		webutils.ReportBadRequest(w, fmt.Sprintf("search query `%s` was not a regular expression: %v", query, err))
		return
	}

	var results []searchResult
	for name, stats := range a.statsByPath {
		if m := re.FindStringSubmatchIndex(name); len(m) > 0 {
			begin, end := m[0], m[1]
			label := template.HTML(fmt.Sprintf("%s<b>%s</b>%s", name[:begin], name[begin:end], name[end:]))
			results = append(results, searchResult{
				Label: label,
				Stats: stats,
			})
		}
	}

	sort.Sort(sort.Reverse(srByCountByName(results)))

	if len(results) > a.cutoff {
		results = results[:a.cutoff]
	}

	err = a.templates.Render(w, "search.html", map[string]interface{}{
		"Results": results,
	})
	if err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *app) nodeURL(node *pythonimports.Node) string {
	anyname := a.anynames[node]
	return fmt.Sprintf("http://graph.kite.com/node/%s", anyname.String())
}

func main() {
	args := struct {
		Port        string
		Path        string
		Graph       string
		InGraphOnly bool
		Cutoff      int
	}{
		Port:        ":3031",
		Graph:       pythonimports.DefaultImportGraph,
		InGraphOnly: true,
		Cutoff:      10000,
	}
	arg.MustParse(&args)

	graph, err := pythonimports.NewGraph(args.Graph)
	if err != nil {
		log.Fatalf("error loading import graph `%s`: %v\n", args.Graph, err)
	}

	anynames := pythonimports.ComputeAnyPaths(graph)

	path := args.Path
	if path == "" {
		if args.InGraphOnly {
			path = pythoncode.DefaultPackageStats
		} else {
			path = pythoncode.DefaultUnfilteredPackageStats
		}
	}
	pkgs, err := pythoncode.LoadGithubPackageStats(path)
	if err != nil {
		log.Fatalf("error loading package stats: %v\n", err)
	}

	root := newStats(graph, anynames, "root", 0)
	statsByPath := make(map[string]*stats)
	for pkg, pkgStats := range pkgs {
		pstats := newStats(graph, anynames, pkg, pkgStats.Count)
		root.Members[pkg] = pstats
		statsByPath[pkg] = root.Members[pkg]

		for _, method := range pkgStats.Methods {
			leaf := root.Members[pkg]
			parts := strings.Split(method.Ident, ".")
			for i := 1; i < len(parts); i++ {
				part := parts[i]
				newLeaf := leaf.Members[part]
				if newLeaf == nil {
					path := strings.Join(parts[:i+1], ".")
					newLeaf = newStats(graph, anynames, path, 0)
					leaf.Members[part] = newLeaf
					statsByPath[path] = newLeaf
				}
				leaf = newLeaf
			}
			leaf.Count += method.Count
		}
	}

	app := &app{
		root:        root,
		statsByPath: statsByPath,
		anynames:    anynames,
		cutoff:      args.Cutoff,
	}

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir}
	app.templates = templateset.NewSet(staticfs, "templates", template.FuncMap{
		"statsURL": statsURL,
		"comma":    comma,
		"sort":     app.sortMembers,
		"nodeURL":  app.nodeURL,
		"color":    color,
		"add":      func(x, y int) int { return x + y },
	})

	r := mux.NewRouter()
	r.HandleFunc("/stats/{slug}", app.handleStats).Methods("GET")
	r.PathPrefix("/static/").Handler(http.FileServer(staticfs))
	r.HandleFunc("/", app.handleTopLevel).Methods("GET")
	r.HandleFunc("/search", app.handleSearch).Methods("GET")
	r.HandleFunc("/lookup/{ident:.+}", app.handleLookup).Methods("GET")

	log.Println("listening on " + args.Port)
	log.Fatal(http.ListenAndServe(args.Port, r))
}
