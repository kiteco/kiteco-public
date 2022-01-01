package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/text"
)

var (
	defaultStatsFile = "s3://kite-emr/datasets/github-stats/2015-10-20_13-39-09-PM/package-stats.emr"
)

// This binary uses import graphs to find the submodules within a package,
// and sort them by popularity on github. The output if a list of PackageStats
// objects.

func main() {
	var (
		packageList string
		statsFile   string
		output      string
		depth       int
	)

	flag.StringVar(&packageList, "p", "", "list of package names (.txt)")
	flag.StringVar(&statsFile, "s", defaultStatsFile, "package stats (package-stats.emr)")
	flag.StringVar(&output, "o", "", "output file containing a list of PackageStats objs (.json)")
	flag.IntVar(&depth, "d", 2, "module depth to explore (package level is 1)")
	flag.Parse()

	if packageList == "" || output == "" || statsFile == "" {
		flag.Usage()
		log.Fatal("must specify --p, --s, --o")
	}
	packages := loadPackageNames(packageList)

	// load import graph
	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		log.Fatal(err)
	}

	// find the package's modules
	packageModules := findPackageModules(graph, packages, depth)

	// filter package stats
	packageStats := filterPackageStats(statsFile, packageModules)

	// save the output to a json file
	fout, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	encoder := json.NewEncoder(fout)
	err = encoder.Encode(packageStats)
	if err != nil {
		log.Fatal(err)
	}
}

// findPackageModules returns a map from a package name to modules
// in this package which are explored at the given depth.
func findPackageModules(graph *pythonimports.Graph, packages []string, depth int) map[string][]string {
	packageModules := make(map[string][]string)
	for _, p := range packages {
		var modules []string
		err := graph.Walk(p, func(name string, node *pythonimports.Node) bool {
			tokens := strings.Split(name, ".")
			if len(tokens) == depth && node.Classification == pythonimports.Module {
				modules = append(modules, name)
			}
			if len(tokens) < depth {
				return true
			}
			return false
		})
		if err != nil {
			log.Println(err)
		}
		modules = text.Uniquify(modules)
		packageModules[p] = modules
	}
	return packageModules
}

// loadPackageNames returns the list of package names specified in packageList.
func loadPackageNames(in string) []string {
	file, err := os.Open(in)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var packages []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		packages = append(packages, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return packages
}

// filterPackageStats loads package stats gathered from github.
func filterPackageStats(in string, packageModules map[string][]string) []*pythoncode.PackageStats {
	f, err := fileutil.NewCachedReader(in)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	statsMap := make(map[string]*pythoncode.PackageStats)

	iter := awsutil.NewEMRIterator(f)
	for iter.Next() {
		var pstats pythoncode.PackageStats
		if err := json.Unmarshal(iter.Value(), &pstats); err != nil {
			log.Fatal(err)
		}
		statsMap[pstats.Package] = &pstats
	}

	if err := iter.Err(); err != nil {
		log.Fatal(err)
	}

	var packageStats []*pythoncode.PackageStats
	for p, modules := range packageModules {
		if pstats, ok := statsMap[p]; ok {
			methods := filterMethodStats(modules, pstats.Methods)
			packageStats = append(packageStats, &pythoncode.PackageStats{
				Package: pstats.Package,
				Count:   pstats.Count,
				Methods: methods,
			})
		} else {
			packageStats = append(packageStats, &pythoncode.PackageStats{Package: p})
		}
	}
	return packageStats
}

// filterMethodStats filters the method stats we gather from github down
// to those for the methods we see in the import graph.
func filterMethodStats(names []string, methods []*pythoncode.MethodStats) []*pythoncode.MethodStats {
	if len(names) < 1 {
		return nil
	}
	depth := len(strings.Split(names[0], "."))
	// find all entities at the given depth in methods
	seenMethods := make(map[string]int)
	for _, m := range methods {
		tokens := strings.Split(m.Ident, ".")
		if len(tokens) >= depth {
			name := strings.Join(tokens[:depth], ".")
			seenMethods[name] += m.Count
		}
	}
	var methodStats []*pythoncode.MethodStats
	for _, name := range names {
		methodStats = append(methodStats, &pythoncode.MethodStats{
			Ident: name,
			Count: seenMethods[name],
		})
	}
	return methodStats
}
