package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"strings"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func isFinite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

func assertFinite(x float64, fmt string, args ...interface{}) {
	if !isFinite(x) {
		log.Fatalf(fmt, args...)
	}
}

func loadFunctionsInGraph(
	tables string,
	verbose bool,
	graph *pythonimports.Graph) map[*pythonimports.Node]*typeinduction.Function {

	var total, skipped int
	var noNode []string
	ftables := make(map[*pythonimports.Node]*typeinduction.Function)
	if err := serialization.Decode(typeinduction.OptionsFromPath(tables).Functions, func(f *typeinduction.Function) {
		total++
		fnode, err := graph.Find(f.Name)
		if err != nil || fnode == nil || graph.AnyPaths[fnode].Empty() {
			if verbose {
				noNode = append(noNode, f.Name)
			}
			skipped++
			return
		}
		ftables[fnode] = f
	}); err != nil {
		log.Fatal(err)
	}

	if verbose {
		fmt.Printf("\n=== Skipped functions from tables ===\n")
		for _, f := range noNode {
			fmt.Printf("  %s\n", f)
		}
	}

	fmt.Printf("Skipped %d (of %d) functions from tables\n", skipped, total)
	return ftables
}

func resolveType(t string, graph *pythonimports.Graph) *pythonimports.Node {
	tn, err := graph.Find(t)
	if tn != nil && err == nil {
		return tn
	}

	// builtins not included for builtin types
	tn, err = graph.Find("builtins." + t)
	if tn != nil && err == nil {
		return tn
	}

	// types not included for members of types package
	tn, err = graph.Find("types." + t)
	if tn != nil && err == nil {
		return tn
	}

	if t == "generator" {
		tn, err = graph.Find("types.GeneratorType")
		if err != nil || tn == nil {
			log.Fatalln("unable to find types.GeneratorType in graph!")
		}
		return tn
	}

	if t == "function" {
		tn, err = graph.Find("types.FunctionType")
		if err != nil || tn == nil {
			log.Fatalln("unable to find types.FunctionType in graph!")
		}
		return tn
	}

	return nil
}

func loadReturnTypes(
	returntypes string,
	verbose bool,
	graph *pythonimports.Graph) map[*pythonimports.Node][]*pythonimports.Node {

	var rawreturns map[string][]string
	if err := serialization.Decode(returntypes, &rawreturns); err != nil {
		log.Fatal(err)
	}

	var skipped int
	fnrtsmap := make(map[*pythonimports.Node]map[*pythonimports.Node]struct{})
	var noNodeFns, noRTFns []string
	noNodeTypes := make(map[string][]string)
	for f, ts := range rawreturns {
		f = strings.TrimSpace(f)

		fnode, err := graph.Find(f)
		if err != nil || fnode == nil || graph.AnyPaths[fnode].Empty() {
			if verbose {
				noNodeFns = append(noNodeFns, f)
			}
			skipped++
			continue
		}

		tnodes := make(map[*pythonimports.Node]struct{})
		var empty int
		for _, t := range ts {
			t = strings.TrimSpace(t)
			if t == "" {
				empty++
				continue
			}

			tnode := resolveType(t, graph)
			if tnode == nil || graph.AnyPaths[tnode].Empty() {
				if verbose {
					noNodeTypes[f] = append(noNodeTypes[f], t)
				}
				continue
			}

			tnodes[tnode] = struct{}{}
		}

		switch {
		case empty == len(ts):
		case len(tnodes) > 0:
			fnrtsmap[fnode] = tnodes
		default:
			if verbose {
				noRTFns = append(noRTFns, f)
			}
			skipped++
		}
	}

	if verbose {
		fmt.Printf("\n=== %d functions with no import graph node or no anypath ===\n", len(noNodeFns))
		for _, f := range noNodeFns {
			fmt.Printf("  %s\n", f)
		}

		fmt.Printf("\n=== %d functions with no return types in import graph ===\n", len(noRTFns))
		seen := make(map[string]bool)
		for _, f := range noRTFns {
			fmt.Printf("  %s\n", f)
			for _, t := range noNodeTypes[f] {
				fmt.Printf("    %s\n", t)
			}
			seen[f] = true
		}

		fmt.Printf("\n=== Missing return types by function ===\n")
		for f, ts := range noNodeTypes {
			if seen[f] {
				continue
			}
			fmt.Printf("  %s\n", f)
			for _, t := range ts {
				fmt.Printf("    %s\n", t)
			}
		}
	}

	fmt.Printf("Skipped %d (of %d) return-functions\n", skipped, len(rawreturns))

	fnrts := make(map[*pythonimports.Node][]*pythonimports.Node)
	for fn, rts := range fnrtsmap {
		for rt := range rts {
			fnrts[fn] = append(fnrts[fn], rt)
		}
	}
	return fnrts
}

// Updates type induction tables to favor results returned for dynamic analysis
// TODO(juan): this should be done as the final step of the typeinduction pipeline
func main() {
	args := struct {
		Tables       string `arg:"help:tables to update"`
		FunctionsOut string `arg:"positional,help:output path for updated function tables"`
		TypesOut     string `arg:"positional,help:output path for updated type tables"`
		Graph        string
		ReturnTypes  string `arg:"help:path to functions and return types from dynamic analysis"`
		Verbose      bool
	}{
		Tables:      "s3://kite-data/type-inference-models/2015-10-09_16-40-58-PM/", // original type induction tables
		ReturnTypes: "s3://kite-emr/datasets/curated-snippets/2016-09-16_14-47-16-PM/return-types.json.gz",
	}
	arg.MustParse(&args)

	start := time.Now()

	// load import graph
	if args.Graph == "" {
		args.Graph = pythonimports.DefaultImportGraph
	}

	graph, err := pythonimports.NewGraph(args.Graph)
	if err != nil {
		log.Fatal(err)
	}
	if err := pythonskeletons.UpdateGraph(graph); err != nil {
		log.Fatal(err)
	}

	// load function tables for functions that have an import node
	ftables := loadFunctionsInGraph(args.Tables, args.Verbose, graph)

	// load function return types for functions and types that have an import node
	fnrts := loadReturnTypes(args.ReturnTypes, args.Verbose, graph)

	var added, updated int
	changed := make(map[string][]string)
	for fn, rts := range fnrts {
		ftable := ftables[fn]
		if ftable == nil {
			// no function table for return type, add a new one
			added++
			ftable = &typeinduction.Function{
				Name: graph.AnyPaths[fn].String(),
			}
			ftables[fn] = ftable
		} else {
			// clear old return types
			updated++
			ftable.ReturnType = ftable.ReturnType[:0]
		}

		switch len(rts) {
		case 0:
			log.Fatalf("no return types for %s! This should not happen!\n", ftable.Name)
		case 1:
			// NOTE: zero value for log probability corresponds to a probability of 1.0
			ftable.ReturnType = append(ftable.ReturnType, typeinduction.Element{
				Name: graph.AnyPaths[rts[0]].String(),
			})
		default:
			// uniform distribution over possible return types
			lp := -math.Log(float64(len(rts)))
			assertFinite(lp, "Nonfinite log probability (%f) for function %s\n", lp, ftable.Name)
			for _, rt := range rts {
				ftable.ReturnType = append(ftable.ReturnType, typeinduction.Element{
					Name:           graph.AnyPaths[rt].String(),
					LogProbability: lp,
				})
			}
		}

		if args.Verbose {
			for _, rt := range ftable.ReturnType {
				changed[ftable.Name] = append(changed[ftable.Name],
					fmt.Sprintf("    %s: %.3f", rt.Name, math.Exp(rt.LogProbability)))
			}
		}
	}

	if args.Verbose {
		fmt.Printf("\n=== %d functions with updated (or added) return types ===\n", len(changed))
		for f, ts := range changed {
			fmt.Printf("  %s\n", f)
			for _, t := range ts {
				fmt.Println(t)
			}
		}
	}

	fmt.Printf("Added %d, updated %d, out of %d function tables\n", added, updated, len(ftables))

	// write out updated functions,
	// NOTE: this will remove functions that were not in the import graph
	fenc, err := serialization.NewEncoder(args.FunctionsOut)
	if err != nil {
		log.Fatal(err)
	}
	defer fenc.Close()

	for _, fn := range ftables {
		if err := fenc.Encode(fn); err != nil {
			log.Fatal(err)
		}
	}

	// write types out unchanged
	in, err := fileutil.NewCachedReader(typeinduction.OptionsFromPath(args.Tables).Types)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	buf, err := ioutil.ReadAll(in)
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile(args.TypesOut, buf, 0777); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Done! Took %v\n", time.Since(start))
}
