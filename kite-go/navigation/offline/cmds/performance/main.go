package main

import (
	"encoding/csv"
	"errors"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/metrics"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type repo struct {
	name        string
	root        localpath.Absolute
	currentPath string
}

func main() {
	args := struct {
		ResultsPath string
	}{}
	arg.MustParse(&args)

	var results []stats
	for _, r := range repos {
		log.Printf("%s starting...\n", r.name)
		s, err := run(r)
		if err != nil {
			log.Fatal(err)
		}
		results = append(results, s)
		log.Printf("%s finished\n", r.name)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].label < results[j].label
	})
	results = append(results, mean(results))

	err := write(args.ResultsPath, results)
	if err != nil {
		log.Fatal(err)
	}
}

type stats struct {
	label                      string
	numFiles                   int
	allocatedMegabytes         int
	heapObjects                int
	buildSeconds               int
	requestFiveMilliseconds    int
	requestTwentyMilliseconds  int
	requestHundredMilliseconds int
}

func mean(batch []stats) stats {
	var total stats
	for _, s := range batch {
		total.numFiles += s.numFiles
		total.allocatedMegabytes += s.allocatedMegabytes
		total.heapObjects += s.heapObjects
		total.buildSeconds += s.buildSeconds
		total.requestFiveMilliseconds += s.requestFiveMilliseconds
		total.requestTwentyMilliseconds += s.requestTwentyMilliseconds
		total.requestHundredMilliseconds += s.requestHundredMilliseconds
	}
	return stats{
		label:                      "mean",
		numFiles:                   round(total.numFiles/len(batch), 100),
		allocatedMegabytes:         total.allocatedMegabytes / len(batch),
		heapObjects:                total.heapObjects / len(batch),
		buildSeconds:               total.buildSeconds / len(batch),
		requestFiveMilliseconds:    total.requestFiveMilliseconds / len(batch),
		requestTwentyMilliseconds:  total.requestTwentyMilliseconds / len(batch),
		requestHundredMilliseconds: total.requestHundredMilliseconds / len(batch),
	}
}

func run(r repo) (stats, error) {
	runtime.GC()
	buildStart := time.Now()
	recommender, err := buildRecommender(r)
	if err != nil {
		return stats{}, err
	}
	buildSeconds := int(time.Since(buildStart).Seconds())

	numFiles := measureNumFiles()

	time.Sleep(time.Second)
	runtime.GC()
	memstats := measureMemStats()
	allocatedMegabytes := roundDivide(int(memstats.Alloc), 1e6)
	heapObjects := roundDivide(int(memstats.HeapObjects), 1e3)

	requestMilliseconds := make(map[int]int)
	for _, batchSize := range []int{5, 20, 100} {
		requestStart := time.Now()
		_, err = makeRequest(r, batchSize, recommender)
		if err != nil {
			return stats{}, err
		}
		requestMilliseconds[batchSize] = int(time.Since(requestStart).Milliseconds())
	}

	s := stats{
		label:                      r.name,
		numFiles:                   numFiles,
		allocatedMegabytes:         allocatedMegabytes,
		heapObjects:                heapObjects,
		buildSeconds:               buildSeconds,
		requestFiveMilliseconds:    requestMilliseconds[5],
		requestTwentyMilliseconds:  requestMilliseconds[20],
		requestHundredMilliseconds: requestMilliseconds[100],
	}
	return s, nil
}

func measureNumFiles() int {
	m := metrics.Read(true)
	return round(int(m["nav_index_num_files"]), 100)
}

func roundDivide(numerator, denominator int) int {
	return int(math.Round(float64(numerator) / float64(denominator)))
}

func round(original, units int) int {
	return roundDivide(original, units) * units
}

func buildRecommender(r repo) (recommend.Recommender, error) {
	ignoreOpts := ignore.Options{
		Root:            r.root,
		IgnoreFilenames: []localpath.Relative{ignore.GitIgnoreFilename},
	}
	ignorer, err := ignore.New(ignoreOpts)
	if err != nil {
		return nil, err
	}

	recOpts := recommend.Options{
		UseCommits:           true,
		ComputedCommitsLimit: git.DefaultComputedCommitsLimit,
		Root:                 r.root,
		MaxFileSize:          1e6,
		MaxFiles:             1e5,
	}
	s, err := git.NewStorage(git.StorageOptions{
		UseDisk: true,
		Path: filepath.Join(
			os.Getenv("GOPATH"),
			"src", "github.com", "kiteco", "kiteco",
			"kite-go", "navigation", "offline", "git-cache.json",
		),
	})
	if err != nil {
		return nil, err
	}
	return recommend.NewRecommender(kitectx.Background(), recOpts, ignorer, s)
}

func makeRequest(r repo, batchSize int, recommender recommend.Recommender) ([]recommend.File, error) {
	request := recommend.Request{
		MaxFileRecs:      -1,
		MaxBlockRecs:     5,
		MaxFileKeywords:  -1,
		MaxBlockKeywords: 3,
		Location: recommend.Location{
			CurrentPath: r.currentPath,
			CurrentLine: 20,
		},
	}
	files, err := recommender.Recommend(kitectx.Background(), request)
	if err != nil {
		return nil, err
	}
	blockRequest := recommend.BlockRequest{
		Request:      request,
		InspectFiles: files[:batchSize],
	}
	return recommender.RecommendBlocks(kitectx.Background(), blockRequest)
}

func measureMemStats() runtime.MemStats {
	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)
	return memstats
}

func write(resultsPath string, results []stats) error {
	if resultsPath == "" {
		return errors.New("path is empty")
	}
	log.Printf("writing to %s\n", resultsPath)

	f, err := os.Create(resultsPath)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	header := []string{
		"label",
		"num files",
		"mem allocated (mb)",
		"heap objects (K)",
		"build (s)",
		"serve 5 (ms)",
		"serve 20 (ms)",
		"serve 100 (ms)",
	}
	err = writer.Write(header)
	if err != nil {
		return err
	}

	for _, r := range results {
		row := []string{
			r.label,
			strconv.Itoa(r.numFiles),
			strconv.Itoa(r.allocatedMegabytes),
			strconv.Itoa(r.heapObjects),
			strconv.Itoa(r.buildSeconds),
			strconv.Itoa(r.requestFiveMilliseconds),
			strconv.Itoa(r.requestTwentyMilliseconds),
			strconv.Itoa(r.requestHundredMilliseconds),
		}
		err = writer.Write(row)
		if err != nil {
			return err
		}
	}
	return nil
}
