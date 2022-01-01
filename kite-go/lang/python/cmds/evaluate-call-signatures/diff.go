package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

type pkgAndMissing struct {
	Pkg     string
	Missing []string
	Total   int64
	Pct     float64
}

type byPercent []pkgAndMissing

func (bm byPercent) Len() int           { return len(bm) }
func (bm byPercent) Swap(i, j int)      { bm[i], bm[j] = bm[j], bm[i] }
func (bm byPercent) Less(i, j int) bool { return bm[i].Pct < bm[j].Pct }

func pkgFromName(name string) string {
	pos := strings.Index(name, ".")
	if pos > -1 {
		return name[:pos]
	}
	return name
}

func patternsByPackage(graph *pythonimports.Graph, patterns *pythoncode.SignaturePatterns) (map[string]map[int64]*pythoncode.MethodPatterns, []string) {
	pkgPatterns := make(map[string]map[int64]*pythoncode.MethodPatterns)
	var skipped []string
	for id, mp := range patterns.Index() {
		node, ok := graph.FindByID(id)
		if !ok {
			log.Fatalf("could not find node for id: %d, patterns: %v\n", id, *mp)
		}

		pkg := node.CanonicalName.Head()
		if pkg == "" {
			pkg = pkgFromName(mp.Method)
			if pkg == "" {
				skipped = append(skipped, mp.Method)
			}
		}

		if pkgPatterns[pkg] == nil {
			pkgPatterns[pkg] = make(map[int64]*pythoncode.MethodPatterns)
		}
		pkgPatterns[pkg][node.ID] = mp
	}
	return pkgPatterns, skipped
}

type diffArgs struct {
	Master      string `arg:"help:Master signature patterns"`
	Candidate   string `arg:"positional,required,help:Candidate signature patterns"`
	Verbose     bool
	Threshold   float64  `arg:"help:Percent [0 - 1] patterns missing to trigger log"`
	MinPatterns uint     `arg:"help:Min number of patterns for package to trigger log"`
	Packages    []string `arg:"positional,help:Restrict diff to patterns for this package"`
	Output      string   `arg:"help:path to output results to"`
}

var diffCmd = cmdline.Command{
	Name:     "diff",
	Synopsis: "diff two sets of signature patterns, only output packages that are missing in candidate but present in master",
	Args: &diffArgs{
		Master:      pythoncode.DefaultSignaturePatterns,
		Threshold:   0.1,
		MinPatterns: 50,
		Output:      "diff.json",
	},
}

// TODO(juan): add way to view new patterns added?
func (args *diffArgs) Handle() error {
	start := time.Now()
	if args.Threshold < 0. || args.Threshold > 1. {
		return fmt.Errorf("Threshold %f must be in interval [0,1]", args.Threshold)
	}

	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		return fmt.Errorf("error loading graph %s: %v", pythonimports.DefaultImportGraph, err)
	}
	anynames := pythonimports.ComputeAnyPaths(graph)

	master, err := pythoncode.NewSignaturePatterns(args.Master, graph, pythoncode.DefaultSignatureOptions)
	if err != nil {
		return fmt.Errorf("error loading signature patterns from %s: %v", args.Master, err)
	}

	candidate, err := pythoncode.NewSignaturePatterns(args.Candidate, graph, pythoncode.DefaultSignatureOptions)
	if err != nil {
		return fmt.Errorf("error loading signature patterns from %s: %v", args.Candidate, err)
	}

	masterPkgs, masterSkipped := patternsByPackage(graph, master)
	fmt.Printf("%d Master patterns missing package and skipped.\n", len(masterSkipped))
	if args.Verbose {
		fmt.Printf("Master patterns skipped: %v\n", masterSkipped)
		fmt.Println()
	}

	candidatePkgs, candidateSkipped := patternsByPackage(graph, candidate)
	fmt.Printf("%d Candidate patterns missing package and skipped. \n", len(candidateSkipped))
	if args.Verbose {
		fmt.Printf("Candidate patterns skipped: %v\n", candidateSkipped)
		fmt.Println()
	}
	fmt.Println()

	var packages []string
	if len(args.Packages) > 0 {
		packages = args.Packages
	} else {
		for pkg := range masterPkgs {
			packages = append(packages, pkg)
		}
	}

	var pms []pkgAndMissing
	for _, pkg := range packages {
		patterns := masterPkgs[pkg]
		if len(patterns) == 0 {
			if args.Verbose {
				fmt.Printf("skipping package %s since master version has no patterns\n", pkg)
			}
			continue
		}
		pm := pkgAndMissing{
			Pkg: pkg,
		}

		candidates := candidatePkgs[pkg]
		if candidates == nil {
			for id := range patterns {
				node, found := graph.FindByID(id)
				if !found {
					log.Fatalf("no node found for id %d\n", id)
				}

				anyname := anynames[node]
				if anyname.Head() != pkg {
					// skip candidates from other packages
					continue
				}
				pm.Total++
				pm.Missing = append(pm.Missing, anyname.String())
			}
			pm.Pct = 100.
			pms = append(pms, pm)
			continue
		}

		for id := range patterns {
			node, found := graph.FindByID(id)
			if !found {
				log.Fatalf("no node found for id %d\n", id)
			}

			anyname := anynames[node]
			if anyname.Head() != pkg {
				// skip functions from other packages
				continue
			}

			pm.Total++
			if _, found := candidates[id]; !found {
				pm.Missing = append(pm.Missing, anyname.String())
			}
		}

		if pm.Total > 0 {
			pm.Pct = 100. * float64(len(pm.Missing)) / float64(pm.Total)
		}

		if len(pm.Missing) > 0 {
			pms = append(pms, pm)
		}
	}
	sort.Sort(sort.Reverse(byPercent(pms)))

	for _, pm := range pms {
		pkg, missing, pct := pm.Pkg, pm.Missing, pm.Pct

		if args.MinPatterns > 0 && len(masterPkgs[pkg]) < int(args.MinPatterns) {
			continue
		}

		if pct < args.Threshold {
			continue
		}

		fmt.Printf("Package %s missing %d of %d patterns (%.1f percent)\n",
			pkg, len(missing), len(masterPkgs[pkg]), pct)
		if args.Verbose {
			fmt.Printf("Missing: %v\n", missing)
			fmt.Println()
		}
	}

	enc, err := serialization.NewEncoder(args.Output)
	if err != nil {
		log.Fatalf("error getting encoder for %s: %v\n", args.Output, err)
	}
	defer enc.Close()

	for _, pm := range pms {
		if err := enc.Encode(pm); err != nil {
			log.Fatalf("error encoding %v: %v\n", pm, err)
		}
	}

	fmt.Printf("Done! Diff took %v\n", time.Since(start))
	return nil
}
