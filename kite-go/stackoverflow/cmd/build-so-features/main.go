package main

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/internal/search"
	"github.com/kiteco/kiteco/kite-golib/jsonutil"
	"github.com/kiteco/kiteco/kite-golib/tfidf"
)

const (
	awsSearchEndpoint = "https://search-stackoverflow-search-dev-0-aookytr4hofuhykxmab7nnygzq.us-west-2.cloudsearch.amazonaws.com"
)

func termNegativeExamples(numExamples int, index search.Index, pageFinder search.PageFinder, query *search.Log) []search.Document {
	var (
		maxNumResults = 50
		st            = stackoverflow.Disjunction
	)
	ids, err := index.Search(query.Query, st, maxNumResults)
	if err != nil {
		log.Fatal(err)
	}
	goodIds := make(map[int64]struct{})
	for _, d := range query.Results {
		goodIds[d.ID] = struct{}{}
	}
	var docs []search.Document
	for _, id := range ids {
		if _, exists := goodIds[id]; !exists {
			page, err := pageFinder.Find(id)
			if err != nil {
				continue
			}
			docs = append(docs, search.Document{
				ID:    page.GetQuestion().GetPost().GetId(),
				Page:  page,
				Score: 0,
			})
		}
	}
	if len(docs) <= numExamples {
		return docs
	}
	var randDocs []search.Document
	perm := rand.Perm(len(docs))
	for i, idx := range perm {
		if i >= numExamples {
			break
		}
		randDocs = append(randDocs, docs[idx])
	}
	return randDocs
}

func main() {
	var (
		dataDirPath    string
		searchLogPath  string
		pageFinderPath string
		outBase        string
		testPct        float64
		rndSeed        int64
		minScore       int
		nNegExPerQuery int
	)
	flag.StringVar(&dataDirPath, "data", "", "path to directory containing tag class data and doc counts (REQUIRED)")
	flag.StringVar(&searchLogPath, "logs", "", "path containg SO SearchLogs (REQUIRED)")
	flag.StringVar(&pageFinderPath, "pages", "", "path to leveldb PageFinder (REQUIRED)")
	flag.StringVar(&outBase, "out", "", "base pathname to write out features and featurers (REQUIRED)")
	flag.Float64Var(&testPct, "testPct", 0.15, "pct of examples to include in test set, in [0,1] (default 0.15)")
	flag.Int64Var(&rndSeed, "rndSeed", 1, "random seed opt use when splitting training and test set (default 1)")
	flag.IntVar(&minScore, "minScore", 5, "Min relevance score for a document to be accepted as relevant (default 5)")
	flag.IntVar(&nNegExPerQuery, "nneg", 10, "Number of negative examples to include for each query (default 10)")
	flag.Parse()

	if dataDirPath == "" || searchLogPath == "" || outBase == "" || pageFinderPath == "" {
		flag.Usage()
		log.Fatal("data, logs, pages, and out parameters REQUIRED")
	}
	if testPct < 0. || testPct > 1. {
		flag.Usage()
		log.Fatal("testPct must be in range [0,1]")
	}

	// 0) Load data
	start := time.Now()

	index, err := search.NewCloudSearchIndex(awsSearchEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	// need PageFinder to get pages for negative examples
	pageFinder, err := search.NewPageFinderLevelDB(pageFinderPath)
	if err != nil {
		log.Fatal(err)
	}

	var logs []*search.Log
	err = jsonutil.DecodeAllFrom(searchLogPath, func(l *search.Log) {
		logs = append(logs, l)
	})
	if err != nil {
		log.Fatal(err)
	}

	fDocCounts, err := os.Open(path.Join(dataDirPath, "docCounts"))
	if err != nil {
		log.Fatal(err)
	}
	defer fDocCounts.Close()
	decoder := gob.NewDecoder(fDocCounts)
	var docCounts map[string]*tfidf.IDFCounter
	err = decoder.Decode(&docCounts)
	if err != nil {
		log.Fatal(err)
	}

	featurers, err := search.NewFeaturers(docCounts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Loaded all data, took ", time.Since(start))

	// 1) Extract features
	cutoffTest := int(testPct * float64(len(logs)))
	rand.Seed(rndSeed)
	perm := rand.Perm(len(logs))
	data := make([][]ranking.Entry, 2)
	var nZeroFeats, nZeroFeatsNegEx, nNegEx, nPosEx int

	for i, idx := range perm {
		sl := logs[idx]
		for _, doc := range sl.Results {
			if doc.Score < minScore {
				continue
			}
			fts := featurers.Features(sl.Query, doc)

			var countNonZero int
			for _, feat := range fts {
				if math.Abs(feat) > 0. {
					countNonZero++
				}
			}
			if countNonZero == 0 {
				nZeroFeats++
				continue
			}
			nPosEx++

			fd := ranking.Entry{
				QueryHash:  sl.Query,
				QueryText:  sl.Query,
				QueryCode:  "googleSO",
				SnapshotID: doc.Page.GetQuestion().GetPost().GetId(),
				Features:   fts,
				Label:      float64(doc.Score),
			}

			var idxData int
			if i >= cutoffTest {
				idxData = 1
			}
			data[idxData] = append(data[idxData], fd)
		}
		docs := termNegativeExamples(nNegExPerQuery, index, pageFinder, sl)
		for _, doc := range docs {
			fts := featurers.Features(sl.Query, doc)
			var countNonZero int
			for _, feat := range fts {
				if math.Abs(feat) > 0 {
					countNonZero++
				}
			}
			if countNonZero == 0 {
				nZeroFeatsNegEx++
				continue
			}
			nNegEx++

			fd := ranking.Entry{
				QueryHash:  sl.Query,
				QueryText:  sl.Query,
				QueryCode:  "googleSO",
				SnapshotID: doc.Page.GetQuestion().GetPost().GetId(),
				Features:   fts,
				Label:      float64(doc.Score),
			}

			var idxData int
			if i >= cutoffTest {
				idxData = 1
			}
			data[idxData] = append(data[idxData], fd)
		}
	}
	fmt.Printf("Totals: \n")
	fmt.Printf("nPosExamples %d, nNegExamples %d \n", nPosEx, nNegEx)
	fmt.Printf("nPosZeroFeat %d, nNegZeroFeat %d \n", nZeroFeats, nZeroFeatsNegEx)

	// 3) Write train and test sets
	prefixes := []string{"test", "train"}

	for i, prefix := range prefixes {
		payload := map[string]interface{}{
			"FeatureLabels":   featurers.Labels(),
			"Data":            data[i],
			"FeaturerOptions": struct{}{},
		}

		f, err := os.Create(outBase + prefix + "-features")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		encoder := json.NewEncoder(f)
		err = encoder.Encode(payload)
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Finished writing features, took ", time.Since(start))
	fmt.Println("nTrainExamples:", len(data[1]), "nTestExamples:", len(data[0]))
}
