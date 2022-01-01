package main

import (
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sort"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/utils"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	args := struct {
		Lang      string
		ModelPath string
		Window    int
		MaxFiles  int
		CacheRoot string
	}{
		ModelPath: "s3://kite-data/run-db/2019-12-12T02:08:29Z_lexical-model-experiments/out_javascript_lexical_context_128_embedding_180_layer_6_head_6_vocab_20000_steps_2000000",
		MaxFiles:  100,
		CacheRoot: "/data/kite",
	}

	arg.MustParse(&args)

	group := lexicalv0.MustLangGroupFromName(args.Lang)

	var testFiles []string
	for _, d := range utils.DatasetForLang(utils.TestDataset, group) {
		ts, err := aggregator.ListDir(d)
		fail(err)
		testFiles = append(testFiles, ts...)
	}
	sort.Strings(testFiles)

	predictor, err := predict.NewPredictor(args.ModelPath, group)
	fail(err)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.NumGo = runtime.NumCPU()
	emrOpts.MaxFileSize = 1 << 17 // 128kb
	emrOpts.MaxRecords = args.MaxFiles
	emrOpts.CacheRoot = args.CacheRoot

	var allIdent, inString int

	src := source.NewEMRDataset("test-corpus", emrOpts, testFiles)
	inspectIdent := dependent.NewFromFunc("inspect-ident", func(s pipeline.Sample) {
		// This is mainly for handling the fact that our dataset is not perfectly shuffled so that we include more diverse files.
		if rand.Float32() < 0.9 {
			return
		}
		kv := s.(pipeline.Keyed)
		bs := []byte(kv.Sample.(sample.ByteSlice))
		tokens, err := predictor.GetEncoder().Lexer.Lex(bs)
		if err != nil {
			log.Println("lex error:", err)
			return
		}
		for i, tok := range tokens {
			if !predictor.GetEncoder().Lexer.IsType(lexer.IDENT, tok) {
				continue
			}
			if i == 0 {
				continue
			}
			allIdent++
			for j := i - 1; j > 0; j-- {
				if !predictor.GetEncoder().Lexer.IsType(lexer.STRING, tokens[j]) {
					continue
				}
				if tokens[j].Lit != tok.Lit {
					continue
				}
				if len(predictor.GetEncoder().EncodeTokens(tokens[j:i])) > args.Window {
					break
				}
				inString++
			}
		}
	})

	var allString, hasIdent int
	inspectString := dependent.NewFromFunc("inspect-string", func(s pipeline.Sample) {
		// This is mainly for handling the fact that our dataset is not perfectly shuffled so that we include more diverse files.
		if rand.Float32() < 0.95 {
			return
		}
		kv := s.(pipeline.Keyed)
		bs := []byte(kv.Sample.(sample.ByteSlice))
		tokens, err := predictor.GetEncoder().Lexer.Lex(bs)
		if err != nil {
			log.Println("lex error:", err)
			return
		}
		for i, tok := range tokens {
			if !predictor.GetEncoder().Lexer.IsType(lexer.STRING, tok) {
				continue
			}
			if i == 0 {
				continue
			}
			allString++
			for j := i - 1; j > 0; j-- {
				if !predictor.GetEncoder().Lexer.IsType(lexer.IDENT, tokens[j]) {
					continue
				}
				if len(tokens[j].Lit) <= 2 {
					continue
				}
				if len(predictor.GetEncoder().EncodeTokens(tokens[j:i])) > args.Window {
					break
				}
				if strings.Contains(tok.Lit, tokens[j].Lit) {
					hasIdent++
				}
			}
		}
	})

	pm := make(pipeline.ParentMap)
	pm.Chain(
		src,
		inspectIdent,
	)
	pm.Chain(
		src,
		inspectString,
	)

	pipe := pipeline.Pipeline{
		Name:    "js-context-inspect",
		Parents: pm,
		Sources: []pipeline.Source{src},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: runtime.NumCPU(),
	})
	fail(err)

	_, err = engine.Run()
	fail(err)

	fmt.Printf("%.3f of ident prediction sites appear in context as string", float64(inString)/float64(allIdent))
	fmt.Printf("%.3f of string prediction sites contain substrings that appear in context as ident", float64(hasIdent)/float64(allString))
}
