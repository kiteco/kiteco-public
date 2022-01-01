package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/internal/search"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/jsonutil"
	"github.com/kiteco/kiteco/kite-golib/languagemodel"
)

const (
	dataS3Path        = "s3://kite-data/stackoverflow/ranking/2015-10-27_18-17-42-PM/"
	awsSearchEndpoint = "https://search-stackoverflow-search-dev-0-aookytr4hofuhykxmab7nnygzq.us-west-2.cloudsearch.amazonaws.com"
	st                = stackoverflow.Conjunction
)

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func median(vals []float64) float64 {
	sort.Sort(sort.Float64Slice(vals))
	l := len(vals) / 2
	if len(vals)%2 == 0 {
		return (vals[l] + vals[l-1]) / 2.
	}
	return vals[l]
}

func stdDeviation(vals []float64) float64 {
	var (
		vMean    float64
		vSqdMean float64
	)
	for _, v := range vals {
		vSqdMean += v * v
		vMean += v
	}
	vSqdMean *= 1. / float64(len(vals))
	vMean *= 1. / float64(len(vals))
	return math.Sqrt(vSqdMean - vMean*vMean)
}

func benchmarkPageFinderPerformance(pf search.PageFinder, logs []*search.Log) {
	var durations []float64
	for _, l := range logs {
		for _, d := range l.Results {
			start := time.Now()
			_, err := pf.Find(d.ID)
			if err != nil {
				continue
			}
			durations = append(durations, time.Since(start).Seconds()*1000.)
		}
	}
	fmt.Printf("PageFinder Duration (per page request): mean: %f (ms), median: %f (ms), stdDev %f (ms)\n", mean(durations), median(durations), stdDeviation(durations))
}

func benchmarkIndexPerformance(index search.Index, logs []*search.Log, numResults int) {
	var durations []float64
	for _, l := range logs {
		start := time.Now()
		_, err := index.Search(l.Query, st, numResults)
		if err != nil {
			log.Fatal(err)
		}
		durations = append(durations, time.Since(start).Seconds())
	}
	fmt.Printf("Index Durations (per query): numResults: %d, mean %f (s), median %f (s), stdDev %f (s) \n", numResults, mean(durations), median(durations), stdDeviation(durations))
}

func recallAtLevel(level int, results []int64, truth []int64) float64 {
	truthSet := make(map[int64]struct{})
	for i, id := range truth {
		if i >= level && level > 0 {
			break
		}
		truthSet[id] = struct{}{}
	}
	var found int
	for i, id := range results {
		if i >= level && level > 0 {
			break
		}
		if _, exists := truthSet[id]; exists {
			found++
		}
	}
	if level == 0 {
		return float64(found) / float64(len(truth))
	}
	return float64(found) / float64(level)
}

func getIDs(results []search.Document) []int64 {
	ids := make([]int64, len(results))
	for i, d := range results {
		ids[i] = d.ID
	}
	return ids
}

func recallString(numResults int, levels []int, recalls [][]float64) string {
	str := "NumResults " + strconv.Itoa(numResults) + ". Mean Recall:"
	for i, level := range levels {
		levelStr := "@"
		if level == 0 {
			levelStr += "any"
		} else {
			levelStr += strconv.Itoa(level)
		}
		str += " " + levelStr + " " + strconv.FormatFloat(mean(recalls[i]), 'f', 3, 64) + ","
	}
	str = str[:len(str)-1]
	return str
}

func benchmarkIndexRelevance(index search.Index, logs []*search.Log, numResults int) {
	var (
		levels  = []int{2, 3, 5, 0}
		recalls = make([][]float64, len(levels))
	)
	for _, l := range logs {
		if len(l.Results) == 0 {
			continue
		}
		results, err := index.Search(l.Query, st, numResults)
		if err != nil {
			log.Fatal(err)
		}
		for i, level := range levels {
			recall := recallAtLevel(level, results, getIDs(l.Results))
			recalls[i] = append(recalls[i], recall)
		}
	}
	fmt.Println(recallString(numResults, levels, recalls))
}

func benchmarkRankerPerformance(ranker *search.Ranker, logs []*search.Log) {
	var durations []float64
	for _, l := range logs {
		if len(l.Results) == 0 {
			continue
		}
		var pages []*stackoverflow.StackOverflowPage
		for _, d := range l.Results {
			pages = append(pages, d.Page)
		}
		start := time.Now()
		ranker.Rank(l.Query, pages)
		durations = append(durations, time.Since(start).Seconds()/float64(len(pages)))
	}
	fmt.Printf("Ranker Durations (per query): mean %f (s), median %f (s), stdDev %f (s) \n", mean(durations), median(durations), stdDeviation(durations))
}

func benchmarkSearcherPerformance(s searcher, logs []*search.Log, numResults int) {
	var durations []float64
	for _, l := range logs {
		start := time.Now()
		_, err := s.Search(l.Query, st, numResults)
		if err != nil {
			log.Fatal(err)
		}
		durations = append(durations, time.Since(start).Seconds())
	}
	fmt.Printf("Searcher Durations (per query): numResults: %d, mean %f (s), median %f (s), stdDev %f (s) \n", numResults, mean(durations), median(durations), stdDeviation(durations))
}

func benchmarkSearcherRelevance(s searcher, logs []*search.Log, numResults int) {
	var (
		levels  = []int{2, 3, 5, 0}
		recalls = make([][]float64, len(levels))
	)
	for _, l := range logs {
		if len(l.Results) == 0 {
			continue
		}
		results, err := s.Search(l.Query, st, numResults)
		if err != nil {
			log.Fatal(err)
		}
		var resultsIDs []int64
		for _, r := range results {
			resultsIDs = append(resultsIDs, r.ID)
		}
		for i, level := range levels {
			recall := recallAtLevel(level, resultsIDs, getIDs(l.Results))
			recalls[i] = append(recalls[i], recall)
		}
	}
	fmt.Println(recallString(numResults, levels, recalls))
}

const (
	benchmark int = iota
	benchmarkIndex
	benchmarkPageFinder
	benchmarkRanker
	benchmarkSearcher
)

func main() {
	var (
		pageFinderPath string
		logsPath       string
		dataPath       string
		mode           int
		kiteMode       int
	)
	flag.StringVar(&pageFinderPath, "pages", "", "path to so PageFinder")
	flag.StringVar(&logsPath, "logs", "", "path to []search.Log (REQUIRED)")
	flag.StringVar(&dataPath, "data", dataS3Path, "data directory path containg ranker + featurers + tag classes (default "+dataS3Path+" )")
	flag.IntVar(&mode, "mode", 0, "Benchmark Mode: 1 -> benchmark Index, 2 -> benchmark PageFinder, 3 -> benchmark Ranker, 4 -> benchmark searcher, 0 -> benchmark all individually")
	flag.IntVar(&kiteMode, "kite", -1, " -1 -> no kite, 0 -> use all kite, 1 -> use kite (cloudsearch) index only, 2 -> use kite PageFinder only")
	flag.Parse()
	if logsPath == "" {
		flag.Usage()
		log.Fatal("logs REQUIRED")
	}

	var index search.Index
	switch mode {
	case benchmark, benchmarkIndex, benchmarkSearcher:
		start := time.Now()
		csi, err := search.NewCloudSearchIndex(awsSearchEndpoint)
		if err != nil {
			log.Fatal(err)
		}
		index = csi
		fmt.Println("Loading Index took:", time.Since(start))
	}

	var pf search.PageFinder
	switch mode {
	case benchmark, benchmarkPageFinder, benchmarkSearcher:
		if pageFinderPath == "" {
			log.Fatal("pages REQUIRED for mode = 0 | 2 | 4")
		}
		start := time.Now()
		if kiteMode == 0 || kiteMode == 2 {
			pfm := search.NewPageFinderMemory(2e6)
			f, err := os.Open(pageFinderPath)
			if err != nil {
				log.Fatal(err)
			}
			err = pfm.LoadFromPagesDump(f)
			if err != nil {
				log.Fatal(err)
			}
			pf = pfm
		} else {
			var err error
			pf, err = search.NewPageFinderLevelDB(pageFinderPath)
			if err != nil {
				log.Fatal(err)
			}
		}
		fmt.Println("Loading PageFinder took:", time.Since(start))
	}

	var ranker *search.Ranker
	switch mode {
	case benchmark, benchmarkRanker, benchmarkSearcher:
		if dataPath == "" {
			log.Fatal("data REQUIRED for mode = 0 | 3 | 4")
		}

		start := time.Now()

		fModel, err := fileutil.NewCachedReader(fileutil.Join(dataPath, "model.json"))
		if err != nil {
			log.Fatal(err)
		}
		defer fModel.Close()

		fDoc, err := fileutil.NewCachedReader(fileutil.Join(dataPath, "docCounts"))
		if err != nil {
			log.Fatal(err)
		}
		defer fDoc.Close()

		ranker, err = search.NewRanker(fModel, fDoc)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Loading Ranker took:", time.Since(start))
	}

	var s searcher
	switch mode {
	case benchmark, benchmarkSearcher:
		start := time.Now()
		f, err := fileutil.NewCachedReader(fileutil.Join(dataPath, "tagData"))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		decoder := gob.NewDecoder(f)
		var tcd search.TagClassData
		err = decoder.Decode(&tcd)
		if err != nil {
			log.Fatal(err)
		}
		resultFilter := search.NewResultFilter(tcd)

		f, err = fileutil.NewCachedReader(fileutil.Join(dataPath, "ldscorer"))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		decoder = gob.NewDecoder(f)
		var scorer languagemodel.Scorer
		err = decoder.Decode(&scorer)
		if err != nil {
			log.Fatal(err)
		}
		langDetector := search.NewLanguageDetector(&scorer, tcd)

		s = searcher{
			index:        index,
			pageFinder:   pf,
			ranker:       ranker,
			langDetector: langDetector,
			resultFilter: resultFilter,
			missed:       make(map[int64]struct{}),
		}
		fmt.Println("Loading Searcher took:", time.Since(start))
	}

	var logs []*search.Log
	err := jsonutil.DecodeAllFrom(logsPath, func(l *search.Log) {
		logs = append(logs, l)
	})
	if err != nil {
		log.Fatal(err)
	}

	numResults := []int{10, 20, 30, 50}

	switch mode {
	case benchmark, benchmarkIndex:
		fmt.Println("Benchmark Index Performance")
		for _, nr := range numResults {
			benchmarkIndexPerformance(index, logs, nr)
		}
		fmt.Println("Benchmark Index Relevance")
		for _, nr := range numResults {
			benchmarkIndexRelevance(index, logs, nr)
		}
	}

	switch mode {
	case benchmark, benchmarkPageFinder:
		fmt.Println("Benchmark PageFinder Performance")
		benchmarkPageFinderPerformance(pf, logs)
	}

	switch mode {
	case benchmark, benchmarkRanker:
		fmt.Println("Benchmark Ranker Performance")
		benchmarkRankerPerformance(ranker, logs)
	}

	switch mode {
	case benchmark, benchmarkSearcher:
		fmt.Println("Benchmark Searcher Performance")
		for _, nr := range numResults {
			benchmarkSearcherPerformance(s, logs, nr)
		}
		fmt.Println("Benchmark Searcher Relevance")
		for _, nr := range numResults {
			benchmarkSearcherRelevance(s, logs, nr)
		}
		fmt.Printf("num not found by page finder %d \n", len(s.missed))
	}
}
