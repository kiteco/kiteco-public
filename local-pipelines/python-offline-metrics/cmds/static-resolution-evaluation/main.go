package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmetrics"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

var parseOpts = pythonparser.Options{
	ErrorMode:   pythonparser.Recover,
	Approximate: true,
}

const (
	target                      = "ultimate" // ultimate or community
	s3Region                    = "us-west-1"
	s3ProjectPath               = "s3://kite-data/offline-processing/analysis-comparison/" + target + "-projects"
	s3RASTStoragePath           = "s3://kite-data/offline-processing/analysis-comparison/" + target + "-results/AST"
	s3JSONStoragePath           = "s3://kite-data/offline-processing/analysis-comparison/" + target + "-results/json"
	refSamplingRandomSeed       = 1
	refComparisonAggregatorName = "ref-resolution-comparison"
)

func maybeQuit(err error) {
	if err != nil {
		panic(err)
	}
}

func resolveWithContext(importer pythonstatic.Importer, parsed *pythonast.Module, filepath string) (*pythonanalyzer.ResolvedAST, error) {
	importer.Path = filepath
	resolver := pythonanalyzer.NewResolverUsingImporter(importer, pythonanalyzer.Options{Path: filepath})
	var rast *pythonanalyzer.ResolvedAST
	err := kitectx.Background().WithTimeout(maxResolvingInterval, func(ctx kitectx.Context) error {
		var err error
		rast, err = resolver.ResolveContext(ctx, parsed, false)
		return err
	})
	return rast, err
}

func processFileInPipeline(importerFactory func() pythonstatic.Importer, rm pythonresource.Manager, astOutputFolder string, project *projectDescription, maxReferencePerProject int, s3RASTOutputPath string, samplingRate float64) func(pipeline.Sample) []pipeline.Sample {
	return func(k pipeline.Sample) []pipeline.Sample {
		keyed := k.(pipeline.Keyed)
		filename := keyed.Key
		parsedFile := keyed.Sample.(pythonpipeline.Parsed)
		referenceMap, err := project.getReferencesForFile(filename, samplingRate, refSamplingRandomSeed)
		maybeQuit(err)
		importer := importerFactory()
		importer.Path = filename
		rast, err := resolveWithContext(importer, parsedFile.Mod, filename)
		maybeQuit(err)

		refComparisons, nodeMap, err := pythonmetrics.ProcessFileOffline(filename, rast, referenceMap)
		maybeQuit(err)

		if astOutputFolder != "" {
			printEnhancedRAST(project.getSourceFolder(), rast, filename, astOutputFolder, nodeMap)
		}

		if s3RASTOutputPath != "" {
			printEnhancedRAST(project.getSourceFolder(), rast, filename, s3RASTOutputPath, nodeMap)
		}

		result := make([]pipeline.Sample, len(refComparisons))
		for i := range refComparisons {
			result[i] = refComparisons[i]
		}
		return result
	}
}

func buildPipeline(project *projectDescription, rm pythonresource.Manager, outputFile string, importerFactory func() pythonstatic.Importer, astOutputFolder string, runDbOutput bool, maxRefPerProject int, execTimestamp string, samplingRate float64) pipeline.Pipeline {
	filelist, err := source.GetFilelist(project.getSourceFolder(), source.NewFileExtensionPredicate(".py"), true)
	maybeQuit(err)

	var s3RASTOutputPath string
	var s3JSONOutputPath string

	if runDbOutput {
		s3ProjectSuffix := fmt.Sprintf("%s/%s", execTimestamp, project.ProjectName)
		s3RASTOutputPath = fileutil.Join(s3RASTStoragePath, s3ProjectSuffix)
		s3JSONOutputPath = fileutil.Join(s3JSONStoragePath, execTimestamp)
	}

	// This pipeline is not thread safe (we can't resolve multiple files at once)
	// The number of reader has to be 1
	srcs := source.NewLocalFiles("localFileSource", 1, filelist, os.Stdout)
	srcFiltered := transform.NewFilter("src-filtered", func(s pipeline.Sample) bool {
		k := s.(pipeline.Keyed)
		return len(k.Sample.(sample.ByteSlice)) < maxSizeBytes
	})
	parsed := transform.NewOneInOneOutKeyed("parsed", pythonpipeline.ParsedNonNil(parseOpts, maxParseInterval))
	fileProcessor := transform.NewMap("fileProcessor", processFileInPipeline(importerFactory, rm,
		astOutputFolder, project, maxRefPerProject, s3RASTOutputPath, samplingRate))
	maybeQuit(err)
	pm := make(pipeline.ParentMap)
	// refComparator is the main part of the pipeline, processing each file, extracting the refs and comparing them
	// to IntelliJ refs
	refComparator := pm.Chain(
		srcs,
		srcFiltered,
		parsed,
		fileProcessor,
	)

	if outputFile != "" {
		// If the user specifies an output folder, a pseudo-json file is generated with all the ref comparisons
		outf, err := os.Create(outputFile)
		maybeQuit(err)
		writer := dependent.NewJSONWriter("outputWriter", outf)
		pm.Chain(refComparator, writer)
	}

	if runDbOutput {
		opts := aggregator.WriterOpts{
			NumGo:      1,
			Logger:     os.Stderr,
			FilePrefix: project.ProjectName,
		}

		writer := aggregator.NewJSONWriter(opts, "s3JSONWriter", s3JSONOutputPath)
		pm.Chain(refComparator, writer)

		// If the user requires a runDB export, the aggregator is added at the end of the pipeline
		pm.Chain(refComparator, newMapAggregator(refComparisonAggregatorName, project.ProjectName))
	}

	pipe := pipeline.Pipeline{
		Name:    "kite-vs-intelliJ-analysis-comparison",
		Parents: pm,
		Sources: []pipeline.Source{srcs},
	}
	return pipe
}

func runPipeline(pipe pipeline.Pipeline, workerCount int) (map[pipeline.Aggregator]pipeline.Sample, error) {
	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: workerCount,
	})
	if err != nil {
		return nil, err
	}
	return engine.Run()
}

func main() {
	computeMetricFromS3Projects()
}

func computeMetricFromS3Projects() {
	maybeQuit(datadeps.Enable())
	args := struct {
		OutBase           string
		RunDbUpload       bool
		ASTOutputFolder   string
		StatsOutputFolder string
		ProjectList       []string
		MaxRefPerProject  int
	}{
		ASTOutputFolder:   "/home/moe/workspace/AnalysisPython/AST",
		StatsOutputFolder: "/home/moe/workspace/AnalysisPython/stats",
		RunDbUpload:       true,
		MaxRefPerProject:  10000,
	}

	arg.MustParse(&args)
	if args.ASTOutputFolder != "" {
		maybeQuit(os.MkdirAll(args.ASTOutputFolder, 0755))
	}
	if args.StatsOutputFolder != "" {
		maybeQuit(os.MkdirAll(args.StatsOutputFolder, 0755))
	}
	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	maybeQuit(<-errc)

	t := time.Now()
	execTimestamp := t.Format("20060102_150405")
	projectList := fetchS3ProjectList()
	if args.ProjectList != nil && len(args.ProjectList) > 0 {
		projectList = intersection(args.ProjectList, projectList)
	}
	sort.Strings(projectList)
	start := time.Now()
	var resultAggregator sample.Addable
	for _, p := range projectList {
		if args.ProjectList != nil && contains(args.ProjectList, p) == -1 {
			continue
		}
		startProject := time.Now()
		project, err := downloadAndPrepareProject(p)
		maybeQuit(err)
		samplingRate := float64(args.MaxRefPerProject) / float64(project.getReferenceCount())
		if err != nil {
			log.Printf("Error while preparing the project %s : %s\nIt will be skipped\n", p, err)
			continue
		}
		var outputFile string
		if args.StatsOutputFolder != "" {
			outputFile = fileutil.Join(args.StatsOutputFolder, project.ProjectName+".json")
		}
		importerFactory := buildContext(project, rm)
		pipe := buildPipeline(project, rm, outputFile, importerFactory, args.ASTOutputFolder, args.RunDbUpload, args.MaxRefPerProject, execTimestamp, samplingRate)
		// This pipeline is not thread safe (the context used for Resolve context can't be shared
		// Do not change the number of worker (or make deep copies of the context when cloning elements of the pipeline
		result, err := runPipeline(pipe, 1)
		maybeQuit(err)
		for agg, s := range result {
			if agg.Name() == refComparisonAggregatorName {
				addableSample := s.(sample.Addable)
				if resultAggregator == nil {
					resultAggregator = addableSample
				} else {
					resultAggregator.Add(addableSample)
				}
			}
		}
		checkNotFoundReferences(project, false)
		log.Println("Time for project ", project.ProjectName, " : ", time.Since(startProject))
	}
	rundbInstance, err := rundb.NewRunDB(rundb.DefaultRunDB)
	maybeQuit(err)
	if args.RunDbUpload {
		runInfo := rundb.NewRunInfo(rundbInstance, "offline-reference-comparisons-"+target, "Summary")
		runInfo.Results = prepareRundbResults(projectList, resultAggregator.(sample.StatsMap), execTimestamp)
		runInfo.SetStatus(rundb.StatusFinished)
		maybeQuit(rundbInstance.SaveRun(runInfo))
	}
	log.Println("Time for all projects : ", time.Since(start))
	log.Println(resultAggregator)
}

func getPieChartDataFromPercentagesValues(data map[string]ratioValue) []rundb.PieChartData {
	result := make([]rundb.PieChartData, 0, len(data))
	for label, v := range data {
		result = append(result, rundb.PieChartData{Label: label, Value: float64(v.Count)})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Label < result[j].Label })
	return result
}

func getSortedKey(m map[string]ratioValue) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

func prepareRundbResults(projectNames []string, aggregatedResults sample.StatsMap, execTimestamp string) []rundb.Result {
	globalProjectKey := allProjectKey[:len(allProjectKey)-1]
	projectNames = append(projectNames, globalProjectKey)

	results := make([]rundb.Result, 0, len(projectNames))
	appendResult := func(name string, value interface{}) {
		results = append(results, rundb.Result{
			Name:       name,
			Aggregator: "offline-references-comparison",
			Value:      value,
		})
	}

	drawer, err := rundb.NewPieChartDrawer(2)
	maybeQuit(err)
	var projectSizes []rundb.PieChartData
	var chartsString string
	// Allow to use some defer to add data at the end of the result list
	func() {
		for _, name := range projectNames {
			percentageMap := make(map[string]ratioValue)
			var totalCount int64
			for entry, value := range aggregatedResults {
				if strings.HasPrefix(entry, name) {
					totalCount += value.Count
					percentageMap[entry[len(name)+1:]] = ratioValue{Count: value.Count}
				}
			}
			for key, value := range percentageMap {
				value.Ratio = float64(value.Count) / float64(totalCount)
				percentageMap[key] = value
			}
			chartName := fmt.Sprintf("%s (%d ref)", name, totalCount)
			chartString, err := drawer.GetPieChartString(getPieChartDataFromPercentagesValues(percentageMap), 400, 350, chartName)
			maybeQuit(err)
			chartsString += chartString

			percentageMap["reference_count"] = ratioValue{Count: totalCount, Ratio: 1}
			if name == globalProjectKey {
				sortedKey := getSortedKey(percentageMap)

				for _, key := range sortedKey {
					value := percentageMap[key]
					appendResult("Global "+key, fmt.Sprintf("%.2f %% (count %d)", value.Ratio*100, value.Count))
				}
			} else {
				projectSizes = append(projectSizes, rundb.PieChartData{Label: name,
					Value: float64(totalCount)})
			}
			jsonRepr, err := json.MarshalIndent(percentageMap, "", " ")
			maybeQuit(err)
			defer appendResult(name, string(jsonRepr))
		}
		chartString, err := drawer.GetPieChartString(projectSizes, 400, 350, "Project sizes")
		maybeQuit(err)
		chartsString += chartString
		appendResult("Charts", chartsString)
		s3RASTOutputPath := fileutil.Join(s3RASTStoragePath, execTimestamp)
		s3JSONOutputPath := fileutil.Join(s3JSONStoragePath, execTimestamp)
		appendResult("RAST S3 weblink", `<a href='https://s3.console.aws.amazon.com/s3/buckets/`+s3RASTOutputPath[5:]+`/' target='_blank'>RAST dumps on S3</a>`)
		appendResult("JSON S3 weblink", "<a href='https://s3.console.aws.amazon.com/s3/buckets/"+s3JSONOutputPath[5:]+"/' target='_blank'>JSON dumps on S3</a>")
		appendResult("RAST S3 dump", s3RASTOutputPath)
		appendResult("JSON S3 dump", s3JSONOutputPath)
	}()
	return results
}

// checkNotFoundReferences verifies if any of the intelliJ references haven't been match
func checkNotFoundReferences(project *projectDescription, verbose bool) {
	if verbose {
		log.Println("References not found for the project ", project.ProjectName, ": ")
	}
	var counter int
	for _, refs := range project.ReferenceLists {
		if verbose {
			log.Println("For file ", refs.Filename)
		}
		for _, ref := range refs.References {
			if !ref.FoundInKite {
				if verbose {
					log.Println(ref)
				}
				counter++
			}
		}
	}
	log.Println("Ref not found for the project ", project.ProjectName, ": ", counter)
}
