package pythongraph

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"sync/atomic"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
)

// ArgPlaceholderTrainSample describes a training sample for predicting value of a keyword argument
type ArgPlaceholderTrainSample struct {
	Feed        GraphFeed       `json:"feed"`
	NameEncoder NameEncoderFeed `json:"name_encoder"`
	NameModel   NameModelFeed   `json:"name_model"`
}

type placeholderSite struct {
	call     *pythonast.CallExpr
	argument *pythonast.Argument
	position int
	node     *Node
}

// ArgPlaceholderTrainInputs groups the inputs required to
// compute training sample for keyword argument value completion.
type ArgPlaceholderTrainInputs struct {
	Hash string
	Inputs
	Symbol pythonresource.Symbol
}

const (
	minClassRatio = .45
	maxClassRatio = 1 - minClassRatio
)

var (
	positiveSampleCounter, negativeSampleCounter, failureCount int64
)

// NewArgPlaceholderTrainSample generate a new sample for the ArgPlaceholder task training
// It will try to balance positive and negative samples over all symbols based on the min/max class ratio constants
func NewArgPlaceholderTrainSample(config TrainConfig, params TrainParams, in ArgPlaceholderTrainInputs) (*InferProductionSample, error) {
	a := newAnalysis(in.RM, in.Words, in.RAST)
	builder1 := newBuilder(kitectx.Background(), a, false, true)
	// There are multiple exit point for failure and only one for success
	// So we always increment failure and decrement it only in case of success (cf end of the function)
	atomic.AddInt64(&failureCount, 1)

	builder1.BuildEdges(config.Graph.EdgeSet)
	// canonicalize symbol
	sym := in.Symbol.Canonical()
	site1, positiveSample := builder1.randomArgPlaceholderSite(kitectx.Background(), params.Rand, sym)

	if site1 == (placeholderSite{}) {
		return nil, fmt.Errorf("placeholder site not found")
	}

	save(
		params.Saver,
		SavedBundle{
			Label:      "original",
			builder:    builder1,
			Buffer:     in.Buffer,
			NodeLabels: nodeLabels(site1.node, "site"),
		},
	)

	lm := linenumber.NewMap(in.Buffer)
	trimEnd := trimEndLineOrStmt(site1.call.End(), site1.call, lm, in.RAST.ParentStmts)

	// munge to get placeholder
	newSrc := bytes.Join([][]byte{
		in.Buffer[:site1.argument.Begin()],
		[]byte(traindata.InferArgPlaceholderMarker),
		[]byte(")"),
		in.Buffer[trimEnd:],
	}, nil)

	save(params.Saver, bufferBundle("munged-no-graph", newSrc))

	a2, err := analyze(kitectx.Background(), in.RM, newSrc)
	if err != nil {
		return nil, err
	}

	builder2 := newBuilder(kitectx.Background(), a2, false, true)
	builder2.BuildEdges(config.Graph.EdgeSet)

	site2 := builder2.findArgPlaceholderAgain()

	if site2 == (placeholderSite{}) {
		return nil, fmt.Errorf("couldn't find arg type site again")
	}

	// Make sure the target argument is of the same position of the same call
	if site2.call.Begin() != site1.call.Begin() || site1.position != site2.position {
		return nil, fmt.Errorf("site mismatch")
	}

	save(
		params.Saver,
		SavedBundle{
			Label:      "munged",
			builder:    builder2,
			NodeLabels: nodeLabels(site2.node, "site"),
			Buffer:     newSrc,
		},
	)

	scope := builder2.ScopeForCall(kitectx.TODO(), site2.call)
	if len(scope) == 0 {
		return nil, fmt.Errorf("no variables in scope")
	}
	builder2.vm.ReduceTo(builder2.a.RAST.Root, scope)

	scopeNodes := builder2.AddScopeNodeAndEdges(scope)
	contextNodes := builder2.ContextTokens(site2.argument)
	modifyArgPlaceholderGraphWithScope(builder2, config.MaxHops, site2, scope, in.RM.SigStats(sym), contextNodes)
	site2.node.Attrs.Types = []string{traindata.InferArgPlaceholderMarker}
	save(
		params.Saver,
		SavedBundle{
			Label:      "pruned",
			builder:    builder2,
			NodeLabels: nodeLabels(site2.node, "site"),
			Buffer:     newSrc,
		},
	)

	if len(builder2.nodes) > maxNumNodesTrainGraph {
		return nil, fmt.Errorf("too many nodes in graph got %d > %d", len(builder2.nodes), maxNumNodesTrainGraph)
	}
	var label int
	if positiveSample {
		label = 1
	}

	var argName string
	if name, ok := site2.argument.Name.(*pythonast.NameExpr); ok {
		argName = name.Ident.Literal
	} else {
		sigStats := in.RM.SigStats(sym)
		if site2.position < len(sigStats.Positional) {
			argName = sigStats.Positional[site2.position].Name
		} else {
			return nil, fmt.Errorf("VarArgs are not supported yet for argPlacheolder task")
		}
	}

	if err != nil {
		return nil, fmt.Errorf("error getting production decoders: %v", err)
	}

	eg, cgToEG := builder2.ExpansionGraph(params.ModelMeta, []*Node{site2.node}, joinNodes(scopeNodes, contextNodes))

	sample := InferProductionSample{
		ContextGraph:   builder2.newGraphFeed(params.ModelMeta),
		ExpansionGraph: eg,
		Production: ProductionModelFeed{
			PredictionNodes: []int32{int32(site2.node.ID)},
			ScopeEncoder:    newNodeIDFeed(scopeNodes, cgToEG),
			ContextTokens:   newNodeIDFeed(contextNodes, cgToEG),
			Labels:          []int{label},
			Corrupted:       traindata.NewSegmentedIndicesFeed(1 - int32(label)),
		},
	}

	symStr := in.Symbol.Canonical().PathString()
	targets := []pythonimports.Hash{
		traindata.IDForChooseArgPlaceholder(symStr, argName, traindata.NoPlaceholder),
		traindata.IDForChooseArgPlaceholder(symStr, argName, traindata.Placeholder),
	}

	for _, t := range targets {
		tid, ok := params.ProductionIndex.Index(t)
		if !ok {
			return nil, fmt.Errorf("no decoder target found for %s", t)
		}
		sample.Production.DecoderTargets.Indices = append(sample.Production.DecoderTargets.Indices, tid)
		sample.Production.DecoderTargets.SampleIDs = append(sample.Production.DecoderTargets.SampleIDs, 0)
	}
	if positiveSample {
		atomic.AddInt64(&positiveSampleCounter, 1)
	} else {
		atomic.AddInt64(&negativeSampleCounter, 1)
	}
	atomic.AddInt64(&failureCount, -1)

	return &sample, nil
}

func (b *graphBuilder) randomArgPlaceholderSite(ctx kitectx.Context, rand *rand.Rand, sym pythonresource.Symbol) (placeholderSite, bool) {
	ctx.CheckAbort()
	var negativeSites, positiveSites []placeholderSite

	pythonast.Inspect(b.a.RAST.Root, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}

		call, ok := node.(*pythonast.CallExpr)
		if !ok {
			return true
		}

		fn := b.astNodes[call.Func]
		if !fn.matchesType(sym, true) {
			return true
		}

		for i, arg := range call.Args {
			// if arg is a complex expression then this site is qualified for the placeholder.
			site := placeholderSite{
				call:     call,
				argument: arg,
				node:     b.astNodes[arg],
				position: i,
			}
			if _, ok := arg.Value.(*pythonast.NameExpr); !ok {
				positiveSites = append(positiveSites, site)
			} else {
				negativeSites = append(negativeSites, site)
			}
		}

		return true
	})

	if len(positiveSites)+len(negativeSites) == 0 {
		return placeholderSite{}, true
	}

	psc := atomic.LoadInt64(&positiveSampleCounter)
	nsc := atomic.LoadInt64(&negativeSampleCounter)

	totalCounter := psc + nsc
	positiveRatio := 0.5
	if totalCounter > 0 {
		positiveRatio = float64(psc) / float64(totalCounter)
	}

	if positiveRatio > minClassRatio && positiveRatio < maxClassRatio {
		idx := rand.Intn(len(positiveSites) + len(negativeSites))
		if idx < len(positiveSites) {
			return positiveSites[idx], true
		}
		return negativeSites[idx-len(positiveSites)], false

	} else if positiveRatio >= maxClassRatio {
		if len(negativeSites) == 0 {
			return placeholderSite{}, false
		}
		return negativeSites[rand.Intn(len(negativeSites))], false
	}

	if len(positiveSites) == 0 {
		return placeholderSite{}, true
	}
	return positiveSites[rand.Intn(len(positiveSites))], true
}

func (b *graphBuilder) findArgPlaceholderAgain() placeholderSite {
	for ast := range b.astNodes {
		call, ok := ast.(*pythonast.CallExpr)
		if !ok {
			continue
		}
		for i, arg := range call.Args {
			value, ok := arg.Value.(*pythonast.NameExpr)
			if !ok || value.Ident.Literal != traindata.InferArgPlaceholderMarker {
				continue
			}

			return placeholderSite{
				argument: arg,
				node:     b.astNodes[arg],
				position: i,
				call:     call,
			}
		}
	}

	return placeholderSite{}
}

func threePopularTypes(pts map[pythonimports.Hash]pythonresource.SigStatTypeInfo) []string {
	var tcs []pythonresource.SigStatTypeInfo
	for _, typeInfo := range pts {
		tcs = append(tcs, typeInfo)
	}
	// sort type based on count
	sort.Slice(tcs, func(i, j int) bool {
		return tcs[i].Count > tcs[j].Count
	})

	if len(tcs) > 3 {
		tcs = tcs[:3]
	}

	var res []string
	for _, tc := range tcs {
		res = append(res, tc.Path)
	}
	return res
}

func modifyArgPlaceholderGraphWithScope(builder *graphBuilder, hops int, site placeholderSite, scope scope, stats *pythonresource.SigStats, contextNodes []*Node) {
	node := site.node
	node.Attrs.Literal = traindata.InferArgPlaceholderMarker
	node.Attrs.Types = []string{traindata.InferArgPlaceholderMarker}
	if name, ok := site.argument.Name.(*pythonast.NameExpr); ok {
		pts := stats.ArgsByName[name.Ident.Literal].Types
		types := threePopularTypes(pts)
		if len(types) > 0 {
			node.Attrs.Types = types
		}
	} else {
		if site.position < len(stats.Positional) {
			pts := stats.Positional[site.position].Types
			types := threePopularTypes(pts)
			if len(types) > 0 {
				node.Attrs.Types = types
			}
		}
	}

	if hops > 0 {
		keep := nodeSet(map[*Node]bool{
			node: true,
		})
		for _, cn := range contextNodes {
			keep[cn] = true
		}

		for _, v := range scope {
			for _, ref := range v.Refs.Names() {
				n := builder.astNodes[ref]
				keep[n] = true
			}
		}
		builder.Prune(kitectx.TODO(), keep, hops)
	}
}
