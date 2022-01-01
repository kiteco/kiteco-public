package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/kennygrant/sanitize"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/templateset"
	"github.com/kiteco/kiteco/kite-golib/xmlvalidation"
)

type handlers struct {
	docModules       pythondocs.Modules
	docstringModules pythondocs.Modules
	packageStats     []*pythoncode.PackageStats
	packageStatsMap  map[string]*pythoncode.PackageStats
	builtins         map[string]string
	graph            *pythonimports.Graph
	corpus           *pythondocs.Corpus
	htmlValidator    *xmlvalidation.HTMLValidator
	templates        *templateset.Set
}

// newHandlers constructs the datasets used for the coverage inspector. It will load the
// target packets, and then determine "allowed" identifiers (identifiers we think are real
// based on how often they occur in different repos/files/etc). It then loads up aggregate
// package stats, only including target packages and allowed identifiers.
func newHandlers(pythonDocs, pythonDocstrings, targetPkgs string, templates *templateset.Set) *handlers {
	targets := loadTargets(targetPkgs)
	allowed := loadAllowedIdentifiers(defaultGroupedStats, targets)
	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		log.Fatalln(err)
	}
	graphstrings, err := pythonimports.LoadGraphStrings(pythonimports.DefaultImportGraphStrings)
	if err != nil {
		log.Fatalln(err)
	}

	docModules, err := pythondocs.NewModules(pythonDocs)
	if err != nil {
		log.Fatalln(err)
	}
	docstringModules, err := pythondocs.NewModules(pythonDocstrings)
	if err != nil {
		log.Fatalln(err)
	}

	h := &handlers{
		docModules:       docModules,
		docstringModules: docstringModules,
		packageStats:     loadPackageStats(defaultPackageStats, allowed),
		packageStatsMap:  make(map[string]*pythoncode.PackageStats),
		builtins:         loadBuiltinDocstrings(graph, graphstrings),
		graph:            graph,
		templates:        templates,
		htmlValidator:    xmlvalidation.NewHTMLValidator(pythondocs.DefaultSchemaPath),
	}

	opts := pythondocs.SearchOptions{
		DocPath:        pythonDocs,
		DocstringsPath: pythonDocstrings,
	}
	h.corpus, err = pythondocs.LoadCorpus(graph, opts)
	if err != nil {
		log.Fatalln(err)
	}

	// construct a map for quick lookup in handlePackage
	for _, stats := range h.packageStats {
		h.packageStatsMap[stats.Package] = stats
	}

	return h
}

// handleIndex is the root of the coverage inspector. It computes the overall
// coverage based on whether it can find various identifiers in the python.Modules
// dataset.
func (h *handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	type pkgData struct {
		Package               string
		Version               string
		Methods               int
		Incantations          int
		PercentOfIncantations float64
		IncantationCoverage   float64
		DocstringCoverage     float64
		CombinedCoverage      float64
		StructuredCoverage    float64
		FuncRoleCoverage      float64
		ClassRoleCoverage     float64
		MethRoleCoverage      float64
		HasRoleCoverage       float64
		RemainingGain         float64
		RowClass              string
		Valid                 string
		Normalized            string
	}

	type indexData struct {
		Packages            []*pkgData
		IncantationCoverage float64
		DocstringCoverage   float64
		CombinedCoverage    float64
		StructuredCoverage  float64
		ValidCoverage       float64
		NormalizedCoverage  float64
		FuncRoleCoverage    float64
		ClassRoleCoverage   float64
		MethRoleCoverage    float64
		HasRoleCoverage     float64
	}

	var (
		totalIncantations             int
		totalCoverageIncantations     int
		totalCoverageDocstrings       int
		totalCoverageCombined         int
		totalCoverageStructured       int
		totalCoverageFuncRole         int
		totalCoverageClassRole        int
		totalCoverageMethRole         int
		totalCoverageHasRole          int
		totalValidIdentifiers         int
		totalInvalidIdentifiers       int
		totalNormalizedIdentifiers    int
		totalNotNormalizedIdentifiers int
	)

	var pkgs []*pkgData
	for _, stats := range h.packageStats {
		var valid, normalized string
		var numValid, numNormalized int
		var incantationCoverage, docstringCoverage, fallbackCoverage, structuredCoverage, funcRoleCoverage, classRoleCoverage, methRoleCoverage, hasRoleCoverage int
		for _, m := range stats.Methods {
			if r, ok := h.corpus.FindIdent(m.Ident); ok {
				entity := r.Entity
				if entity.StructuredDoc != nil && entity.StructuredDoc.DescriptionHTML != "" {
					structuredCoverage += m.Count
					f := strings.Contains(entity.StructuredDoc.DescriptionHTML, "func_role")
					if f {
						funcRoleCoverage += m.Count
					}

					c := strings.Contains(entity.StructuredDoc.DescriptionHTML, "class_role")
					if c {
						classRoleCoverage += m.Count
					}

					me := strings.Contains(entity.StructuredDoc.DescriptionHTML, "meth_role")
					if me {
						methRoleCoverage += m.Count
					}

					if f || c || me {
						hasRoleCoverage += m.Count
					}

					if h.htmlValidator.Validate(entity.StructuredDoc.DescriptionHTML) == nil {
						totalValidIdentifiers++
						numValid++
					} else {
						totalInvalidIdentifiers++
						valid = "NO"
					}

					if cleanHTML(entity.StructuredDoc.DescriptionHTML) == entity.Doc {
						totalNormalizedIdentifiers++
						numNormalized++
					} else {
						totalNotNormalizedIdentifiers++
						normalized = "NO"
					}
				}
			}
		}
		if valid == "" && numValid > 0 {
			valid = "YES"
		}
		if normalized == "" && numNormalized > 0 {
			normalized = "YES"
		}

		totalIncantations += stats.Count
		totalCoverageIncantations += incantationCoverage
		totalCoverageDocstrings += docstringCoverage
		totalCoverageCombined += fallbackCoverage + incantationCoverage
		totalCoverageStructured += structuredCoverage
		totalCoverageFuncRole += funcRoleCoverage
		totalCoverageClassRole += classRoleCoverage
		totalCoverageMethRole += methRoleCoverage
		totalCoverageHasRole += hasRoleCoverage

		var version string
		if m := h.docModules[stats.Package]; m != nil {
			version = m.Version
		}

		pd := &pkgData{
			Package:             stats.Package,
			Version:             version,
			Methods:             len(stats.Methods),
			Incantations:        stats.Count,
			IncantationCoverage: percentage(incantationCoverage, stats.Count),
			DocstringCoverage:   percentage(docstringCoverage, stats.Count),
			CombinedCoverage:    percentage(fallbackCoverage+incantationCoverage, stats.Count),
			StructuredCoverage:  percentage(structuredCoverage, stats.Count),
			FuncRoleCoverage:    percentage(funcRoleCoverage, stats.Count),
			ClassRoleCoverage:   percentage(classRoleCoverage, stats.Count),
			MethRoleCoverage:    percentage(methRoleCoverage, stats.Count),
			HasRoleCoverage:     percentage(hasRoleCoverage, stats.Count),
			Valid:               valid,
			Normalized:          normalized,
		}

		pkgs = append(pkgs, pd)
	}

	// New package for builtins
	// Builtins are not reflected in the total counts
	var builtinsCoverage, builtinsDocstring, builtinsFallback, builtinsStructured int
	for ident, docstring := range h.builtins {
		r, found := h.corpus.FindIdent(ident)
		if found {
			builtinsCoverage++
			if r.Entity.StructuredDoc != nil && r.Entity.StructuredDoc.DescriptionHTML != "" {
				builtinsStructured++
			}
		}
		if len(docstring) > 0 {
			if !found {
				builtinsFallback++
			}
			builtinsDocstring++
		}
	}

	pkgs = append(pkgs, &pkgData{
		Package:             python.BuiltinPackage,
		Methods:             len(h.builtins),
		Incantations:        len(h.builtins),
		IncantationCoverage: percentage(builtinsCoverage, len(h.builtins)),
		DocstringCoverage:   percentage(builtinsDocstring, len(h.builtins)),
		CombinedCoverage:    percentage(builtinsFallback+builtinsCoverage, len(h.builtins)),
		StructuredCoverage:  percentage(builtinsStructured, len(h.builtins)),
	})

	for _, d := range pkgs {
		if d.Package != python.BuiltinPackage {
			d.PercentOfIncantations = percentage(d.Incantations, totalIncantations)
			d.RemainingGain = (100.0 - d.CombinedCoverage) * d.PercentOfIncantations / 100.0
		}
		if d.IncantationCoverage == 0 && d.DocstringCoverage == 0 {
			d.RowClass = "kite_danger"
		}
	}

	idxData := &indexData{
		Packages:            pkgs,
		IncantationCoverage: percentage(totalCoverageIncantations, totalIncantations),
		DocstringCoverage:   percentage(totalCoverageDocstrings, totalIncantations),
		CombinedCoverage:    percentage(totalCoverageCombined, totalIncantations),
		StructuredCoverage:  percentage(totalCoverageStructured, totalIncantations),
		ValidCoverage:       percentage(totalValidIdentifiers, totalValidIdentifiers+totalInvalidIdentifiers),
		NormalizedCoverage:  percentage(totalNormalizedIdentifiers, totalNormalizedIdentifiers+totalNotNormalizedIdentifiers),
		FuncRoleCoverage:    percentage(totalCoverageFuncRole, totalIncantations),
		ClassRoleCoverage:   percentage(totalCoverageClassRole, totalIncantations),
		MethRoleCoverage:    percentage(totalCoverageMethRole, totalIncantations),
		HasRoleCoverage:     percentage(totalCoverageHasRole, totalIncantations),
	}

	if err := h.templates.Render(w, "index.html", idxData); err != nil {
		log.Println(err.Error())
	}
}

type identifierData struct {
	Identifier       string
	Incantations     int
	PercentOfPackage float64
	HasDescription   string
	ValidHTML        string
	RowClass         string
	Normalized       string
	FuncRole         string
	ClassRole        string
	MethRole         string
}

type packageData struct {
	Package             string
	IncantationCoverage float64
	DocstringCoverage   float64
	CombinedCoverage    float64
	ValidHTMLCoverage   float64
	NormalizedCoverage  float64
	StructuredCoverage  float64
	Identifiers         []*identifierData
}

// handlePackages displays individual identifiers within a package, showing the most popular
// identifiers and whether we have them in the python documentation dataset.
func (h *handlers) handlePackage(w http.ResponseWriter, r *http.Request) {
	pkg := r.URL.Query().Get("q")
	if pkg == python.BuiltinPackage {
		h.handleBuiltinPackage(w, r)
		return
	}
	pkgStats, exists := h.packageStatsMap[pkg]
	if !exists {
		http.Error(w, fmt.Sprintf("%s not found", pkg), http.StatusNotFound)
		return
	}

	pkgData := &packageData{Package: pkg}

	for _, m := range pkgStats.Methods {
		pkgData.Identifiers = append(pkgData.Identifiers, &identifierData{
			Identifier:   m.Ident,
			Incantations: m.Count,
		})
	}

	var incantations, docstrings, fallback, structured, valid, invalid, normalized, notNormalized int
	for _, ident := range pkgData.Identifiers {
		ident.PercentOfPackage = percentage(ident.Incantations, pkgStats.Count)
		ident.RowClass = "kite_danger"

		node, err := h.graph.Find(ident.Identifier)
		if err != nil {
			continue
		}

		if entity, ok := h.corpus.Entity(node); ok {
			if entity.StructuredDoc != nil && entity.StructuredDoc.DescriptionHTML != "" {
				structured += ident.Incantations

				if strings.Contains(entity.StructuredDoc.DescriptionHTML, "func_role") {
					ident.FuncRole = "YES"
				}
				if strings.Contains(entity.StructuredDoc.DescriptionHTML, "class_role") {
					ident.ClassRole = "YES"
				}
				if strings.Contains(entity.StructuredDoc.DescriptionHTML, "meth_role") {
					ident.MethRole = "YES"
				}

				if h.htmlValidator.Validate(entity.StructuredDoc.DescriptionHTML) == nil {
					ident.ValidHTML = "YES"
					valid++
				} else {
					ident.ValidHTML = "NO"
					invalid++
				}
				if cleanHTML(entity.StructuredDoc.DescriptionHTML) == entity.Doc {
					ident.Normalized = "YES"
					normalized++
				} else {
					ident.Normalized = "NO"
					notNormalized++
				}
			}
		}
	}

	pkgData.IncantationCoverage = percentage(incantations, pkgStats.Count)
	pkgData.DocstringCoverage = percentage(docstrings, pkgStats.Count)
	pkgData.CombinedCoverage = percentage(fallback+incantations, pkgStats.Count)
	pkgData.StructuredCoverage = percentage(structured, pkgStats.Count)
	pkgData.ValidHTMLCoverage = percentage(valid, valid+invalid)
	pkgData.NormalizedCoverage = percentage(normalized, normalized+notNormalized)
	if err := h.templates.Render(w, "package.html", pkgData); err != nil {
		log.Println(err.Error())
	}
}

func (h *handlers) handleBuiltinPackage(w http.ResponseWriter, r *http.Request) {
	pkgData := &packageData{Package: python.BuiltinPackage}

	for ident := range h.builtins {
		pkgData.Identifiers = append(pkgData.Identifiers, &identifierData{
			Identifier:   ident,
			Incantations: 1,
		})
	}

	var incantations, docstrings, fallback int
	for _, ident := range pkgData.Identifiers {
		ident.PercentOfPackage = percentage(ident.Incantations, len(pkgData.Identifiers))
		ident.RowClass = "danger"

		if docstring, ok := h.builtins[ident.Identifier]; ok && len(docstring) > 0 {
			if ident.RowClass == "kite_danger" {
				fallback++
			}
			docstrings++
			ident.RowClass = ""
		}
	}

	pkgData.IncantationCoverage = percentage(incantations, len(pkgData.Identifiers))
	pkgData.DocstringCoverage = percentage(docstrings, len(pkgData.Identifiers))
	pkgData.CombinedCoverage = percentage(fallback+incantations, len(pkgData.Identifiers))
	if err := h.templates.Render(w, "package.html", pkgData); err != nil {
		log.Println(err.Error())
	}
}

// handleDocumentation shows the signature and description of a given identifier
func (h *handlers) handleDocumentation(w http.ResponseWriter, r *http.Request) {
	ident := r.URL.Query().Get("q")
	n, err := h.graph.Find(ident)
	if err != nil {
		http.Error(w, fmt.Sprintf("no doc entries for %s", ident), http.StatusNotFound)
		return
	}

	entity, exists := h.corpus.Entity(n)
	if !exists {
		http.Error(w, fmt.Sprintf("no doc entries for %s", ident), http.StatusNotFound)
		return
	}

	type parameter struct {
		Type    string
		Name    string
		Default string

		Description string
	}

	type docData struct {
		Identifier      string
		Kind            string
		Description     string
		DescriptionHTML string
		Docstring       string
		Parameters      []parameter
		Entity          *pythondocs.LangEntity
	}

	var params []parameter
	var descr string
	var html string
	if entity.StructuredDoc != nil {
		for _, p := range entity.StructuredDoc.Parameters {
			params = append(params, parameter{
				Type:        p.Type.String(),
				Name:        p.Name,
				Default:     p.Default,
				Description: p.DescriptionHTML,
			})
		}
		descr = entity.Doc
		html = entity.StructuredDoc.DescriptionHTML
	}

	data := &docData{
		Identifier:      ident,
		Kind:            entity.Kind.String(),
		Description:     descr,
		DescriptionHTML: html,
		Parameters:      params,
		Entity:          entity,
	}

	if err := h.templates.Render(w, "documentation.html", data); err != nil {
		log.Println(err.Error())
	}
}

// --

func cleanHTML(htmlStr string) string {
	htmlStr = sanitize.HTML(htmlStr)
	var ret string
	for _, c := range htmlStr {
		switch c {
		case rune('\u00B6'): // filter our paragraph symbol
		default:
			ret += string(c)
		}
	}
	return ret
}

func percentage(numerator, denominator int) float64 {
	return 100.0 * float64(numerator) / float64(denominator)
}
