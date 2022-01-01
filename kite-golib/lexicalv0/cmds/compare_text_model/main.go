package main

import (
	"fmt"
	"math/rand"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/utils"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

type predictorBundle struct {
	predict.Predictor
	Search predict.SearchConfig
}

func newPredictorBundle(path string, l lexicalv0.LangGroup) predictorBundle {
	predictor, err := predict.NewPredictor(path, l)
	fail(err)
	search, err := predict.NewSearchConfigFromModelPath(path)
	fail(err)

	return predictorBundle{
		Predictor: predictor,
		Search:    search,
	}
}

func main() {
	args := struct {
		TextModel      string
		NativeModel    string
		NativeLang     string
		MaxFiles       int
		Seed           int64
		OutMetrics     string
		OutRenderDiffs string
	}{
		MaxFiles:       50,
		OutMetrics:     "results.tsv",
		OutRenderDiffs: "render-diffs.txt",
	}
	arg.MustParse(&args)

	rand.Seed(args.Seed)

	nativeLang := lang.MustFromName(args.NativeLang)

	native := newPredictorBundle(args.NativeModel, lexicalv0.NewLangGroup(nativeLang))

	text := newPredictorBundle(args.TextModel, lexicalv0.NewLangGroup(lang.Text, nativeLang))

	files, err := utils.LocalFilesKiteco(lexicalv0.NewLangGroup(nativeLang), "")
	fail(err)

	rand.Shuffle(len(files), func(i, j int) {
		files[i], files[j] = files[j], files[i]
	})

	if args.MaxFiles > 0 && len(files) > args.MaxFiles {
		files = files[:args.MaxFiles]
	}

	sampleRates := sampleRates{
		atleast1Ident:  .05,
		atleast2Idents: .05,
	}

	fmt.Println("Computing metrics...")
	computeMetrics(args.OutMetrics, sampleRates, files, native, text)
	fmt.Println("Computing render diffs...")
	computeRenderDiffs(args.OutRenderDiffs, sampleRates, files, native, text)
}
