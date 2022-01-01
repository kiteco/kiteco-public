package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/type-induction/data"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		MaxEvents   int
		Out         string
		NumAnalysis int
		Packages    string
		RunDBPath   string
		RunName     string
		Role        pipeline.Role
		Port        int
		Endpoints   []string
	}{
		Packages:    "all-packages.txt",
		MaxEvents:   0,
		NumAnalysis: 2,
		Port:        0,
		RunDBPath:   rundb.DefaultRunDB,
	}

	arg.MustParse(&args)

	pkgs, err := traindata.LoadPackageList(args.Packages)
	fail(err)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	startDate, err := analyze.ParseDate("2018-01-01")
	fail(err)
	endDate, err := analyze.ParseDate("2019-01-20")
	fail(err)

	recreator, err := servercontext.NewRecreator(servercontext.DefaultBucketsByRegion)
	fail(err)

	trackOpts := pythonpipeline.DefaultTrackingEventsOpts
	trackOpts.MaxEvents = args.MaxEvents
	trackOpts.ShardByUMF = true
	trackOpts.NumReaders = 2
	trackOpts.Logger = os.Stdout

	events := pythonpipeline.NewTrackingEvents(startDate, endDate, pythontracking.ServerSignatureFailureEvent, trackOpts)
	deduped := transform.NewFilter("deduped", pythonpipeline.DedupeEvents())
	analyzed := transform.NewOneInOneOut("analyzed", pythonpipeline.AnalyzeEvents(recreator, false))

	extractor := data.NewExtractor(rm, pkgs)
	extracted := transform.NewMap("extracted", func(s pipeline.Sample) []pipeline.Sample {
		ev := s.(pythonpipeline.AnalyzedEvent)
		return extractor.Extract(ev.Context.Resolved)
	})

	agg := aggregator.NewSumAggregator("aggregated", func() sample.Addable {
		return make(data.SampleByPkg)
	}, func(s pipeline.Sample) sample.Addable {
		ex := s.(data.Sample)
		return data.SampleByPkg{ex.Pkg: []data.Sample{ex}}
	})

	pm := make(pipeline.ParentMap)
	pm.Chain(
		events,
		deduped,
		analyzed,
		extracted,
		agg,
	)

	pipe := pipeline.Pipeline{
		Name:    "python-type-induction-validation-examples",
		Parents: pm,
		Sources: []pipeline.Source{events},
	}

	var runDBPath string
	if args.RunName != "" {
		runDBPath = args.RunDBPath
	}

	eOpts := pipeline.DefaultEngineOptions
	eOpts.RunDBPath = runDBPath
	eOpts.RunName = args.RunName
	eOpts.Role = args.Role
	eOpts.Port = args.Port
	eOpts.ShardEndpoints = args.Endpoints
	eOpts.NumWorkers = args.NumAnalysis

	start := time.Now()
	engine, err := pipeline.NewEngine(pipe, eOpts)
	fail(err)

	res, err := engine.Run()
	fail(err)

	samples := res[agg].(data.SampleByPkg)

	for k, v := range samples {
		if len(v) > 0 {
			fmt.Printf("Found %d validation samples for package %s\n", len(v), k)
			f, err := fileutil.NewBufferedWriter(fmt.Sprintf("%s/%s.json", args.Out, k))
			fail(err)
			fail(json.NewEncoder(f).Encode(&v))
			fail(f.Close())
		}
	}

	fmt.Printf("Done! Took %v to extract samples. Validation samples were extracted and saved to %s.\n", time.Since(start), args.Out)
}
