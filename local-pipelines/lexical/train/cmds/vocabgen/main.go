package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/bpe"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/utils"
)

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		Lang           string
		Resume         string
		Words          string
		WordCount      int
		TopP           float64
		Output         string
		MaxVocabSize   int
		MaxFiles       int
		CacheRoot      string
		CheckpointsDir string
		UseBytes       bool
	}{
		Resume:         "",
		Words:          "",
		Output:         "ident-vocab.bpe",
		MaxVocabSize:   30000,
		MaxFiles:       100e6,
		CacheRoot:      "/data/kite",
		CheckpointsDir: "vocab-checkpoints",
		UseBytes:       true,
	}

	arg.MustParse(&args)

	var builder *bpe.Builder
	if args.Resume == "" {
		builder = bpe.NewBuilder(args.UseBytes)
	} else {
		log.Println("resuming bpe with", args.Resume)
		var err error
		builder, err = bpe.NewBuilderWithVocab(args.Resume)
		maybeQuit(err)
	}

	fromScratch := func() {
		langGroup := lexicalv0.MustLangGroupFromName(args.Lang)

		langLexer, err := lexicalv0.NewLexer(langGroup.Lexer)
		if err != nil {
			log.Fatalln(err)
		}

		start := time.Now()

		dirs := utils.DatasetForLang(utils.TrainDataset, langGroup)
		dirs = append(dirs, utils.DatasetForLang(utils.ValidateDataset, langGroup)...)
		sort.Strings(dirs)

		var files []string
		for _, dir := range dirs {
			log.Println("using input dir:", dir)
			dirFiles, err := aggregator.ListDir(dir)
			maybeQuit(err)
			files = append(files, dirFiles...)
		}

		sort.Strings(files)

		emrOpts := source.DefaultEMRDatasetOpts
		emrOpts.NumGo = runtime.NumCPU()
		emrOpts.MaxFileSize = 1 << 17 // 128kb
		emrOpts.MaxRecords = args.MaxFiles
		emrOpts.CacheRoot = args.CacheRoot

		srcs := source.NewEMRDataset("train-and-validate-corpus", emrOpts, files)

		var vocabFiles int
		vocab := dependent.NewFromFunc("vocab-from-test-train", func(s pipeline.Sample) {
			kv := s.(pipeline.Keyed)
			if strings.HasSuffix(kv.Key, ".min.js") {
				return
			}
			bs := []byte(kv.Sample.(sample.ByteSlice))
			tokens, err := langLexer.Lex(bs)
			if err != nil {
				log.Println("lex error:", err)
				return
			}

			var toks []string
			for _, tok := range tokens {
				if parts, ok := langLexer.ShouldBPEEncode(tok); ok {
					toks = append(toks, parts...)
				}
			}

			vocabFiles++
			builder.Add(toks)
		})

		pm := make(pipeline.ParentMap)
		pm.Chain(
			srcs,
			vocab,
		)

		pipe := pipeline.Pipeline{
			Name:    "lexical-vocabgen",
			Parents: pm,
			Sources: []pipeline.Source{srcs},
			ResultsFn: func(map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
				res := []rundb.Result{
					{
						Name:  "Duration",
						Value: fmt.Sprintf("%v", time.Since(start)),
					},
					{
						Name:  "Vocab files",
						Value: vocabFiles,
					},
					{
						Name:  "Vocab words",
						Value: builder.Words(),
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
		maybeQuit(err)

		_, err = engine.Run()
		maybeQuit(err)
	}

	if args.Words == "" {
		fromScratch()
	} else {
		err := builder.LoadWords(args.Words, bpe.LoadOptions{WordCount: args.WordCount, TopP: args.TopP})
		if err != nil {
			log.Fatalln(err)
		}
	}

	mergeStart := time.Now()
	log.Println("running bpe merge operation with max vocab of", args.MaxVocabSize)

	err := builder.Merge(bpe.MergeOptions{
		MaxVocabSize:  args.MaxVocabSize,
		Logging:       true,
		Concurrency:   2 * runtime.NumCPU(),
		CheckpointDir: args.CheckpointsDir,
	})
	maybeQuit(err)

	log.Println("bpe merge took", time.Since(mergeStart))

	f, err := os.Create(args.Output)
	maybeQuit(err)

	defer f.Close()
	_, err = builder.WriteTo(f)
	maybeQuit(err)
}
