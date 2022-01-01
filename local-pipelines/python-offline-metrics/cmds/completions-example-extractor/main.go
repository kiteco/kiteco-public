package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/kiteco/kiteco/kite-golib/segment/analyze"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"

	"github.com/kiteco/kiteco/kite-go/lang/python"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-golib/kitectx"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

const (
	maxSizeBytes     = 10000
	maxParseInterval = 1 * time.Second
	outDir           = "/data/kite/mixing/"
)

var (
	datasetPath           = pythoncode.DedupedCodeDumpPath
	maxFilesToScan        = int(1e5)
	examplesPerFile       = 1
	getExampleFromSegment = true
	parseOpts             = pythonparser.Options{
		ErrorMode:   pythonparser.Recover,
		Approximate: true,
	}
)

type parsedWithBuffer struct {
	Mod    *pythonast.Module
	Words  []pythonscanner.Word
	Buffer []byte
}

func (parsedWithBuffer) SampleTag() {}

type resolvedWithBuffer struct {
	Mod    *pythonast.Module
	RAST   *pythonanalyzer.ResolvedAST
	Words  []pythonscanner.Word
	Buffer []byte
}

func (resolvedWithBuffer) SampleTag() {}

type codeExample struct {
	Buffer string
	Cursor int64

	// Symbol of the relevant value on which we want to do completions
	Symbol string

	// Expected identifier for this example
	Expected string
	// Provided is a map of model name to the information provided by the model for the example
	Provided map[string]example.Provided
}

func (codeExample) SampleTag() {}

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type exampleData struct {
	node pythonast.Node
	desc string
	call pythonast.Node
}

func selectExpressions(rm pythonresource.Manager, models *pythonmodels.Models) func(s pipeline.Sample) []pipeline.Sample {
	return func(s pipeline.Sample) []pipeline.Sample {
		resolved := s.(pipeline.Keyed).Sample.(resolvedWithBuffer)
		var expressions []exampleData
		pythonast.Inspect(resolved.Mod, func(node pythonast.Node) bool {
			call, ok := node.(*pythonast.CallExpr)
			if !ok {
				return true
			}

			val := resolved.RAST.References[call.Func]
			symb, err := python.GetExternalSymbol(kitectx.Background(), rm, val)
			if err != nil || models.Expr.CallSupported(rm, symb) != nil {
				return true
			}

			for i, arg := range call.Args {
				if !pythonast.IsNil(arg.Name) {
					desc := fmt.Sprintf("[NAME] %d/%d", i, len(call.Args))
					expressions = append(expressions, exampleData{arg.Name, desc, call})
				}
				desc := fmt.Sprintf("[POS] %d/%d", i, len(call.Args))
				expressions = append(expressions, exampleData{arg.Value, desc, call})
			}
			return true
		})
		result := make([]pipeline.Sample, 0, examplesPerFile)
		if len(expressions) < examplesPerFile {
			for _, ex := range expressions {
				result = append(result, NewCodeExample(ex, resolved))
			}
		} else {
			for _, i := range rand.Perm(len(expressions))[:examplesPerFile] {
				result = append(result, NewCodeExample(expressions[i], resolved))
			}
		}
		return result
	}
}

// NewCodeExample build a CodeExample struct from exampleData and return it as a pipeline.Sample
func NewCodeExample(ex exampleData, resolved resolvedWithBuffer) pipeline.Sample {
	return codeExample{
		Buffer:   string(resolved.Buffer),
		Cursor:   int64(ex.node.Begin()),
		Expected: string(resolved.Buffer[ex.node.Begin():ex.call.End()]),
		Symbol:   ex.desc,
	}
}

func parseFile(s pipeline.Sample) pipeline.Sample {
	if len(s.(sample.ByteSlice)) > maxSizeBytes {
		return nil
	}
	parsedSampleRaw := pythonpipeline.ParsedNonNil(parseOpts, maxParseInterval)(s)
	if parsedSampleRaw == nil {
		return nil
	}
	parsedSample := parsedSampleRaw.(pythonpipeline.Parsed)

	return parsedWithBuffer{
		Mod:    parsedSample.Mod,
		Words:  parsedSample.Words,
		Buffer: s.(sample.ByteSlice),
	}
}

func resolveFile(rm pythonresource.Manager) func(s pipeline.Sample) pipeline.Sample {
	return func(s pipeline.Sample) pipeline.Sample {
		inputData := s.(parsedWithBuffer)

		var rast *pythonanalyzer.ResolvedAST
		err := kitectx.Background().WithTimeout(maxParseInterval, func(ctx kitectx.Context) error {
			var err error
			rast, err = pythonanalyzer.NewResolver(rm, pythonanalyzer.Options{
				Path: "/src.py",
			}).ResolveContext(ctx, inputData.Mod, false)
			return err
		})

		if err != nil {
			return nil
		}

		return resolvedWithBuffer{
			Mod:    inputData.Mod,
			Words:  inputData.Words,
			Buffer: inputData.Buffer,
			RAST:   rast,
		}
	}
}

func getGithubCrawlSource() (pipeline.Feed, pipeline.Source) {
	keys, err := aggregator.ListDir(datasetPath)
	maybeQuit(err)
	sourceOpts := source.DefaultEMRDatasetOpts
	sourceOpts.MaxRecords = maxFilesToScan
	result := source.NewEMRDataset("dedupe_files", sourceOpts, keys)
	return result, result
}

func getSegmentSampleSource(pm pipeline.ParentMap) (pipeline.Feed, pipeline.Source) {
	compEvents := pythonpipeline.NewTrackingEvents(
		analyze.NewDate(2018, 12, 01),
		analyze.NewDate(2019, 04, 15),
		pythontracking.ServerCompletionsFailureEvent,
		pythonpipeline.TrackingEventsOpts{
			NumReaders: 8,
		})

	return pm.Chain(compEvents,
		transform.NewFilter("compSampleEvents", func(s pipeline.Sample) bool {
			ev := s.(pythonpipeline.Event)
			return ev.Completions.Failure == pythontracking.CompletionsSample
		}),
		transform.NewFilter("onePerFile", pythonpipeline.DedupeEvents()),
		transform.NewOneInOneOut("SegmentToByteArray", func(s pipeline.Sample) pipeline.Sample {
			ev := s.(pythonpipeline.Event)
			return pipeline.Keyed{Key: ev.Filename,
				Sample: sample.ByteSlice(ev.Buffer)}
		})), compEvents
}

func buildPipeline() pipeline.Pipeline {
	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	maybeQuit(<-errc)

	models, err := pythonmodels.New(pythonmodels.DefaultOptions)
	maybeQuit(err)

	pm := make(pipeline.ParentMap)

	var exampleFeed pipeline.Feed
	var exampleSource pipeline.Source
	filePrefix := "examples"
	if getExampleFromSegment {
		exampleFeed, exampleSource = getSegmentSampleSource(pm)
		filePrefix += "_segment"
	} else {
		exampleFeed, exampleSource = getGithubCrawlSource()
		filePrefix += "_github"
	}

	parsed := transform.NewOneInOneOutKeyed("parsed", parseFile)
	resolved := transform.NewOneInOneOutKeyed("resolved", resolveFile(rm))
	extracted := transform.NewMap("extracted", selectExpressions(rm, models))
	writer := aggregator.NewJSONWriter(aggregator.WriterOpts{
		FilePrefix: filePrefix,
	}, "json_writer", outDir)

	pm.Chain(exampleFeed, parsed, resolved, extracted, writer)

	pipe := pipeline.Pipeline{
		Name:    "call-example-extractor",
		Parents: pm,
		Sources: []pipeline.Source{exampleSource},
	}
	return pipe
}

func runPipeline(pipe pipeline.Pipeline) error {
	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: 10,
	})
	if err != nil {
		return err
	}
	_, err = engine.Run()
	return err
}

func main() {
	maybeQuit(datadeps.Enable())
	pipe := buildPipeline()
	maybeQuit(runPipeline(pipe))
}
