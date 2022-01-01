package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/performance"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	args := struct {
		Lang              string
		ModelPath         string
		ResultPath        string
		Search            string
		SeriesLength      int
		AccuracyTopN      int
		RandomSeed        int64
		ModelConfig       bool
		MinMuddle         int
		UnmuddledWindow   int
		UseSearcher       bool
		UseCache          bool
		NumThreads        int
		LocalDataRoot     string
		DetailedOutput    bool
		AllowedExtensions string
	}{
		AccuracyTopN:   5,
		SeriesLength:   5,
		RandomSeed:     2019,
		UseCache:       true,
		NumThreads:     runtime.NumCPU() / 2,
		UseSearcher:    false,
		DetailedOutput: false,
	}

	arg.MustParse(&args)

	tensorflow.SetTensorflowThreadpoolSize(runtime.NumCPU())

	langGroup := lexicalv0.MustLangGroupFromName(args.Lang)
	allExts := make(map[string]bool)
	if args.AllowedExtensions != "" {
		for _, e := range strings.Split(args.AllowedExtensions, ",") {
			allExts[e] = true
		}
	} else {
		for _, l := range langGroup.Langs {
			for _, e := range l.Extensions() {
				allExts[e] = true
			}
		}
	}

	var predictor predict.Predictor
	if args.UseSearcher {
		var err error
		predictor, err = predict.NewTFSearcherFromS3(args.ModelPath, langGroup)
		fail(err)
	} else {
		p, err := predict.NewPredictor(args.ModelPath, langGroup)
		fail(err)
		switch t := p.(type) {
		case *predict.TFPredictor:
			t.SetUseCache(args.UseCache)
		case *predict.PrefixSuffixPredictor:
			t.SetUseCache(args.UseCache)
		}
		predictor = p
	}
	predictor.SetStrictChecking(true)

	if args.ModelConfig {
		args.Search = predict.SearchConfigPathFromModelPath(args.ModelPath)
	}
	// Various measurements and their sampling ratios
	measurements := map[performance.Measurement]float64{
		performance.Lexical: 0,
		performance.Word:    0.1,
		performance.Series:  0,
		// CharValueAdded is computed along TokenValueAdded, they both use the sampling rate associated with TokenValueAdded
		performance.TokenValueAdded:    0.1,
		performance.CharValueAdded:     0.1,
		performance.CorrectTokensAhead: 0.05,
	}

	// Minimum number of prediction sites for each measurement
	minNums := minNumSites{
		performance.Word:               300,
		performance.TokenValueAdded:    300, // CharValueAdded is computed along TokenValueAdded with the same min num
		performance.CorrectTokensAhead: 300,
	}

	search, err := predict.NewSearchConfig(args.Search)
	fail(err)

	start := time.Now()
	gen, err := inspect.NewCodeGeneratorWithOpts(langGroup, langGroup.Lexer != lang.Text, "", args.LocalDataRoot, args.RandomSeed)

	fail(err)

	extractor := performance.Extractor{
		Encoder:      predictor.GetEncoder(),
		Rand:         rand.New(rand.NewSource(args.RandomSeed)),
		RandomSeed:   args.RandomSeed,
		Measurements: measurements,
		SeriesLength: args.SeriesLength,
	}

	pathToSites := collectSites(gen, extractor, minNums, allExts)
	fmt.Printf("Evaluating sites on %d files.\n", len(pathToSites))
	var completed int32
	var m sync.Mutex
	var jobs []workerpool.Job
	var evaluators []performance.Evaluator
	pool := workerpool.New(args.NumThreads)
	for path, sites := range pathToSites {
		localPath := path
		localSites := sites
		jobs = append(jobs, func() error {
			evaluator := performance.Evaluator{
				Filename:        localPath,
				Encoder:         predictor.GetEncoder(),
				Predictor:       predictor,
				Search:          search,
				SeriesLength:    args.SeriesLength,
				TopN:            args.AccuracyTopN,
				Rand:            rand.New(rand.NewSource(args.RandomSeed)),
				RandomSeed:      args.RandomSeed,
				Measurements:    measurements,
				MinMuddle:       args.MinMuddle,
				UnmuddledWindow: args.UnmuddledWindow,
			}

			start := time.Now()
			log.Printf("starting on %s", localPath)

			if err := evaluator.Eval(localSites); err != nil {
				log.Printf("error measuring model performance on %s: %v", path, err)
				panic(err)
			}

			finished := atomic.AddInt32(&completed, 1)
			log.Printf("(%d/%d) completed in %s: %s", finished, len(pathToSites), time.Since(start), localPath)

			m.Lock()
			defer m.Unlock()
			evaluators = append(evaluators, evaluator)
			return nil
		})
	}

	pool.AddBlocking(jobs)
	err = pool.Wait()
	fail(err)

	duration := time.Since(start)
	fmt.Printf("took %v to process %d files, aggregating the results...\n", duration, len(pathToSites))

	f, err := os.Create(args.ResultPath)
	fail(err)
	defer f.Close()

	// Log the model name and search configs
	fmt.Fprintf(f, "num_files\t%v\n", len(pathToSites))
	fmt.Fprintf(f, "processing_time\t%v\n", time.Since(start))
	fmt.Fprintf(f, "model_path\t%s\n", args.ModelPath)
	fmt.Fprintf(f, "search\t%+v\n", search)
	fmt.Fprintf(f, "\n\n")

	fail(performance.AggregateAndWrite(evaluators, f, args.DetailedOutput))
}
