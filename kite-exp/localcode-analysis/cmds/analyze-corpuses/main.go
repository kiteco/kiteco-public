package main

import (
	"hash/fnv"
	"log"
	"math/rand"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-exp/localcode-analysis/index"
	"github.com/kiteco/kiteco/kite-exp/localcode-analysis/localfiles"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

const (
	maxCallsPerFile   = 10
	maxFilesPerCorpus = 5
	parseTimeout      = 3 * time.Second
	resolveTimeout    = 3 * time.Second
)

const (
	localFilesS3Bucket = "kite-local-content"
	localFilesS3Region = "us-west-1"
)

var (
	localFilesDBURI = envutil.MustGetenv("PROD_WESTUS2_LOCALFILES_DB_URI")
)

type analysisPipeline struct {
	Pipeline pipeline.Pipeline
	WriteAgg pipeline.Aggregator
}

type resources struct {
	rm pythonresource.Manager
	mi pythonexpr.MetaInfo
}

type options struct {
	MaxCorpora int
	OutDir     string
}

func (o options) Params() map[string]interface{} {
	return map[string]interface{}{
		"MaxCorpora": o.MaxCorpora,
		"OutDir":     o.OutDir,
	}
}

func createPipeline(res resources, opts options) analysisPipeline {
	pm := make(pipeline.ParentMap)

	conf := source.LocalFilesDBConfig{
		DBURI:            localFilesDBURI,
		S3Bucket:         localFilesS3Bucket,
		S3Region:         localFilesS3Region,
		UserMachineIndex: source.WestUS2UserMachineIndex,
		MaxRecords:       opts.MaxCorpora,
	}

	localFiles, err := source.NewLocalFilesDB(conf)
	fail(err)

	records := pm.Chain(
		localFiles,
		transform.NewOneInOneOut("build", func(s pipeline.Sample) pipeline.Sample {
			c := s.(sample.Corpus)

			idx, err := index.NewLocalIndex(c, res.rm)
			if err != nil {
				return pipeline.WrapError("error getting file info", err)
			}
			return idx
		}),
		transform.NewMap("select-files", func(s pipeline.Sample) []pipeline.Sample {
			idx := s.(index.LocalIndex)
			var nonLib []sample.FileInfo
			categorized := localfiles.CategorizeLocalFiles(idx.Files)
			for _, fi := range categorized {
				if !fi.IsLibrary {
					nonLib = append(nonLib, fi.FileInfo)
				}
			}
			if len(nonLib) == 0 {
				return []pipeline.Sample{pipeline.NewError("no non-library files")}
			}

			files := selectUpTo(nonLib, maxFilesPerCorpus, idx.Corpus.ID())

			out := make([]pipeline.Sample, 0, len(files))
			for _, f := range files {
				buf, err := idx.Corpus.Get(f.Name)
				if err != nil {
					out = append(out, pipeline.WrapError("cannot get selected file", err))
					continue
				}
				out = append(out, fileRecord{
					LocalIndex: idx,
					File:       f,
					Buffer:     buf,
				})
			}
			return out
		}),
		transform.NewOneInOneOut("resolve", func(s pipeline.Sample) pipeline.Sample {
			fr := s.(fileRecord)

			parseOpts := pythonparser.Options{
				ErrorMode:   pythonparser.Recover,
				Approximate: true,
			}
			ast, words, err := pythonpipeline.Parse(parseOpts, parseTimeout, fr.Buffer)
			if err != nil {
				return pipeline.WrapError("parse error", err)
			}

			rast, err := pythonpipeline.Resolve(res.rm, resolveTimeout, pythonpipeline.Parsed{Mod: ast, Words: words})
			if err != nil {
				return pipeline.WrapError("resolve error", err)
			}

			fr.RAST = rast
			return fr
		}),
		transform.NewMap("get-records", func(s pipeline.Sample) []pipeline.Sample {
			return getCallRecords(res, s.(fileRecord))
		}))

	summaryAgg := aggregator.NewSumAggregator("summary",
		func() sample.Addable {
			return paramSummary{}
		}, func(s pipeline.Sample) sample.Addable {
			return newParamSummary(s.(callRecord))
		})

	pm.Chain(records, summaryAgg)

	var writeAgg pipeline.Aggregator
	if opts.OutDir != "" {
		writeAgg = aggregator.NewJSONWriter(aggregator.DefaultWriterOpts, "write-dir", opts.OutDir)
		pm.Chain(records, writeAgg)
	}

	resFn := func(res map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
		s := res[summaryAgg].(paramSummary)
		percent := func(num, denom int) float64 { return 100. * float64(num) / float64(denom) }
		result := func(name string, value interface{}) rundb.Result {
			return rundb.Result{Aggregator: summaryAgg.Name(), Name: name, Value: value}
		}

		return []rundb.Result{
			result("Count", s.Count),
			result("Resolved %", percent(s.Resolved, s.Count)),
			result("Resolved to Global %", percent(s.ResolvedToGlobal, s.Count)),
			result("Resolved to Local %", percent(s.ResolvedToLocal, s.Count)),
			result("All Name Subtokens Recognized %", percent(s.AllNameSubtokensRecognized, s.Count)),
		}
	}

	return analysisPipeline{
		Pipeline: pipeline.Pipeline{
			Name:      "local-code-analysis",
			Parents:   pm,
			Sources:   []pipeline.Source{localFiles},
			Params:    opts.Params(),
			ResultsFn: resFn,
		},
		WriteAgg: writeAgg,
	}
}

func main() {
	args := struct {
		MaxCorpora int
		OutDir     string
		RunDBPath  string
	}{
		MaxCorpora: 10,
		RunDBPath:  rundb.DefaultRunDB,
	}
	arg.MustParse(&args)

	datadeps.Enable()
	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		log.Fatalln(err)
	}

	mi, err := pythonexpr.NewMetaInfo(pythonmodels.DefaultOptions.ExprModelPath)
	fail(err)

	res := resources{
		rm: rm,
		mi: mi,
	}

	opts := options{
		MaxCorpora: args.MaxCorpora,
		OutDir:     args.OutDir,
	}

	pipe := createPipeline(res, opts)

	eOpts := pipeline.DefaultEngineOptions
	eOpts.NumWorkers = 4
	eOpts.RunDBPath = args.RunDBPath
	engine, err := pipeline.NewEngine(pipe.Pipeline, eOpts)
	fail(err)

	_, err = engine.Run()
	fail(err)
}

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func selectUpTo(files []sample.FileInfo, max int, seedSource string) []sample.FileInfo {
	count := len(files)
	if count > max {
		count = max
	}
	out := make([]sample.FileInfo, 0, count)
	h := fnv.New64()
	h.Write([]byte(seedSource))
	r := rand.New(rand.NewSource(int64(h.Sum64())))
	for _, i := range r.Perm(len(files))[:count] {
		out = append(out, files[i])
	}
	return out
}
