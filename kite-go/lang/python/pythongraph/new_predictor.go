package pythongraph

import (
	"bytes"
	"fmt"
	"go/token"
	"io"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

const (
	beamSize              = 3
	maxHops               = 3
	defaultBeamSearchStep = 15
)

var feedConfig = GraphFeedConfig{
	EdgeSet: []EdgeType{
		ASTChild,
		NextToken,
		DataFlow,
	},
}

var stoppingExpansionTasks = map[ExpansionTask]struct{}{
	lexicalGrammar.CallDone:    struct{}{},
	lexicalGrammar.AttrDone:    struct{}{},
	lexicalGrammar.ExprDone:    struct{}{},
	lexicalGrammar.NameDone:    struct{}{},
	lexicalGrammar.Placeholder: struct{}{},
}

// PredictorNew is a wrapper around the expansion graph and expansion graph updates, that links
// it directly to python.
type PredictorNew struct {
	t      *tracer
	update EgUpdate

	saver *beamSaver

	// for backwards compatibility
	site            pythonast.Node
	scopeSize       int
	numOrigArgs     int
	funcSym         pythonresource.Symbol
	predictorInputs PredictorInputs
	remaining       int
	searcNodeWalker searchNodeWalker
	textReplaced    string
}

// ClosingParenthesisPresent tells if the closing parenthesis is already present or not in the call completion
func (p PredictorNew) ClosingParenthesisPresent() bool {
	if p.searcNodeWalker.callBuilder == nil {
		return false
	}
	return p.searcNodeWalker.callBuilder.closingParenPresent
}

// SetClosingParenthesisPresent tells if the closing parenthesis is already present or not in the call completion
func (p PredictorNew) SetClosingParenthesisPresent(b bool) {
	if p.searcNodeWalker.callBuilder == nil {
		return
	}
	p.searcNodeWalker.callBuilder.closingParenPresent = b
}

// GetLastSymbol gets the last symbol of the searchNode.
func (p PredictorNew) GetLastSymbol() pythonresource.Symbol {
	return p.searcNodeWalker.lastSymbol
}

// PredictedCallBuilder build predicted calls
type PredictedCallBuilder struct {
	calledFunction      pythonresource.Symbol
	nextArg             PredictedCallArg
	args                []PredictedCallArg
	numOrigArgs         int
	prob                float32
	scopeSize           int
	closingParenPresent bool
	nextCommaPresent    bool
}

// NewCallBuilder builds a call builder that builds calls during the walking of the searchNode tree
func NewCallBuilder(function pythonresource.Symbol, numOrigArgs int, initProb float32, scopeSize int, alreadyInCall bool, nextCommaPresent bool) *PredictedCallBuilder {
	return &PredictedCallBuilder{
		calledFunction:      function,
		nextArg:             NewPredictedCallArg(),
		args:                nil,
		numOrigArgs:         numOrigArgs,
		prob:                initProb,
		scopeSize:           scopeSize,
		closingParenPresent: alreadyInCall,
		nextCommaPresent:    nextCommaPresent,
	}

}

func (cb *PredictedCallBuilder) deepCopy() *PredictedCallBuilder {
	var args []PredictedCallArg
	args = append(args, cb.args...)
	return &PredictedCallBuilder{
		calledFunction:      cb.calledFunction,
		nextArg:             cb.nextArg,
		args:                args,
		numOrigArgs:         cb.numOrigArgs,
		prob:                cb.prob,
		scopeSize:           cb.scopeSize,
		closingParenPresent: cb.closingParenPresent,
		nextCommaPresent:    cb.nextCommaPresent,
	}
}

type searchNodeWalker struct {
	callBuilder *PredictedCallBuilder
	prob        float64
	scopeSize   int
	lastSymbol  pythonresource.Symbol
	callDone    *PredictedCall
}

func (snw searchNodeWalker) deepCopy() searchNodeWalker {
	result := searchNodeWalker{
		callBuilder: nil,
		prob:        snw.prob,
		scopeSize:   snw.scopeSize,
		lastSymbol:  snw.lastSymbol,
	}
	if snw.callBuilder != nil {
		result.callBuilder = snw.callBuilder.deepCopy()
	}
	return result
}

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

func nonEmptyArgCount(call *pythonast.CallExpr) int {
	var result int
	for _, a := range call.Args {
		if a.Begin() != a.End() {
			result++
		}
	}
	return result
}

// NewNewPredictor ...
func NewNewPredictor(ctx kitectx.Context, config ContextGraphConfig, in PredictorInputs) (PredictorNew, error) {
	defer buildContextGraphDuration.DeferRecord(time.Now())

	if pythonast.IsNil(in.Site) {
		return PredictorNew{}, errors.Errorf("site is nil")
	}
	config.Propagate = true

	cg, err := NewContextGraph(ctx, config, ContextGraphInputs{
		ModelMeta: in.ModelMeta,
		Model:     in.Model,
		In:        in.In,
		Site:      in.Site,
	})
	if err != nil {
		return PredictorNew{}, errors.Errorf("error building context graph: %v", err)
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
		return PredictorNew{}, err
	}

	var numOrigArgs int
	var funcSym pythonresource.Symbol
	var closingParentPresent bool
	var nextCommaPresent bool
	var textReplaced string
	if call, ok := in.Site.(*pythonast.CallExpr); ok {
		numOrigArgs = nonEmptyArgCount(call)
		if call.RightParen != nil {
			closingParentPresent = true
		}
		if len(call.Commas) >= numOrigArgs {
			nextCommaPresent = true
		}
		if syms := cg.builder.a.ResolveToSymbols(ctx, call.Func); len(syms) > 0 {
			funcSym = syms[0]
		}
	} else if attr, ok := in.Site.(*pythonast.AttributeExpr); ok {
		if attr.Attribute != nil {
			textReplaced = attr.Attribute.Literal
		}
	}
	scopeSize := len(cg.builder.vm.Variables)
	snw := searchNodeWalker{
		callBuilder: nil,
		prob:        1.0,
		scopeSize:   scopeSize,
	}
	if !funcSym.Nil() {
		snw.callBuilder = &PredictedCallBuilder{
			calledFunction:      funcSym,
			nextArg:             NewPredictedCallArg(),
			args:                nil,
			numOrigArgs:         numOrigArgs,
			prob:                1,
			scopeSize:           scopeSize,
			closingParenPresent: closingParentPresent,
			nextCommaPresent:    nextCommaPresent,
		}
	}

	return PredictorNew{
		t: t,
		update: EgUpdate{
			graph: eg,
			meta:  eg.meta,
			prob:  1.,
			stack: stack,
		},
		site:            in.Site,
		saver:           bs,
		scopeSize:       scopeSize,
		numOrigArgs:     numOrigArgs,
		funcSym:         funcSym,
		predictorInputs: in,
		remaining:       defaultBeamSearchStep,
		searcNodeWalker: snw,
		textReplaced:    textReplaced,
	}, nil
}

// NewNewPredictorForExpand ...
func (p PredictorNew) NewNewPredictorForExpand(nextUpdate EgUpdate, remaining int) PredictorNew {
	// We don't copy textReplaced as its is only intended to be used during the first round of Expand
	return PredictorNew{
		t:               p.t,
		update:          nextUpdate,
		site:            p.site,
		saver:           p.saver,
		scopeSize:       p.scopeSize,
		numOrigArgs:     p.numOrigArgs,
		funcSym:         p.funcSym,
		predictorInputs: p.predictorInputs,
		remaining:       remaining,
	}
}

// Prediction ...
type Prediction struct {
	CompTokens    []compToken
	Value         string
	Predictor     *PredictorNew
	PredictedCall *PredictedCall
	Score         float64
	StoppingTask  string
	Symbol        pythonresource.Symbol
}

// Expand the prediction tree based on the current state and return the results along
// with the new predictors.
// Returns:
//   - `predictions` is a slice containing information about the task that was completed along with a value.
//   - `predictors`  is a slice of `PredictorNew`s, calling `Expand` on one of these predictors will conduct another round
//     of search using the returned prediction as the base.
// NOTE:
//   - If there are no expansions or predictions left then `nil,nil,nil` is returned.
//   - We always have that `len(predictions) == len(predictors)`.
//   - Under the hood we may conduct multiple inference passes to ensure that we continue
//     searching until we can return a set of concrete results to the client.
//   - Tt is NOT safe to call this from multiple go routines.
func (p PredictorNew) Expand(ctx kitectx.Context) ([]Prediction, error) {
	defer predictExprDuration.DeferRecord(time.Now())
	root := &searchNode{
		Update: p.update,
		Prob:   1,
	}
	defer predictExprCallDuration.DeferRecord(time.Now())
	err := p.decodePartialExpr(ctx, p.remaining, root, true)
	if err != nil {
		return nil, err
	}

	preds, err := p.searcNodeWalker.processSearchNode(ctx, root, 1, nil, "", "")
	if err != nil {
		return nil, err
	}

	p.saveBeamSearchBundles(root)
	return preds, nil
}

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

	Stop          bool
	DiscardBranch bool

	Children []*searchNode

	// for debugging / saver
	Bundles []SavedBundle

	Logged bytes.Buffer

	DebugStr      string
	nextPredictor *PredictorNew

	Symbol    pythonresource.Symbol
	Value     pythontype.Value
	Remaining int
}

func (cb *PredictedCallBuilder) getCall(partialCall bool) *PredictedCall {
	result := PredictedCall{
		NumOrigArgs: cb.numOrigArgs,
		Args:        cb.args,
		Prob:        float32(cb.prob),
		Symbol:      cb.calledFunction,
		PartialCall: partialCall,
		ScopeSize:   cb.scopeSize,
	}
	if cb.nextArg.Name != "" && cb.nextArg.Prob != 1 {
		result.Args = append(result.Args, cb.nextArg)
		result.Prob *= cb.nextArg.Prob
	}
	return &result
}

type compToken struct {
	Value    string
	Position int    // if not an arg  then -1
	Name     string // if not an arg ""
	Symbol   pythonresource.Symbol
}

func (ct compToken) Nil() bool {
	if ct.Value == "" {
		return true
	}
	return false
}
func (snw searchNodeWalker) processSearchNode(ctx kitectx.Context, s *searchNode, score float64, toks []compToken, completion string, prefix string) ([]Prediction, error) {
	ctx.CheckAbort()
	if s.DiscardBranch {
		return nil, nil
	}
	var callBuilder *PredictedCallBuilder
	if snw.callBuilder != nil {
		callBuilder = snw.callBuilder
	}
	newCompToken := compToken{
		Value:    completionFromWords(s.Words),
		Position: -1,
	}
	completion += completionFromWords(s.Words)

	score *= float64(s.Prob)
	switch s.Task {
	case lexicalGrammar.NameDone, lexicalGrammar.Placeholder, lexicalGrammar.GenericNoPlaceholder,
		lexicalGrammar.KeywordDone, lexicalGrammar.CallDone, lexicalGrammar.ChooseArgType:
		if callBuilder == nil {
			return nil, errors.Errorf("Error, got a %s outside of a call", s.Task.EgClientData())
		}
	}
	if !s.Symbol.Nil() {
		snw.lastSymbol = s.Symbol
	}

	switch s.Task {
	case lexicalGrammar.NameDone, lexicalGrammar.Placeholder:
		nextArg := callBuilder.nextArg
		if nextArg.Prob == 0 {
			nextArg.Prob = 1
		}
		callBuilder.nextArg = NewPredictedCallArg()
		nextArg.Value = s.Words[0].Lit
		nextArg.Prob *= s.Prob
		nextArg.Symbol = s.Symbol

		newCompToken.Position = len(callBuilder.args)
		newCompToken.Name = nextArg.Name
		newCompToken.Symbol = callBuilder.calledFunction
		callBuilder.args = append(callBuilder.args, nextArg)
		callBuilder.prob *= nextArg.Prob
	case lexicalGrammar.ChooseArgType:

		if s.Update.Peek().ChosenProdData() != lexicalGrammar.Stop {
			// We add the parenthesis when we decide to add a new argument
			if (len(callBuilder.args) > 0 || callBuilder.numOrigArgs > 0) && !callBuilder.nextCommaPresent {
				newCompToken.Value = token.COMMA.String() + " "
				completion += newCompToken.Value
			}
			callBuilder.nextCommaPresent = false
			callBuilder.nextArg.Prob *= s.Prob
		}
	case lexicalGrammar.InferName:
		callBuilder.nextArg.Prob *= s.Prob
	case lexicalGrammar.KeywordDone:
		if callBuilder.nextArg.Name != "" {
			return nil, errors.Errorf("New keyword found in prediction with another keyword already registered")
		}
		callBuilder.nextArg.Name = s.Words[0].Lit
		callBuilder.nextArg.Prob *= s.Prob
	case lexicalGrammar.CallDone:
		callBuilder.prob *= s.Prob
		snw.callDone = callBuilder.getCall(false)
		if !callBuilder.closingParenPresent {
			// We add a closing parenthesis token if we were not already in the call
			newCompToken = compToken{
				Value:    token.RPAREN.String(),
				Position: -1,
			}
			completion += newCompToken.Value
		}
		callBuilder = nil
		snw.callBuilder = nil
	case lexicalGrammar.Call:
		callBuilder = NewCallBuilder(snw.lastSymbol, 0, s.Prob, snw.scopeSize, false, false)
		snw.callBuilder = callBuilder
	}

	if !newCompToken.Nil() {
		toks = append([]compToken{}, toks...)
		toks = append(toks, newCompToken)
	}

	if len(s.Children) > 0 {
		var result []Prediction
		for i, c := range s.Children {

			if callBuilder != nil {
				snw.callBuilder = callBuilder.deepCopy()
			}
			newExpr, err := snw.processSearchNode(ctx, c, score, toks, completion, fmt.Sprintf("%s-%d", prefix, i))
			if err != nil {
				return nil, err
			}
			result = append(result, newExpr...)
		}
		return result, nil
	}
	var call *PredictedCall
	if snw.callDone != nil {
		call = snw.callDone
	} else if callBuilder != nil {
		call = callBuilder.getCall(true)
	}

	if s.nextPredictor != nil {
		s.nextPredictor.searcNodeWalker = snw.deepCopy()
	}
	if len(toks) > 0 {
		return []Prediction{{
			CompTokens:    toks,
			Value:         completion,
			Predictor:     s.nextPredictor,
			PredictedCall: call,
			Score:         score,
			StoppingTask:  fmt.Sprintf("TaskClientData: : %s dbgString %s", s.Task.EgClientData(), s.DebugStr),
			Symbol:        snw.lastSymbol,
		}}, nil
	}
	return nil, nil
}

func (p PredictorNew) decodePartialExpr(ctx kitectx.Context, remaining int, bsn *searchNode, noopCompletionInit bool) error {
	if p.saver != nil {
		p.saver.node = bsn
	}

	if remaining == 0 {
		// We don't want to emit any completion that don't end with a Stopping token
		// So we add a child indicating the branch should be discarded
		bsn.Children = append(bsn.Children, &searchNode{
			DiscardBranch: true,
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
		noopCompletion := noopCompletionInit
		if i >= beamSize {
			break
		}
		child := &searchNode{
			Update:    update,
			Prob:      update.Prob(),
			Remaining: remaining,
		}
		bsn.Children = append(bsn.Children, child)

		if head := update.Peek(); head.Type != "" {
			child.DebugStr = head.Client.EgClientData()
			if head.Type == InferNameTaskCompleted {
				child.Words = append(child.Words, word{
					Token: pythonscanner.Ident,
					Lit:   head.Site.Attrs.Literal,
				})
				child.Task = lexicalGrammar.NameDone
				// TODO extract symbol from name (to store if ever it's the name of a function)
				// snw.lastSymbol = pythonresource.Symbol(head.ChosenProdData().(inferProdAttr))
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
				case lexicalGrammar.Attr:
					child.Words = append(child.Words, word{
						Token: pythonscanner.Period,
					})
				case lexicalGrammar.InferAttr:
					child.Symbol = pythonresource.Symbol(head.ChosenProdData().(inferProdAttr))
					child.Words = append(child.Words, word{Token: pythonscanner.Ident,
						Lit: child.Symbol.Path().Last()})
				case lexicalGrammar.Call:
					child.Words = append(child.Words, word{
						Token: pythonscanner.Lparen,
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
		if len(child.Words) > 0 {
			if len(child.Words) != 1 || child.Words[0].String() != p.textReplaced {
				noopCompletion = false
			}
		}
		if _, ok := stoppingExpansionTasks[child.Task]; ok && !noopCompletion {
			pNew := p.NewNewPredictorForExpand(update, remaining)
			child.nextPredictor = &pNew
			continue
		}

		if err := p.decodePartialExpr(ctx, remaining, child, noopCompletion); err != nil {
			return err
		}
	}
	return nil
}

func completionFromWords(words []word) string {
	var parts []string
	for _, word := range words {
		if !word.Token.IsLiteral() {
			if word.Token == pythonscanner.Illegal {
				parts = append(parts, word.Lit)
				continue
			}
			parts = append(parts, word.Token.String())
			continue
		}
		parts = append(parts, word.Lit)
	}
	return strings.Join(parts, "")
}

func (p PredictorNew) saveBeamSearchBundles(node *searchNode) {
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
}
