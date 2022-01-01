package decisiontree

// A Node represents a splitting decision of the form "x[FeatureIndex] < Threshold ?" in a decision tree
type Node struct {
	// FeatureIndex indicates which feature is used in this splitting decision
	FeatureIndex int `json:"feature_index"`
	// Threshold indicates the cutoff value between the left and right subtrees
	Threshold float64 `json:"threshold"`
	// LeftChild is the index of the node representing the left subtree
	LeftChild int `json:"left_child"`
	// LeftIsLeaf indicates whether the left subtree is a leaf node
	LeftIsLeaf bool `json:"left_is_leaf"`
	// RightChild is the index of the node representing the right subtree
	RightChild int `json:"right_child"`
	// RightIsLeaf indicates wherther the right subtree is a leaf node
	RightIsLeaf bool `json:"right_is_leaf"`
}

// A DecisionTree is a mapping from a feature space to real numbers implemented with a decision tree
type DecisionTree struct {
	// Nodes is a flat list of all nodes in the tree
	Nodes []Node `json:"nodes"`
	// Outputs is an array containing the outputs for each bin
	Outputs []float64 `json:"outputs"`
	// FeatureSize is the length of feature vectors processed by this tree
	FeatureSize int `json:"feature_size"`
	// Depth is the maximum depth of any leaf in the tree
	Depth int `json:"depth"`
}

// Bin drops a feature vector down a decision tree and returns the index of the bin that it ends up in
func (t *DecisionTree) Bin(x []float64) int {
	if len(x) != t.FeatureSize {
		panic("feature vector had incorrect length")
	}
	if t.Nodes == nil {
		panic("tree not initialized")
	}
	cur := t.Nodes[0]
	for i := 0; i < t.Depth; i++ {
		if x[cur.FeatureIndex] < cur.Threshold {
			if cur.LeftIsLeaf {
				return cur.LeftChild
			}
			cur = t.Nodes[cur.LeftChild]
		} else {
			if cur.RightIsLeaf {
				return cur.RightChild
			}
			cur = t.Nodes[cur.RightChild]
		}
	}
	panic("tree traversal did not terminate")
}

// Evaluate drops a feature vector down a decision tree and returns the output associated with the bin
// it ends up in.
func (t *DecisionTree) Evaluate(x []float64) float64 {
	return t.Outputs[t.Bin(x)]
}

// An Ensemble outputs the sum of several decision trees
type Ensemble struct {
	Trees []DecisionTree `json:"trees"`
}

// Evaluate computes the sum of the outputs of the component decision trees
func (e *Ensemble) Evaluate(x []float64) float64 {
	var sum float64
	for _, t := range e.Trees {
		sum += t.Evaluate(x)
	}
	return sum
}

// Print satisfies the ranking.Scorer interface
func (e *Ensemble) Print() {
}
