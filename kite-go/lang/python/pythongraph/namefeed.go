package pythongraph

import (
	"fmt"
	"path"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
)

// NameEncoderFeed for a name encoder model
type NameEncoderFeed struct {
	// Usages are the node ids for the usage nodes associated with each name,
	// samplesIDs[i] = j implies variable i is part of sample j
	Usages traindata.SegmentedIndicesFeed `json:"usages"`

	// Names of the names, only for training
	Names []string `json:"names"`
	// Types of the names, only for training
	Types []string `json:"types"`
}

func newNameEncoderFeedFromNameSite(astNodes astToNode, nameSite nameSite) NameEncoderFeed {
	numCands := len(nameSite.Candidates)
	nameFeed := NameEncoderFeed{
		Types:  make([]string, 0, numCands),
		Names:  make([]string, 0, numCands),
		Usages: traindata.NewSegmentedIndicesFeed(),
	}

	assertTrue(len(nameSite.Candidates) == len(nameSite.Scope), "bad times")

	for _, cand := range nameSite.Candidates {
		nameFeed.Usages.Indices = append(nameFeed.Usages.Indices, int32(cand.Usage.ID))
		nameFeed.Usages.SampleIDs = append(nameFeed.Usages.SampleIDs, 0)

		origin := astNodes[cand.Variable.Origin]

		ts := getNodeTypes(origin, maxNumTypesPerNode)
		nameFeed.Types = append(nameFeed.Types, asciiOnly(strings.Join(ts, ":")))

		subTokens := getNodeSubtokens(origin, maxNumSubtokensPerNode)
		nameFeed.Names = append(nameFeed.Names, asciiOnly(strings.Join(subTokens, ":")))
	}

	return nameFeed
}

// FeedDict for the encoder
func (n NameEncoderFeed) FeedDict(prefix string) map[string]interface{} {
	return n.Usages.FeedDict(path.Join(prefix, "placeholders", "usages"))
}

// NumVariables in the feed
func (n NameEncoderFeed) NumVariables() int {
	return len(n.Usages.Indices)
}

func (n NameEncoderFeed) append(other NameEncoderFeed, sampleID int, nodeOffset NodeID) NameEncoderFeed {
	n.Usages = n.Usages.Append(other.Usages, int32(sampleID), int32(nodeOffset))
	for i := range other.Names {
		n.Names = append(n.Names, other.Names[i])
		n.Types = append(n.Types, other.Types[i])
	}

	return n
}

// InferNameSample ...
type InferNameSample struct {
	ContextGraph   GraphFeed
	ExpansionGraph ExpansionGraphTrainFeed
	Name           NameModelFeed
}

// NameModelFeed for a choose name model
type NameModelFeed struct {
	PredictionNodes []int32                        `json:"prediction_nodes"`
	Corrupted       traindata.SegmentedIndicesFeed `json:"corrupted"`
	Labels          []VariableID                   `json:"labels"`
	Types           traindata.SegmentedIndicesFeed `json:"types"`
	Subtokens       traindata.SegmentedIndicesFeed `json:"subtokens"`
	Names           NameEncoderFeed                `json:"names"`
}

func newNameModelFeed() NameModelFeed {
	return NameModelFeed{
		PredictionNodes: []int32{},
		Corrupted:       traindata.NewSegmentedIndicesFeed(),
		Labels:          []VariableID{},
		Types:           traindata.NewSegmentedIndicesFeed(),
		Subtokens:       traindata.NewSegmentedIndicesFeed(),
		Names:           newNameEncoderFeedFromNameSite(nil, nameSite{}),
	}
}

// FeedDict for the name model
func (n NameModelFeed) FeedDict(prefix string) map[string]interface{} {
	name := func(n string) string {
		return path.Join(prefix, "placeholders", n)
	}

	fd := map[string]interface{}{
		name("prediction_nodes"): n.PredictionNodes,
	}

	for k, v := range n.Names.FeedDict(path.Join(prefix, "name_encoder")) {
		fd[k] = v
	}

	for k, v := range n.Types.FeedDict(name("types")) {
		fd[k] = v
	}

	for k, v := range n.Subtokens.FeedDict(name("subtokens")) {
		fd[k] = v
	}

	return fd
}

func (n NameModelFeed) append(other NameModelFeed, sampleID int, nodeOffset NodeID, varOffset VariableID) NameModelFeed {
	// update prediction nodes
	if len(other.PredictionNodes) != 1 {
		panic(fmt.Sprintf("expected one prediction node, got %d", len(other.PredictionNodes)))
	}
	n.PredictionNodes = append(n.PredictionNodes, int32(nodeOffset)+other.PredictionNodes[0])

	// update corrupted
	n.Corrupted = n.Corrupted.Append(other.Corrupted, int32(sampleID), int32(varOffset))

	// update label
	if len(other.Labels) != 1 {
		panic(fmt.Sprintf("expected one label, got %d", len(other.Labels)))
	}
	n.Labels = append(n.Labels, varOffset+other.Labels[0])

	n.Types = n.Types.Append(other.Types, int32(sampleID), 0)
	n.Subtokens = n.Subtokens.Append(other.Subtokens, int32(sampleID), 0)

	n.Names = n.Names.append(other.Names, sampleID, nodeOffset)

	return n
}
