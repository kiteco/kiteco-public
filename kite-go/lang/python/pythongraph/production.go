package pythongraph

import (
	"fmt"
	"path"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
)

// InferProductionSample ...
type InferProductionSample struct {
	ContextGraph   GraphFeed
	ExpansionGraph ExpansionGraphTrainFeed
	Production     ProductionModelFeed
}

// ProductionModelFeed for a choose production model
type ProductionModelFeed struct {
	PredictionNodes []int32                        `json:"prediction_nodes"`
	Labels          []int                          `json:"labels"`
	DecoderTargets  traindata.SegmentedIndicesFeed `json:"decoder_targets"`
	Corrupted       traindata.SegmentedIndicesFeed `json:"corrupted"`
	ScopeEncoder    traindata.SegmentedIndicesFeed `json:"scope_encoder"`
	ContextTokens   traindata.SegmentedIndicesFeed `json:"context_tokens"`
}

func (p ProductionModelFeed) append(pp ProductionModelFeed, sampleID int, nodeOffset NodeID) ProductionModelFeed {
	if len(pp.PredictionNodes) != 1 || len(pp.Labels) != 1 {
		panic(fmt.Sprintf("bad times, pp has more than one entry"))
	}

	p.PredictionNodes = append(p.PredictionNodes, pp.PredictionNodes[0]+int32(nodeOffset))

	labelOffset := len(p.DecoderTargets.Indices)
	p.Labels = append(p.Labels, pp.Labels[0]+labelOffset)

	p.DecoderTargets = p.DecoderTargets.Append(pp.DecoderTargets, int32(sampleID), 0)

	p.Corrupted = p.Corrupted.Append(pp.Corrupted, int32(sampleID), int32(labelOffset))

	p.ScopeEncoder = p.ScopeEncoder.Append(pp.ScopeEncoder, int32(sampleID), int32(nodeOffset))

	p.ContextTokens = p.ContextTokens.Append(pp.ContextTokens, int32(sampleID), int32(nodeOffset))

	return p
}

// FeedDict ...
func (p ProductionModelFeed) FeedDict(nameScope string) map[string]interface{} {
	fd := scopeEncoderFeedDict(path.Join(nameScope, "scope_encoder"), p.ScopeEncoder)

	name := func(n string) string {
		return path.Join(nameScope, "placeholders", n)
	}

	fd[name("prediction_nodes")] = p.PredictionNodes
	for k, v := range p.DecoderTargets.FeedDict(name("decoder_targets")) {
		fd[k] = v
	}

	for k, v := range p.ContextTokens.FeedDict(name("context_tokens")) {
		fd[k] = v
	}

	return fd
}

func newEmptyProductionModelFeed() ProductionModelFeed {
	return ProductionModelFeed{
		PredictionNodes: []int32{},
		Labels:          []int{},
		DecoderTargets:  traindata.NewSegmentedIndicesFeed(),
		Corrupted:       traindata.NewSegmentedIndicesFeed(),
		ScopeEncoder:    traindata.NewSegmentedIndicesFeed(),
		ContextTokens:   traindata.NewSegmentedIndicesFeed(),
	}
}

func scopeEncoderFeedDict(nameScope string, f traindata.SegmentedIndicesFeed) map[string]interface{} {
	return f.FeedDict(path.Join(nameScope, "placeholders", "variable_node_ids"))
}
