package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/diskmap"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

type key struct {
	node *pythonimports.Node
	path string
}

type byPath []key

func (xs byPath) Len() int           { return len(xs) }
func (xs byPath) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }
func (xs byPath) Less(i, j int) bool { return xs[i].path < xs[j].path }

type byScore []*stackoverflow.Result

func (xs byScore) Len() int           { return len(xs) }
func (xs byScore) Swap(i, j int)      { xs[i], xs[j] = xs[j], xs[i] }
func (xs byScore) Less(i, j int) bool { return xs[i].Score < xs[j].Score }

var attrRegexp = regexp.MustCompile(`([a-zA-Z_][\.a-zA-Z0-9_]*)`)

var errLimitReached = errors.New("limit reached")

func fail(msg interface{}, parts ...interface{}) {
	fmt.Printf(fmt.Sprintf("%v", msg)+"\n", parts...)
	os.Exit(1)
}

func main() {
	var args struct {
		Input   string `arg:"positional,required"`
		Output  string `arg:"positional,required"`
		Limit   int
		Verbose bool
	}
	arg.MustParse(&args)

	// open output stream to surface errors early
	w, err := os.Create(args.Output)
	if err != nil {
		fail(err)
	}

	// load import graph
	graph, err := pythonimports.NewGraph(pythonimports.SmallImportGraph)
	if err != nil {
		fail(err)
	}

	// initialize index
	index := make(map[*pythonimports.Node]*stackoverflow.ResultSet)

	// iterate over stackoverflow posts
	var count int
	err = serialization.Decode(args.Input, func(post *stackoverflow.XMLPost) error {
		count++
		if args.Limit != 0 && count > args.Limit {
			return errLimitReached
		}
		if count%10000 == 0 {
			log.Println(count)
		}
		if args.Verbose {
			log.Printf(`processing "%s"`, post.Title)
		}

		// parse html
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(post.Body))
		if err != nil {
			return fmt.Errorf("error parsing html: %v", err)
		}

		// initialize the "seen" map for this post
		seen := make(map[*pythonimports.Node]*stackoverflow.Result)

		// find code snippets
		codeBlocks := doc.Find("code")
		if args.Verbose {
			log.Printf("found %d code elements", len(codeBlocks.Nodes))
		}
		codeBlocks.Each(func(i int, s *goquery.Selection) {
			// find attribute expressions
			attrs := attrRegexp.FindAllString(s.Text(), -1)
			if args.Verbose {
				log.Printf("  found %d attribute expressions", len(attrs))
			}

			for _, attr := range attrs {
				path := pythonimports.NewPath(strings.Split(attr, ".")...)
				if args.Verbose {
					log.Println("    ", path)
				}
				if node, _ := graph.Navigate(path); node != nil {
					if args.Verbose {
						log.Println("       resolved to ", node.CanonicalName)
					}

					results, found := index[node]
					if !found {
						results = new(stackoverflow.ResultSet)
						index[node] = results
					}

					// check whether we already have this result
					if _, isseen := seen[node]; !isseen {
						result := stackoverflow.Result{
							Title:  post.Title,
							PostID: post.Id,
							Score:  post.Score,
						}
						results.Results = append(results.Results, &result)
						seen[node] = &result
					}
				}
			}
		})
		return nil
	})
	if err != nil && !strings.Contains(err.Error(), "limit reached") {
		fail(err)
	}

	// sort results by stars
	for _, r := range index {
		sort.Sort(sort.Reverse(byScore(r.Results)))
		if len(r.Results) > 50 {
			r.Results = r.Results[:50]
		}
	}

	// sort the keys
	var keys []key
	for node := range index {
		anypath, ok := graph.AnyPaths[node]
		if !ok {
			log.Println("missing anypath for", node)
			continue
		}
		keys = append(keys, key{
			node: node,
			path: anypath.String(),
		})
	}
	// TODO: when on go 1.8, use sort.Slice here
	sort.Sort(byPath(keys))

	// write to diskmap
	b := diskmap.NewStreamBuilder(w)
	for _, key := range keys {
		err := diskmap.JSON.Add(b, key.path, index[key.node])
		if err != nil {
			fail("error writing to diskmap:", err)
		}
	}
	b.Close()
}
