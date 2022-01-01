package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

func nodeStr(n *pythonimports.Node) string {
	if n == nil {
		return "<nil>"
	}
	if n.CanonicalName.Empty() {
		return "<no name>"
	}
	return n.CanonicalName.String()
}

func compareStrings(before, after string) string {
	if before == after {
		return ""
	}
	return before + " -> " + after
}

func compareNodes(before, after *pythonimports.Node) string {
	if before == nil && after == nil {
		return ""
	}
	if before != nil && after != nil && before.CanonicalName.Hash == after.CanonicalName.Hash {
		return ""
	}
	return compareStrings(nodeStr(before), nodeStr(after))
}

func compareMembers(before, after *pythonimports.Node) string {
	var added, removed []string
	for attr := range before.Members {
		if _, found := after.Members[attr]; !found {
			removed = append(removed, attr)
		}
	}
	for attr := range after.Members {
		if _, found := before.Members[attr]; !found {
			added = append(added, attr)
		}
	}
	if len(added) == 0 && len(removed) == 0 {
		return ""
	}
	return fmt.Sprintf("added %d (%s) removed %d (%s)",
		len(added), strings.Join(added, ","), len(removed), strings.Join(removed, ","))
}

type pkgCount struct {
	Package string
	Count   int
}

type byCount []pkgCount

func (xs byCount) Len() int           { return len(xs) }
func (xs byCount) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }
func (xs byCount) Less(i, j int) bool { return xs[i].Count < xs[j].Count }

func sortedKeys(m map[string]int) []string {
	var items []pkgCount
	for pkg, count := range m {
		items = append(items, pkgCount{pkg, count})
	}
	sort.Sort(byCount(items))
	var keys []string
	for _, item := range items {
		keys = append(keys, item.Package)
	}
	return keys
}

func main() {
	var args struct {
		MasterGraph    string `arg:"positional,required"`
		CandidateGraph string `arg:"positional,required"`
		Missing        string
		Verbose        bool
		Package        string `arg:"help:limit output to this package only"`
	}
	arg.MustParse(&args)

	if args.MasterGraph == "current" {
		args.MasterGraph = pythonimports.DefaultImportGraph
	}

	beforeGraph, err := pythonimports.NewGraph(args.MasterGraph)
	if err != nil {
		log.Fatal(err)
	}

	afterGraph, err := pythonimports.NewGraph(args.CandidateGraph)
	if err != nil {
		log.Fatal(err)
	}

	// construct a lookup map into the after map
	afterByName := make(map[pythonimports.Hash]*pythonimports.Node)
	for i := range afterGraph.Nodes {
		n := &afterGraph.Nodes[i]
		if !n.CanonicalName.Empty() {
			afterByName[n.CanonicalName.Hash] = &afterGraph.Nodes[i]
		}
	}

	// construct a lookup map into the before map
	beforeByName := make(map[pythonimports.Hash]*pythonimports.Node)
	for i := range beforeGraph.Nodes {
		n := &beforeGraph.Nodes[i]
		if !n.CanonicalName.Empty() {
			beforeByName[n.CanonicalName.Hash] = &beforeGraph.Nodes[i]
		}
	}

	// initialize counters
	countByPkgBefore := make(map[string]int)
	changesByPkg := make(map[string]int)
	reportDiff := func(op string, node *pythonimports.Node, msg interface{}) {
		changesByPkg[node.CanonicalName.Head()]++
		if args.Verbose {
			fmt.Printf("%20s %60v %s\n", op, node.CanonicalName, msg)
		}
	}

	// check each node
	var removed, kindChanged, typeChanged, membersChanged int
	for i := range beforeGraph.Nodes {
		before := &beforeGraph.Nodes[i]
		if before.CanonicalName.Empty() {
			continue
		}

		pkg := before.CanonicalName.Head()
		if args.Package != "" && pkg != args.Package {
			continue
		}
		countByPkgBefore[pkg]++

		after, found := afterByName[before.CanonicalName.Hash]
		if !found {
			removed++
			reportDiff("removed", before, "")
			continue
		}

		if d := compareStrings(before.Classification.String(), after.Classification.String()); d != "" {
			kindChanged++
			reportDiff("kind changed", before, d)
		}
		if d := compareNodes(before.Type, after.Type); d != "" {
			typeChanged++
			reportDiff("type changed", before, d)
		}
		if d := compareMembers(before, after); d != "" {
			membersChanged++
			reportDiff("members changed", before, d)
		}
	}

	var numNew int
	for i := range afterGraph.Nodes {
		after := &afterGraph.Nodes[i]
		if after.CanonicalName.Empty() {
			continue
		}

		pkg := after.CanonicalName.Head()
		if args.Package != "" && pkg != args.Package {
			continue
		}

		_, found := beforeByName[after.CanonicalName.Hash]
		if !found {
			numNew++
			reportDiff("new node", after, "")
			log.Printf("New: %s\n", after.CanonicalName.String())
		}
	}

	// make a map of which packages are present in the candidate graph
	countByPkgAfter := make(map[string]int)
	for i := range afterGraph.Nodes {
		countByPkgAfter[afterGraph.Nodes[i].CanonicalName.Head()]++
	}

	// list changes summarized by package
	fmt.Println()
	for _, pkg := range sortedKeys(changesByPkg) {
		fmt.Printf("%-25s %d changes (%d nodes before, %d nodes after)\n",
			pkg, changesByPkg[pkg], countByPkgBefore[pkg], countByPkgAfter[pkg])
	}

	// list packages that were removed
	fmt.Println()
	for pkg := range countByPkgBefore {
		if _, found := countByPkgAfter[pkg]; !found {
			fmt.Printf("%s package removed\n", pkg)
		}
	}

	fmt.Println()
	fmt.Printf("%d nodes before, %d nodes after\n", len(beforeGraph.Nodes), len(afterGraph.Nodes))
	fmt.Printf("%d nodes removed\n", removed)
	fmt.Printf("%d nodes new\n", numNew)
	fmt.Printf("%d nodes with updated kind\n", kindChanged)
	fmt.Printf("%d nodes with updated type\n", typeChanged)
	fmt.Printf("%d nodes with updated members\n", membersChanged)
}
