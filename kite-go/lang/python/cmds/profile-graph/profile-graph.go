package main

import (
	"log"
	"reflect"
	"runtime"
	"time"

	arg "github.com/alexflint/go-arg"
	humanize "github.com/dustin/go-humanize"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

type stat struct {
	t     reflect.Type
	count uint64
	bytes uint64
}

type byCount []stat

func (xs byCount) Len() int           { return len(xs) }
func (xs byCount) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }
func (xs byCount) Less(i, j int) bool { return xs[i].count < xs[j].count }

func main() {
	var args struct {
		Path  string
		Limit int
	}
	args.Path = pythonimports.DefaultImportGraph
	arg.MustParse(&args)

	// Look up memory stats
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Convert to full graph
	graph, err := pythonimports.NewGraph(args.Path)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("graph has %d nodes", len(graph.Nodes))

	// Given the GC time to clean up the mess...
	log.Println("giving GC time to clean up the mess...")
	runtime.GC()
	time.Sleep(60 * time.Second)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Print results
	log.Println("allocs before:", humanize.Bytes(memBefore.Alloc))
	log.Println("allocs after:", humanize.Bytes(memAfter.Alloc))

	// // Compute size of various objects
	// log.Println("computing sizes...")
	// p := objectgraph.NewProfile(graph)

	// // Generate report
	// var stats []stat
	// for t, sz := range p.SizeByType {
	// 	stats = append(stats, stat{
	// 		t:     t,
	// 		bytes: sz,
	// 		count: p.CountByType[t],
	// 	})
	// }
	// sort.Sort(byCount(stats))
	// for _, st := range stats {
	// 	fmt.Printf("%8d %8s  %s\n", st.count, humanize.Bytes(st.bytes), st.t.String())
	// }
	// fmt.Printf("         %8s  TOTAL\n", humanize.Bytes(p.TotalBytes))
}
