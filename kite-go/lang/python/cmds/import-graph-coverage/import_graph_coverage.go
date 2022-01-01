//go:generate go-bindata -o bindata.go templates/...

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/jsonutil"
)

const defaultUsages = "s3://kite-emr/users/tarak/python-code-examples/2015-05-19_10-28-59-PM/merge_group_obj_usages/output/part-00000"

// struct to hold per-package Counts
type packageCounts struct {
	Name     string
	Found    int
	NotFound int
	Counts   map[string]int
}

type byNotFound []*packageCounts

func (x byNotFound) Len() int           { return len(x) }
func (x byNotFound) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x byNotFound) Less(i, j int) bool { return x[i].NotFound < x[j].NotFound }

type byNumUnique []*packageCounts

func (x byNumUnique) Len() int           { return len(x) }
func (x byNumUnique) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x byNumUnique) Less(i, j int) bool { return len(x[i].Counts) < len(x[j].Counts) }

type identifierResult struct {
	Name  string
	Count int
}

type packageResult struct {
	Package     string
	Found       int
	NotFound    int
	TopFound    []identifierResult
	TopNotFound []identifierResult
}

type byCount []identifierResult

func (x byCount) Len() int           { return len(x) }
func (x byCount) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x byCount) Less(i, j int) bool { return x[i].Count < x[j].Count }

func topN(xs []identifierResult, n int) []identifierResult {
	if len(xs) > n {
		return xs[:n]
	}
	return xs
}

func writeReport(w io.Writer, pkgs []*packageCounts) error {
	// Compile the results
	var results []packageResult
	for _, pkg := range pkgs {
		// compile lists of identifiers
		var found, notfound []identifierResult
		for k, v := range pkg.Counts {
			if v < 0 {
				notfound = append(notfound, identifierResult{k, -v})
			} else {
				found = append(found, identifierResult{k, v})
			}
		}

		// take the top N of each list
		sort.Sort(sort.Reverse(byCount(found)))
		sort.Sort(sort.Reverse(byCount(notfound)))
		found = topN(found, 20)
		notfound = topN(notfound, 20)

		results = append(results, packageResult{
			Package:     pkg.Name,
			Found:       pkg.Found,
			NotFound:    pkg.NotFound,
			TopFound:    found,
			TopNotFound: notfound,
		})
	}

	// Construct template
	data := MustAsset("templates/result.html")
	tpl, err := template.New("results").Parse(string(data))
	if err != nil {
		return fmt.Errorf("error parsing template: %v", err)
	}

	// Render template
	err = tpl.Execute(w, map[string]interface{}{
		"Results": results,
	})
	if err != nil {
		return fmt.Errorf("error executing template: %v", err)
	}

	return nil
}

// If the root package in the dotten name S is PREFIX then replace it with REPLACE
func rewritePackage(s, prefix, replace string) string {
	// transform "np" -> "numpy" and "np.zeros" -> "numpy.zeros" but not npoly
	if s == prefix {
		return replace
	}
	prefix += "."
	if strings.HasPrefix(s, prefix) {
		return replace + "." + strings.TrimPrefix(s, prefix)
	}
	return s
}

// Map from fully qualified names to an indicator of whether that name exists in the import graph
var cache = make(map[string]bool)

// Determine whether a fully qualified name exists in the import graph
func isPresent(graph *pythonimports.Graph, fqn string) bool {
	if present, cached := cache[fqn]; cached {
		return present
	}
	present := isPresentImpl(graph, fqn)
	cache[fqn] = present
	return present
}

// Determine whether a fully qualified name exists in the import graph
func isPresentImpl(graph *pythonimports.Graph, fqn string) bool {
	fqn = rewritePackage(fqn, "np", "numpy")
	fqn = rewritePackage(fqn, "QtGui", "PyQt4.QtGui")
	fqn = rewritePackage(fqn, "QtCore", "PyQt4.QtCore")
	fqn = rewritePackage(fqn, "plt", "matplotlib.pyplot")

	_, err := graph.Find(fqn)
	if err != nil {
		_, err = graph.Find("builtins." + fqn)
	}
	return err == nil
}

func main() {
	var verbose bool
	var reportlimit int
	var importgraph, usages, report, found, counts, loadcounts, pkgfilter string
	flag.StringVar(&importgraph, "importgraph", pythonimports.DefaultImportGraph, "path to import graph")
	flag.StringVar(&usages, "usages", defaultUsages, "path to github usages")
	flag.StringVar(&report, "report", "", "path to which HTML counts will be written")
	flag.IntVar(&reportlimit, "reportlimit", 10000, "limit on number of items to report")
	flag.StringVar(&found, "found", "", "path to write symbols that were found in the import graph")
	flag.StringVar(&counts, "counts", "", "path to write package counts to")
	flag.StringVar(&loadcounts, "loadcounts", "",
		"read package counts from a file (rather than computing them), "+
			"the format is the same as that output by this command using '-counts PATH'")
	flag.BoolVar(&verbose, "verbose", false, "verbose output mode")
	flag.StringVar(&pkgfilter, "pkgfilter", "", "only process usages that belong to this package")
	flag.Parse()

	var pkgs []*packageCounts
	if loadcounts != "" {
		// Read package stats from a file
		err := jsonutil.DecodeAllFrom(loadcounts, func(pkg *packageCounts) {
			pkgs = append(pkgs, pkg)
		})
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		// Load import graph
		graph, err := pythonimports.NewGraph(importgraph)
		if err != nil {
			log.Fatalln(err)
		}

		// Open the usage file
		f, err := fileutil.NewCachedReader(usages)
		if err != nil {
			log.Fatalln(err)
		}
		defer f.Close()

		// Open the JSON output file
		var enc *json.Encoder
		if counts != "" {
			jsonWr, err := os.Create(counts)
			if err != nil {
				log.Fatalln(err)
			}
			defer jsonWr.Close()
			enc = json.NewEncoder(jsonWr)
		}

		// Open the found and notfound text files
		var textWr io.WriteCloser
		if found != "" {
			textWr, err = os.Create(found)
			if err != nil {
				log.Fatalln(err)
			}
			defer textWr.Close()
		}

		var numRecords, numUnique, recordsFound, recordsNotFound, uniqueFound, uniqueNotFound int
		var pkg *packageCounts

		// Construct a reporting function so that we can report intermediate results in the same
		// format as we use at the end.
		printCoverage := func() {
			log.Printf("Overall coverage: (based on %d records)", numRecords)
			recordsTotal := recordsFound + recordsNotFound
			uniqueTotal := uniqueFound + uniqueNotFound
			recordsFoundFrac := float64(recordsFound) / float64(recordsTotal)
			uniqueFoundFrac := float64(uniqueFound) / float64(uniqueTotal)
			log.Printf("  %.2f%% usages found (%d of %d)\n", recordsFoundFrac*100., recordsFound, recordsTotal)
			log.Printf("  %.2f%% unique names found (%d of %d)\n", uniqueFoundFrac*100., uniqueFound, uniqueTotal)
		}

		r := awsutil.NewEMRIterator(f)
		for r.Next() {
			numRecords++
			if numRecords%100000 == 0 {
				printCoverage()
			}

			// Get the package name
			fqn := r.Key()
			pkgName := fqn
			if pos := strings.Index(pkgName, "."); pos != -1 {
				pkgName = pkgName[:pos]
			}
			if pkgfilter != "" && pkgName != pkgfilter {
				continue
			}

			if pkg == nil || pkgName != pkg.Name {
				if pkg != nil {
					pkgs = append(pkgs, pkg)
					if enc != nil {
						enc.Encode(pkg)
					}
				}
				pkg = &packageCounts{
					Name:   pkgName,
					Counts: make(map[string]int),
				}
			}

			// Find the node in the graph
			_, seen := pkg.Counts[fqn]
			present := isPresent(graph, fqn)

			var status string
			if present {
				status = "FOUND"
				pkg.Counts[fqn]++
				pkg.Found++
				recordsFound++
				if !seen {
					uniqueFound++
				}
			} else {
				status = "MISSING"
				pkg.Counts[fqn]--
				pkg.NotFound++
				recordsNotFound++
				if !seen {
					uniqueNotFound++
				}
			}

			if !seen {
				numUnique++
				if textWr != nil {
					fmt.Fprintf(textWr, "%s\t%s\n", status, fqn)
				}
				if verbose {
					log.Printf("%s\t%s\n", status, fqn)
				}
			}
		}

		// Write final summary
		printCoverage()
	}

	// sort the packages
	sort.Sort(sort.Reverse(byNotFound(pkgs)))
	if len(pkgs) > reportlimit {
		pkgs = pkgs[:reportlimit]
	}

	// Construct html
	if report != "" {
		w, err := os.Create(report)
		if err != nil {
			log.Fatalln(err)
		}
		defer w.Close()

		err = writeReport(w, pkgs)
		if err != nil {
			log.Fatalf("Error rendering report: %v", err)
		}
	}
}
