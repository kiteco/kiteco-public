// +build darwin linux

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"

	"github.com/alexflint/go-arg"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/benchmark"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/protobuf/tensorflow/core/protobuf"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	args := struct {
		Lang       string
		Model      string
		Search     string
		Input      string
		ThreadPool int
		Iters      int
		Verbose    bool
		Out        string
		XLA        bool
	}{
		ThreadPool: 1,
		Iters:      100,
		Out:        "results.tsv",
		XLA:        false,
	}
	arg.MustParse(&args)

	var globalJitLevel protobuf.OptimizerOptions_GlobalJitLevel
	if args.XLA {
		globalJitLevel = 2
		os.Setenv("TF_XLA_FLAGS", "--tf_xla_cpu_global_jit")
	}

	tfConfig := protobuf.ConfigProto{
		IntraOpParallelismThreads: int32(args.ThreadPool),
		InterOpParallelismThreads: int32(args.ThreadPool),
		DeviceCount:               map[string]int32{"CPU": 1, "GPU": 1},
		GraphOptions: &protobuf.GraphOptions{
			OptimizerOptions: &protobuf.OptimizerOptions{
				OptLevel:           0,
				GlobalJitLevel:     globalJitLevel,
				DoFunctionInlining: true,
			},
		},
	}

	tensorflow.SetSessionOptions(&tfConfig)

	l := lexicalv0.MustLangGroupFromName(args.Lang)

	predictor, err := predict.NewPredictor(args.Model, l)
	fail(err)

	if args.Search == "" {
		args.Search = predict.SearchConfigPathFromModelPath(args.Model)
	}

	search, err := predict.NewSearchConfig(args.Search)
	fail(err)

	// set MinP to zero to get worst case times for long completions
	search.MinP = 0

	buf, err := ioutil.ReadFile(args.Input)
	fail(err)

	sbuf, err := json.MarshalIndent(search, "", "  ")
	fail(err)

	w := io.Writer(os.Stdout)
	if args.Out != "" {
		f, err := os.Create(args.Out)
		fail(err)
		defer f.Close()
		w = io.MultiWriter(f, w)
	}

	fmt.Fprintf(w, "For model @ %s\n", args.Model)
	fmt.Fprintf(w, "and search config @ %s\n", args.Search)
	fmt.Fprintf(w, "using search config: %s\n", string(sbuf))

	optBuf, err := json.MarshalIndent(*tensorflow.GetSessionOptions(), "", "  ")
	fail(err)

	fmt.Fprintf(w, "using tensorflow session options: %s\n", string(optBuf))

	b := benchmark.Benchmarker{
		Iters:     args.Iters,
		Predictor: predictor,
		Search:    search,
		Buf:       buf,
		Verbose:   args.Verbose,
	}

	fmt.Fprintf(w, "\n\n===== Partial run stats, beam-in-golang =====\n\n")

	newPRM := func(in predict.Inputs) *predict.PartialRunModel {
		ctx := predictor.GetEncoder().EncodeTokens(in.Tokens)
		langTag := predictor.GetEncoder().LangTagForPath(in.FilePath)
		prm, err := predict.NewPartialRunModel(predictor.GetModel(), toInt64(ctx), predictor.GetHParams(), search, false, langTag)
		fail(err)
		return prm
	}

	b.MustBenchmark("initialize-partial-run-and-embed-initial-context", nil,
		func(model lexicalmodels.ModelBase, in predict.Inputs) {
			newPRM(in)
		},
	).Print(w)

	var prm *predict.PartialRunModel
	b.MustBenchmark("close-partial-run",
		func(model lexicalmodels.ModelBase, in predict.Inputs) {
			prm = newPRM(in)
		},
		func(model lexicalmodels.ModelBase, in predict.Inputs) {
			fail(prm.Close())
		},
	).Print(w)

	b.MustBenchmark("probs-partial-run",
		func(model lexicalmodels.ModelBase, in predict.Inputs) {
			prm = newPRM(in)
		},
		func(model lexicalmodels.ModelBase, in predict.Inputs) {
			ctx := toInt64(model.GetEncoder().EncodeTokens(in.Tokens))
			if len(ctx) > 6 {
				ctx = ctx[:6]
			}
			newCtx := [][]int64{ctx, ctx, ctx, ctx, ctx}
			_, err = prm.Query(newCtx)
			fail(err)
		},
	).Print(w)

	b.MustBenchmark("predict-partial-run-state-already-initialized", nil,
		func(model lexicalmodels.ModelBase, in predict.Inputs) {
			predChan, errChan := predictor.PredictChan(kitectx.Background(), in)
			for range predChan {
			}
			fail(<-errChan)
		},
	).Print(w)

	b.MustBenchmark("predict-end-to-end-partial-run", nil,
		func(model lexicalmodels.ModelBase, in predict.Inputs) {
			predChan, errChan := predictor.PredictChan(kitectx.Background(), in)
			for range predChan {
			}
			fail(<-errChan)
		},
	).Print(w)
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
