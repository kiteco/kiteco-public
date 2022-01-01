package pythonpipeline

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

// TrimMethod defines how to trim the buffer before performing model-powered completions.
type TrimMethod string

const (
	// TrimAttribute just replaces the attribute with attrPlaceholder
	TrimAttribute TrimMethod = "attribute"
	// TrimLine trims the buffer from the replaced attribute to the end of the current line, or to the end of the
	// deepest statement, whichever is nearer to the cursor.
	TrimLine TrimMethod = "line"
	// TrimAfter trims all of the buffer after the replaced attribute.
	TrimAfter TrimMethod = "after"
)

// attrPlaceholder is the attribute name used to replace the real attribute for attribute prediction.
const attrPlaceholder = "GUESS_ME_123"

// AttributeCompSituation describes a situation in which the user is typing a partial attribute.
type AttributeCompSituation struct {
	AnalyzedEvent AnalyzedEvent

	AttrExpr *pythonast.AttributeExpr
	Expected string
	Symbol   string

	TypedPrefix string
}

// SampleTag implements Sample
func (AttributeCompSituation) SampleTag() {}

// ExprAttributeSituations builds attribute prediction situations from viable EventExpr samples
func ExprAttributeSituations(s pipeline.Sample) pipeline.Sample {
	expr := s.(EventExpr)

	attr, ok := expr.Expr.(*pythonast.AttributeExpr)
	if !ok {
		return pipeline.NewError("expr is not an AttributeExpr")
	}

	// we want at least a few characters to have been typed
	attrLen := len(attr.Attribute.Literal)
	if attrLen == 0 {
		return pipeline.NewError("attribute literal is empty")
	}

	parentVal := expr.AnalyzedEvent.Context.Resolved.References[attr.Value]
	sym, err := python.GetExternalSymbol(kitectx.Background(), expr.AnalyzedEvent.Context.Importer.Global, parentVal)
	if err != nil {
		return pipeline.NewError("parent val is not external")
	}

	return AttributeCompSituation{
		AnalyzedEvent: expr.AnalyzedEvent,
		AttrExpr:      attr,
		Expected:      attr.Attribute.Literal,
		Symbol:        sym.PathString(),
		TypedPrefix:   "", // assume the prediction problem is after the attribute dot
	}
}

// Completion along with its score
type Completion struct {
	Identifier string
	Score      float64
}

// AttributeCompletions represents a list of ranked completions along with the expected completion, as well as some
// metadata.
type AttributeCompletions struct {
	Situation    AttributeCompSituation
	Provided     []Completion
	MungedBuffer string
}

// SampleTag implements pipeline.Sample
func (AttributeCompletions) SampleTag() {}

// Rank returns the rank of the expected completion within the provided ones, returning -1 if it was not found.
func (a AttributeCompletions) Rank() int {
	for i, comp := range a.Provided {
		if comp.Identifier == a.Situation.Expected {
			return i
		}
	}
	return -1
}

// InTopN returns true if the completions are within top-N.
func (a AttributeCompletions) InTopN(n int) bool {
	rank := a.Rank()
	return rank >= 0 && rank < n
}

// TopNRecall for the given values of N
func (a AttributeCompletions) TopNRecall(ns []int) TopNRecall {
	topN := make(map[int]int, len(ns))
	for _, n := range ns {
		var count int
		if a.InTopN(n) {
			count = 1
		}
		topN[n] = count
	}

	return TopNRecall{
		Count: 1,
		TopN:  topN,
	}
}

// ToProvided returns an example.Provided struct representing the completions that can be put into an example.Example.
func (a AttributeCompletions) ToProvided() example.Provided {
	cs := make([]example.Completion, 0, len(a.Provided))
	for _, p := range a.Provided {
		cs = append(cs, example.Completion{
			Identifier: p.Identifier,
			Score:      p.Score,
		})
	}
	return example.Provided{
		Completions:  cs,
		MungedBuffer: a.MungedBuffer,
	}
}

// AttributeCompletionsGroup represents completions for a situation given by different providers.
type AttributeCompletionsGroup struct {
	Situation AttributeCompSituation
	// Provided is a map of provider name => either an AttributeCompletions struct or an error
	Provided map[string]pipeline.Sample
}

// SampleTag implements pipeline.Sample
func (AttributeCompletionsGroup) SampleTag() {}

// Successful returns the successfully provided completions
func (a AttributeCompletionsGroup) Successful() map[string]AttributeCompletions {
	ret := make(map[string]AttributeCompletions)
	for k, v := range a.Provided {
		if ac, ok := v.(AttributeCompletions); ok {
			ret[k] = ac
		}
	}
	return ret
}

// Filter filters the providers to the ones given.
func (a AttributeCompletionsGroup) Filter(providers ...string) AttributeCompletionsGroup {
	provided := make(map[string]pipeline.Sample)
	for _, p := range providers {
		if s, ok := a.Provided[p]; ok {
			provided[p] = s
		}
	}

	return AttributeCompletionsGroup{
		Situation: a.Situation,
		Provided:  provided,
	}
}

// ToExample creates an example from the given set of completions.
func (a AttributeCompletionsGroup) ToExample() example.Example {
	provided := make(map[string]example.Provided, len(a.Provided))
	for k, comps := range a.Successful() {
		provided[k] = comps.ToProvided()
	}

	return example.Example{
		Buffer:   a.Situation.AnalyzedEvent.Event.Buffer,
		Cursor:   int64(a.Situation.AttrExpr.Dot.End),
		Symbol:   a.Situation.Symbol,
		Expected: a.Situation.Expected,
		Provided: provided,
	}
}

// TopNRecall for each provider for the given values of N
func (a AttributeCompletionsGroup) TopNRecall(ns []int) TopNRecallMap {
	m := make(TopNRecallMap)
	for p, comps := range a.Successful() {
		m[p] = comps.TopNRecall(ns)
	}
	return m
}

// AttributeSituationsAllowedByModel filters attribute situations to ones for which:
// - the Value is a NameExpr
// - the Value of the attribute expr resolves to a type supported by the attribute model
func AttributeSituationsAllowedByModel(model pythonexpr.Model) transform.IncludeFn {
	return func(s pipeline.Sample) bool {
		situation := s.(AttributeCompSituation)

		parentVal := situation.AnalyzedEvent.Context.Resolved.References[situation.AttrExpr.Value]
		if parentVal == nil {
			return false
		}

		rm := situation.AnalyzedEvent.Context.Importer.Global
		sym, err := python.GetExternalSymbol(kitectx.Background(), rm, parentVal)
		if err != nil {
			return false
		}

		if err := model.AttrSupported(rm, sym); err != nil {
			return false
		}

		return true
	}
}

// PopAttributeCompletions takes as input a AttributeCompSituation
// and returns a AttributeCompletions sample representing the popularity-based attribute completions for the sample
// and the expected completion.
func PopAttributeCompletions(s pipeline.Sample) pipeline.Sample {
	sit := s.(AttributeCompSituation)

	var comps []Completion

	global := pythonproviders.Global{
		ResourceManager: sit.AnalyzedEvent.Context.Importer.Global,
		Models:          pythonmodels.Mock(),
		LocalIndex:      sit.AnalyzedEvent.Context.LocalIndex,
		Product:         licensing.Pro,
	}
	inps, err := pythonproviders.NewInputsFromPyCtx(kitectx.Background(), global, data.NewBuffer(string(sit.AnalyzedEvent.Context.Buffer)).Select(data.Cursor(int(sit.AnalyzedEvent.Context.Cursor))), false, sit.AnalyzedEvent.Context, false)
	if err != nil {
		panic(err)
	}
	pythonproviders.Attributes{UseDefaultReferences: true}.Provide(kitectx.Background(), global, inps,
		func(_ kitectx.Context, _ data.SelectedBuffer, c pythonproviders.MetaCompletion) {
			comps = append(comps, Completion{
				Identifier: c.Snippet.Text,
				Score:      c.Score,
			})
		})

	sort.Slice(comps, func(i, j int) bool {
		return comps[i].Score > comps[j].Score
	})

	return AttributeCompletions{
		Situation: sit,
		Provided:  comps,
	}
}

// GGNNAttributeCompletions returns attribute-model-powered completions for attribute situations, or an error
func GGNNAttributeCompletions(recreator *servercontext.Recreator, models *pythonmodels.Models, tm TrimMethod, inferOnName bool, localIndex bool) transform.OneInOneOutFn {
	return func(s pipeline.Sample) pipeline.Sample {
		sit := s.(AttributeCompSituation)

		attrExpr := sit.AttrExpr

		var parentName *pythonast.NameExpr
		if inferOnName {
			n, ok := sit.AttrExpr.Value.(*pythonast.NameExpr)
			if !ok {
				return pipeline.NewError("AttrExpr.Value is not a NameExpr")
			}
			parentName = n
		}

		// Munge the buffer in various ways, and then re-analyze in order to not leak type information.
		// All of these methods replace the attribute of the AttributeExpr with attrPlaceholder as well.
		src := sit.AnalyzedEvent.Event.Event.Buffer

		trimStart := sit.AttrExpr.Dot.End
		placeholder := attrPlaceholder
		if inferOnName {
			trimStart = parentName.End()
			placeholder = ""
		}

		var newSrc string
		switch tm {
		case TrimAttribute:
			newSrc = src[:trimStart] + placeholder + src[sit.AttrExpr.End():]
		case TrimLine:
			lm := linenumber.NewMap([]byte(src))
			line := lm.Line(int(trimStart))
			_, trimEnd := lm.LineBounds(line)
			// If the deepest statement spans multiple lines, trim to its end
			stmt := sit.AnalyzedEvent.Context.Resolved.ParentStmts[attrExpr]
			if pythonast.IsNil(stmt) {
				return pipeline.NewError("parent stmt is nil")
			}
			if int(stmt.End()) > trimEnd {
				trimEnd = int(stmt.End())
			}
			newSrc = src[:trimStart] + placeholder + src[trimEnd:]
		case TrimAfter:
			newSrc = src[:trimStart] + placeholder
		default:
			panic(fmt.Errorf("unrecognized trim method: %s", tm))
		}

		mutated := sit.AnalyzedEvent.Event.Event
		mutated.Buffer = newSrc
		mutated.Offset = int64(trimStart)

		newCtx, err := recreator.RecreateContext(&mutated, localIndex)
		if err != nil {
			return pipeline.WrapError("error recreating context", err)
		}

		// Find the expression in the new AST that we want to infer on
		var newExpr pythonast.Expr
		pythonast.Inspect(newCtx.AST, func(n pythonast.Node) bool {
			if inferOnName {
				if n, ok := n.(*pythonast.NameExpr); ok {
					if n.Begin() == parentName.Begin() {
						if n.Ident.Literal == parentName.Ident.Literal {
							newExpr = n
						}
					}
				}
			} else {
				if n, ok := n.(*pythonast.AttributeExpr); ok {
					if n.Dot.Begin == attrExpr.Dot.Begin && n.Attribute.Literal == attrPlaceholder {
						newExpr = n
					}
				}
			}
			return true
		})

		if pythonast.IsNil(newExpr) {
			return pipeline.NewError("can't find expr in new AST")
		}

		in := pythonexpr.Input{
			RM:                  newCtx.Importer.Global,
			RAST:                newCtx.Resolved,
			Words:               newCtx.IncrLexer.Words(),
			Src:                 []byte(newSrc),
			Expr:                newExpr,
			MungeBufferForAttrs: true,
			Depth:               1,
		}

		var tree *pythongraph.PredictionTreeNode
		err = kitectx.Background().WithTimeout(3*time.Second, func(ctx kitectx.Context) error {
			ggnn, err := models.Expr.Predict(ctx, in)
			tree = ggnn.OldPredictorResult
			return err
		})
		if err != nil {
			log.Println(err)
			return pipeline.WrapError("expr model failure", err)
		}

		var preds []Completion
		probs := []float64{1.}

		pythongraph.Inspect(tree, func(n *pythongraph.PredictionTreeNode) bool {
			lastProb := len(probs) - 1
			if n == nil {
				probs = probs[:lastProb]
				return false
			}
			prob := probs[lastProb] * float64(n.Prob)
			probs = append(probs, prob)

			switch {
			case !n.Attr.Nil():
				preds = append(preds, Completion{
					Identifier: n.Attr.Path().Last(),
					Score:      prob,
				})
			}
			return true
		})

		sort.Slice(preds, func(i, j int) bool {
			return preds[i].Score > preds[j].Score
		})

		sb := data.NewBuffer(string(newCtx.Buffer)).Select(data.Cursor(int(newCtx.Cursor)))
		var filtered []Completion
		for _, p := range preds {
			_, valid := data.Completion{
				Snippet: data.NewSnippet(p.Identifier),
				Replace: data.Selection{Begin: int(newExpr.Begin()), End: int(newCtx.Cursor)},
			}.Validate(sb)
			if valid {
				filtered = append(filtered, p)
			}
		}

		return AttributeCompletions{
			Situation:    sit,
			Provided:     filtered,
			MungedBuffer: newSrc,
		}
	}
}

// TopNRecall is an aggregation of top-N stats. It contains a total count of samples as well as counts of samples that
// are in top-N for different values of N.
type TopNRecall struct {
	Count int
	TopN  map[int]int
}

// Add together two TopNRecalls (mutates original)
func (t TopNRecall) Add(other TopNRecall) TopNRecall {
	t.Count += other.Count
	if t.TopN == nil {
		t.TopN = make(map[int]int)
	}
	for n, c := range other.TopN {
		t.TopN[n] += c
	}
	return t
}

// TopNRecallMap containing top-n recall for a set of providers
type TopNRecallMap map[string]TopNRecall

// SampleTag implements pipeline.Sample
func (TopNRecallMap) SampleTag() {}

// Add implements sample.Addable
func (t TopNRecallMap) Add(other sample.Addable) sample.Addable {
	for k, v := range other.(TopNRecallMap) {
		t[k] = t[k].Add(v)
	}
	return t
}
