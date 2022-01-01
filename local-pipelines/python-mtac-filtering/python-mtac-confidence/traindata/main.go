package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/mtacconf"
	mtacutils "github.com/kiteco/kiteco/local-pipelines/python-mtac-filtering/internal/utils"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

const (
	maxFileSize = 50000
)

var (
	// scanOpts and parseOpts should match the options in the driver (or whatever is running inference with the model)
	scanOpts = pythonscanner.Options{
		ScanComments: true,
		ScanNewLines: true,
	}

	parseOpts = pythonparser.Options{
		ScanOptions: pythonscanner.Options{
			ScanComments: false,
			ScanNewLines: false,
		},
		ErrorMode: pythonparser.Recover,
	}

	datasetPath = pythoncode.DedupedCodeDumpPath
)

type rng struct {
	r *rand.Rand
	m sync.Mutex
}

func newRNG(seed int64) *rng {
	return &rng{
		r: rand.New(rand.NewSource(seed)),
	}
}

func (r *rng) Random() *rand.Rand {
	r.m.Lock()
	defer r.m.Unlock()

	seed := r.r.Int63()
	return rand.New(rand.NewSource(seed))
}

type trainSample mtacconf.TrainSample

func (trainSample) SampleTag() {}

type options struct {
	MaxFiles int
	// OutDir can be a local or S3 directory
	OutDir string
}

type pipe struct {
	pipeline.Pipeline
	Writer *aggregator.Writer
}

func createPipeline(res mtacutils.Resources, opts options) (pipe, error) {
	rng := newRNG(1)

	files, err := aggregator.ListDir(datasetPath)
	if err != nil {
		return pipe{}, err
	}
	if len(files) == 0 {
		return pipe{}, fmt.Errorf("no files present in %s", datasetPath)
	}

	emrOpts := source.DefaultEMRDatasetOpts
	emrOpts.MaxRecords = opts.MaxFiles
	emrOpts.MaxFileSize = maxFileSize

	dataset := source.NewEMRDataset("dataset", emrOpts, files)

	inputs := transform.NewOneInOneOut("sample-inputs", func(s pipeline.Sample) pipeline.Sample {
		k := s.(pipeline.Keyed)
		buf := k.Sample.(sample.ByteSlice)

		in, err := getSampleInputs(k.Key, buf, res, rng.Random())
		if err != nil {
			return nil
		}

		return in
	})

	samples := transform.NewOneInOneOut("samples", func(s pipeline.Sample) pipeline.Sample {
		in := s.(sampleInputs)

		ts, err := getTrainSample(in, res)
		if err != nil {
			return nil
		}
		return trainSample(ts)
	})

	writer := aggregator.NewJSONWriter(aggregator.DefaultWriterOpts, "writer", opts.OutDir)

	pm := make(pipeline.ParentMap)

	pm.Chain(
		dataset,
		inputs,
		samples,
		writer)

	resFn := func(out map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
		var results []rundb.Result
		for _, file := range out[writer].(sample.StringSlice) {
			results = append(results, rundb.Result{
				Name:  "File",
				Value: file,
			})
		}
		return results
	}

	return pipe{
		Pipeline: pipeline.Pipeline{
			Name:    "mtac-conf-traindata",
			Parents: pm,
			Sources: []pipeline.Source{dataset},
			Params: map[string]interface{}{
				"MaxFiles": opts.MaxFiles,
				"OutDir":   opts.OutDir,
			},
			ResultsFn: resFn,
		},
		Writer: writer,
	}, nil
}

func main() {
	datadeps.Enable()
	args := struct {
		MaxFiles  int
		OutDir    string
		RunDBPath string
		Role      pipeline.Role
		Port      int
		Endpoints []string
	}{
		OutDir:    "./out",
		MaxFiles:  10000,
		RunDBPath: rundb.DefaultRunDB,
		Port:      0,
		Endpoints: nil,
	}

	arg.MustParse(&args)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		log.Fatal(err)
	}

	models, err := pythonmodels.New(pythonmodels.DefaultOptions)
	if err != nil {
		log.Fatalln(err)
	}

	res := mtacutils.Resources{RM: rm, Models: models}

	opts := options{
		MaxFiles: args.MaxFiles,
		OutDir:   args.OutDir,
	}

	start := time.Now()

	pipe, err := createPipeline(res, opts)
	if err != nil {
		log.Fatal(err)
	}
	eOpts := pipeline.DefaultEngineOptions
	outf, err := os.Create("log.txt")
	if err != nil {
		log.Fatal(err)
	}
	eOpts.Logger = outf

	eOpts.RunDBPath = args.RunDBPath
	eOpts.Role = args.Role
	eOpts.Port = args.Port
	eOpts.ShardEndpoints = args.Endpoints

	engine, err := pipeline.NewEngine(pipe.Pipeline, eOpts)
	if err != nil {
		log.Fatal(err)
	}

	out, err := engine.Run()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("files written:")
	for _, filename := range out[pipe.Writer].(sample.StringSlice) {
		log.Printf(filename)
	}

	log.Printf("Done! took %v", time.Since(start))
}
