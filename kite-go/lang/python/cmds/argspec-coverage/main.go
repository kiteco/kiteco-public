package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

func main() {
	var (
		importGraphPath     string
		importGraphArgSpecs string
		typeshedArgSpecs    string
		csvPath             string
		outputPath          string
	)

	flag.StringVar(&importGraphPath, "importGraph", pythonimports.DefaultImportGraph, "import graph path")
	flag.StringVar(&importGraphArgSpecs, "importGraphArgSpecs", pythonimports.DefaultImportGraphArgSpecs, "arg specs path")
	flag.StringVar(&typeshedArgSpecs, "typeshedArgSpecs", pythonimports.DefaultTypeshedArgSpecs, "arg specs path")
	flag.StringVar(&csvPath, "csv", "./packages.csv", "packages csv file path")
	flag.StringVar(&outputPath, "output", "./coverage.out", "output file path")

	csv, err := os.Create(csvPath)
	if err != nil {
		log.Fatalln("error opening csv file:", err)
	}
	defer csv.Close()

	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatalln("error opening output file:", err)
	}
	defer f.Close()

	log.Printf("loading graph from %s...", importGraphPath)
	importGraph, err := pythonimports.NewGraph(importGraphPath)
	if err != nil {
		log.Fatalln("error loading graph:", err)
	}

	log.Printf("loading arg specs from %s, %s...", importGraphArgSpecs, typeshedArgSpecs)
	argSpecs, err := pythonimports.LoadArgSpecs(importGraph, importGraphArgSpecs, typeshedArgSpecs)
	if err != nil {
		log.Fatalln("error loading arg specs:", err)
	}

	var tot int
	pkgs := make(map[string]int)
	var missingFunctions, missingTypes []string

	for pkg := range importGraph.PkgToNode {
		importGraph.Walk(pkg, func(name string, node *pythonimports.Node) bool {
			if node == nil {
				return false
			}
			if node.Classification == pythonimports.Function {
				tot++
				if spec := argSpecs.Find(node); spec == nil {
					pkgs[pkg]++
					missingFunctions = append(missingFunctions, name)
				}
			}
			if node.Classification == pythonimports.Type {
				tot++
				if spec := argSpecs.Find(node); spec == nil {
					pkgs[pkg]++
					missingTypes = append(missingTypes, name)
				}
			}
			return true
		})
	}

	log.Printf("%d functions or types (%.2f%% coverage)\n", tot, 100.0*(1.0-float64(len(missingFunctions)+len(missingTypes))/float64(tot)))
	log.Printf("%d missing functions", len(missingFunctions))
	log.Printf("%d missing types", len(missingTypes))

	for pkg, cnt := range pkgs {
		csv.WriteString(fmt.Sprintf("%s,%d\n", pkg, cnt))
	}
	csv.Sync()

	for _, n := range missingFunctions {
		f.WriteString(n + "\n")
	}
	for _, n := range missingTypes {
		f.WriteString(n + "\n")
	}
	f.Sync()
}
