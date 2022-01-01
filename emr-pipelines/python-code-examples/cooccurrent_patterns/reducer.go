package main

import (
	"container/heap"
	"encoding/json"
	"io"
	"log"
	"os"
	"sort"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/clustering"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

const (
	numClusters      = 5
	minClusterSize   = 100
	featVecSize      = 103
	maxSnippets      = 10
	patternThreshold = 0.5
)

type snippetCooccur struct {
	Functions []string
	Hash      string
	Score     int
}

func (s *snippetCooccur) ID() uint64 {
	return spooky.Hash64([]byte(s.Hash))
}

func (s *snippetCooccur) Values() []float64 {
	features := make([]float64, featVecSize)
	for _, f := range s.Functions {
		features[spooky.Hash64([]byte(f))%featVecSize] = 1.0
	}
	return features
}

type snippetHeap []*snippetCooccur

func (s snippetHeap) Len() int { return len(s) }
func (s snippetHeap) Less(i, j int) bool {
	// We use greater-than here to make this a max-heap
	return s[i].Score > s[j].Score
}
func (s snippetHeap) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s *snippetHeap) Push(x interface{}) {
	*s = append(*s, x.(*snippetCooccur))
}

func (s *snippetHeap) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

func main() {
	r := awsutil.NewEMRReader(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	trainer, err := clustering.NewKMeans(numClusters)
	if err != nil {
		log.Fatal(err)
	}

	var lastKey string
	nodeIndex := make(map[uint64]*snippetCooccur)
	for {
		key, value, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if key != lastKey && len(nodeIndex) > 0 {
			// First, cluster the snippetCoocur objects
			var nodes []clustering.Node
			for _, n := range nodeIndex {
				nodes = append(nodes, n)
			}

			clusters := trainer.Train(nodes)
			total := len(nodeIndex)

			// For each cluster, detect the pattern and find the best
			// examples for that pattern.
			for idx, c := range clusters {
				if len(c.Members) < minClusterSize {
					continue
				}

				sheap := snippetHeap{}
				funcCounts := make(map[string]int)

				// For each member, count the functions, and add the
				// snippetCooccur to the heap keeping track of the
				// best scoring snippets.
				var totalSnippets int
				for _, m := range c.Members {
					snip := nodeIndex[m.ID()]
					for _, f := range snip.Functions {
						funcCounts[f]++
					}

					heap.Push(&sheap, snip)
					if sheap.Len() > 5*maxSnippets {
						heap.Pop(&sheap)
					}

					totalSnippets++
				}

				pattern := findPattern(funcCounts, totalSnippets, patternThreshold)

				// Pull out the top snippets
				var topSnippets []*snippetCooccur
				for sheap.Len() > 0 {
					snip := heap.Pop(&sheap).(*snippetCooccur)
					if containsPattern(snip, pattern) && len(topSnippets) < maxSnippets {
						topSnippets = append(topSnippets, snip)
					}
				}

				// Sort them correctly
				sort.Sort(sort.Reverse(snippetHeap(topSnippets)))

				// Collect the hashes for the snippets
				var topHashes []string
				for _, snip := range topSnippets {
					topHashes = append(topHashes, snip.Hash)
				}

				ret := pythoncode.CooccurrencePattern{
					Method:       lastKey,
					Pattern:      pattern,
					Hashes:       topHashes,
					Frequency:    float64(len(c.Members)) / float64(total),
					MethodCount:  total,
					ClusterID:    idx,
					ClusterCount: len(c.Members),
				}

				if len(ret.Pattern) <= 1 {
					continue
				}

				buf, err := json.Marshal(ret)
				if err != nil {
					log.Fatal(err)
				}

				err = w.Emit("cooccurrence", buf)
				if err != nil {
					log.Fatal(err)
				}
			}

			nodeIndex = make(map[uint64]*snippetCooccur)
		}

		var co snippetCooccur
		err = json.Unmarshal(value, &co)
		if err != nil {
			log.Fatal(err)
		}

		lastKey = key

		// Keep a map of snippetCoocur id to the actual object
		nodeIndex[co.ID()] = &co
	}
}

// findPattern takes counts of occurances of methods within a particular cluster and returns an
// array of strings that contains the methods representing the cooccurrence pattern
func findPattern(counts map[string]int, total int, threshold float64) []string {
	var pattern []string
	for f, c := range counts {
		if float64(c)/float64(total) > threshold {
			pattern = append(pattern, f)
		}
	}
	return pattern
}

// containsPattern returns whether a particular pattern is contained in a snippetCoooccur.
func containsPattern(snip *snippetCooccur, pattern []string) bool {
	matches := make(map[string]struct{})
	for _, f := range snip.Functions {
		matches[f] = struct{}{}
	}
	for _, p := range pattern {
		if _, exists := matches[p]; !exists {
			return false
		}
	}
	return true
}
