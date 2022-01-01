package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

func main() {
	var (
		importGraphPath   string
		matchingCsvPath   string
		mismatchedCsvPath string
	)

	defaultOpts := python.DefaultServiceOptions

	flag.StringVar(&importGraphPath, "importGraph", defaultOpts.ImportGraph, "import graph path")
	flag.StringVar(&matchingCsvPath, "matchingOutput", "./matching.csv", "matching names csv file path")
	flag.StringVar(&mismatchedCsvPath, "mismatchOutput", "./mismatch.csv", "mismatched names packages csv file path")

	matchingCsv, err := os.Create(matchingCsvPath)
	if err != nil {
		log.Fatalln("error opening csv file:", err)
	}
	defer matchingCsv.Close()

	mismatchedCsv, err := os.Create(mismatchedCsvPath)
	if err != nil {
		log.Fatalln("error opening csv file:", err)
	}
	defer mismatchedCsv.Close()

	log.Printf("loading graph from %s...", importGraphPath)
	importGraph, err := pythonimports.NewGraph(importGraphPath)
	if err != nil {
		log.Fatalln("error loading graph:", err)
	}

	var tot, totMissing, totFilled, totMatching int
	matchingNames := make(map[string]string)
	mismatchedNames := make(map[string]string)

	for pkg := range importGraph.PkgToNode {
		importGraph.Walk(pkg, func(name string, node *pythonimports.Node) bool {
			tot++
			if node == nil {
				totMissing++
				return false
			}
			if node.CanonicalName.Empty() {
				totMissing++
				path := importGraph.AnyPaths[node]
				if !path.Empty() {
					totFilled++
					if path.Head() == pkg {
						totMatching++
						matchingNames[name] = path.String()
					} else {
						mismatchedNames[name] = path.String()
					}
				}
			}
			return true
		})
	}

	log.Printf("%d missing names (%.2f%% missing), %d filled (%.2f%% coverage), %d matching (%.2f%% coverage)",
		totMissing, 100.0*float64(totMissing)/float64(tot),
		totFilled, 100.0*float64(totFilled)/float64(totMissing),
		totMatching, 100.0*float64(totMatching)/float64(totMissing))

	for n, m := range matchingNames {
		matchingCsv.WriteString(fmt.Sprintf("%s,%s\n", n, m))
	}
	matchingCsv.Sync()

	for n, m := range mismatchedNames {
		mismatchedCsv.WriteString(fmt.Sprintf("%s,%s\n", n, m))
	}
	mismatchedCsv.Sync()
}
