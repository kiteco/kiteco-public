package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"

	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"

	"github.com/kiteco/kiteco/kite-golib/complete/data"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

// Args ...
type Args struct {
	repoTarget string
}

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

var parseOpts = pythonparser.Options{
	ErrorMode:   pythonparser.Recover,
	Approximate: true,
}

const maxUnionPerFile = 0
const maxSizeBytes = 1000000
const maxParseInterval = 1 * time.Second

func maybeQuit(err error) {
	if err != nil {
		panic(err)
	}
}

// PairComparison ...
type PairComparison struct {
	beforeRestriction int
	afterRestriction  int
}

// SampleTag implements pipeline.Sample
func (PairComparison) SampleTag() {}

// PairComparisonPair ...
type PairComparisonPair struct {
	size PairComparison
	rank PairComparison
}

// SampleTag ...
func (PairComparisonPair) SampleTag() {}

func pairSelector(selectRank bool) func(s pipeline.Sample) pipeline.Sample {
	return func(s pipeline.Sample) pipeline.Sample {
		pp := s.(PairComparisonPair)
		if selectRank {
			return pp.rank
		}
		return pp.size
	}
}

type rastAndUnion struct {
	rast  resolvedWithBuffer
	union *pythonast.AttributeExpr
}

type pairCompAggregation map[int]map[int]int

func (pairCompAggregation) SampleTag() {}

func (pca pairCompAggregation) aggregateInplace(other pairCompAggregation) {
	for k, m := range other {
		inmap, ok := pca[k]
		if !ok {
			inmap = make(map[int]int)
			pca[k] = inmap
		}
		for k2, v := range m {
			inmap[k2] = inmap[k2] + v
		}
	}
}

func (pca pairCompAggregation) addPair(p PairComparison) {
	if _, ok := pca[p.beforeRestriction]; !ok {
		pca[p.beforeRestriction] = make(map[int]int)
	}
	pca[p.beforeRestriction][p.afterRestriction]++
}

func (pca pairCompAggregation) Add(other sample.Addable) sample.Addable {
	o := other.(pairCompAggregation)
	result := make(pairCompAggregation)
	result.aggregateInplace(pca)
	result.aggregateInplace(o)
	return result
}

func newPairCompAgg() sample.Addable {
	return make(pairCompAggregation)
}

func convertSample(s pipeline.Sample) sample.Addable {
	pair := s.(PairComparison)
	pca := newPairCompAgg().(pairCompAggregation)
	pca.addPair(pair)
	return pca
}

func newPairAggregator(name string) pipeline.Aggregator {
	return aggregator.NewSumAggregator(name, newPairCompAgg, convertSample)
}

func (rastAndUnion) SampleTag() {}

func compareUnionSize(s pipeline.Sample) []pipeline.Sample {
	resolved := s.(resolvedWithBuffer)
	results := make([]pipeline.Sample, 0)

	for expr, val := range resolved.RAST.References {
		if _, ok := expr.(*pythonast.NameExpr); !ok {
			// We only consider union associated with NameExpr
			continue
		}
		if u, ok := val.(pythontype.Union); ok {
			afterVal := resolved.RAST.RefinedValue(expr)
			comp := PairComparison{
				beforeRestriction: len(u.Constituents),
				afterRestriction:  1,
			}
			if u2, ok := afterVal.(pythontype.Union); ok {
				comp.afterRestriction = len(u2.Constituents)
			}
			results = append(results, comp)
		}
	}
	return results
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

func selectExpressions(s pipeline.Sample) []pipeline.Sample {
	resolved := s.(resolvedWithBuffer)
	var expressions []*pythonast.AttributeExpr
	pythonast.Inspect(resolved.RAST.Root, func(node pythonast.Node) bool {
		if node == nil {
			return false
		}
		att, ok := node.(*pythonast.AttributeExpr)
		if !ok {
			return true
		}
		val := resolved.RAST.References[att.Value]
		if _, ok := att.Value.(*pythonast.NameExpr); !ok {
			// We don't want attributes expression where the base is not a name
			return true
		}
		if _, ok := val.(pythontype.Union); ok {
			expressions = append(expressions, att)
		}
		return true
	})
	if len(expressions) > maxUnionPerFile && maxUnionPerFile > 0 {
		rand.Shuffle(len(expressions), func(i, j int) { expressions[i], expressions[j] = expressions[j], expressions[i] })
		expressions = expressions[:maxUnionPerFile]
	}
	result := make([]pipeline.Sample, 0, len(expressions))
	for _, e := range expressions {
		result = append(result, rastAndUnion{
			rast:  resolved,
			union: e,
		})
	}
	return result
}

func findLastUnion(s pipeline.Sample) pipeline.Sample {
	resolved := s.(resolvedWithBuffer)
	var maxRank int
	var attr *pythonast.AttributeExpr
	for exp, rank := range resolved.RAST.Order {
		if attExp, ok := exp.(*pythonast.AttributeExpr); ok && rank > maxRank {
			attr = attExp
			maxRank = rank
		}
	}
	if attr == nil {
		return pipeline.NewError("Can't find the last union, all hope is lost!")
	}
	attrVal, ok := resolved.RAST.References[attr.Value]
	if !ok {
		return pipeline.NewError("Can't find the last union in the reference array")
	}
	if _, ok := attrVal.(pythontype.Union); !ok {
		return pipeline.NewError("The value of the last attribute expr is not a union, all hope is lost")
	}

	return rastAndUnion{
		rast:  resolved,
		union: attr,
	}
}

func compareLastUnionSize(s pipeline.Sample) pipeline.Sample {
	ru := s.(rastAndUnion)
	attrVal := ru.rast.RAST.References[ru.union.Value]
	u1 := attrVal.(pythontype.Union)
	v2 := ru.rast.RAST.RefinedValue(ru.union.Value)
	result := PairComparison{
		beforeRestriction: len(u1.Constituents),
		afterRestriction:  1,
	}
	if u2, ok := v2.(pythontype.Union); ok {
		result.afterRestriction = len(u2.Constituents)
	}
	return result
}

func compareLastUnionCompletions(rm pythonresource.Manager, models *pythonmodels.Models) func(s pipeline.Sample) pipeline.Sample {
	return func(s pipeline.Sample) pipeline.Sample {

		attrName := s.(pipeline.Keyed).Key
		ru := s.(pipeline.Keyed).Sample.(rastAndUnion)
		g := pythonproviders.Global{
			ResourceManager: rm,
			Models:          models,
			UserID:          18,
			MachineID:       "My pink laptop with sparkles everywhere",
			FilePath:        "/test.py",
			LocalIndex:      nil,
			Product:         licensing.Pro,
		}
		buf := data.NewBuffer(string(ru.rast.Buffer)).Select(data.NewSelection(ru.union.Dot.End, ru.union.Dot.End))
		inputs, err := pythonproviders.NewInputs(kitectx.TODO(), g, buf, false, false)
		if err != nil {
			return pipeline.CoerceError(err)
		}
		var completionsBefore, completionsAfter []pythonproviders.MetaCompletion

		err = pythonproviders.Attributes{UseDefaultReferences: true}.Provide(kitectx.Background(), g, inputs, func(ctx kitectx.Context, sb data.SelectedBuffer, mc pythonproviders.MetaCompletion) {
			completionsBefore = append(completionsBefore, mc)
		})
		if err != nil {
			return pipeline.CoerceError(err)
		}

		sort.Slice(completionsBefore, func(i, j int) bool {
			return completionsBefore[i].Score > completionsBefore[j].Score
		})
		indexBefore := -1
		for i, c := range completionsBefore {
			if c.Snippet.Text == attrName {
				indexBefore = i
				break
			}
		}

		err = pythonproviders.Attributes{}.Provide(kitectx.Background(), g, inputs, func(ctx kitectx.Context, sb data.SelectedBuffer, mc pythonproviders.MetaCompletion) {
			completionsAfter = append(completionsAfter, mc)
		})
		if err != nil {
			return pipeline.CoerceError(err)
		}

		sort.Slice(completionsAfter, func(i, j int) bool {
			return completionsAfter[i].Score > completionsAfter[j].Score
		})

		indexAfter := -1
		for i, c := range completionsAfter {
			if c.Snippet.Text == attrName {
				indexAfter = i
				break
			}
		}
		//printCompletions(completionsBefore, completionsAfter, attrName)
		result := PairComparisonPair{
			size: PairComparison{
				beforeRestriction: len(completionsBefore),
				afterRestriction:  len(completionsAfter),
			},
			rank: PairComparison{
				beforeRestriction: indexBefore,
				afterRestriction:  indexAfter,
			},
		}
		return result
	}
}

func printCompletions(compBefore, compAfter []pythonproviders.MetaCompletion, target string) {
	fmt.Println("Target : ", target)
	fmt.Println("Completion before restriction : ")
	for i, c := range compBefore {
		targetHit := ""
		if c.Snippet.Text == target {
			targetHit = "***** TARGET *****"
		}
		fmt.Printf("%d : %s   %s\n", i, c.Snippet.Text, targetHit)
	}
	fmt.Println("Completion after restriction : ")
	for i, c := range compBefore {
		targetHit := ""
		if c.Snippet.Text == target {
			targetHit = "***** TARGET *****"
		}
		fmt.Printf("%d : %s   %s\n", i, c.Snippet.Text, targetHit)
	}
}

func truncate(s pipeline.Sample) pipeline.Sample {
	ru := s.(rastAndUnion)
	newText := ru.rast.Buffer[:ru.union.Dot.End]

	return pipeline.Keyed{
		Key:    ru.union.Attribute.Literal,
		Sample: sample.ByteSlice(newText),
	}
}

func buildPipeline(repoTarget string, seed int64, numReader int, logger io.Writer) (pipeline.Pipeline, func()) {

	randGen := rand.New(rand.NewSource(seed))

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	maybeQuit(<-errc)

	models, err := pythonmodels.New(pythonmodels.DefaultOptions)
	maybeQuit(err)

	pm := make(pipeline.ParentMap)

	source, cleanFunc := source.NewGitHubRepo(repoTarget, "master", "", "py", randGen, numReader, 0, logger)
	keyRemover := func(name string) *transform.OneInOneOut {
		return transform.NewOneInOneOut(name, func(s pipeline.Sample) pipeline.Sample {
			return s.(pipeline.Keyed).Sample
		})
	}
	srcFiltered := transform.NewFilter("src-filtered", func(s pipeline.Sample) bool {
		return len(s.(sample.ByteSlice)) < maxSizeBytes
	})
	parsed := transform.NewOneInOneOut("parsed", parseFile)
	resolved := transform.NewOneInOneOut("resolved", resolveFile(rm))
	resolveChain := pm.Chain(source, keyRemover("keyRemover I"), srcFiltered, parsed, resolved)

	unionSizeComparator := transform.NewMap("size comparator", compareUnionSize)
	sizeCompAgg := newPairAggregator("size comparison")
	pm.Chain(resolveChain, unionSizeComparator, sizeCompAgg)

	extracted := transform.NewMap("extracted", selectExpressions)
	truncator := transform.NewOneInOneOut("truncator", truncate)
	parsed2 := transform.NewOneInOneOutKeyed("parsed again", parseFile)
	resolved2 := transform.NewOneInOneOutKeyed("resolved again", resolveFile(rm))
	lastUnionFinder := transform.NewOneInOneOutKeyed("last union finder", findLastUnion)
	truncAndResolve := pm.Chain(resolveChain, extracted, truncator, parsed2, resolved2, lastUnionFinder)

	lastUnionSizeComp := transform.NewOneInOneOutKeyed("last union size comparator", compareLastUnionSize)
	lastUnionSizeAgg := newPairAggregator("Last union size (truncated src)")
	pm.Chain(truncAndResolve, lastUnionSizeComp, keyRemover("keyRemover II"), lastUnionSizeAgg)

	lastUnionCompComp := transform.NewOneInOneOut("last union completion comparator", compareLastUnionCompletions(rm, models))
	completionComparator := pm.Chain(truncAndResolve, lastUnionCompComp)
	rankSelect := transform.NewOneInOneOut("Rank selector", pairSelector(true))
	sizeSelect := transform.NewOneInOneOut("Size selector", pairSelector(false))
	lastUnionRankAgg := newPairAggregator("completion rank comparator (trunc src)")
	lastUnionCompSizeAgg := newPairAggregator("completion list size comparator (trunc src)")
	pm.Chain(completionComparator, rankSelect, lastUnionRankAgg)
	pm.Chain(completionComparator, sizeSelect, lastUnionCompSizeAgg)
	return pipeline.Pipeline{
		Name:      "union-restriction-evaluation",
		Parents:   pm,
		Sources:   []pipeline.Source{source},
		ResultsFn: compileResults(repoTarget),
	}, cleanFunc
}

func runPipeline(pipe pipeline.Pipeline, repoName string) error {
	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: 10,
		RunDBPath:  rundb.DefaultRunDB,
		RunName:    repoName,
	})
	if err != nil {
		return err
	}
	_, err = engine.Run()
	return err
}

func main() {
	//repo := "git@github.com:kiteco/etl.git"
	repo := "git@github.com:home-assistant/home-assistant.git"
	//repo := "git@github.com:Miserlou/Zappa.git"
	maybeQuit(datadeps.Enable())
	pipe, cleanFunc := buildPipeline(repo, 1, 2, os.Stderr)
	defer cleanFunc()
	maybeQuit(runPipeline(pipe, repo))
}
