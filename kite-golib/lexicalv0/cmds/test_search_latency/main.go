// +build darwin linux

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/benchmark"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

func fail(err error) {
	if err != nil {
		log.Println(err)
	}
}

func main() {
	args := struct {
		Lang             string
		Input            string
		Iters            int
		Verbose          bool
		FixedContextSize bool
		CompareResults   bool
		CompareLatency   bool
	}{
		Lang:             "golang",
		Iters:            10,
		FixedContextSize: false,
	}

	go http.ListenAndServe(":9501", nil)

	arg.MustParse(&args)
	l := lexicalv0.MustLangGroupFromName(args.Lang)

	var modelConfig lexicalmodels.ModelConfig

	switch l.Lexer {
	case lang.Golang:
		modelConfig = lexicalmodels.DefaultModelOptions.WithRemoteModels(lexicalmodels.DefaultRemoteHost).TextMiscGroup
	case lang.Python, lang.JavaScript:
		// add support when ready
	default:
		log.Fatalf("%s not supported", l.Name())
	}

	tfpred, err := predict.NewPredictor(modelConfig.ModelPath, l)
	fail(err)

	tfsearch, err := predict.NewTFSearcherFromS3(modelConfig.ModelPath, l)
	fail(err)

	tfremote, err := predict.NewTFServingSearcher(modelConfig.TFServing)
	fail(err)

	buf, err := ioutil.ReadFile(args.Input)
	fail(err)

	config, err := predict.NewSearchConfigFromModelPath(modelConfig.ModelPath)
	fail(err)

	b := benchmark.Benchmarker{
		Predictor: tfpred,
		Search:    config,
		Iters:     args.Iters,
		Buf:       buf,
		Verbose:   args.Verbose,
	}

	if args.CompareLatency {
		b.MustBenchmark("predictor", nil,
			func(predictor lexicalmodels.ModelBase, in predict.Inputs) {
				preds, err := tfpred.PredictChan(kitectx.Background(), in)
				for range preds {
				}
				fail(<-err)
			},
		).Print(os.Stdout)

		b.MustBenchmark("searcher", nil,
			func(predictor lexicalmodels.ModelBase, in predict.Inputs) {
				preds, err := tfsearch.PredictChan(kitectx.Background(), in)
				for range preds {
				}
				fail(<-err)
			},
		).Print(os.Stdout)

		b.MustBenchmark("searcher-remote", nil,
			func(predictor lexicalmodels.ModelBase, in predict.Inputs) {
				preds, err := tfremote.PredictChan(kitectx.Background(), in)
				for range preds {
				}
				fail(<-err)
			},
		).Print(os.Stdout)
	}

	if args.CompareResults {
		var allPreds, allSearches, allRemotes [][]predict.Predicted
		b.MustBenchmark("predict-vs-search-vs-remote", nil,
			func(predictor lexicalmodels.ModelBase, in predict.Inputs) {
				preds, errChan := tfpred.PredictChan(kitectx.Background(), in)
				var pred []predict.Predicted
				for p := range preds {
					pred = append(pred, p)
				}
				err = <-errChan
				if err != nil {
					err = errors.Wrapf(err, "tfpred")
					fail(err)
				}

				preds, errChan = tfsearch.PredictChan(kitectx.Background(), in)
				var search []predict.Predicted
				for p := range preds {
					search = append(search, p)
				}
				err = <-errChan
				if err != nil {
					err = errors.Wrapf(err, "tfsearch")
					fail(err)
				}

				preds, errChan = tfremote.PredictChan(kitectx.Background(), in)
				var remote []predict.Predicted
				for p := range preds {
					remote = append(remote, p)
				}
				err = <-errChan
				if err != nil {
					err = errors.Wrapf(err, "tfsearch_batched")
					fail(err)
				}

				sort.Slice(pred, func(i, j int) bool {
					return pred[i].Prob > pred[j].Prob
				})
				sort.Slice(search, func(i, j int) bool {
					return search[i].Prob > search[j].Prob
				})
				sort.Slice(remote, func(i, j int) bool {
					return remote[i].Prob > remote[j].Prob
				})

				allPreds = append(allPreds, pred)
				allSearches = append(allSearches, search)
				allRemotes = append(allRemotes, remote)
			},
		).Print(ioutil.Discard)

		tabw := tabwriter.NewWriter(os.Stdout, 16, 4, 4, ' ', 0)
		for i := 0; i < len(allPreds); i++ {
			preds, searches, remotes := allPreds[i], allSearches[i], allRemotes[i]
			fmt.Fprintln(tabw, "iter:idx\tpred\tprob\tsearch\tprob\tremote\tprob")
			for j := 0; j < len(preds) || j < len(searches) || j < len(remotes); j++ {
				var s, p, r predict.Predicted
				if j < len(preds) {
					p = preds[j]
				}
				if j < len(searches) {
					s = searches[j]
				}
				if j < len(remotes) {
					r = remotes[j]
				}
				fmt.Fprintf(tabw, "%d:%d\t%v\t%.04f\t%v\t%.04f\t%v\t%.04f\n", i, j,
					p.TokenIDs, p.Prob, s.TokenIDs, s.Prob, r.TokenIDs, r.Prob)
			}
		}
		tabw.Flush()
	}
}

func toInt(ctx []int64) []int {
	var new []int
	for _, c := range ctx {
		new = append(new, int(c))
	}
	return new
}

func toInt64(ctx []int) []int64 {
	var new []int64
	for _, c := range ctx {
		new = append(new, int64(c))
	}
	return new
}
