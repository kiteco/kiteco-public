package main

import (
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"sync"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/status"
)

const (
	// sigSymbolThreshold is used when computing the list of fallback symbols - it determines the number of samples that
	// need to be present for a given symbol to determine whether popularity completions are better than
	// GGNN completions
	sigSymbolThreshold = 10
	// statsRankThreshold determines whether a set of completions is used for adding to the per-symbol stats.
	// If the ranks of both the popularity and GGNN completions are below this threshold, the sample is rejected from
	// the stats.
	statsRankThreshold = 5
)

const (
	// completions in alphabetical order
	alphaLabel = "alpha"
	// completions ranked by popularity stats
	popLabel = "pop"
	// predict on the AttributeExpr, chop off the buffer until the end of the line
	ggnnLineAttrLabel = "ggnn-line-attr"
	// predict on the NameExpr (i.e. chop off the attribute); chop off the buffer until the end of the line
	ggnnLineNameLabel = "gnn-line-name"
	// predict on the attribute, chop off the entire buffer after the AttributeExpr
	ggnnLineAfterLabel = "ggnn-line-after"
)

var (
	topNValues = []int{1, 3, 5, 10}
)

var (
	section         = status.NewSection("python-offline-metrics/ggnn-attribute-completions")
	predictDuration = section.SampleDuration("Prediction duration")
	memUsage        = section.SampleInt64("Memory usage")
)

func maybeQuit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type symbolStat struct {
	Count      int
	GGNNBetter int
	PopBetter  int
}

func (s symbolStat) Add(o symbolStat) symbolStat {
	return symbolStat{
		Count:      s.Count + o.Count,
		GGNNBetter: s.GGNNBetter + o.GGNNBetter,
		PopBetter:  s.PopBetter + o.PopBetter,
	}
}

// symbolStats tracks attribute vs GGNN performance per-symbol.
type symbolStats map[string]symbolStat

func newSymbolStats(comps pythonpipeline.AttributeCompletionsGroup) pipeline.Sample {
	successful := comps.Successful()

	if _, ok := successful[popLabel]; !ok {
		return pipeline.NewError("no pop comps")
	}

	ggnnToUse := ggnnLineNameLabel
	if _, ok := successful[ggnnToUse]; !ok {
		return pipeline.NewError("no ggnn-line-name comps")
	}
	popRank := successful[popLabel].Rank()
	ggnnRank := successful[ggnnToUse].Rank()

	popFound := popRank >= 0
	ggnnFound := ggnnRank >= 0

	if (!popFound || popRank >= statsRankThreshold) && (!ggnnFound || ggnnRank >= statsRankThreshold) {
		return pipeline.WrapError(
			"rank(s) over threshold", fmt.Errorf("pop ran: %d, ggnn rank: %d", popRank, ggnnRank))

	}

	stat := symbolStat{Count: 1}

	switch {
	case popFound && !ggnnFound:
		stat.PopBetter = 1
	case ggnnFound && !popFound:
		stat.GGNNBetter = 1
	case popRank < ggnnRank:
		stat.PopBetter = 1
	case ggnnRank < popRank:
		stat.GGNNBetter = 1
	}

	return symbolStats{
		comps.Situation.Symbol: stat,
	}
}

func (s symbolStats) Add(a sample.Addable) sample.Addable {
	for k, v := range a.(symbolStats) {
		s[k] = s[k].Add(v)
	}
	return s
}

func (symbolStats) SampleTag() {}

// attributeExprs returns a deterministic pseudo-random sampling of up to max AttributeExprs in an analyzed event.
func attributeExprs(s pipeline.Sample, max int) []pipeline.Sample {
	ev := s.(pythonpipeline.AnalyzedEvent)

	var exprs []*pythonast.AttributeExpr

	pythonast.Inspect(ev.Context.AST, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}
		if expr, ok := n.(*pythonast.AttributeExpr); ok {
			exprs = append(exprs, expr)
		}
		return true
	})

	if len(exprs) == 0 {
		return []pipeline.Sample{pipeline.NewError("no AttributeExprs found")}
	}

	count := len(exprs)
	if count > max {
		count = max
	}

	shuffled := make([]*pythonast.AttributeExpr, 0, count)

	h := fnv.New64()
	h.Write([]byte(ev.Event.Buffer))
	r := rand.New(rand.NewSource(int64(h.Sum64())))

	samples := make([]pipeline.Sample, 0, len(shuffled))
	for _, i := range r.Perm(len(exprs))[0:count] {
		samples = append(samples, pythonpipeline.EventExpr{
			AnalyzedEvent: ev,
			Expr:          exprs[i],
		})
	}

	return samples
}

func sortCompletions(s pipeline.Sample) pipeline.Sample {
	comps := s.(pythonpipeline.AttributeCompletions)

	sorted := make([]pythonpipeline.Completion, 0, len(comps.Provided))
	for _, comp := range comps.Provided {
		sorted = append(sorted, comp)
	}

	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Identifier < sorted[j].Identifier })

	newComps := comps
	newComps.Provided = sorted

	return newComps
}

// record is written to JSON for downstream analysis (e.g. via pandas)
type record struct {
	MessageID      string `json:"message_id"`
	Cursor         int64  `json:"cursor"`
	Symbol         string `json:"symbol"`
	ParentExprType string `json:"parent_expr_type"`
	Label          string `json:"label"`
	Expected       string `json:"expected"`
	Count          int    `json:"count"` // number of provided completions
	First          string `json:"first"` // the first completion that was provided
	Index          int    `json:"index"` // index of the expected completion in the provided completions, -1 if not found
	Top1           int    `json:"top_1"`
	Top3           int    `json:"top_3"`
	Top5           int    `json:"top_5"`
	Top10          int    `json:"top_10"`
}

func (record) SampleTag() {}

func analyzeCompletions(comps pythonpipeline.AttributeCompletions, label string) record {
	idx := -1
	for i, a := range comps.Provided {
		if a.Identifier == comps.Situation.Expected {
			idx = i
			break
		}
	}

	var first string
	if len(comps.Provided) > 0 {
		first = comps.Provided[0].Identifier
	}

	toInt := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}

	parentExprType := reflect.TypeOf(comps.Situation.AttrExpr.Value).String()

	rec := record{
		MessageID:      comps.Situation.AnalyzedEvent.Event.Meta.ID.String(),
		Cursor:         int64(comps.Situation.AttrExpr.Dot.End),
		Symbol:         comps.Situation.Symbol,
		ParentExprType: parentExprType,
		Label:          label,
		Expected:       comps.Situation.Expected,
		Count:          len(comps.Provided),
		First:          first,
		Index:          idx,
		Top1:           toInt(comps.InTopN(1)),
		Top3:           toInt(comps.InTopN(3)),
		Top5:           toInt(comps.InTopN(5)),
		Top10:          toInt(comps.InTopN(10)),
	}

	return rec
}

type exampleWriter struct {
	OutDir      string
	NumExamples int

	examples []example.Example
	m        sync.Mutex
}

func (e *exampleWriter) maybeAdd(ex example.Example) {
	e.m.Lock()
	defer e.m.Unlock()

	if len(e.examples) >= e.NumExamples {
		return
	}

	e.examples = append(e.examples, ex)
	if len(e.examples) >= e.NumExamples {
		coll := example.Collection{Examples: e.examples}
		maybeQuit(coll.WriteToDir(e.OutDir))
		log.Printf("wrote %d examples to %s", len(e.examples), e.OutDir)
	}
}

type ggnnPipeline struct {
	pipeline.Pipeline
	globalTopN  pipeline.Aggregator
	symbolStats pipeline.Aggregator
}

type options struct {
	OutDir      string
	MaxEvents   int
	StartDate   analyze.Date
	EndDate     analyze.Date
	ExampleDir  string
	NumExamples int
}

func createPipeline(recreator *servercontext.Recreator, models *pythonmodels.Models, opts options) ggnnPipeline {
	pm := make(pipeline.ParentMap)

	trackOpts := pythonpipeline.DefaultTrackingEventsOpts
	trackOpts.MaxEvents = opts.MaxEvents
	trackOpts.ShardByUMF = true
	trackOpts.NumReaders = 2
	trackOpts.Logger = os.Stdout

	events := pythonpipeline.NewTrackingEvents(
		opts.StartDate, opts.EndDate, pythontracking.ServerSignatureFailureEvent, trackOpts)

	comps := pm.Chain(
		events,
		transform.NewFilter("deduped", pythonpipeline.DedupeEvents()),
		transform.NewOneInOneOut("analyzed", pythonpipeline.AnalyzeEvents(recreator, false)),
		transform.NewMap("exprs", func(s pipeline.Sample) []pipeline.Sample {
			return attributeExprs(s, 5)
		}),
		transform.NewOneInOneOut("attrib-situations", pythonpipeline.ExprAttributeSituations),
		transform.NewFilter("chosen-situations",
			pythonpipeline.AttributeSituationsAllowedByModel(models.Expr)),
		transform.NewOneInOneOut("completions", func(s pipeline.Sample) pipeline.Sample {
			sit := s.(pythonpipeline.AttributeCompSituation)

			popComps := pythonpipeline.PopAttributeCompletions(s)

			start := time.Now()
			nameComps := pythonpipeline.GGNNAttributeCompletions(
				recreator, models, pythonpipeline.TrimLine, true, false)(s)
			if ac, ok := nameComps.(pythonpipeline.AttributeCompletions); ok {
				if len(ac.Provided) > 0 {
					predictDuration.RecordDuration(time.Since(start))
				}
			}

			mc := pythonpipeline.AttributeCompletionsGroup{
				Situation: sit,
				Provided: map[string]pipeline.Sample{
					popLabel:   popComps,
					alphaLabel: sortCompletions(popComps),
					ggnnLineAttrLabel: pythonpipeline.GGNNAttributeCompletions(
						recreator, models, pythonpipeline.TrimLine, false, false)(s),
					ggnnLineNameLabel: nameComps,
					ggnnLineAfterLabel: pythonpipeline.GGNNAttributeCompletions(
						recreator, models, pythonpipeline.TrimAfter, false, false)(s),
				},
			}
			return mc
		}),
	)

	globalTopN := aggregator.NewSumAggregator("global-top-n", func() sample.Addable {
		return make(pythonpipeline.TopNRecallMap)
	}, func(s pipeline.Sample) sample.Addable {
		return s.(pythonpipeline.AttributeCompletionsGroup).TopNRecall(topNValues)
	})

	pm.Chain(comps, globalTopN)

	if opts.ExampleDir != "" {
		maybeQuit(os.MkdirAll(opts.ExampleDir, os.ModePerm))
		ew := exampleWriter{OutDir: opts.ExampleDir, NumExamples: opts.NumExamples}
		pm.Chain(comps,
			dependent.NewFromFunc("maybe-add-example", func(s pipeline.Sample) {
				mc := s.(pythonpipeline.AttributeCompletionsGroup)
				toInclude := []string{popLabel, ggnnLineAttrLabel, ggnnLineNameLabel}
				filtered := mc.Filter(toInclude...)
				if len(filtered.Provided) < len(toInclude) {
					// Don't add the example if any providers failed to give completions
					return
				}
				ew.maybeAdd(filtered.ToExample())
			}))
	}

	split := pm.Chain(
		comps,
		transform.NewMap("split-completions", func(s pipeline.Sample) []pipeline.Sample {
			mc := s.(pythonpipeline.AttributeCompletionsGroup)
			samples := make([]pipeline.Sample, 0, len(mc.Provided))
			for provider := range mc.Provided {
				samples = append(samples, pipeline.Keyed{
					Key:    provider,
					Sample: mc.Provided[provider],
				}.FlattenError())
			}
			return samples
		}))

	if len(opts.OutDir) > 0 {
		maybeQuit(os.MkdirAll(opts.OutDir, os.ModePerm))

		aOpts := aggregator.DefaultWriterOpts
		aOpts.NumGo = 2
		pm.Chain(
			split,
			transform.NewOneInOneOut("to-record", func(s pipeline.Sample) pipeline.Sample {
				keyed := s.(pipeline.Keyed)
				return analyzeCompletions(keyed.Sample.(pythonpipeline.AttributeCompletions), keyed.Key)
			}),
			aggregator.NewJSONWriter(aOpts, "write-dir", opts.OutDir),
		)
	}

	symStats := transform.NewOneInOneOut("symbol-stats", func(s pipeline.Sample) pipeline.Sample {
		comps := s.(pythonpipeline.AttributeCompletionsGroup)
		return newSymbolStats(comps)
	})

	symbolAgg := aggregator.NewSumAggregator("symbol-agg", func() sample.Addable {
		return make(symbolStats)
	}, func(s pipeline.Sample) sample.Addable {
		return s.(symbolStats)
	})

	pm.Chain(comps, symStats, symbolAgg)

	resFn := func(res map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
		var results []rundb.Result

		topNMap := res[globalTopN].(pythonpipeline.TopNRecallMap)

		var labels []string
		for l := range topNMap {
			labels = append(labels, l)
		}
		sort.Strings(labels)

		for _, label := range labels {
			topN := topNMap[label]

			results = append(results, rundb.Result{
				Name:       fmt.Sprintf("%s count", label),
				Aggregator: globalTopN.Name(),
				Value:      topN.Count,
			})

			ns := make([]int, 0, len(topN.TopN))
			for n := range topN.TopN {
				ns = append(ns, n)
			}
			sort.Ints(ns)

			for _, n := range ns {
				var avg float64
				if topN.Count > 0 {
					avg = float64(topN.TopN[n]) / float64(topN.Count)
				}

				results = append(results, rundb.Result{
					Name:       fmt.Sprintf("%s top-%d", label, n),
					Aggregator: globalTopN.Name(),
					Value:      avg,
				})
			}
		}

		return results
	}

	return ggnnPipeline{
		Pipeline: pipeline.Pipeline{
			Name:      "ggnn-attribute-completions-validate",
			Parents:   pm,
			Sources:   []pipeline.Source{events},
			ResultsFn: resFn,
			Params: map[string]interface{}{
				"ExprModel": models.Expr.Dir(),
				"OutDir":    opts.OutDir,
				"MaxEvents": opts.MaxEvents,
				"StartDate": time.Time(opts.StartDate),
				"EndDate":   time.Time(opts.EndDate),
			},
		},
		globalTopN:  globalTopN,
		symbolStats: symbolAgg,
	}
}

func writeFallbackFile(stats symbolStats, path string) error {
	outf, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outf.Close()

	var fallbackSyms []string

	for path, stat := range stats {
		if stat.Count < sigSymbolThreshold {
			continue
		}
		fmt.Printf("stats for sym %s: %+v\n", path, stat)

		if stat.PopBetter > stat.GGNNBetter {
			fallbackSyms = append(fallbackSyms, path)
		}
	}

	sort.Strings(fallbackSyms)

	for _, path := range fallbackSyms {
		if _, err := outf.WriteString(path + "\n"); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	args := struct {
		ExprModel    string
		MaxEvents    int
		OutDir       string
		FallbackPath string

		ExampleDir  string
		NumExamples int

		FeedPath       string
		NumFeedRecords int

		RunDBPath string
		RunName   string
		Role      pipeline.Role
		Port      int
		Endpoints []string
	}{
		ExprModel:      pythonmodels.DefaultOptions.ExprModelShards[0].ModelPath,
		MaxEvents:      5000,
		NumExamples:    1000,
		NumFeedRecords: 100,
		RunDBPath:      rundb.DefaultRunDB,
		Port:           0,
		RunName:        "validate-attrs",
	}
	arg.MustParse(&args)

	modelOpts := pythonmodels.DefaultOptions
	modelOpts.ExprModelShards = pythonexpr.ShardsFromModelPath(args.ExprModel)
	if args.FeedPath != "" {
		log.Printf("will write %d feed records to %s", args.NumFeedRecords, args.FeedPath)
		fw := feedWriter{Filename: args.FeedPath, Count: args.NumFeedRecords}
		modelOpts.ExprModelOpts.TFCallback = fw.Callback
	}

	models, err := pythonmodels.New(modelOpts)
	maybeQuit(err)

	recreator, err := servercontext.NewRecreator(servercontext.DefaultBucketsByRegion)
	maybeQuit(err)

	start := time.Now()

	startDate, err := analyze.ParseDate("2018-10-20")
	maybeQuit(err)

	endDate, err := analyze.ParseDate("2018-12-20")
	maybeQuit(err)

	opts := options{
		OutDir:      args.OutDir,
		MaxEvents:   args.MaxEvents,
		StartDate:   startDate,
		EndDate:     endDate,
		ExampleDir:  args.ExampleDir,
		NumExamples: args.NumExamples,
	}
	pipe := createPipeline(recreator, models, opts)

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
	eOpts.NumWorkers = 2

	engine, err := pipeline.NewEngine(pipe.Pipeline, eOpts)
	maybeQuit(err)

	mr := newMemRecorder(memUsage)

	res, err := engine.Run()
	if err != nil {
		log.Fatalln(err)
	}

	mr.Stop()

	topNMap := res[pipe.globalTopN].(pythonpipeline.TopNRecallMap)

	var labels []string
	for k := range topNMap {
		labels = append(labels, k)
	}
	sort.Strings(labels)

	for _, label := range labels {
		topN := topNMap[label]
		fmt.Printf("%s top-N (count = %d):\n", label, topN.Count)

		ns := make([]int, 0, len(topN.TopN))
		for n := range topN.TopN {
			ns = append(ns, n)
		}
		sort.Ints(ns)
		for _, n := range ns {
			sum := topN.TopN[n]
			avg := float64(sum) / float64(topN.Count)
			fmt.Printf("    %d: %d (%.3f %%)\n", n, sum, avg*100)
		}
	}

	if args.FallbackPath != "" {
		if err := writeFallbackFile(res[pipe.symbolStats].(symbolStats), args.FallbackPath); err != nil {
			log.Fatal(fmt.Errorf("error writing to fallback file: %s", err))
		}
	}

	printSampleInt64s("mem usage", section.Percentiles(), memUsage.Values())
	printSampleDurations("prediction duration", section.Percentiles(), predictDuration.Values())

	fmt.Println("Done! Took", time.Since(start))
}
