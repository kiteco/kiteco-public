package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
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

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		Lang         string
		Vocab        string
		Words        string
		WordCount    int
		TopP         float64
		Output       string
		UseSubtokens bool
		MaxFiles     int
		CacheRoot    string
	}{
		Output:    "ident-vocab-count.json",
		Vocab:     "ident-vocab.bpe",
		MaxFiles:  5e6,
		CacheRoot: "/data/kite",
	}

	arg.MustParse(&args)

	builder, err := bpe.NewBuilderWithVocab(args.Vocab)
	fail(err)

	if args.Words != "" {
		err := builder.LoadWords(args.Words, bpe.LoadOptions{WordCount: args.WordCount, TopP: args.TopP})
		fail(err)
	} else {
		group := lexicalv0.MustLangGroupFromName(args.Lang)

		langLexer, err := lexicalv0.NewLexer(group.Lexer)
		if err != nil {
			log.Fatalln(err)
		}

		start := time.Now()

		dirs := utils.DatasetForLang(utils.TrainDataset, group)
		dirs = append(dirs, utils.DatasetForLang(utils.ValidateDataset, group)...)
		sort.Strings(dirs)

		var files []string
		for _, dir := range dirs {
			dirFiles, err := aggregator.ListDir(dir)
			fail(err)
			files = append(files, dirFiles...)
		}

		sort.Strings(files)

		emrOpts := source.DefaultEMRDatasetOpts
		emrOpts.NumGo = runtime.NumCPU()
		emrOpts.MaxFileSize = 1 << 18 // 256kb
		emrOpts.MaxRecords = args.MaxFiles
		emrOpts.CacheRoot = args.CacheRoot

		srcs := source.NewEMRDataset("go-train-and-validate-corpus", emrOpts, files)
		count := dependent.NewFromFunc("vocab-from-test-train", func(s pipeline.Sample) {
			kv := s.(pipeline.Keyed)
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

			builder.Add(toks)
		})

		pm := make(pipeline.ParentMap)
		pm.Chain(
			srcs,
			count,
		)

		pipe := pipeline.Pipeline{
			Name:    "golang-lexical-vocab-count",
			Parents: pm,
			Sources: []pipeline.Source{srcs},
			ResultsFn: func(map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
				res := []rundb.Result{
					{
						Name:  "Duration",
						Value: fmt.Sprintf("%v", time.Since(start)),
					},
					{
						Name:  "Vocab Size",
						Value: len(builder.CurrentVocab()),
					},
				}
				for _, r := range res {
					fmt.Println(r.Name, r.Value)
				}
				return res
			},
		}

		engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
			NumWorkers: runtime.NumCPU(),
		})
		fail(err)

		_, err = engine.Run()
		fail(err)
	}

	// Count the popularity
	counts := builder.CurrentTokens()

	f, err := os.Create(args.Output)
	fail(err)

	defer f.Close()
	fail(json.NewEncoder(f).Encode(counts))
}
