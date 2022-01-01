package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/githubdata"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/performance"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

type evalRecorder struct {
	evaluator     performance.Evaluator
	site          githubdata.PredictionSiteWithMetrics
	metrics       []float64
	ifHasNewWords []float64
}

func newEvaluator(predictor *predict.TFPredictor, search predict.SearchConfig, seed int64, samplingRate float64) performance.Evaluator {
	measurements := map[performance.Measurement]float64{
		performance.TokenValueAdded: samplingRate,
	}

	return performance.Evaluator{
		Predictor:    predictor,
		Encoder:      predictor.GetEncoder(),
		Search:       search,
		Rand:         rand.New(rand.NewSource(seed)),
		RandomSeed:   seed,
		Measurements: measurements,
	}
}

func newEvalRecorder(localSite githubdata.PredictionSiteWithMetrics, evaluator performance.Evaluator) evalRecorder {
	return evalRecorder{
		evaluator: evaluator,
		site:      localSite,
	}
}

func (e *evalRecorder) extractTokens(text string) ([]lexer.Token, error) {
	lexed, err := e.evaluator.Encoder.Lexer.Lex([]byte(text))
	if err != nil {
		return nil, err
	}
	if len(lexed) > 0 {
		lexed = lexed[:len(lexed)-1]
	}
	return lexed, nil
}

func (e *evalRecorder) hasNewWord(newWords []string, window []lexer.Token) float64 {
	for _, t := range window {
		if subtokens, ok := e.evaluator.Encoder.Lexer.ShouldBPEEncode(t); ok {
			for _, s := range subtokens {
				for _, w := range newWords {
					if strings.Contains(s, w) {
						return 1.0
					}
				}
			}
		}
	}
	return 0.0
}

func (e *evalRecorder) eval(ctx string, series string, newEntries []string) error {
	e.evaluator.Rand.Seed(e.evaluator.RandomSeed)
	initial, err := e.extractTokens(ctx)
	if err != nil {
		return err
	}
	window, err := e.extractTokens(series)
	if err != nil {
		return err
	}
	if len(window) >= 5 {
		for i := 0; i <= len(window)-5; i++ {
			var context []lexer.Token
			context = append(context, initial...)
			context = append(context, window[:i]...)
			currentWindow := window[i : i+5]
			if performance.ValidForValueAdded(e.evaluator.Encoder.Lexer, currentWindow) {
				// Make sure at least one valid sample is evaluated for each site
				if len(e.metrics) == 0 || e.evaluator.Rand.Float64() < e.evaluator.Measurements[performance.TokenValueAdded] {
					_, tva, _, err := e.evaluator.ValueAdded(context, currentWindow)
					if err != nil {
						return err
					}
					e.metrics = append(e.metrics, tva.Accurate)
					e.ifHasNewWords = append(e.ifHasNewWords, e.hasNewWord(newEntries, currentWindow))
				}
			}
		}
	}
	return nil
}

func average(nums []float64) float64 {
	var total float64
	for _, num := range nums {
		total += num
	}
	return total / float64(len(nums))
}

func main() {
	args := struct {
		SitesFile      string
		ModelPath      string
		SamplingRate   float64
		RandomSeed     int64
		ReportPath     string
		NewEntriesPath string
		Language       string
	}{
		RandomSeed:   4,
		SamplingRate: 0.2,
	}
	arg.MustParse(&args)
	if args.ModelPath == "" {
		args.ModelPath = lexicalmodels.DefaultModelOptions.TextMiscGroup.ModelPath
	}

	var sites []githubdata.PredictionSiteWithMetrics
	f, err := ioutil.ReadFile(args.SitesFile)
	fail(err)
	fail(json.Unmarshal(f, &sites))

	var newEntries []string
	if args.NewEntriesPath != "" {
		f, err = ioutil.ReadFile(args.NewEntriesPath)
		fail(err)
		fail(json.Unmarshal(f, &newEntries))
	}

	group := lexicalv0.MustLangGroupFromName(args.Language)
	predictor, err := predict.NewTFPredictorFromS3(args.ModelPath, group)
	search, err := predict.NewSearchConfigFromModelPath(args.ModelPath)
	fail(err)

	start := time.Now()

	var m sync.Mutex
	var jobs []workerpool.Job
	var recordersOldContext []evalRecorder
	var recordersNewContext []evalRecorder
	pool := workerpool.New(runtime.NumCPU() - 1)
	for _, site := range sites {
		localSite := site
		jobs = append(jobs, func() error {
			evaluator := newEvaluator(predictor, search, args.RandomSeed, args.SamplingRate)
			e := newEvalRecorder(localSite, evaluator)
			fail(e.eval(localSite.SrcContextBefore, localSite.DstWindow, newEntries))

			m.Lock()
			recordersOldContext = append(recordersOldContext, e)
			m.Unlock()

			evaluator = newEvaluator(predictor, search, args.RandomSeed, args.SamplingRate)
			e = newEvalRecorder(localSite, evaluator)
			fail(e.eval(localSite.DstContextBefore, localSite.DstWindow, newEntries))

			m.Lock()
			recordersNewContext = append(recordersNewContext, e)
			m.Unlock()

			return nil
		})
	}

	pool.AddBlocking(jobs)
	err = pool.Wait()
	fail(err)

	fmt.Printf("took %v to process %d files, aggregating the results...\n", time.Since(start), len(sites))

	// Aggregate and write report
	report, err := os.Create(args.ReportPath)
	fail(err)
	defer report.Close()

	aggregate := func(recorders []evalRecorder, label string) {
		report.WriteString(label + "\n")
		report.WriteString("pr\tpath\ttime\tnum_lines\tnum_evaluation\taverage_metric\tnew_word_percentage\t" +
			"lev\ttb\tta\tib\tia\ttd\tti\tid\tii\n")

		var aggregated []float64
		for _, e := range recorders {
			if len(e.metrics) > 0 {
				ave := average(e.metrics)
				aggregated = append(aggregated, ave)
				report.WriteString(fmt.Sprintf("%d\t%s\t%s\t%d\t%d\t%.5f\t%.3f\t"+
					"%.3f\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
					e.site.PullNumber, e.site.FilePath, e.site.PullTime, e.site.AdditionSize,
					len(e.metrics), ave, average(e.ifHasNewWords), e.site.RelativeLevenshtein,
					e.site.NumTokensBefore, e.site.NumTokensAfter,
					e.site.NumIdentsBefore, e.site.NumIdentsAfter,
					e.site.NumTokensDeletion, e.site.NumTokensInsertion,
					e.site.NumIdentsDeletion, e.site.NumIdentsInsertion))
			}
		}
		summary := fmt.Sprintf("%d/%d sites are feasible, average token value add using %s: %f\n",
			len(aggregated), len(sites), label, average(aggregated),
		)
		fmt.Println(summary)
		report.WriteString(summary)
	}

	aggregate(recordersOldContext, "Old context")
	aggregate(recordersNewContext, "New context")
}
