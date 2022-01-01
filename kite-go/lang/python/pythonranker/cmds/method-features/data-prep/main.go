// This binary takes in manually labelled method prediction training data
// and computes (by using labelled ranking data) to find out the score
// each method receives.
// The input data should follow this format:
// {"Query":"Plot multiple plots on one plot",
//  "Method":"matplotlib.pyplot.subplot, matplotlib.pyplot.figure, matplotlib.pyplot.plot, matplotlib.pyplot.bar"}

package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-golib/text"
	_ "github.com/mattn/go-sqlite3"
)

// rawDatum represents an entry of the human-labelled test data
// for method prediction
type rawDatum struct {
	Query  string
	Method string
}

// trainingDatum contains the score a method gets for the query
type trainingDatum struct {
	Query   string
	Method  string
	Package string
	Score   float64
}

const (
	defaultAttributePath = "/var/kite/ranking_data/ranked-snippets-attributes.emr"
	defaultQueryOutput   = "queries.list"
)

func main() {
	var (
		input         string
		output        string
		attributePath string
		queryOutput   string
	)
	flag.StringVar(&input, "in", "", "input data (method-data.json) see the beginning of manual.go to find out the proper format")
	flag.StringVar(&output, "out", "", "output data (.json)")
	flag.StringVar(&attributePath, "attribute", defaultAttributePath, "path to the attribute file (ranked-snippets-attributes.emr)")
	flag.StringVar(&queryOutput, "query", defaultQueryOutput, "output file which contains all the queries used in the data set")
	flag.Parse()

	if input == "" || output == "" {
		flag.Usage()
		log.Fatal("must specify --in, --out")
	}

	queryToLabels := loadRankingDB()
	snippets, attributes := loadAttributes(attributePath)

	in, err := os.Open(input)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	queries, data := scoreMethods(queryToLabels, snippets, attributes, in)
	out, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	encoder := json.NewEncoder(out)
	err = encoder.Encode(data)
	if err != nil {
		log.Fatal(err)
	}

	qout, err := os.Create(queryOutput)
	if err != nil {
		log.Fatal(err)
	}
	defer qout.Close()

	for _, query := range queries {
		qout.WriteString(query + "\n")
	}
}

func scoreMethods(queryToLabels map[string][]ranking.Label,
	snippets map[int64]*pythoncuration.Snippet,
	attributes map[int64][]pythoncuration.Attribute, r io.Reader) ([]string, []*trainingDatum) {

	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		log.Fatal(err)
	}

	var trainingData []*trainingDatum
	queryToData := make(map[string][]*trainingDatum)

	decoder := json.NewDecoder(r)
	for {
		var datum rawDatum
		err := decoder.Decode(&datum)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var methods []string
		packages := make(map[string]struct{})

		for _, m := range strings.Split(datum.Method, ",") {
			methods = append(methods, strings.TrimSpace(m))
			tokens := strings.Split(m, ".")
			if len(tokens) > 0 {
				packages[tokens[0]] = struct{}{}
			}
		}

		scores := make(map[string]float64)
		labels := queryToLabels[datum.Query]

		for _, l := range labels {
			snip, found := snippets[l.SnapshotID]
			if !found {
				continue
			}

			// build a chart that maps from selector to fully qualified names
			attrs, _ := attributes[l.SnapshotID]
			lookup := buildLookUpTable(packages, attrs, methods)

			candidates := findFuncCandidates(snip.Curated.Snippet.Code)
			overlaps := overlap(candidates, methods)

			for _, m := range overlaps {
				names := findCanonicalNames(graph, lookup[m])
				for _, name := range text.Uniquify(names) {
					scores[name] += l.Rank
				}
			}
		}

		for name, score := range scores {
			d := &trainingDatum{
				Query:   datum.Query,
				Method:  name,
				Package: strings.Split(name, ".")[0],
				Score:   score,
			}
			trainingData = append(trainingData, d)
			queryToData[datum.Query] = append(queryToData[datum.Query], d)
		}
	}
	var queries []string

	for query, data := range queryToData {
		sort.Sort(sort.Reverse(byScore(data)))
		scale := 1.0
		if data[0].Score > 4 {
			scale = data[0].Score / 4.0
		}
		for _, d := range data {
			d.Score /= scale
		}
		queries = append(queries, query)
	}
	return queries, trainingData
}
