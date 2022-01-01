package main

import (
	"context"
	"log"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/local-pipelines/python-call-filtering/internal/utils"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func canPredictCall(call *pythonast.CallExpr) bool {
	if call.LeftParen == nil || call.RightParen == nil {
		return false
	}

	if !pythonast.IsNil(call.Kwarg) || !pythonast.IsNil(call.Vararg) {
		return false
	}

	for _, argument := range call.Args {
		if _, ok := argument.Value.(*pythonast.NameExpr); !ok {
			return false
		}
	}
	return true
}

func findCallSites(ast *pythonast.Module) []*pythonast.CallExpr {
	var calls []*pythonast.CallExpr
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		switch n := n.(type) {
		case *pythonast.CallExpr:
			calls = append(calls, n)
		}

		return true
	})

	return calls
}

func newRecord(src []byte, res utils.Resources) (record, error) {
	ast, _, err := utils.Parse(src)
	if err != nil {
		return record{}, err
	}

	nodes := findCallSites(ast)
	if len(nodes) == 0 {
		return record{}, errors.Errorf("no call expressions found")
	}

	rast, err := utils.Resolve(ast, res.RM)
	if err != nil {
		return record{}, errors.Errorf("error resolving ast: %v", err)
	}

	var lparens []int
	for _, n := range nodes {
		val := rast.References[n.Func]
		if val == nil {
			continue
		}
		if !utils.ValueSupported(res, val) {
			continue
		}
		if !canPredictCall(n) {
			continue
		}
		lparens = append(lparens, int(n.LeftParen.Begin))
	}

	if len(lparens) == 0 {
		return record{}, errors.Errorf("no valid call sites")
	}

	return record{
		Source:  src,
		PosList: lparens,
	}, nil
}

type record struct {
	Source  []byte
	PosList []int
}

func (record) SampleTag() {}

func main() {
	fail(datadeps.Enable())
	args := struct {
		Out         string
		MaxFiles    int
		NumReaders  int
		NumAnalysis int
		RunDB       string
		ExprShards  string
		RunName     string
	}{
		Out:         "./out",
		MaxFiles:    100,
		NumReaders:  4,
		NumAnalysis: 4,
	}
	arg.MustParse(&args)

	var runDB string
	if args.RunName != "" {
		runDB = args.RunDB
	}

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	modelOpts := pythonmodels.DefaultOptions
	if args.ExprShards != "" {
		shards, err := pythonexpr.ShardsFromFile(args.ExprShards)
		fail(err)
		modelOpts.ExprModelShards = shards
	}

	expr, err := pythonexpr.NewShardedModel(context.Background(), modelOpts.ExprModelShards, modelOpts.ExprModelOpts)
	if err != nil {
		panic(err)
	}

	models := &pythonmodels.Models{
		Expr: expr,
	}

	res := utils.Resources{RM: rm, Models: models}

	start := time.Now()

	files, err := aggregator.ListDir(pythoncode.DedupedCodeDumpPath)
	fail(err)

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.MaxRecords = args.MaxFiles
	emrOpts.NumGo = args.NumReaders

	dataset := source.NewEMRDataset("dataset", emrOpts, files)
	samples := transform.NewOneInOneOut("samples", func(s pipeline.Sample) pipeline.Sample {
		k := s.(pipeline.Keyed)
		buf := k.Sample.(sample.ByteSlice)

		in, err := newRecord([]byte(buf), res)
		if err != nil {
			return pipeline.WrapError("error building record", err)
		}

		return in
	})

	writer := aggregator.NewJSONWriter(aggregator.DefaultWriterOpts, "writer", args.Out)

	pm := make(pipeline.ParentMap)

	pm.Chain(
		dataset,
		samples,
		writer,
	)

	eOpts := pipeline.DefaultEngineOptions
	eOpts.RunDBPath = runDB
	eOpts.RunName = args.RunName
	eOpts.NumWorkers = args.NumAnalysis

	pipe := pipeline.Pipeline{
		Name:    "call-prob-gt-data",
		Parents: pm,
		Sources: []pipeline.Source{dataset},
		Params: map[string]interface{}{
			"MaxFiles": args.MaxFiles,
			"OutDir":   args.Out,
		},
	}

	engine, err := pipeline.NewEngine(pipe, eOpts)
	fail(err)

	out, err := engine.Run()
	fail(err)

	log.Printf("files written:")
	for _, filename := range out[writer].(sample.StringSlice) {
		log.Println(filename)
	}

	log.Printf("Done! took %v", time.Since(start))
}
