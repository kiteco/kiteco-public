package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/templateset"
	"github.com/kr/pretty"
)

// TODO(juan): add popularity info?
type coverageArgs struct {
	Patterns            string
	ImportGraphArgSpecs string
	TypeshedArgSpecs    string
	Kwargs              string
	Source              string
	Packages            []string `arg:"positional,help:restrict coverage to the provided packages"`
	Verbose             bool
	Graph               string `arg:"help:import graph to use during evaluation"`
	MinSeen             int64  `arg:"help:only include functions that have been seen a minimum number of times"`
	Port                string
	NoSpecs             bool `arg:"help:do not load argspec data"`
	NoDocs              bool `arg:"help:do not load doc data"`
	NoPatterns          bool `arg:"help:do not load signature patterns"`
	NoKwargs            bool `arg:"help:do not load possible **kwargs"`
	NoSource            bool `arg:"help:do not load source code for definitions"`
}

var coverageCmd = cmdline.Command{
	Name:     "coverage",
	Synopsis: "evaluate coverage of signature patterns relative to the import graph",
	Args: &coverageArgs{
		Patterns:            pythoncode.DefaultSignaturePatterns,
		ImportGraphArgSpecs: pythonimports.DefaultImportGraphArgSpecs,
		TypeshedArgSpecs:    pythonimports.DefaultImportGraphArgSpecs,
		Kwargs:              pythoncode.DefaultKwargs,
		Graph:               pythonimports.DefaultImportGraph,
		Port:                ":3027",
	},
}

func popularityStats(graph *pythonimports.Graph, stats map[string]pythoncode.PackageStats) map[*pythonimports.Node]int64 {
	popularity := make(map[*pythonimports.Node]int64)
	for pkg, ps := range stats {
		node := graph.Root.Members[pkg]
		if node == nil {
			continue
		}
		popularity[node] = int64(ps.Count)

		for _, ms := range ps.Methods {
			node, err := graph.Find(ms.Ident)
			if node == nil || err != nil {
				continue
			}

			popularity[node] = int64(ms.Count)
		}
	}
	return popularity
}

type methodStats struct {
	Name    string
	AnyName string
	Node    *pythonimports.Node
	// Popularity is the number of times this method has appeared in the github data.
	Popularity int64

	HaveArgSpec bool
	ArgSpec     *pythonimports.ArgSpec

	HavePatterns bool
	Patterns     *pythoncode.MethodPatterns

	HaveDoc bool
	Doc     *pythondocs.LangEntity

	HaveKwargs bool
	Kwargs     *response.PythonKwargs

	HaveSource bool
}

type packageStats struct {
	Name    string
	AnyName string
	// Popularity is the number of times the package has appeared in the github data
	Popularity int64

	Node *pythonimports.Node
	// Members contains the stats for child packages of the current package
	Members map[string]*packageStats
	// Methods contains the stats for methods in the current package
	Methods map[string]*methodStats

	// NumSpecs is the number of argument specs we have for methods in the current package
	NumSpecs int64
	// PctSpecs is the percentage of methods in the current package for which we have arg specs
	PctSpecs float64
	// NumPatterns is the number of signature patterns we have for methods in the current pacakge
	NumPatterns int64
	// PctPatterns is the percentage of methods in the current package for which we have signature patterns
	PctPatterns float64
	// NumDocs is the number of methods in the current package for which we have docs
	NumDocs int64
	// PctDocs is the percentage of methods in the current package for which we have docs
	PctDocs float64
	// NumKwargs is the number of methods for which we have possible **kwargs info
	NumKwargs int64
	// PctKwargs is teh percentage of methods for which we have possible **kwargs info
	PctKwargs float64
	// NumSource is the number of methods for which we have source code definitions
	NumSource int64
	// PctSource is the percentage of methods for which we have source code definitions
	PctSource float64

	// NumSpecsChildren is the number of methods in child packages (and children of child packages...etc) for which we have arg specs.
	NumSpecsChildren int64
	// PctSpecsChildren is the percentage of methods in child packages (and children of child packages...etc) for which we have arg specs
	PctSpecsChildren float64
	// NumPatternsChildren is the number of methods in child packages (and children of child packages...etc) for which we have signature patterns
	NumPatternsChildren int64
	// PctPatternsChildren is the percentage of methods in child packages (and children of child packages...etc) for which we have signature patterns
	PctPatternsChildren float64
	// NumDocsChildren is the number of methods in child packages (and children of child packages...etc) for which we have docs
	NumDocsChildren int64
	// PctDocsChildren is the percentage of methods in child packages (and children of child packages...etc) for which we have docs
	PctDocsChildren float64
	// NumKwargsChildren is the number of methods in child packages (and children of child packages ...etc) for which we have possible **kwargs info for
	NumKwargsChildren int64
	// PctKwargsChildren is the percentage of methods in child packages (and children of child packages ...etc) for which we have possible **kwargs info for
	PctKwargsChildren float64
	// NumSourceChildren is the number of methods in all child packages for which we have source code definitions
	NumSourceChildren int64
	// PctSourceChildren is the percentaoge of methods in child packages for which we have source code definitions
	PctSourceChildren float64
	// NumFuncsChildren is the number of methods in child packages (and children of child packages...etc)
	NumFuncsChildren int64
	// PopularityChildren is the sum popularity of all the children of the current package
	PopularityChildren int64

	// NumSpecsChildrenAndCurrent is the number of methods in child packages
	// (and children of child packages...etc) and the current package for which we have arg specs
	NumSpecsChildrenAndCurrent int64
	// PctSpecsChildrenAndCurrent is the percentage of methods in child packages
	// (and children of child packages...etc) and the current package for which we have arg specs
	PctSpecsChildrenAndCurrent float64
	// NumPatternsChildrenAndCurrent is the number of methods in child packages
	// (and children of child packages...etc) and the current package for which we have signature patterns
	NumPatternsChildrenAndCurrent int64
	// PctPatternsChildrenAndCurrent is the percentage of methods in child packages
	// (and children of child packages...etc) and the current package for which we have signature patterns
	PctPatternsChildrenAndCurrent float64
	// NumDocsChildrenAndCurrent is the number of methods in child packages
	// (and children of child packages...etc) and the current package for which we have docs
	NumDocsChildrenAndCurrent int64
	// PctDocsChildrenAndCurrent is the percentage of methods in child packages
	// (and children of child packages...etc) and the current package for which we have docs
	PctDocsChildrenAndCurrent float64
	// NumKwargsChildrenAndCurrent is the number of methods in child packages and the current package for which we have possible **kwargs info for
	NumKwargsChildrenAndCurrent int64
	// PctKwargsChildrenAndCurrent is the percentage of methods in child packages and the current package for which we have possible **kwargs info for
	PctKwargsChildrenAndCurrent float64
	// NumSourceChildrenAndCurrent is the number of methods in child packages and the current package for which we have source code definitions
	NumSourceChildrenAndCurrent int64
	// PctSourceChildrenAndCurrent is the percentage of methods in child packages and the current package for which we have source code definitions
	PctSourceChildrenAndCurrent float64
	// NumFuncsChildrenAndCurrent is the number of methods in child packages (and children of child packages...etc) and the current package
	NumFuncsChildrenAndCurrent int64
	// PopularityChildrenAndCurrent is the sum popularity of child packages (and children of child packages...etc) and the current package
	PopularityChildrenAndCurrent int64
}

func newPackageStats(name, anyname string, node *pythonimports.Node, popularity int64) *packageStats {
	return &packageStats{
		Name:       name,
		AnyName:    anyname,
		Popularity: popularity,
		Node:       node,
		Members:    make(map[string]*packageStats),
		Methods:    make(map[string]*methodStats),
	}
}

type byPkgPopChildrenAndCurrent []*packageStats

func (bp byPkgPopChildrenAndCurrent) Len() int      { return len(bp) }
func (bp byPkgPopChildrenAndCurrent) Swap(i, j int) { bp[i], bp[j] = bp[j], bp[i] }
func (bp byPkgPopChildrenAndCurrent) Less(i, j int) bool {
	return bp[i].PopularityChildrenAndCurrent < bp[j].PopularityChildrenAndCurrent
}

func sortMembers(members map[string]*packageStats) []*packageStats {
	var m []*packageStats
	for _, member := range members {
		m = append(m, member)
	}
	sort.Sort(sort.Reverse(byPkgPopChildrenAndCurrent(m)))
	return m
}

type byMethodPopularity []*methodStats

func (bp byMethodPopularity) Len() int           { return len(bp) }
func (bp byMethodPopularity) Swap(i, j int)      { bp[i], bp[j] = bp[j], bp[i] }
func (bp byMethodPopularity) Less(i, j int) bool { return bp[i].Popularity < bp[j].Popularity }

func sortMethods(methods map[string]*methodStats) []*methodStats {
	var m []*methodStats
	for _, ms := range methods {
		m = append(m, ms)
	}
	sort.Sort(sort.Reverse(byMethodPopularity(m)))
	return m
}

type countData struct {
	ArgSpecs   int64
	Docs       int64
	Patterns   int64
	Kwargs     int64
	Source     int64
	Funcs      int64
	Popularity int64
}

func countCallSignatures(pkg *packageStats, seen map[*pythonimports.Node]*packageStats) countData {
	if s := seen[pkg.Node]; s != nil {
		return countData{
			ArgSpecs:   s.NumSpecsChildrenAndCurrent,
			Docs:       s.NumDocsChildrenAndCurrent,
			Patterns:   s.NumPatternsChildrenAndCurrent,
			Kwargs:     s.NumKwargsChildrenAndCurrent,
			Source:     s.NumSourceChildrenAndCurrent,
			Funcs:      s.NumFuncsChildrenAndCurrent,
			Popularity: s.PopularityChildrenAndCurrent,
		}
	}
	seen[pkg.Node] = pkg

	var cd countData
	for _, method := range pkg.Methods {
		cd.Funcs++
		if method.HaveArgSpec {
			cd.ArgSpecs++
		}
		if method.HaveDoc {
			cd.Docs++
		}
		if method.HavePatterns {
			cd.Patterns++
		}
		if method.HaveKwargs {
			cd.Kwargs++
		}
		if method.HaveSource {
			cd.Source++
		}
		pkg.PopularityChildren += method.Popularity
	}

	pkg.NumDocs = cd.Docs
	pkg.NumSpecs = cd.ArgSpecs
	pkg.NumPatterns = cd.Patterns
	pkg.NumKwargs = cd.Kwargs
	pkg.NumSource = cd.Source
	if cd.Funcs > 0 {
		invFn := 1. / float64(cd.Funcs)
		pkg.PctDocs = float64(pkg.NumDocs) * invFn
		pkg.PctSpecs = float64(pkg.NumSpecs) * invFn
		pkg.PctPatterns = float64(pkg.NumPatterns) * invFn
		pkg.PctKwargs = float64(pkg.NumKwargs) * invFn
		pkg.PctSource = float64(pkg.NumSource) * invFn
	}

	for _, member := range pkg.Members {
		cdChild := countCallSignatures(member, seen)
		pkg.NumSpecsChildren += cdChild.ArgSpecs
		pkg.NumDocsChildren += cdChild.Docs
		pkg.NumPatternsChildren += cdChild.Patterns
		pkg.NumKwargsChildren += cdChild.Kwargs
		pkg.NumSourceChildren += cdChild.Source
		pkg.NumFuncsChildren += cdChild.Funcs
		pkg.PopularityChildren += cdChild.Popularity
	}

	if pkg.NumFuncsChildren > 0 {
		invFn := 1. / float64(pkg.NumFuncsChildren)
		pkg.PctDocsChildren = float64(pkg.NumDocsChildren) * invFn
		pkg.PctPatternsChildren = float64(pkg.NumPatternsChildren) * invFn
		pkg.PctSpecsChildren = float64(pkg.NumSpecsChildren) * invFn
		pkg.PctKwargsChildren = float64(pkg.NumKwargsChildren) * invFn
		pkg.PctSourceChildren = float64(pkg.NumSourceChildren) * invFn
	}

	pkg.NumDocsChildrenAndCurrent = pkg.NumDocsChildren + cd.Docs
	pkg.NumPatternsChildrenAndCurrent = pkg.NumPatternsChildren + cd.Patterns
	pkg.NumSpecsChildrenAndCurrent = pkg.NumSpecsChildren + cd.ArgSpecs
	pkg.NumKwargsChildrenAndCurrent = pkg.NumKwargsChildren + cd.Kwargs
	pkg.NumSourceChildrenAndCurrent = pkg.NumSourceChildren + cd.Source
	pkg.NumFuncsChildrenAndCurrent = cd.Funcs + pkg.NumFuncsChildren
	pkg.PopularityChildrenAndCurrent = pkg.Popularity + pkg.PopularityChildren
	if pkg.NumFuncsChildrenAndCurrent > 0 {
		invFn := 1. / float64(pkg.NumFuncsChildrenAndCurrent)
		pkg.PctDocsChildrenAndCurrent = float64(pkg.NumDocsChildrenAndCurrent) * invFn
		pkg.PctPatternsChildrenAndCurrent = float64(pkg.NumPatternsChildrenAndCurrent) * invFn
		pkg.PctSpecsChildrenAndCurrent = float64(pkg.NumSpecsChildrenAndCurrent) * invFn
		pkg.PctKwargsChildrenAndCurrent = float64(pkg.NumKwargsChildrenAndCurrent) * invFn
		pkg.PctSourceChildrenAndCurrent = float64(pkg.NumSourceChildrenAndCurrent) * invFn
	}

	return countData{
		ArgSpecs:   pkg.NumSpecsChildrenAndCurrent,
		Docs:       pkg.NumDocsChildrenAndCurrent,
		Patterns:   pkg.NumPatternsChildrenAndCurrent,
		Kwargs:     pkg.NumKwargsChildrenAndCurrent,
		Source:     pkg.NumSourceChildrenAndCurrent,
		Funcs:      pkg.NumFuncsChildrenAndCurrent,
		Popularity: pkg.PopularityChildrenAndCurrent,
	}
}

type coverageApp struct {
	root          *packageStats
	templates     *templateset.Set
	pkgsByPath    map[string]*packageStats
	methodsByPath map[string]*methodStats
}

func (a *coverageApp) handleTopLevel(w http.ResponseWriter, r *http.Request) {
	pkgs := sortMembers(a.root.Members)

	if err := a.templates.Render(w, "coverage/toplevel.html", map[string]interface{}{
		"Total":       int64(len(a.root.Members)),
		"NumFuncs":    a.root.NumFuncsChildren,
		"NumSpecs":    a.root.NumSpecsChildren,
		"PctSpecs":    a.root.PctSpecsChildren,
		"NumPatterns": a.root.NumPatternsChildren,
		"PctPatterns": a.root.PctPatternsChildren,
		"NumDocs":     a.root.NumDocsChildren,
		"PctDocs":     a.root.PctDocsChildren,
		"NumKwargs":   a.root.NumKwargsChildren,
		"PctKwargs":   a.root.PctKwargsChildren,
		"NumSource":   a.root.NumSourceChildren,
		"PctSource":   a.root.PctSourceChildren,
		"Pkgs":        pkgs,
	}); err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *coverageApp) handlePackage(w http.ResponseWriter, r *http.Request) {
	pkg := mux.Vars(r)["pkg"]
	stats := a.pkgsByPath[pkg]
	if stats == nil {
		webutils.ReportNotFound(w, fmt.Sprintf("unknown package `%s`", pkg))
		return
	}
	if err := a.templates.Render(w, "coverage/package.html", map[string]interface{}{
		"Pkg": stats,
	}); err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (a *coverageApp) handleMethod(w http.ResponseWriter, r *http.Request) {
	method := mux.Vars(r)["method"]
	ms := a.methodsByPath[method]
	if ms == nil {
		webutils.ReportNotFound(w, fmt.Sprintf("unknown method `%s`", method))
		return
	}
	if err := a.templates.Render(w, "coverage/method.html", map[string]interface{}{
		"Method": ms,
	}); err != nil {
		webutils.ReportError(w, err.Error())
	}
}

func (args *coverageArgs) Handle() error {
	start := time.Now()
	graph, err := pythonimports.NewGraph(args.Graph)
	if err != nil {
		return fmt.Errorf("error loading graph %s: %v", args.Graph, err)
	}
	log.Printf("took %v to load import graph\n", time.Since(start))

	start = time.Now()
	argSpecs := &pythonimports.ArgSpecs{}
	if !args.NoSpecs {
		argSpecs, err = pythonimports.LoadArgSpecs(graph, args.ImportGraphArgSpecs, args.TypeshedArgSpecs)
		if err != nil {
			return fmt.Errorf("error loading argspecs %s %s: %v", args.ImportGraphArgSpecs, args.TypeshedArgSpecs, err)
		}
		log.Printf("took %v to load arg specs\n", time.Since(start))
	}

	anynames := pythonimports.ComputeAnyPaths(graph)

	start = time.Now()
	patterns := make(map[int64]*pythoncode.MethodPatterns)
	if !args.NoPatterns {
		sigs, err := pythoncode.NewSignaturePatterns(args.Patterns, graph, pythoncode.DefaultSignatureOptions)
		if err != nil {
			return fmt.Errorf("error loading signature patterns from %s: %v", args.Patterns, err)
		}
		patterns = sigs.Index()
		log.Printf("took %v to load signature patterns\n", time.Since(start))
	}

	start = time.Now()
	var docs *pythondocs.Corpus
	if !args.NoDocs {
		docs, err = pythondocs.LoadCorpus(graph, pythondocs.DefaultSearchOptions)
		if err != nil {
			return fmt.Errorf("error loading docs: %v", err)
		}
		log.Printf("took %v to load docs\n", time.Since(start))
	}

	start = time.Now()
	pkgStats, err := pythoncode.LoadGithubPackageStats(pythoncode.DefaultPackageStats)
	if err != nil {
		return fmt.Errorf("error loading github package stats")
	}
	popularity := popularityStats(graph, pkgStats)
	log.Printf("took %v to load pkg stats\n", time.Since(start))

	start = time.Now()
	kwargs := pythoncode.NewKwargsIndex()
	if !args.NoKwargs {
		kwargs, err = pythoncode.LoadKwargsIndex(graph, pythoncode.DefaultKwargsOptions, args.Kwargs)
		if err != nil {
			return fmt.Errorf("error loading possible **kwargs `%s`: %v", args.Kwargs, err)
		}
		log.Printf("took %v to load kwargs\n", time.Since(start))

	}

	var packages []string
	if len(args.Packages) > 0 {
		packages = args.Packages
	} else {
		for pkg := range graph.Root.Members {
			packages = append(packages, pkg)
		}
	}

	start = time.Now()
	pkgsByPath := make(map[string]*packageStats)
	methodsByPath := make(map[string]*methodStats)
	var buildStats func(node *pythonimports.Node, prefix string, seen map[*pythonimports.Node]*packageStats) *packageStats
	buildStats = func(node *pythonimports.Node, prefix string, seen map[*pythonimports.Node]*packageStats) *packageStats {
		switch {
		case node == nil:
			log.Fatal("nil node!")
		case seen[node] != nil:
			return seen[node]
		}

		rootAnyName := anynames[node]
		root := newPackageStats(rootAnyName.Last(), rootAnyName.String(), node, popularity[node])
		seen[node] = root
		for name, member := range node.Members {
			anyname := anynames[member].String()
			switch {
			case member == nil:
				continue
			case !strings.HasPrefix(anyname, prefix):
				continue
			case strings.TrimPrefix(anyname, prefix+".") != name:
				continue
			}

			switch member.Classification {
			case pythonimports.Function, pythonimports.Type:
				if args.MinSeen < 1 || popularity[member] >= args.MinSeen {
					var doc *pythondocs.LangEntity
					if docs != nil {
						if res, found := docs.FindIdent(anyname); found {
							doc = res.Entity
						}
					}

					ms := &methodStats{
						Name:         name,
						AnyName:      anyname,
						Node:         member,
						Popularity:   popularity[member],
						HaveArgSpec:  argSpecs.Find(member) != nil,
						ArgSpec:      argSpecs.Find(member),
						HavePatterns: patterns[member.ID] != nil,
						Patterns:     patterns[member.ID],
						HaveDoc:      doc != nil,
						Doc:          doc,
						HaveKwargs:   kwargs.Kwargs(member) != nil,
						Kwargs:       kwargs.Kwargs(member),
					}
					root.Methods[name] = ms
					methodsByPath[anyname] = ms
				}
				if member.Classification == pythonimports.Type {
					root.Members[name] = buildStats(member, prefix+"."+name, seen)
				}
			case pythonimports.Module:
				root.Members[name] = buildStats(member, prefix+"."+name, seen)
			case pythonimports.Object:
				anyname = anynames[member.Type].String()
				switch {
				case member.Type == nil:
					continue
				case !strings.HasPrefix(anyname, prefix):
					continue
				case strings.TrimPrefix(anyname, prefix+".") != name:
					continue
				}

				root.Members[name] = buildStats(member.Type, prefix+"."+name, seen)
			}
		}

		pkgsByPath[rootAnyName.String()] = root
		return root
	}

	root := newPackageStats("root", "root", nil, 0)
	seen := make(map[*pythonimports.Node]*packageStats)
	for _, pkg := range packages {
		node, _ := graph.Find(pkg)
		if node == nil {
			fmt.Printf("No node found for package %s, skipping\n", pkg)
			continue
		}
		root.Members[pkg] = buildStats(node, pkg, seen)
	}

	seen = make(map[*pythonimports.Node]*packageStats)
	for _, member := range root.Members {
		countCallSignatures(member, seen)
		root.NumDocsChildren += member.NumDocsChildrenAndCurrent
		root.NumSpecsChildren += member.NumSpecsChildrenAndCurrent
		root.NumPatternsChildren += member.NumPatternsChildrenAndCurrent
		root.NumKwargsChildren += member.NumKwargsChildrenAndCurrent
		root.NumSourceChildren += member.NumSourceChildrenAndCurrent
		root.NumFuncsChildren += member.NumFuncsChildrenAndCurrent
		root.PopularityChildren += member.PopularityChildrenAndCurrent
	}
	if root.NumFuncsChildren > 0 {
		invFuncs := 1. / float64(root.NumFuncsChildren)
		root.PctDocsChildren = float64(root.NumDocsChildren) * invFuncs
		root.PctPatternsChildren = float64(root.NumPatternsChildren) * invFuncs
		root.PctSpecsChildren = float64(root.NumSpecsChildren) * invFuncs
		root.PctKwargsChildren = float64(root.NumKwargsChildren) * invFuncs
		root.PctSourceChildren = float64(root.NumSourceChildren) * invFuncs
	}
	log.Printf("took %v to build stats graph\n", time.Since(start))

	app := &coverageApp{
		root:          root,
		pkgsByPath:    pkgsByPath,
		methodsByPath: methodsByPath,
	}

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir}
	app.templates = templateset.NewSet(staticfs, "templates", template.FuncMap{
		"comma":          humanize.Comma,
		"sortMembers":    sortMembers,
		"sortMethods":    sortMethods,
		"len":            func(x interface{}) int64 { return int64(reflect.ValueOf(x).Len()) },
		"ftoa":           func(x float64) string { return fmt.Sprintf("%.2f", x) },
		"pkgURL":         func(anyname string) string { return fmt.Sprintf("/pkg/%s", anyname) },
		"nodeURL":        func(anyname string) string { return fmt.Sprintf("http://graph.kite.com/node/%s", anyname) },
		"methodURL":      func(anyname string) string { return fmt.Sprintf("/method/%s", anyname) },
		"argSpecString":  func(as *pythonimports.ArgSpec) string { return pretty.Sprintf("%# v", as) },
		"patternsString": func(p *pythoncode.MethodPatterns) string { return pretty.Sprintf("%# v", p) },
		"docString":      func(d *pythondocs.LangEntity) string { return pretty.Sprintf("%# v", d) },
		"kwargsString":   func(k *response.PythonKwargs) string { return pretty.Sprintf("%# v", k) },
		"color": func(m *methodStats, field string) string {
			var have bool
			switch field {
			case "docs":
				have = m.HaveDoc
			case "patterns":
				have = m.HavePatterns
			case "spec":
				have = m.HaveArgSpec
			case "kwargs":
				have = m.HaveKwargs
			case "source":
				have = m.HaveSource
			}
			if have {
				return "green"
			}
			return "red"
		},
		"mult": func(x, y float64) float64 { return x * y },
		"add":  func(x, y int) int { return x + y },
	})

	r := mux.NewRouter()
	r.HandleFunc("/pkg/{pkg}", app.handlePackage).Methods("GET")
	r.HandleFunc("/method/{method}", app.handleMethod).Methods("GET")
	r.PathPrefix("/static/").Handler(http.FileServer(staticfs))
	r.HandleFunc("/", app.handleTopLevel).Methods("GET")

	log.Println("listening on " + args.Port)
	return http.ListenAndServe(args.Port, r)
}
