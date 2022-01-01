package main

import (
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/markup"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

var (
	showCmd = cmdline.Command{
		Name:     "show",
		Synopsis: "Show coverage for static analysis",
		Args: &showArgs{
			Port:     ":3031",
			LibDepth: 1,
		},
	}
)

type showArgs struct {
	Port     string
	Corpus   string `arg:"positional"`
	LibDepth int
}

type project struct {
	// Path is the path to the top level dir in the project
	Path string

	Stats batchStats

	// Attrs is the number of attr expressions in the project
	Attrs int64
	// ResolvedBases is the number of bases of attr expressions that resolved
	ResolvedBases int64
	// PctResolvedBases is the pct of bases of attrs resolved
	PctResolvedBases float64
	// MembersResolved is the number of attr expressions that had the rhs in the completions for the lhs
	MembersResolved int64
	// PctMembersResolved is the pct of attr expressions that had the rhs in the completions for the lhs
	PctMembersResolved float64

	// Sources is the marked up source for each file in the project
	Sources map[string]template.HTML
}

type projectByPath []project

func (p projectByPath) Len() int           { return len(p) }
func (p projectByPath) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p projectByPath) Less(i, j int) bool { return p[i].Path < p[j].Path }

func sortedProjects(projects map[string]project) []project {
	var elems []project
	for _, p := range projects {
		elems = append(elems, p)
	}
	sort.Sort(projectByPath(elems))
	return elems
}

type srcAndPath struct {
	Src  template.HTML
	Path string
	Base string
}

type srcByPath []srcAndPath

func (s srcByPath) Len() int           { return len(s) }
func (s srcByPath) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s srcByPath) Less(i, j int) bool { return s[i].Path < s[j].Path }

func sortedSrcs(sources map[string]template.HTML) []srcAndPath {
	var elems []srcAndPath
	for path, src := range sources {
		elems = append(elems, srcAndPath{
			Src:  src,
			Path: path,
		})
	}
	sort.Sort(srcByPath(elems))
	return elems
}

// show usage decides whether to show a node based on its evaluate/assign/delete/import usage
func showUsage(u pythonast.Usage) bool {
	return u == pythonast.Evaluate || u == pythonast.Import
}

type app struct {
	Projects       map[string]project
	Templates      *templateset.Set
	ProcessingTime string
}

func (a *app) handleTopLevel(w http.ResponseWriter, r *http.Request) {
	var resolvedBases, membersResolved, attrs int64
	var stats batchStats
	for _, project := range a.Projects {
		resolvedBases += project.ResolvedBases
		membersResolved += project.MembersResolved
		attrs += project.Attrs
		stats.Added += project.Stats.Added
		stats.Files += project.Stats.Files
		stats.TooLarge += project.Stats.TooLarge
		stats.ParseErrors += project.Stats.ParseErrors
		stats.ProcessingTime += project.Stats.ProcessingTime
	}
	if err := a.Templates.Render(w, "toplevel.html", map[string]interface{}{
		"Projects":           sortedProjects(a.Projects),
		"ResolvedBases":      resolvedBases,
		"PctResolvedBases":   float64(resolvedBases) / float64(attrs),
		"MembersResolved":    membersResolved,
		"PctMembersResolved": float64(membersResolved) / float64(attrs),
		"Attrs":              attrs,
		"Stats":              stats,
	}); err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *app) handleFile(w http.ResponseWriter, r *http.Request) {
	pn := mux.Vars(r)["project"]
	if pn == "" {
		http.Error(w, "project query parameter empty", http.StatusBadRequest)
		return
	}

	project, found := a.Projects[pn]
	if !found {
		http.Error(w, fmt.Sprintf("unknown project %s", pn), http.StatusNotFound)
		return
	}

	fn := mux.Vars(r)["file"]
	if fn == "" {
		http.Error(w, "file query parameter empty", http.StatusBadRequest)
		return
	}

	src, found := project.Sources[fn]
	if !found {
		http.Error(w, fmt.Sprintf("unable to find file %s in project %s", fn, pn), http.StatusNotFound)
		return
	}

	if err := a.Templates.Render(w, "file.html", map[string]interface{}{
		"MarkedUpSource": src,
		"Path":           fn,
	}); err != nil {
		http.Error(w, fmt.Sprintf("error generate template for file %s in project %s: %v", fn, pn, err), http.StatusInternalServerError)
	}
}

func (a *app) handleTrace(w http.ResponseWriter, r *http.Request) {
	pn := mux.Vars(r)["project"]
	if pn == "" {
		http.Error(w, "project query parameter empty", http.StatusBadRequest)
		return
	}

	project, found := a.Projects[pn]
	if !found {
		http.Error(w, fmt.Sprintf("unknown project %s", pn), http.StatusNotFound)
		return
	}

	if err := a.Templates.Render(w, "trace.html", map[string]interface{}{
		"Trace": project.Stats.Trace,
	}); err != nil {
		http.Error(w, fmt.Sprintf("error generate template for trace in project %s: %v", pn, err), http.StatusInternalServerError)
	}
}

func (args *showArgs) Handle() error {
	start := time.Now()
	projects := make(map[string]project)
	wp := walkParams{
		Corpus:       args.Corpus,
		LibraryDepth: args.LibDepth,
	}
	if err := walk(wp, func(sources map[string]sourceFile, collector *collector, stats batchStats) error {
		corpus := strings.Replace(strings.TrimPrefix(stats.Corpus, args.Corpus), "/", ":", -1)
		project := project{
			Path:    corpus,
			Stats:   stats,
			Sources: make(map[string]template.HTML),
		}

		for _, file := range sources {
			parents := pythonast.ConstructParentTable(file.AST, 0)

			// Markup original source
			var n int
			var m markup.Markupper
			pythonast.Inspect(file.AST, func(node pythonast.Node) bool {
				expr, isexpr := node.(pythonast.Expr)
				if !isexpr {
					return true
				}

				parent := parents[expr]

				if attr, isAttr := expr.(*pythonast.AttributeExpr); isAttr {
					project.Attrs++
					if val, found := collector.exprs[attr.Value]; found && val != nil {
						project.ResolvedBases++
						if res, err := pythontype.AttrNoCtx(val, attr.Attribute.Literal); err == nil && res.Value() != nil {
							project.MembersResolved++
						}
					}
				}

				// show names, attributes, and any expression used as the base of an attribute
				var render bool
				openpos, closepos := expr.Begin(), expr.End()
				switch expr := expr.(type) {
				case *pythonast.NameExpr:
					render = showUsage(expr.Usage)
				case *pythonast.AttributeExpr:
					render = showUsage(expr.Usage)
					openpos, closepos = expr.Attribute.Begin, expr.Attribute.End
				}
				if _, parentattr := parent.(*pythonast.AttributeExpr); parentattr {
					render = true
				}

				if !render {
					return true
				}

				var tag string
				if val, found := collector.exprs[expr]; found && val != nil {
					placement := "top"
					if n%2 == 1 {
						placement = "bottom"
					}
					n++
					tag = fmt.Sprintf(`<span class="ast resolved" title="%v" data-placement="%s">`,
						html.EscapeString(fmt.Sprintf("%v", val)), placement)
				} else {
					tag = `<span class="ast unresolved">`
				}

				m.Add(int(openpos), int(closepos), tag, "</span>")
				return true
			})

			path := strings.Replace(strings.TrimPrefix(file.Path, stats.Corpus), "/", ":", -1)
			project.Sources[path] = m.Render(file.Contents)
		}

		project.PctResolvedBases = float64(project.ResolvedBases) / float64(project.Attrs)
		project.PctMembersResolved = float64(project.MembersResolved) / float64(project.Attrs)

		projects[corpus] = project

		fmt.Printf("%s took %v, contained %d python files, %d were too large, %d contained parse errors, %d were added to the batch\n",
			stats.Corpus, stats.ProcessingTime, stats.Files, stats.TooLarge, stats.ParseErrors, stats.Added)
		return nil
	}); err != nil {
		return fmt.Errorf("error walking corpus `%s`: %v", args.Corpus, err)
	}
	duration := time.Since(start)
	fmt.Println("Done loading! Took", duration)

	// Construct static assets
	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir}

	app := &app{
		Projects:       projects,
		ProcessingTime: fmt.Sprintf("%v", duration),
		Templates: templateset.NewSet(staticfs, "templates", template.FuncMap{
			"sortFiles":  sortedSrcs,
			"prettyPct":  func(f float64) string { return fmt.Sprintf("%.3f%%", 100.*f) },
			"comma":      func(i int64) string { return humanize.Comma(i) },
			"prettyTime": func(d time.Duration) string { return fmt.Sprintf("%v", d) },
		}),
	}

	// Construct router
	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler(http.FileServer(staticfs))
	r.HandleFunc("/", app.handleTopLevel).Methods("GET")
	r.HandleFunc("/file/{project}/{file}", app.handleFile).Methods("GET")
	r.HandleFunc("/trace/{project}", app.handleTrace).Methods("GET")

	// Listen
	log.Println("listening on " + args.Port)
	return http.ListenAndServe(args.Port, r)
}
