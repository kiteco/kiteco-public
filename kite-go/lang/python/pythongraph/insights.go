package pythongraph

import (
	"fmt"
	"math"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

type insightsTask string

const (
	inferProdInsights = "infer_prod_insights"
	inferNameInsights = "infer_name_insights"
	graphInsights     = "graph_insights"
)

type insightsType string

const (
	finalPropagationStep insightsType = "final_propagation_step"
	allPropagationSteps  insightsType = "all_propagation_steps"
)

const (
	initNodeStatesOp = "test/expansion_graph/build_state/init_node_states"

	// # [num decoder targets, 1]
	scopeWeightOp = "test/infer_production/build_context/scope_weight_1"
	siteWeightOp  = "test/infer_production/build_context/site_weight"
	tokenWeightOp = "test/infer_production/build_context/token_weight"

	// [num decoder targets, context depth]
	scopeStateOp = "test/infer_production/build_context/scope_state_expanded"
	siteStateOp  = "test/infer_production/site_state/site_state_expanded"
	tokenStateOp = "test/infer_production/token_state/token_state_final_expanded"

	targetsEmbeddedOp           = "test/infer_production/embed_decoder_targets/targets_embedded"
	targetsEmbeddedOpCompressed = "test/infer_production/embed_decoder_targets/targets_embedded/lookup/SparseTensorDenseMatMul"

	inferProdLogitsOp = "test/infer_production/prediction/logits"
	inferNameLogitsOp = "test/infer_name/prediction/logits"
)

// EdgeValue contains the raw and normalized values for the edge
type EdgeValue struct {
	Raw        float64 `json:"raw"`
	Normalized float64 `json:"normalized"`
}

// ContextWeights collects weights in the prediction context
type ContextWeights struct {
	// The label of Content[a][b] corresponds to YLabels[a] and XLabels[b]
	XLabels []string    `json:"xlabels"`
	YLabels []string    `json:"ylabels"`
	Content [][]float32 `json:"content"`
}

func magnitude(messages [][]float32) []float32 {
	mags := make([]float32, 0, len(messages))
	for _, emb := range messages {
		mag := math.Sqrt(float64(innerProduct(emb, emb)) / float64(len(emb)))
		mags = append(mags, float32(mag))
	}
	return mags
}

// EdgeIDStr produces an id string for edge
func edgeIDStr(edgeType EdgeType, from int32, to int32) string {
	return fmt.Sprintf("%s:%d:%d", edgeType, from, to)
}

// EdgeToIDStr produces an id string from *Edge
func EdgeToIDStr(edge *SavedEdge) string {
	return edgeIDStr(edge.Type, int32(edge.From.Node.ID), int32(edge.To.Node.ID))
}

func normalize(nums map[string]float64) map[string]float64 {
	var max float64
	for _, n := range nums {
		if math.Abs(n) > max {
			max = math.Abs(n)
		}
	}

	normalized := make(map[string]float64)
	for k := range nums {
		normalized[k] = nums[k]
		if max > 1e-7 {
			normalized[k] /= max
		}
	}
	return normalized
}

type insightBuilderParams struct {
	MaxHops              int
	Model                *tensorflow.Model
	Graph                *SavedGraph
	Edges                map[string][][2]int32
	Feed                 map[string]interface{}
	Labels               NodeLabels
	UseUncompressedModel bool
}

type insightBuilder struct {
	params insightBuilderParams

	res                map[string]interface{}
	nodeStatesAfterOps []string
	attentionOps       []string
	messagesOps        []string
}

func newInsightBuilder(params insightBuilderParams, task insightsTask) insightBuilder {
	ib := insightBuilder{
		params: params,
	}

	ops := []string{initNodeStatesOp}
	switch task {
	case inferProdInsights:

		ops = append(ops,
			scopeWeightOp,
			siteWeightOp,
			tokenWeightOp,
			scopeStateOp,
			siteStateOp,
			tokenStateOp,
			inferProdLogitsOp,
			ib.inferProdTargetsOp(),
		)
	case inferNameInsights:
		ops = append(ops, inferNameLogitsOp)
	default:
	}

	for i := 0; i < params.MaxHops; i++ {
		ib.nodeStatesAfterOps = append(ib.nodeStatesAfterOps,
			fmt.Sprintf("test/expansion_graph/graph/propagate_%d/updated_states_%d", i, i))
		ib.messagesOps = append(ib.messagesOps,
			fmt.Sprintf("test/expansion_graph/graph/propagate_%d/messages", i))

		ib.attentionOps = append(ib.attentionOps, fmt.Sprintf("test/expansion_graph/graph/propagate_%d/attentions", i))
	}

	ops = append(ops, ib.nodeStatesAfterOps...)
	ops = append(ops, ib.attentionOps...)
	ops = append(ops, ib.messagesOps...)

	res, err := params.Model.Run(params.Feed, ops)
	if err != nil {
		panic(fmt.Sprintf("error building insights: %v", err))
	}

	ib.res = res

	return ib
}

func (ib insightBuilder) inferProdTargetsOp() string {
	if ib.params.UseUncompressedModel {
		return targetsEmbeddedOp
	}
	return targetsEmbeddedOpCompressed
}

// Here we replicate the logic from https://github.com/kiteco/kiteco/blob/master/kite-python/kite_ml/kite/graph_encoder/encoder.py#L172
// where we sort the edge keys first then iterate through the edges
// This way we'll be able to match the edge values to the actual edges
func (ib insightBuilder) matchEdgeValues(evs []float32) map[string]float64 {
	var edgeTypeKeys []string
	for k := range ib.params.Edges {
		edgeTypeKeys = append(edgeTypeKeys, k)
	}

	sort.Strings(edgeTypeKeys)

	var edgeIDSeq []string
	for _, k := range edgeTypeKeys {
		for _, p := range ib.params.Edges[k] {
			// TODO: why dont we care about direction?
			t, _ := typeFromEdgeKey(k)
			edgeIDSeq = append(edgeIDSeq, edgeIDStr(t, p[0], p[1]))
		}
	}

	edgeIDToValue := make(map[string]float64)
	for i := range edgeIDSeq {
		edgeIDToValue[edgeIDSeq[i]] = float64(evs[i])
	}

	return edgeIDToValue
}

func (ib insightBuilder) initNodeStates() [][]float32 {
	return ib.res[initNodeStatesOp].([][]float32)
}

func (ib insightBuilder) nodeStatesAfter(pStep int) [][]float32 {
	op := ib.nodeStatesAfterOps[pStep]
	return ib.res[op].([][]float32)
}

func (ib insightBuilder) rawEdgeValues(pStep int) []float32 {
	messagesOp := ib.messagesOps[pStep]
	attentionsOp := ib.attentionOps[pStep]

	messages := ib.res[messagesOp].([][]float32)
	messageMags := magnitude(messages)
	attentions := ib.res[attentionsOp].([]float32)

	if len(messageMags) != len(attentions) || len(attentions) != len(ib.params.Graph.Edges) {
		panic(fmt.Sprintf("should have same size of edges, weights and attentions"))
	}

	edgeValues := make([]float32, 0, len(attentions))
	for i := range messageMags {
		edgeValues = append(edgeValues, messageMags[i]*attentions[i])
	}

	return edgeValues
}

func (ib insightBuilder) finalEdgeValues() map[string]EdgeValue {
	rawValues := ib.rawEdgeValues(ib.params.MaxHops - 1)

	raw := ib.matchEdgeValues(rawValues)
	normalized := normalize(raw)
	final := make(map[string]EdgeValue)
	for k := range raw {
		final[k] = EdgeValue{
			Raw:        raw[k],
			Normalized: normalized[k],
		}
	}

	return final
}

// Extracts  the sub-graph representing the computational graph within certain steps of hops from the provided node
// Each node is labeled as original-label-STATE%d to represent different step of the propagate
func (ib insightBuilder) subGraph(sites []*SavedNode) (*SavedGraph, NodeLabels, map[string]EdgeValue) {
	var newNodes []*SavedNode
	labels := make(NodeLabels)
	addNode := func(orig *SavedNode, level int, state []float32) *SavedNode {
		new := &SavedNode{
			Node:  orig.Node.deepCopy(),
			Level: level,
		}
		new.Node.ID = NodeID(len(newNodes))

		label := fmt.Sprintf("STATE_%d", level)
		if old := ib.params.Labels[orig.Node.ID]; old != "" {
			label = fmt.Sprintf("%s::%s", old, label)
		}
		labels[new.Node.ID] = label

		new.Level = level
		new.Hover = fmt.Sprintf("Magnitude: %.4f", l2Norm(state))
		newNodes = append(newNodes, new)
		return new
	}

	nodeStates := ib.nodeStatesAfter(ib.params.MaxHops - 1)

	currentOldToNew := make(map[*SavedNode]*SavedNode)
	for _, site := range sites {
		currentOldToNew[site] = addNode(site, ib.params.MaxHops, nodeStates[site.Node.ID])
	}

	var newEdges []*SavedEdge
	edgeValues := make(map[string]EdgeValue)

	// Trace backwards
	for i := ib.params.MaxHops - 1; i >= 0; i-- {
		matched := ib.matchEdgeValues(ib.rawEdgeValues(i))

		var nodeStates [][]float32
		if i-1 < 0 {
			nodeStates = ib.initNodeStates()
		} else {
			nodeStates = ib.nodeStatesAfter(i - 1)
		}

		newOldToNew := make(map[*SavedNode]*SavedNode)
		newEdgeValues := make(map[string]float64)
		for old := range currentOldToNew {
			// Look for edges in the original graph that pass information to the current node
			// and create corresponding flow edges in the computational graph
			for _, e := range ib.params.Graph.Edges {
				if e.To != old {
					continue
				}

				// Add a new node to the current layer if needed
				if _, ok := newOldToNew[e.From]; !ok {
					newOldToNew[e.From] = addNode(e.From, i, nodeStates[e.From.Node.ID])
				}

				newEdge := &SavedEdge{
					From:    newOldToNew[e.From],
					To:      currentOldToNew[e.To],
					Type:    e.Type,
					Forward: e.Forward,
				}

				newEdges = append(newEdges, newEdge)

				newEdgeValues[EdgeToIDStr(newEdge)] = matched[EdgeToIDStr(e)]
			}
		}

		// Normalize the values separately for each step
		for k, v := range normalize(newEdgeValues) {
			edgeValues[k] = EdgeValue{
				Raw:        newEdgeValues[k],
				Normalized: v,
			}
		}
		currentOldToNew = newOldToNew
	}

	return &SavedGraph{
		Nodes: newNodes,
		Edges: newEdges,
	}, labels, edgeValues
}

func innerProduct(a []float32, b []float32) float32 {
	var p float32
	for i := range a {
		p += a[i] * b[i]
	}
	return p
}

func l2Norm(a []float32) float32 {
	return float32(math.Sqrt(float64(innerProduct(a, a))))
}

func (ib insightBuilder) extractProductionContext(targets []string) ContextWeights {
	ci := make(map[string][]float32)
	for i := range targets {
		target := ib.res[ib.inferProdTargetsOp()].([][]float32)[i]
		ci["scope_weight"] = append(ci["scope_weight"], ib.res[scopeWeightOp].([][]float32)[i][0])
		ci["site_weight"] = append(ci["site_weight"], ib.res[siteWeightOp].([][]float32)[i][0])
		ci["token_weight"] = append(ci["token_weight"], ib.res[tokenWeightOp].([][]float32)[i][0])
		ci["scope_product"] = append(ci["scope_product"], innerProduct(target, ib.res[scopeStateOp].([][]float32)[i]))
		ci["site_product"] = append(ci["site_product"], innerProduct(target, ib.res[siteStateOp].([][]float32)[i]))
		ci["token_product"] = append(ci["token_product"], innerProduct(target, ib.res[tokenStateOp].([][]float32)[i]))
		ci["logit"] = append(ci["logit"], ib.res[inferProdLogitsOp].([]float32)[i])
	}

	cw := ContextWeights{
		XLabels: targets,
	}

	// Iterate the map in fixed order.
	ys := []string{"scope_weight", "site_weight", "token_weight", "scope_product", "site_product", "token_product", "logit"}

	for _, y := range ys {
		cw.YLabels = append(cw.YLabels, y)
		cw.Content = append(cw.Content, ci[y])
	}

	return cw
}

func (ib insightBuilder) extractNameContext(usages []*SavedNode) ContextWeights {
	cw := ContextWeights{
		YLabels: []string{"logit"},
		Content: [][]float32{ib.res[inferNameLogitsOp].([]float32)},
	}

	for _, u := range usages {
		cw.XLabels = append(cw.XLabels, u.Node.Attrs.Literal)
	}

	return cw
}

func (ib insightBuilder) addGraphInsights(it insightsType, sites []*SavedNode) (*SavedGraph, NodeLabels, map[string]EdgeValue) {
	switch it {
	case finalPropagationStep:
		edgeValues := ib.finalEdgeValues()
		initStates := ib.initNodeStates()
		finalStates := ib.nodeStatesAfter(ib.params.MaxHops - 1)

		for _, n := range ib.params.Graph.Nodes {
			initMag := l2Norm(initStates[n.Node.ID])
			finalMag := l2Norm(finalStates[n.Node.ID])

			n.Hover = fmt.Sprintf("MagnitudeInit: %.4f -> MagnitudeFinal: %.4f", initMag, finalMag)
		}

		return ib.params.Graph, ib.params.Labels, edgeValues
	case allPropagationSteps:
		return ib.subGraph(sites)
	default:
		panic(fmt.Sprintf("insight type not supported %v", it))
	}
}

func (ib insightBuilder) GraphInsights(it insightsType, site *SavedNode) SavedBundle {
	graph, labels, edgeValues := ib.addGraphInsights(it, []*SavedNode{site})
	return SavedBundle{
		Graph:      graph,
		NodeLabels: labels,
		EdgeValues: edgeValues,
	}
}

func (ib insightBuilder) InferProdInsights(it insightsType, targets []string, site *SavedNode) SavedBundle {
	graph, labels, edgeValues := ib.addGraphInsights(it, []*SavedNode{site})

	weights := ib.extractProductionContext(targets)

	return SavedBundle{
		Graph:      graph,
		NodeLabels: labels,
		EdgeValues: edgeValues,
		Weights:    weights,
	}
}

func (ib insightBuilder) InferNameInsights(it insightsType, usages []*SavedNode, site *SavedNode) SavedBundle {
	graph, labels, edgeValues := ib.addGraphInsights(it, append([]*SavedNode{site}, usages...))

	weights := ib.extractNameContext(usages)

	return SavedBundle{
		Graph:      graph,
		NodeLabels: labels,
		EdgeValues: edgeValues,
		Weights:    weights,
	}
}
