package main

import (
	"log"
	"sort"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/tensorflow/bench"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

const (
	maxIdx = 100
)

func main() {
	args := struct {
		In    string `arg:"required"`
		Out   string `arg:"required"`
		Nodes int    // if non-zero, every feed in graph will have this many nodes
		Edges int    // if non-zero, every feed in graph will have this many edges
	}{}
	arg.MustParse(&args)

	if (args.Nodes != 0) != (args.Edges != 0) {
		log.Fatal("nodes and edges params need to be set together")
	}
	if args.Nodes > args.Edges {
		log.Fatal("edges must be >= nodes such that each node has >=1 incoming edge")
	}

	recs, err := bench.LoadFeedRecords(args.In)
	fail(err)
	log.Printf("loaded %d feeds from %s", len(recs), args.In)

	for _, rec := range recs {
		rewriteIndices(rec)
		if args.Nodes != 0 {
			setNodeCount(rec, args.Nodes)
		}
		if args.Edges != 0 {
			setEdgeCount(rec, args.Edges)
		}

		connectAllNodes(rec)
		//printFeeds(rec)
	}

	fail(bench.SaveFeedRecords(args.Out, recs))
	log.Printf("wrote %d feeds to %s", len(recs), args.Out)
}

func int2D(rows, cols int) [][]int32 {
	s := make([][]int32, rows)
	for i := range s {
		s[i] = make([]int32, cols)
	}
	return s
}

func setNodeCount(rec bench.FeedRecord, nodes int) {
	typesTensor := "graph/inputs/nodes/node_types"
	numTypes := len(rec.Feeds[typesTensor].([][]int32)[0])
	rec.Feeds[typesTensor] = int2D(nodes, numTypes)

	subtokensTensor := "graph/inputs/nodes/node_subtokens"
	numSubtokens := len(rec.Feeds[subtokensTensor].([][]int32)[0])
	rec.Feeds[subtokensTensor] = int2D(nodes, numSubtokens)

	rec.Feeds["graph/inputs/nodes/target_bits"] = make([]float32, nodes)
}

func setEdgeCount(rec bench.FeedRecord, edges int) {
	var edgeFeeds []string
	var totalEdges int
	for name, feed := range rec.Feeds {
		if strings.HasPrefix(name, "graph/inputs/edges/") {
			totalEdges += len(feed.([][2]int32))
			edgeFeeds = append(edgeFeeds, name)
		}
	}

	// we evenly distribute the edges over the edge types present in the feed
	scaling := float64(edges) / float64(totalEdges)

	sort.Strings(edgeFeeds)
	var newTotal int

	for i, name := range edgeFeeds {
		feed := rec.Feeds[name]

		count := int(scaling * float64(len(feed.([][2]int32))))
		// Note that if we scale every feed to this count, we'll undercount since we always round down
		// in the float -> int conversion. If we're at the last edge type, we add some extra if necessary
		// to make up for this.
		if i == len(edgeFeeds)-1 {
			count = edges - newTotal
		}
		newTotal += count
		rec.Feeds[name] = make([][2]int32, count)
	}

	if newTotal != edges {
		log.Fatalf("new total edges (%d) is not the desired number (%d)", newTotal, edges)
	}
}

// rewriteIndices rewrites the value of every int slice tensor to a number in [0,maxIdx) - this makes it possible
// to decrease embedding sizes without tf.gather failing on indices that are too large.
func rewriteIndices(rec bench.FeedRecord) {
	for name, feed := range rec.Feeds {
		switch feed := feed.(type) {
		case int32:
			rec.Feeds[name] = feed % maxIdx
		case []int32:
			for i, v := range feed {
				feed[i] = v % maxIdx
			}
		case [][]int32:
			for i, row := range feed {
				for j, v := range row {
					feed[i][j] = v % maxIdx
				}
			}

		case [][2]int32:
			for i, row := range feed {
				for j, v := range row {
					feed[i][j] = v % maxIdx
				}
			}
		}
	}
}

// Ensure every node has at least one incoming edge
// Otherwise, the model will complain about mismatching dimensions
func connectAllNodes(rec bench.FeedRecord) {
	totalNodes := len(rec.Feeds["graph/inputs/nodes/node_types"].([][]int32))

	var edgeFeeds []string
	for name := range rec.Feeds {
		if strings.HasPrefix(name, "graph/inputs/edges/") {
			edgeFeeds = append(edgeFeeds, name)
		}
	}
	sort.Strings(edgeFeeds)

	var node int

SetEdges:
	for _, name := range edgeFeeds {
		feed := rec.Feeds[name].([][2]int32)

		for i := range feed {
			feed[i][1] = int32(node)
			node++
			if node >= totalNodes {
				break SetEdges
			}
		}
	}

	if node < totalNodes {
		log.Fatalf("only connected %d/%d nodes", node, totalNodes)
	}
}

func printFeeds(rec bench.FeedRecord) {
	min := func(a, b int) int {
		if a > b {
			return b
		}
		return a
	}

	m := 5

	for name, feed := range rec.Feeds {
		log.Printf("feed: %s (%T)", name, feed)
		switch feed := feed.(type) {
		case int32:
			log.Printf("%d", feed)
		case []int32:
			log.Printf("%v", feed[:min(len(feed), m)])
		case [][]int32:
			log.Printf("%v", feed[0][:min(len(feed[0]), m)])
		case [][2]int32:
			log.Printf("%v", feed[0][:min(len(feed[0]), m)])
		default:
			log.Printf("skipping %s (%T)", name, feed)
		}
	}
}
