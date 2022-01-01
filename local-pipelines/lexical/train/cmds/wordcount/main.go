package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync/atomic"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/bpe"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/words"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/utils"
)

// Increase or decrease this value to use more or less memory
const maxWordCount = 20e6

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	args := struct {
		Lang       string
		Output     string
		SplitsDir  string
		MaxFiles   int
		CacheRoot  string
		MinCount   int
		SampleRate float64
		Seed       int64
	}{
		Output:     "wordcounts.json",
		SplitsDir:  "splits",
		MaxFiles:   100e7,
		CacheRoot:  "/data/kite",
		MinCount:   200,
		SampleRate: .25,
		Seed:       42,
	}
	arg.MustParse(&args)

	err := os.MkdirAll(args.SplitsDir, os.ModePerm)
	fail(err)

	rand.Seed(args.Seed)
	langGroup := lexicalv0.MustLangGroupFromName(args.Lang)

	langLexer, err := lexicalv0.NewLexer(langGroup.Lexer)
	fail(err)

	start := time.Now()

	dirs := utils.DatasetForLang(utils.TrainDataset, langGroup)
	dirs = append(dirs, utils.DatasetForLang(utils.ValidateDataset, langGroup)...)
	sort.Strings(dirs)

	var files []string
	for _, dir := range dirs {
		log.Println("using input dir:", dir)
		dirFiles, err := aggregator.ListDir(dir)
		fail(err)
		files = append(files, dirFiles...)
	}

	sort.Strings(files)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = runtime.NumCPU() * 2
	emrOpts.MaxFileSize = 1 << 17 // 128kb
	emrOpts.MaxRecords = args.MaxFiles
	emrOpts.CacheRoot = args.CacheRoot
	emrOpts.PanicOnError = false

	srcs := source.NewEMRDataset("train-and-validate-corpus", emrOpts, files)

	var numFiles int32
	var filtered int32

	aggregator, err := words.NewAggregator(args.SplitsDir)
	fail(err)

	vocab := dependent.NewFromFunc("wordcount-from-test-train", func(s pipeline.Sample) {
		kv := s.(pipeline.Keyed)
		bs := []byte(kv.Sample.(sample.ByteSlice))

		if utils.FilterFile(kv.Key, bs) {
			atomic.AddInt32(&filtered, 1)
			return
		}

		tokens, err := langLexer.Lex(bs)
		if err != nil {
			log.Println("lex error:", err)
			return
		}

		ext := filepath.Ext(kv.Key)

		wc := make(words.Counts)
		for _, tok := range tokens {
			if parts, ok := langLexer.ShouldBPEEncode(tok); ok {
				for _, part := range parts {
					wc.Hit(part, ext, 1)
				}
			}
		}

		atomic.AddInt32(&numFiles, 1)
		aggregator.Add(wc)
	})

	pm := make(pipeline.ParentMap)
	pm.Chain(
		srcs,
		vocab,
	)

	pipe := pipeline.Pipeline{
		Name:    "wordcount",
		Parents: pm,
		Sources: []pipeline.Source{srcs},
		ResultsFn: func(map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
			res := []rundb.Result{
				{
					Name:  "Duration",
					Value: fmt.Sprintf("%v", time.Since(start)),
				},
				{
					Name:  "Files",
					Value: numFiles,
				},
				{
					Name:  "Filtered",
					Value: filtered,
				},
			}
			for _, r := range res {
				fmt.Println(r.Name, r.Value)
			}
			return res
		},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: runtime.NumCPU() * 3,
	})
	fail(err)

	_, err = engine.Run()
	fail(err)

	err = aggregator.Flush()
	fail(err)

	fmt.Printf("filtered %d files out of %d\n", filtered, numFiles)
	fmt.Printf("done with counts, took %v\n", time.Since(start))

	wordcount, err := aggregator.Merge(args.MinCount)
	fail(err)

	normalized := wordcount.Normalized(args.MinCount)

	fmt.Printf("done with normalization, took %v\n", time.Since(start))
	var wc []bpe.BuilderWordCount
	for word, count := range normalized {
		wBytes := []byte(word)
		if word != string(wBytes) {
			panic(fmt.Sprintf("bad times, original word %s != %s (string([]byte(word))), bytes %x", word, string(wBytes), wBytes))
		}
		wc = append(wc, bpe.BuilderWordCount{
			Word:  wBytes,
			Count: count,
		})
	}

	sort.Slice(wc, func(i, j int) bool {
		return wc[i].Count > wc[j].Count
	})

	f, err := os.Create(args.Output)
	fail(err)
	defer f.Close()

	err = json.NewEncoder(f).Encode(&wc)
	fail(err)
}
