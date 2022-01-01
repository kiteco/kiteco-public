package pythongraph

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

/*
Present in the new predictor
const (
	beamSize = 3
)
*/

// Predictor is a wrapper around the expansion graph and expansion graph updates, that links
// it directly to python.
type Predictor struct {
	t      *tracer
	update EgUpdate

	saver *beamSaver

	// for backwards compatibility
	site        pythonast.Node
	scopeSize   int
	numOrigArgs int
	funcSym     pythonresource.Symbol
}

/*
Present in NewPredictor
// PredictorInputs ...
type PredictorInputs struct {
	ModelMeta ModelMeta
	Model     *tensorflow.Model
	In        Inputs

	UseUncompressedModel bool

	// we use this for pruning if it is provided
	Site pythonast.Node

	Tracer io.Writer

	Saver Saver

	Callbacks ExprCallbacks
}
*/

// NewPredictor ...
func NewPredictor(ctx kitectx.Context, config ContextGraphConfig, in PredictorInputs) (Predictor, error) {
	defer buildContextGraphDuration.DeferRecord(time.Now())

	if pythonast.IsNil(in.Site) {
		return Predictor{}, fmt.Errorf("site is nil")
	}
	config.Propagate = true

	cg, err := NewContextGraph(ctx, config, ContextGraphInputs{
		ModelMeta: in.ModelMeta,
		Model:     in.Model,
		In:        in.In,
		Site:      in.Site,
	})
	if err != nil {
		return Predictor{}, fmt.Errorf("error building context graph: %v", err)
	}

	// this is kind of annoying but we cannot declare
	// saver as type *beamSaver because then the interface will have a non nil type
	// but the value will be nil and this causes annoying weirdness in all the places
	// where we use a `Saver != nil` check.
	var saver Saver
	var bs *beamSaver
	if in.Saver != nil {
		root := &searchNode{}
		bs = &beamSaver{root: root, node: root, base: in.Saver}
		saver = bs
	}

	var t *tracer
	if in.Tracer != nil {
		w := in.Tracer
		if bs != nil {
			w = io.MultiWriter(in.Tracer, bs)
		}
		t = &tracer{w: w}
	}

	ld := lexicalDecoder{
		t:      t,
		saver:  saver,
		rm:     in.In.RM,
		cbs:    in.Callbacks,
		meta:   in.ModelMeta,
		buffer: in.In.Buffer,
		ast:    in.In.RAST.Root,
	}

	eg := &ExpansionGraph{
		state: newExpansionGraphState(cg),
		meta: expansionGraphMeta{
			model:                in.Model,
			modelMeta:            in.ModelMeta,
			cb:                   ld,
			tracer:               t,
			saver:                saver,
			buffer:               in.In.Buffer,
			ast:                  in.In.RAST.Root,
			useUncompressedModel: in.UseUncompressedModel,
		},
	}

	// prepare expansion graph for first prediction
	stack, err := ld.PrepareForInference(ctx, cg, eg, in.Site)
	if err != nil {
		return Predictor{}, err
	}

	var numOrigArgs int
	var funcSym pythonresource.Symbol
	if call, ok := in.Site.(*pythonast.CallExpr); ok {
		numOrigArgs = len(call.Args)
		if syms := cg.builder.a.ResolveToSymbols(ctx, call.Func); len(syms) > 0 {
			funcSym = syms[0]
		}
	}

	return Predictor{
		t: t,
		update: EgUpdate{
			graph: eg,
			meta:  eg.meta,
			prob:  1.,
			stack: stack,
		},
		site:        in.Site,
		saver:       bs,
		scopeSize:   len(cg.builder.vm.Variables),
		numOrigArgs: numOrigArgs,
		funcSym:     funcSym,
	}, nil
}

/*
// Prediction ...
type Prediction struct {
	Task  EgTaskType
	Value string
}
*/

// Expand the prediction tree based on the current state and return the results along
// with the new predictors.
// Returns:
//   - `predictions` is a slice containing information about the task that was completed along with a value.
//   - `predictors`  is a slice of `Predictor`s, calling `Expand` on one of these predictors will conduct another round
//     of search using the returned prediction as the base.
// NOTE:
//   - If there are no expansions or predictions left then `nil,nil,nil` is returned.
//   - We always have that `len(predictions) == len(predictors)`.
//   - Under the hood we may conduct multiple inference passes to ensure that we continue
//     searching until we can return a set of concreate results to the client.
//   - Tt is NOT safe to call this from multiple go routines.
func (p Predictor) Expand(ctx kitectx.Context) ([]Prediction, []Predictor, error) {
	panic("not implemented yet")
}

// PredictExpr predicts the complete expression that was sent in via `NewPredictor`,
// this is for backwards compatibility and benchmarking.
func (p Predictor) PredictExpr(ctx kitectx.Context) (*PredictionTreeNode, error) {
	defer predictExprDuration.DeferRecord(time.Now())
	switch p.site.(type) {
	case *pythonast.CallExpr:
		defer predictExprCallDuration.DeferRecord(time.Now())

		root := &searchNode{
			Update: p.update,
			Prob:   1,
		}

		if err := p.decodeCall(ctx, 12, root); err != nil {
			return nil, err
		}

		p.saveBeamSearchBundles(root)

		predictedCalls := p.extractCalls(ctx, root)

		return &PredictionTreeNode{
			Task: predictionTreeRootMarker,
			Prob: 1.,
			Children: []*PredictionTreeNode{{
				Task: InferCallTask,
				Prob: 1.,
				Call: PredictedCallSummary{
					ScopeSize: p.scopeSize,
					Symbol:    p.funcSym,
					Predicted: predictedCalls,
				},
			}},
		}, nil

	case *pythonast.AttributeExpr:
		root := &PredictionTreeNode{
			Task: predictionTreeRootMarker,
			Prob: 1.,
		}
		if err := p.decodeAttr(ctx, root); err != nil {
			return nil, fmt.Errorf("error decoding attribute: %v", err)
		}
		return root, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", p.site)
	}
}

/*
Defined in the new_predictor

type word struct {
	Token pythonscanner.Token
	Lit   string
}

func (w word) String() string {
	if w.Lit == "" {
		return w.Token.String()
	}
	return w.Lit
}

type searchNode struct {
	Update EgUpdate

	Prob float32

	// This is mostly for backwards compatibility
	Task ExpansionTask

	Words []word

	Stop bool

	Children []*searchNode

	// for debugging / saver
	Bundles []SavedBundle

	Logged bytes.Buffer

	DebugStr string
}
*/

func (p Predictor) decodeAttr(ctx kitectx.Context, ptn *PredictionTreeNode) error {
	updates, err := p.update.Expand(ctx)
	if err != nil {
		return err
	}

	for _, update := range updates {
		t := update.Peek()
		ptn.Children = append(ptn.Children, &PredictionTreeNode{
			Task: InferAttrTask,
			Prob: update.Prob(),
			Attr: pythonresource.Symbol(t.ChosenProdData().(inferProdAttr)),
		})
	}

	return nil
}

func (p Predictor) extractCalls(ctx kitectx.Context, s *searchNode) []PredictedCall {
	var recur func(float32, int, []PredictedCallArg, *searchNode) []PredictedCall
	recur = func(prob float32, pl int, prefix []PredictedCallArg, s *searchNode) []PredictedCall {
		ctx.CheckAbort()

		prob += float32(math.Log(float64(s.Prob)))
		switch s.Task {
		case lexicalGrammar.NameDone, lexicalGrammar.Placeholder:
			prefix = append(prefix, PredictedCallArg{
				Value: s.Words[0].Lit,
				Prob:  s.Prob,
			})
			pl++
		case lexicalGrammar.KeywordDone:
			prefix = append(prefix, PredictedCallArg{
				Name: s.Words[0].Lit,
			})
		}

		var calls []PredictedCall
		for _, c := range s.Children {
			cpy := append([]PredictedCallArg{}, prefix...)
			calls = append(calls, recur(prob, pl, cpy, c)...)
		}

		if len(s.Children) == 0 {
			// we have hit the end of a path, so we can append
			// the call
			calls = append(calls, PredictedCall{
				Prob: float32(math.Exp(float64(prob / float32(pl)))),
				Args: prefix,
			})
		}

		return calls
	}

	calls := recur(0, 1, nil, s)
	for i, call := range calls {
		// clean up keyword args by merging them
		var newArgs []PredictedCallArg
		for j := 0; j < len(call.Args); {
			arg := call.Args[j]
			if arg.Name != "" {
				if j+1 < len(call.Args) {
					arg.Value = call.Args[j+1].Value
				} else {
					// hit the depth limit so we just make the value a placeholder
					arg.Value = PlaceholderPlaceholder
				}
				j += 2
			} else {
				j++
			}
			newArgs = append(newArgs, arg)
		}
		call.Args = newArgs

		if len(call.Args) == 0 || !call.Args[len(call.Args)-1].Stop {
			call.Args = append(call.Args, PredictedCallArg{
				Stop: true,
				Prob: 1,
			})
		}
		calls[i] = call
	}

	var sum float32
	for _, call := range calls {
		sum += call.Prob
	}

	for i := range calls {
		calls[i].Prob /= sum
	}

	sort.Slice(calls, func(i, j int) bool {
		return calls[i].Prob > calls[j].Prob
	})

	// Lastly we need to dedupe the calls becuase
	// calls in which we hit the depth limit have their search stopped.
	// If the search had continued then they would be unique but since
	// the search did not continue we can get duplicate calls.
	// We just dedupe these here and since the calls
	// the calls are sorted a simple iteration + map lookup works
	var newCalls []PredictedCall
	seen := make(map[string]bool)
	for _, call := range calls {
		s := call.id()
		if seen[s] {
			continue
		}
		seen[s] = true
		call.NumOrigArgs = p.numOrigArgs
		newCalls = append(newCalls, call)
	}

	return newCalls
}

func (p Predictor) decodeCall(ctx kitectx.Context, remaining int, bsn *searchNode) error {
	if p.saver != nil {
		p.saver.node = bsn
	}

	if remaining == 0 {
		bsn.Children = append(bsn.Children, &searchNode{
			Prob:     1,
			Stop:     true,
			DebugStr: "<MaxDepth>",
		})
		return nil
	}
	if bsn.Update.Peek().Type != NoInferTask {
		remaining--
	}

	updates, err := bsn.Update.Expand(ctx)
	if err != nil {
		return err
	}

	for i, update := range updates {
		if i >= 3 {
			break
		}

		child := &searchNode{
			Update: update,
			Prob:   update.Prob(),
		}

		if head := update.Peek(); head.Type != "" {
			child.DebugStr = head.Client.EgClientData()
			if head.Type == InferNameTaskCompleted {
				child.Words = append(child.Words, word{
					Token: pythonscanner.Ident,
					Lit:   head.Site.Attrs.Literal,
				})
				child.Task = lexicalGrammar.NameDone
			} else {
				if head.Type.Completed() && len(head.InferProdClient) > 0 {
					child.DebugStr = head.ChosenProdData().EgClientData()
				}

				child.Task = head.Client.(ExpansionTask)
				switch child.Task {
				case lexicalGrammar.KeywordDone:
					child.Words = append(child.Words,
						word{
							Token: pythonscanner.Ident,
							Lit:   head.Site.Attrs.Literal,
						},
						word{
							Token: pythonscanner.Assign,
						},
					)
				case lexicalGrammar.ArgDone:
					child.Words = append(child.Words, word{
						Token: pythonscanner.Comma,
					})
				case lexicalGrammar.Attr:
					child.Words = append(child.Words, word{
						Token: pythonscanner.Period,
					})
				case lexicalGrammar.Call:
					child.Words = append(child.Words, word{
						Token: pythonscanner.Lparen,
					})
				case lexicalGrammar.CallDone:
					child.Words = append(child.Words, word{
						Token: pythonscanner.Rparen,
					})
				case lexicalGrammar.ChooseTerminalType:
					if head.ChosenProdData().(ExpansionTask) == lexicalGrammar.Placeholder {
						// this just makes decoding easier
						child.Task = lexicalGrammar.Placeholder
						child.Words = append(child.Words, word{
							Lit: PlaceholderPlaceholder,
						})
					}
				}
			}
		} else {
			child.DebugStr = "<Stop>"
		}

		bsn.Children = append(bsn.Children, child)
		if err := p.decodeCall(ctx, remaining, child); err != nil {
			return err
		}
	}

	return nil
}

func (p Predictor) saveBeamSearchBundles(node *searchNode) {
	if p.saver == nil {
		return
	}

	// make saved bundle for root
	sb := SavedBundle{
		Label:   "<Context>",
		Entries: p.saver.root.Bundles,
		Children: []SavedBundle{{
			Prob:  1.,
			Label: "<Root>",
		}},
		Prob:   1.,
		Buffer: p.saver.root.Logged.Bytes(),
	}

	var recur func(*SavedBundle, *searchNode)
	recur = func(sb *SavedBundle, sn *searchNode) {
		sb.Entries = sn.Bundles

		for _, child := range sn.Children {
			sbb := &SavedBundle{
				Prob:   child.Prob,
				Label:  child.DebugStr,
				Buffer: child.Logged.Bytes(),
			}

			if len(child.Words) > 0 {
				var parts []string
				for _, w := range child.Words {
					parts = append(parts, w.String())
				}
				sbb.Label += " :: " + strings.Join(parts, "")
			}

			recur(sbb, child)
			sb.Children = append(sb.Children, *sbb)
		}
	}

	recur(&sb.Children[0], node)

	p.saver.base.Save(sb)
}

/*
Present in the new predictor
type beamSaver struct {
	base Saver
	root *searchNode
	node *searchNode
}

func (s *beamSaver) Save(b SavedBundle) {
	s.node.Bundles = append(s.node.Bundles, b)
}

func (s *beamSaver) Write(buf []byte) (int, error) {
	s.node.Logged.Write(buf)

	return len(buf), nil
}*/
